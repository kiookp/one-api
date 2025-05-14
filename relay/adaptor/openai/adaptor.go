package openai

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/alibailian"
	"github.com/songquanpeng/one-api/relay/adaptor/baiduv2"
	"github.com/songquanpeng/one-api/relay/adaptor/doubao"
	"github.com/songquanpeng/one-api/relay/adaptor/geminiv2"
	"github.com/songquanpeng/one-api/relay/adaptor/minimax"
	"github.com/songquanpeng/one-api/relay/adaptor/novita"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

type Adaptor struct {
	ChannelType int
}

func (a *Adaptor) Init(meta *meta.Meta) {
	a.ChannelType = meta.ChannelType
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	switch meta.ChannelType {
	case channeltype.Azure:
		if meta.Mode == relaymode.ImagesGenerations {
			fullRequestURL := fmt.Sprintf("%s/openai/deployments/%s/images/generations?api-version=%s", meta.BaseURL, meta.ActualModelName, meta.Config.APIVersion)
			return fullRequestURL, nil
		}
		requestURL := strings.Split(meta.RequestURLPath, "?")[0]
		requestURL = fmt.Sprintf("%s?api-version=%s", requestURL, meta.Config.APIVersion)
		task := strings.TrimPrefix(requestURL, "/v1/")
		model_ := meta.ActualModelName
		model_ = strings.Replace(model_, ".", "", -1)
		requestURL = fmt.Sprintf("/openai/deployments/%s/%s", model_, task)
		return GetFullRequestURL(meta.BaseURL, requestURL, meta.ChannelType), nil
	case channeltype.Minimax:
		return minimax.GetRequestURL(meta)
	case channeltype.Doubao:
		return doubao.GetRequestURL(meta)
	case channeltype.Novita:
		return novita.GetRequestURL(meta)
	case channeltype.BaiduV2:
		return baiduv2.GetRequestURL(meta)
	case channeltype.AliBailian:
		return alibailian.GetRequestURL(meta)
	case channeltype.GeminiOpenAICompatible:
		return geminiv2.GetRequestURL(meta)
	default:
		return GetFullRequestURL(meta.BaseURL, meta.RequestURLPath, meta.ChannelType), nil
	}
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	adaptor.SetupCommonRequestHeader(c, req, meta)

	if meta.ChannelType == channeltype.Azure {
		req.Header.Set("api-key", meta.APIKey)
		return nil
	}

	req.Header.Set("Authorization", "Bearer "+meta.APIKey)

	if strings.Contains(meta.BaseURL, "inference.smallcloud.ai") {
		req.Header.Set("User-Agent", "refact-lsp 0.10.19")
		req.Header.Set("Accept", "application/json")
	}

	if meta.ChannelType == channeltype.OpenRouter {
		req.Header.Set("HTTP-Referer", "https://github.com/songquanpeng/one-api")
		req.Header.Set("X-Title", "One API")
	}

	fmt.Println("==== Outgoing Request Debug Info ====")
	fmt.Printf("URL     : %s\n", req.URL.String())
	fmt.Printf("Method  : %s\n", req.Method)
	fmt.Printf("Host    : %s\n", req.URL.Host)
	fmt.Printf("Scheme  : %s\n", req.URL.Scheme)
	fmt.Printf("Path    : %s\n", req.URL.Path)
	if req.URL.RawQuery != "" {
		fmt.Printf("Query   : %s\n", req.URL.RawQuery)
	}
	fmt.Println("Headers :")
	for key, values := range req.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}
	fmt.Println("=====================================")

	return nil
}

func (a *Adaptor) ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}

	if request.StreamOptions == nil {
		request.StreamOptions = &model.StreamOptions{}
	}
	request.StreamOptions.IncludeUsage = true

	return request, nil
}

func (a *Adaptor) ConvertImageRequest(request *model.ImageRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	bodyBytes, _ := io.ReadAll(requestBody)
	fmt.Println("==== 🔍 Outgoing JSON Payload ====")
	fmt.Println(string(bodyBytes))
	fmt.Println("==================================")

	requestBody = io.NopCloser(strings.NewReader(string(bodyBytes)))

	return adaptor.DoRequestHelper(a, c, meta, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (usage *model.Usage, err *model.ErrorWithStatusCode) {
    if resp != nil {
        // ✅ 打印响应状态和原始 Body 内容
        fmt.Println("==== 🔁 Raw Response From Model Provider ====")
        fmt.Println("Status:", resp.Status)

        bodyBytes, _ := io.ReadAll(resp.Body)
        fmt.Println("Body:")
        fmt.Println(string(bodyBytes))
        fmt.Println("=============================================")

        // ✅ 重建 resp.Body 供后续逻辑使用
        resp.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
    }

    // 由于已设置为非流式模式，直接处理非流式响应
    switch meta.Mode {
    case relaymode.ImagesGenerations:
        err, _ = ImageHandler(c, resp)
    default:
        err, usage = Handler(c, resp, meta.PromptTokens, meta.ActualModelName)
    }

    return
}

func (a *Adaptor) GetModelList() []string {
	_, modelList := GetCompatibleChannelMeta(a.ChannelType)
	return modelList
}

func (a *Adaptor) GetChannelName() string {
	channelName, _ := GetCompatibleChannelMeta(a.ChannelType)
	return channelName
}