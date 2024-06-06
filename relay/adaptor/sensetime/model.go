package sensetime

import (
	"github.com/songquanpeng/one-api/relay/model"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Plugin struct {
	WebSearch struct {
		SearchEnable bool `json:"search_enable"`
		ResultEnable bool `json:"result_enable"`
	} `json:"web_search,omitempty"`
	AssociatedKnowledge struct {
		Content string `json:"content"`
		Mode    string `json:"mode"`
	} `json:"associated_knowledge,omitempty"`
}

type PluginResult struct {
	WebSearch struct {
		OnlineSearchCount int `json:"online_search_count,omitempty"`
		Results           []struct {
			Index     int    `json:"index,omitempty"`
			URL       string `json:"url,omitempty"`
			URLSource string `json:"url_source,omitempty"`
			Title     string `json:"title,omitempty"`
			Icon      string `json:"icon,omitempty"`
		} `json:"results,omitempty"`
	} `json:"web_search,omitempty"`
	AssociatedKnowledge struct {
		Content string `json:"content"`
		Mode    string `json:"mode"`
	} `json:"associated_knowledge,omitempty"`
}

type KnowledgeBaseConfig struct {
	KnowID       string  `json:"know_id"`
	FAQThreshold float64 `json:"faq_threshold"`
}

type KnowledgeConfig struct {
	ControlLevel         string                `json:"control_level"`
	KnowledgeBaseResult  bool                  `json:"knowledge_base_result"`
	KnowledgeBaseConfigs []KnowledgeBaseConfig `json:"knowledge_base_configs"`
}

type Request struct {
	Model             string           `json:"model"`
	N                 int              `json:"n,omitempty"`
	KnowIds           []string         `json:"know_ids,omitempty"`
	MaxNewTokens      int              `json:"max_new_tokens,omitempty"`
	Messages          []Message        `json:"messages"`
	RepetitionPenalty float64          `json:"repetition_penalty,omitempty"`
	Stream            bool             `json:"stream,omitempty"`
	Temperature       float64          `json:"temperature,omitempty"`
	TopP              float64          `json:"top_p,omitempty"`
	User              string           `json:"user,omitempty"`
	KnowledgeConfig   *KnowledgeConfig `json:"knowledge_config,omitempty"`
	Plugins           *Plugin          `json:"plugins,omitempty"`
}

type Choice struct {
	Message      string `json:"message,omitempty"`
	FinishReason string `json:"finish_reason,omitempty"`
	Index        int    `json:"index,omitempty"`
	Role         string `json:"role,omitempty"`
	Delta        string `json:"delta,omitempty"`
}

type KnowledgeBaseResult struct {
	KnowID  string `json:"know_id,omitempty"`
	Results []struct {
		Score     float64 `json:"score,omitempty"`
		Result    string  `json:"result,omitempty"`
		ExtraInfo struct {
			FileID string `json:"file_id,omitempty"`
			Page   int    `json:"page,omitempty"`
		} `json:"extra_info,omitempty"`
	} `json:"results,omitempty"`
}

type ResponseData struct {
	ID                   string                `json:"id,omitempty"`
	Choices              []Choice              `json:"choices,omitempty"`
	KnowledgeBaseResults []KnowledgeBaseResult `json:"knowledge_base_results,omitempty"`
	Plugins              *PluginResult         `json:"plugins,omitempty"`
	model.Usage          `json:"usage"`
}

type ResponseStatus struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type Response struct {
	Status ResponseStatus `json:"status"`
	Data   ResponseData   `json:"data"`
}

type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type EmbeddingResponse struct {
	Embeddings  []EmbeddingData `json:"embeddings"`
	model.Usage `json:"usage"`
}

type EmbeddingData struct {
	Index         int       `json:"index"`
	StatusCode    int       `json:"status_code"`
	StatusMessage string    `json:"status_message"`
	Embedding     []float64 `json:"embedding"`
}

type ImageRequest struct {
	Model        string `json:"model"`
	Instructions string `json:"instructions"`
}
