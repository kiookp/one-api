package refact

import (
	"bufio"
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
	body := map[string]any{
		"model":       request.Model,
		"messages":    request.Messages,
		"stream":      c.Query("stream") == "true" || request.Stream, // 自动判断流式
		"temperature": request.Temperature,
		"top_p":       request.TopP,
		"max_tokens":  request.MaxTokens,
	}
	return body, nil
}

func (a *RefactAdaptor) ConvertImageRequest(request *model.ImageRequest) (any, error) {
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

func (a *RefactAdaptor) DoResponse(c *gin.Context, resp *http.Response, m *meta.Meta) (*model.Usage, *model.ErrorWithStatusCode) {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &model.ErrorWithStatusCode{
			StatusCode: resp.StatusCode,
			Error:      model.Error{Message: "refact returned non-200 status"},
		}
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
				if err == io.EOF {
					break
				}
				return nil, &model.ErrorWithStatusCode{
					StatusCode: http.StatusInternalServerError,
					Error:      model.Error{Message: "error reading stream: " + err.Error()},
				}
			}
			_, err = c.Writer.Write([]byte("data: " + string(line) + "\n\n"))
			if err != nil {
				return nil, &model.ErrorWithStatusCode{
					StatusCode: http.StatusInternalServerError,
					Error:      model.Error{Message: "error writing stream: " + err.Error()},
				}
			}
			c.Writer.Flush()
		}
		// For streaming, usage might not be available; return nil or estimate if required
		return nil, nil
	}

	var result struct {
		Choices []struct {
			Message ChatMessage `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil || len(result.Choices) == 0 {
		return nil, &model.ErrorWithStatusCode{
			StatusCode: http.StatusInternalServerError,
			Error:      model.Error{Message: "invalid response format: " + err.Error()},
		}
	}

	// Construct usage data
	usage := &model.Usage{
		PromptTokens:     result.Usage.PromptTokens,
		CompletionTokens: result.Usage.CompletionTokens,
		TotalTokens:      result.Usage.TotalTokens,
	}

	// Return JSON response
	c.JSON(http.StatusOK, gin.H{
		"choices": []gin.H{
			{"message": result.Choices[0].Message},
		},
		"usage": usage,
	})

	return usage, nil
}

func (a *RefactAdaptor) GetModelList() []string {
	return []string{"gpt-4.1", "code-complete-alpha"}
}

func (a *RefactAdaptor) GetChannelName() string {
	return "Refact.ai"
}