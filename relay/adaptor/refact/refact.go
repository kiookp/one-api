package refact

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
)

type RefactAdaptor struct{}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RefactRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

func (a *RefactAdaptor) Init(m *meta.Meta) {}

func (a *RefactAdaptor) GetRequestURL(m *meta.Meta) (string, error) {
	return "https://inference.smallcloud.ai/v1/chat/completions", nil
}

func (a *RefactAdaptor) SetupRequestHeader(c *gin.Context, req *http.Request, m *meta.Meta) error {
	req.Header.Set("Authorization", "Bearer "+m.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "refact-lsp 0.10.19")
	return nil
}

func (a *RefactAdaptor) ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error) {
	messages := make([]ChatMessage, len(request.Messages))
	for i, msg := range request.Messages {
		messages[i] = ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return RefactRequest{
		Model:    request.Model,
		Messages: messages,
		Stream:   request.Stream,
	}, nil
}

func (a *RefactAdaptor) ConvertResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (*model.GeneralOpenAIResponse, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result model.GeneralOpenAIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.New("unmarshal_response_body_failed")
	}
	return &result, nil
}

func (a *RefactAdaptor) Do(c *gin.Context, request any, requestURL string, meta *meta.Meta) (*http.Response, error) {
	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}
	err = a.SetupRequestHeader(c, req, meta)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	return client.Do(req)
}