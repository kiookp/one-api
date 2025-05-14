package apitype

const (
	OpenAI = iota
	Anthropic
	PaLM
	Baidu
	Zhipu
	Ali
	Xunfei
	AIProxyLibrary
	Tencent
	Gemini
	Ollama
	AwsClaude
	Coze
	Cohere
	Cloudflare
	DeepL
	VertexAI
	Proxy
	Replicate
    Refact // ✅ 新增 Refact 适配器类型
	Dummy // this one is only for count, do not add any channel after this
)
