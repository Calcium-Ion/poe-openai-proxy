package poe

import (
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/pkoukk/tiktoken-go"

	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/juzeon/poe-openai-proxy/conf"
	"github.com/juzeon/poe-openai-proxy/util"
	"github.com/lwydyby/poe-api"
)

var httpClient *resty.Client
var clients []*Client
var clientIx = 0
var clientLock = &sync.Mutex{}

func Setup() {
	//httpClient = resty.New().SetBaseURL(conf.Conf.Gateway)
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
	Token     string
	Usage     []time.Time
	Lock      bool
	PoeClient *poe_api.Client
}

func GetRealModel(model string, token []int) string {
	if model == "gpt-4" && len(token) > 2200 {
		model = "gpt-4-32k"
		util.Logger.Info("Token len ", len(token), " out of limit, use gpt-4-32k")
	}
	return model
}

func GetBotName(model string) string {
	if strings.HasPrefix(model, "gpt-4") {
		if strings.HasPrefix(model, "gpt-4-32k") {
			return "GPT-4-32k"
		}
		return "GPT-4"
	}
	return "GPT-4"
}

func NewClient(token string) (*Client, error) {
	util.Logger.Info("registering client: " + token)
	var uri *url.URL
	if conf.Conf.Proxy == "" {
		uri = nil
	} else {
		url, err := url.Parse(conf.Conf.Proxy)
		if err != nil {
			return nil, err
		}
		uri = url
	}
	// defer recover
	defer func() {
		if r := recover(); r != nil {
			//fmt.Println("Recovered in f", r)
			util.Logger.Error(r)
		}
	}()
	c := poe_api.NewClient(token, uri)
	//resp, err := c.SendMessage("GPT-4", "Test1", false, 10*time.Second)
	//if err != nil {
	//	return nil, err
	//}
	util.Logger.Info("ok")
	return &Client{Token: token, Usage: nil, Lock: false, PoeClient: c}, nil
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
		if len(messages) == 1 && messages[0].Role == "u*-=ser" ||
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

func (c *Client) Stream(messages []Message, model string) (<-chan map[string]interface{}, error) {

	content := c.getContentToSend(messages)

	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		err = fmt.Errorf("getEncoding: %v", err)
		return nil, err
	}

	token := tkm.Encode(content, nil, nil)

	util.Logger.Info("Token len ", len(token))

	if model == "gpt-4" && len(token) > 8000 {
		return nil, errors.New("Token len " + strconv.Itoa(len(token)) + " out of limit, max token len is 8000")
	}

	model = GetRealModel(model, token)

	util.Logger.Info("Stream using bot", GetBotName(model))

	//defer func() {
	//	if e := recover(); e != nil {
	//		//util.Logger.Error(e)
	//		panic(e)
	//	}
	//}()

	resp, err := c.PoeClient.SendMessage(GetBotName(model), content, false, time.Duration(conf.Conf.ApiTimeout)*time.Second)
	if err != nil {
		return nil, err
	}
	return resp, err
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

	if model == "gpt-4" && len(token) > 8000 {
		return nil, errors.New("Token len " + strconv.Itoa(len(token)) + " out of limit, max token len is 8000")
	}

	model = GetRealModel(model, token)

	util.Logger.Info("Ask using bot", GetBotName(model))

	resp, err := c.PoeClient.SendMessage(GetBotName(model), content, true, time.Duration(conf.Conf.ApiTimeout)*time.Second)
	if err != nil {
		return nil, err
	}
	return &Message{
		Role:    "assistant",
		Content: poe_api.GetFinalResponse(resp),
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

func CheckClient() {
	defer func() {
		if r := recover(); r != nil {
			//fmt.Println("Recovered in f", r)
			util.Logger.Error(r)
		}
	}()
	util.Logger.Info("开始定时重启")
	for i := range clients {
		client := clients[i]
		needUpdate := false
		if len(client.Usage) > 0 {
			lastUsage := client.Usage[len(client.Usage)-1]
			if time.Since(lastUsage) > time.Duration(conf.Conf.AutoReload)*time.Minute {
				needUpdate = true
			}
			util.Logger.Info("Client:", client.Token, " last usage:", lastUsage, " need update:", needUpdate)
		} else {
			needUpdate = true
			util.Logger.Info("Client:", client.Token, " never usage", " need update:", needUpdate)
		}

		if needUpdate {
			clientNew, err := NewClient(client.Token)
			if err != nil {
				util.Logger.Error(err)
				continue
			}
			clientNew.Usage = make([]time.Time, 0)
			util.Logger.Info("Update client:", client.Token)
		}
	}
}
