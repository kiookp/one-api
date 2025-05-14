package refact

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/meta"
)

type RefactAdaptor struct{}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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

func (a *RefactAdaptor) ConvertRequest(c *gin.Context, relayMode int, request any) (any, error) {
	req, ok := request.(map[string]any)
	if !ok {
		return nil, errors.New("invalid request format")
	}
	req["stream"] = true
	return req, nil
}

func (a *RefactAdaptor) ConvertImageRequest(request any) (any, error) {
	return nil, errors.New("refact does not support image generation")
}

func (a *RefactAdaptor) DoRequest(c *gin.Context, m *meta.Meta, body io.Reader) (*http.Response, error) {
	fullURL, _ := a.GetRequestURL(m)
	req, err := http.NewRequest("POST", fullURL, body)
	if err != nil {
		return nil, err
	}
	a.SetupRequestHeader(c, req, m)
	return http.DefaultClient.Do(req)
}

func (a *RefactAdaptor) DoResponse(c *gin.Context, resp *http.Response, m *meta.Meta) (any, error) {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("refact returned non-200 status")
	}

	if m.IsStream {
		c.Status(http.StatusOK)
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				break
			}
			c.Writer.Write([]byte("data: " + string(line) + "\n\n"))
			c.Writer.Flush()
		}
		return nil, nil
	}

	var result struct {
		Choices []struct {
			Message ChatMessage `json:"message"`
		} `json:"choices"`
	}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil || len(result.Choices) == 0 {
		return nil, errors.New("invalid response format")
	}

	c.JSON(http.StatusOK, gin.H{
		"choices": []gin.H{
			{"message": result.Choices[0].Message},
		},
	})
	return nil, nil
}

func (a *RefactAdaptor) GetModelList() []string {
	return []string{"gpt-4.1", "code-complete-alpha"}
}

func (a *RefactAdaptor) GetChannelName() string {
	return "Refact.ai"
}