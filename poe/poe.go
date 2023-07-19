package poe

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
	"github.com/juzeon/poe-openai-proxy/conf"
	"github.com/juzeon/poe-openai-proxy/util"
	"github.com/pkoukk/tiktoken-go"
)

var httpClient *resty.Client
var clients []*Client
var clientIx = 0
var clientLock = &sync.Mutex{}

func Setup() {
	httpClient = resty.New().SetBaseURL(conf.Conf.Gateway)
	for _, token := range conf.Conf.Tokens {
		client, err := NewClient(token)
		if err != nil {
			util.Logger.Error(err)
			continue
		}
		clients = append(clients, client)
	}
}

type Client struct {
	Token string
	Usage []time.Time
	Lock  bool
}

func NewClient(token string) (*Client, error) {
	util.Logger.Info("registering client: " + token)
	resp, err := httpClient.R().SetFormData(map[string]string{
		"token": token,
	}).Post("/add_token")
	if err != nil {
		return nil, errors.New("registering client error: " + err.Error())
	}
	util.Logger.Info("registering client: " + resp.String())
	return &Client{Token: token, Usage: nil, Lock: false}, nil
}
func (c *Client) getContentToSend(messages []Message) string {
	leadingMap := map[string]string{
		"system":    "Instructions",
		"user":      "User",
		"assistant": "Assistant",
		"function":  "Information",
	}
	content := ""
	var simulateRoles bool
	switch conf.Conf.SimulateRoles {
	case 0:
		simulateRoles = false
	case 1:
		simulateRoles = true
	case 2:
		if len(messages) == 1 && messages[0].Role == "user" ||
			len(messages) == 1 && messages[0].Role == "system" ||
			len(messages) == 2 && messages[0].Role == "system" && messages[1].Role == "user" {
			simulateRoles = false
		} else {
			simulateRoles = true
		}
	}
	for _, message := range messages {
		if simulateRoles {
			content += "||>" + leadingMap[message.Role] + ":\n" + message.Content + "\n"
		} else {
			content += message.Content + "\n"
		}
	}
	if simulateRoles {
		content += "||>Assistant:\n"
	}
	util.Logger.Debug("Generated content to send: " + content)
	return content
}
func (c *Client) Stream(messages []Message, model string) (<-chan string, error) {
	channel := make(chan string, 1024)
	content := c.getContentToSend(messages)

	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		err = fmt.Errorf("getEncoding: %v", err)
		return nil, err
	}

	token := tkm.Encode(content, nil, nil)

	util.Logger.Info("Token len ", len(token))

	if model == "gpt-4" && len(token) > 2200 {
		model = "gpt-4-32k"
		util.Logger.Info("Token len ", len(token), " out of limit, use gpt-4-32k")
	}

	conn, _, err := websocket.DefaultDialer.Dial(conf.Conf.GetGatewayWsURL()+"/stream", nil)
	if err != nil {
		return nil, err
	}

	bot, ok := conf.Conf.Bot[model]
	if !ok {
		bot = "capybara"
	}
	util.Logger.Info("Stream using bot", bot)

	err = conn.WriteMessage(websocket.TextMessage, []byte(c.Token))
	if err != nil {
		return nil, err
	}
	err = conn.WriteMessage(websocket.TextMessage, []byte(bot))
	if err != nil {
		return nil, err
	}
	err = conn.WriteMessage(websocket.TextMessage, []byte(content))
	if err != nil {
		return nil, err
	}
	go func(conn *websocket.Conn, channel chan string) {
		defer close(channel)
		defer conn.Close()
		for {
			_, v, err := conn.ReadMessage()
			if strings.HasPrefix(string(v), "An exception of type") {
				util.Logger.Error(string(v))
				err = errors.New("connect to openai error code 2")
			}

			if strings.Contains(string(v), "RSV1 set, FIN") {
				util.Logger.Error(string(v))
				err = errors.New("connect to openai error code 3")
			}

			channel <- string(v)
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					util.Logger.Error(err)
					channel <- "\n\n[ERROR] " + err.Error()
				}
				channel <- "[DONE]"
				break
			}
		}
	}(conn, channel)
	return channel, nil
}
func (c *Client) Ask(messages []Message, model string) (*Message, error) {
	content := c.getContentToSend(messages)

	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		err = fmt.Errorf("getEncoding: %v", err)
		return nil, err
	}

	token := tkm.Encode(content, nil, nil)

	util.Logger.Info("Token len ", len(token))

	if model == "gpt-4" && len(token) > 2200 {
		model = "gpt-4-32k"
		util.Logger.Info("Token len ", len(token), " out of limit, use gpt-4-32k")
	}

	bot, ok := conf.Conf.Bot[model]
	if !ok {
		bot = "capybara"
	}
	util.Logger.Info("Ask using bot", bot)

	resp, err := httpClient.R().SetFormData(map[string]string{
		"token":   c.Token,
		"bot":     bot,
		"content": content,
	}).Post("/ask")
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(resp.String(), "An exception of type") {
		util.Logger.Error(resp.String())
		err = errors.New("connect to openai error code 2")
	}

	if strings.Contains(resp.String(), "RSV1 set, FIN") {
		util.Logger.Error(resp.String())
		err = errors.New("connect to openai error code 3")
	}
	return &Message{
		Role:    "assistant",
		Content: resp.String(),
		Name:    "",
	}, nil
}
func (c *Client) Release() {
	clientLock.Lock()
	defer clientLock.Unlock()
	c.Lock = false
}

func GetClient() (*Client, error) {
	clientLock.Lock()
	defer clientLock.Unlock()
	if len(clients) == 0 {
		return nil, errors.New("no client is available")
	}
	for i := 0; i < len(clients); i++ {
		client := clients[clientIx%len(clients)]
		clientIx++
		if client.Lock {
			continue
		}
		if len(client.Usage) > 0 {
			lastUsage := client.Usage[len(client.Usage)-1]
			if time.Since(lastUsage) < time.Duration(conf.Conf.CoolDown)*time.Second {
				continue
			}
		}
		if len(client.Usage) < conf.Conf.RateLimit {
			client.Usage = append(client.Usage, time.Now())
			client.Lock = true
			return client, nil
		} else {
			usage := client.Usage[len(client.Usage)-conf.Conf.RateLimit]
			if time.Since(usage) <= 1*time.Minute {
				continue
			}
			client.Usage = append(client.Usage, time.Now())
			client.Lock = true
			return client, nil
		}
	}
	return nil, errors.New("no available client")
}
