package sensetime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/model"
	"io"
	"net/http"
	"strings"
	"time"
)

func GetToken(ak string, sk string) string {
	payload := jwt.MapClaims{
		"iss": ak,
		"exp": time.Now().Add(1800 * time.Second).Unix(),
		"nbf": time.Now().Add(-5 * time.Second).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	signedToken, err := token.SignedString([]byte(sk))
	if err != nil {
		fmt.Println("Error encoding JWT token:", err)
		return ""
	}
	return signedToken
}

func ConvertRequest(request model.GeneralOpenAIRequest) *Request {
	messages := make([]Message, 0, len(request.Messages))
	for _, message := range request.Messages {
		messages = append(messages, Message{
			Role:    message.Role,
			Content: message.StringContent(),
		})
	}
	return &Request{
		Model:             request.Model,
		Messages:          messages,
		Temperature:       request.Temperature,
		TopP:              request.TopP,
		MaxNewTokens:      request.MaxTokens,
		RepetitionPenalty: request.PresencePenalty,
		Stream:            request.Stream,
	}
}

func responseSensetimeOpenAI(response *Response) *openai.TextResponse {
	fullTextResponse := openai.TextResponse{
		Id:      response.Data.ID,
		Object:  "chat.completion",
		Created: helper.GetTimestamp(),
		Choices: make([]openai.TextResponseChoice, 0, len(response.Data.Choices)),
		Usage:   response.Data.Usage,
	}
	for i, choice := range response.Data.Choices {
		openaiChoice := openai.TextResponseChoice{
			Index: i,
			Message: model.Message{
				Role:    choice.Role,
				Content: strings.Trim(choice.Message, "\""),
			},
			FinishReason: "",
		}
		if i == len(response.Data.Choices)-1 {
			openaiChoice.FinishReason = "stop"
		}
		fullTextResponse.Choices = append(fullTextResponse.Choices, openaiChoice)
	}
	return &fullTextResponse
}

func streamResponseSensetimeOpenAI(sensetimeResponseStr string) *openai.ChatCompletionsStreamResponse {
	var choice openai.ChatCompletionsStreamResponseChoice
	var sensetimeResponse Response
	err := json.Unmarshal([]byte(sensetimeResponseStr), &sensetimeResponse)
	if err != nil {
		logger.SysError("error unmarshalling stream response: " + err.Error())
		return nil
	}

	choice.Delta.Content = sensetimeResponse.Data.Choices[0].Delta
	choice.Delta.Role = sensetimeResponse.Data.Choices[0].Role
	response := openai.ChatCompletionsStreamResponse{
		Id:      sensetimeResponse.Data.ID,
		Object:  "chat.completion.chunk",
		Created: helper.GetTimestamp(),
		Model:   "SenseChat-5",
		Choices: []openai.ChatCompletionsStreamResponseChoice{choice},
		Usage:   &sensetimeResponse.Data.Usage,
	}
	return &response
}

func StreamHandler(c *gin.Context, resp *http.Response) (*model.ErrorWithStatusCode, *model.Usage) {
	var usage *model.Usage
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := strings.Index(string(data), "\n\n"); i >= 0 && strings.Index(string(data), ":") >= 0 {
			return i + 2, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})
	dataChan := make(chan string)
	stopChan := make(chan bool)
	go func() {
		for scanner.Scan() {
			data := scanner.Text()
			if len(data) < 5 { // ignore blank line or wrong format
				continue
			}
			if data[:5] == "data:" {
				if data[5:] != "[DONE]" {
					dataChan <- data[5:]
				}
			}
		}
		stopChan <- true
	}()
	common.SetEventStreamHeaders(c)
	c.Stream(func(w io.Writer) bool {
		select {
		case data := <-dataChan:
			response := streamResponseSensetimeOpenAI(data)
			jsonResponse, err := json.Marshal(response)
			if err != nil {
				logger.SysError("error marshalling stream response: " + err.Error())
				return true
			}
			c.Render(-1, common.CustomEvent{Data: "data: " + string(jsonResponse)})
			return true
		case <-stopChan:
			c.Render(-1, common.CustomEvent{Data: "data: [DONE]"})
			return false
		}
	})
	err := resp.Body.Close()
	if err != nil {
		return openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}
	return nil, usage
}

func Handler(c *gin.Context, resp *http.Response) (*model.ErrorWithStatusCode, *model.Usage) {
	var sensetimeResponse Response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	err = resp.Body.Close()
	if err != nil {
		return openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}
	err = json.Unmarshal(responseBody, &sensetimeResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}
	if len(sensetimeResponse.Status.Msg) > 0 {
		return &model.ErrorWithStatusCode{
			Error: model.Error{
				Message: sensetimeResponse.Status.Msg,
				Type:    "sensetime_error",
				Param:   "",
				Code:    sensetimeResponse.Status.Code,
			},
			StatusCode: resp.StatusCode,
		}, nil
	}
	fullTextResponse := responseSensetimeOpenAI(&sensetimeResponse)
	fullTextResponse.Model = "SenseChat-5"
	jsonResponse, err := json.Marshal(fullTextResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError), nil
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = c.Writer.Write(jsonResponse)
	return nil, &fullTextResponse.Usage
}

func EmbeddingsHandler(c *gin.Context, resp *http.Response) (*model.ErrorWithStatusCode, *model.Usage) {
	var sensetimeResponse EmbeddingResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	err = resp.Body.Close()
	if err != nil {
		return openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}
	err = json.Unmarshal(responseBody, &sensetimeResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}
	fullTextResponse := embeddingResponseSensetimeOpenAI(&sensetimeResponse)
	jsonResponse, err := json.Marshal(fullTextResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError), nil
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = c.Writer.Write(jsonResponse)
	return nil, &fullTextResponse.Usage
}

func embeddingResponseSensetimeOpenAI(response *EmbeddingResponse) *openai.EmbeddingResponse {
	openAIEmbeddingResponse := openai.EmbeddingResponse{
		Object: "list",
		Data:   make([]openai.EmbeddingResponseItem, 0, len(response.Embeddings)),
		Model:  "nova-embedding-stable",
		Usage: model.Usage{
			PromptTokens:     response.PromptTokens,
			CompletionTokens: response.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
	}

	for _, item := range response.Embeddings {
		openAIEmbeddingResponse.Data = append(openAIEmbeddingResponse.Data, openai.EmbeddingResponseItem{
			Object:    `embedding`,
			Index:     item.Index,
			Embedding: item.Embedding,
		})
	}
	return &openAIEmbeddingResponse
}

func ImageHandler(c *gin.Context, resp *http.Response) (*model.ErrorWithStatusCode, *model.Usage) {
	return nil, nil
}
