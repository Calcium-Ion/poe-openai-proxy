package router

import (
	"encoding/json"
	"fmt"
	poe_api "github.com/Calcium-Ion/poe-api-go"
	"github.com/pkoukk/tiktoken-go"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/juzeon/poe-openai-proxy/conf"
	"github.com/juzeon/poe-openai-proxy/poe"
	"github.com/juzeon/poe-openai-proxy/util"
)

//func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
//	msg := tgbotapi.NewMessage(chatID, text)
//
//	_, err := bot.Send(msg)
//	if err != nil {
//		log.Panic(err)
//	}
//}

func Setup(engine *gin.Engine) {

	getModels := func(c *gin.Context) {
		SetCORS(c)
		c.JSON(http.StatusOK, conf.Models)
	}

	engine.GET("/models", getModels)
	engine.GET("/v1/models", getModels)

	postCompletions := func(c *gin.Context) {
		SetCORS(c)
		var req poe.CompletionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, "bad request")
			return
		}
		for _, msg := range req.Messages {
			if msg.Role != "system" && msg.Role != "user" && msg.Role != "assistant" && msg.Role != "function" {
				c.JSON(400, "role of message validation failed: "+msg.Role)
				return
			}
		}
		client, err := poe.GetClient()
		defer client.Release()
		if err != nil {
			c.JSON(500, err)
			return
		}
		defer client.Release()
		if req.Stream {
			util.Logger.Info("stream using client: " + client.Token)
			Stream(c, req, client)
		} else {
			util.Logger.Info("ask using client: " + client.Token)
			Ask(c, req, client)
		}
	}

	engine.POST("/chat/completions", postCompletions)
	engine.POST("/v1/chat/completions", postCompletions)

	// OPTIONS /v1/chat/completions

	optionsCompletions := func(c *gin.Context) {
		SetCORS(c)
		c.JSON(200, "")
	}

	engine.OPTIONS("/chat/completions", optionsCompletions)
	engine.OPTIONS("/v1/chat/completions", optionsCompletions)
}
func Stream(c *gin.Context, req poe.CompletionRequest, client *poe.Client) {
	defer func() {
		if err := recover(); err != nil {
			util.Logger.Error(err, client.Token)
			openAIError := poe.OpenAIError{
				Type:    "openai_api_error",
				Message: "The server had an error while processing your request",
				Code:    "do_request_failed",
			}

			c.JSON(500, gin.H{
				"error": openAIError,
			})
		}
	}()
	resp, err := client.Stream(req.Messages, req.Model)
	if err != nil {
		panic(err)
		return
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	w := c.Writer
	flusher, _ := w.(http.Flusher)
	timeout := time.Duration(conf.Conf.Timeout) * time.Second
	ticker := time.NewTimer(timeout)

	defer ticker.Stop()

	conversationID := "chatcmpl-" + util.RandStringRunes(29)

	createSSEResponse := func(content string, haveRole bool) {
		done := content == "[DONE]"
		var finishReason *string
		delta := map[string]string{}
		if done {
			_str := "stop"
			finishReason = &_str
		} else if haveRole {
			delta["role"] = "assistant"
		} else {
			delta["content"] = content
		}
		data := poe.CompletionSSEResponse{
			Choices: []poe.SSEChoice{{
				Index:        0,
				Delta:        delta,
				FinishReason: finishReason,
			}},
			Created: time.Now().Unix(),
			Id:      conversationID,
			Model:   req.Model,
			Object:  "chat.completion.chunk",
		}
		dataV, _ := json.Marshal(&data)
		_, err := io.WriteString(w, "data: "+string(dataV)+"\n\n")
		if err != nil {
			util.Logger.Error(err)
		}
		flusher.Flush()
		if done {
			_, err := io.WriteString(w, "data: [DONE]\n\n")
			if err != nil {
				util.Logger.Error(err)
			}
			flusher.Flush()
		}
	}
	createSSEResponse("", true)

	respCount := 0
	for m := range poe_api.GetTextStream(resp) {
		createSSEResponse(m, false)
		respCount++
		log.Printf("stream: %s len %d\n", m, len(m))
	}
	if respCount == 0 {
		//createSSEResponse("你的提问被屏蔽，解决办法：不要在问题中进行对话（角色）模拟", false)
		c.Writer.Header().Set("Content-Type", "application/json")
		openAIError := poe.OpenAIError{
			Type:    "openai_api_error",
			Message: "你的提问被屏蔽，解决办法：不要在问题中进行对话（角色）模拟",
			Code:    "do_request_failed",
		}

		c.JSON(500, gin.H{
			"error": openAIError,
		})
		return
	}
	util.Logger.Info("stream count: ", respCount)
	createSSEResponse("[DONE]", false)

}
func Ask(c *gin.Context, req poe.CompletionRequest, client *poe.Client) {
	defer func() {
		if err := recover(); err != nil {
			util.Logger.Error(err)
			openAIError := poe.OpenAIError{
				Type:    "openai_api_error",
				Message: "The server had an error while processing your request",
				Code:    "do_request_failed",
			}
			c.JSON(500, gin.H{
				"error": openAIError,
			})
		}
	}()
	message, promptTokens, err := client.Ask(req.Messages, req.Model)
	if err != nil {
		panic(err)
		return
	}
	tkm, err := tiktoken.EncodingForModel("gpt-4")
	var token = make([]int, 0)
	if err != nil {
		err = fmt.Errorf("getEncoding: %v", err)
	} else {
		token = tkm.Encode(message.Content, nil, nil)
	}
	c.JSON(200, poe.CompletionResponse{
		ID:      "chatcmpl-" + util.RandStringRunes(29),
		Object:  "chat.completion",
		Created: int(time.Now().Unix()),
		Choices: []poe.Choice{{
			Index:        0,
			Message:      *message,
			FinishReason: "stop",
		}},
		Usage: poe.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: len(token),
			TotalTokens:      promptTokens + len(token),
		},
	})
}

func SetCORS(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	c.Writer.Header().Set("Access-Control-Max-Age", "86400")
	c.Writer.Header().Set("Content-Type", "application/json")
}
