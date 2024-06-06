package sensetime

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
	"io"
	"math"
	"net/http"
)

type Adaptor struct {
}

func (a *Adaptor) Init(meta *meta.Meta) {

}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	switch meta.Mode {
	case relaymode.ImagesGenerations:
		return fmt.Sprintf("%s/v1/assistants", meta.BaseURL), nil
	case relaymode.Embeddings:
		return fmt.Sprintf("%s/v1/llm/embeddings", meta.BaseURL), nil
	}
	return fmt.Sprintf("%s/v1/llm/chat-completions", meta.BaseURL), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	adaptor.SetupCommonRequestHeader(c, req, meta)
	token := GetToken(meta.Config.AK, meta.Config.SK)
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

func (a *Adaptor) ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	switch relayMode {
	case relaymode.Embeddings:
		sensetimeEmbeddingRequest, err := ConvertEmbeddingRequest(*request)
		return sensetimeEmbeddingRequest, err
	default:
		// TopP (0.0, 1.0)
		request.TopP = math.Min(0.99, request.TopP)
		request.TopP = math.Max(0.01, request.TopP)

		// Temperature (0.0, 1.0)
		request.Temperature = math.Min(0.99, request.Temperature)
		request.Temperature = math.Max(0.01, request.Temperature)
		return ConvertRequest(*request), nil
	}
}

func (a *Adaptor) ConvertImageRequest(request *model.ImageRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	newRequest := ImageRequest{
		Model:        request.Model,
		Instructions: request.Prompt,
	}
	return newRequest, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	return adaptor.DoRequestHelper(a, c, meta, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (usage *model.Usage, err *model.ErrorWithStatusCode) {
	switch meta.Mode {
	case relaymode.Embeddings:
		err, usage = EmbeddingsHandler(c, resp)
		return
	case relaymode.ImagesGenerations:
		err, usage = ImageHandler(c, resp)
		return
	}
	if meta.IsStream {
		err, usage = StreamHandler(c, resp)
	} else {
		err, usage = Handler(c, resp)
	}
	return
}

func ConvertEmbeddingRequest(request model.GeneralOpenAIRequest) (*EmbeddingRequest, error) {
	inputs := request.ParseInput()
	if len(inputs) != 1 {
		return nil, errors.New("invalid input length, sensetime only support one input")
	}
	return &EmbeddingRequest{
		Model: request.Model,
		Input: inputs,
	}, nil
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return "sensetime"
}
