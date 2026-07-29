package ensure

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

// Client is a thin Ollama HTTP client scoped to the `ensure` verb.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient resolves the Ollama base URL from (in order): OLLAMA_BASE_URL,
// OLLAMA_HOST + OLLAMA_PORT, then the 11434 default. Callers that need a
// custom client (tests) can construct the struct directly.
func NewClient() *Client {
	return &Client{
		BaseURL: resolveBaseURL(),
		HTTP:    &http.Client{Timeout: 0}, // streaming; timeout via ctx
	}
}

func resolveBaseURL() string {
	if raw := strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL")); raw != "" {
		return strings.TrimRight(raw, "/")
	}
	host := strings.TrimSpace(os.Getenv("OLLAMA_HOST"))
	port := strings.TrimSpace(os.Getenv("OLLAMA_PORT"))
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "11434"
	}
	// OLLAMA_HOST may itself be "host:port" (upstream convention). Respect it.
	if strings.Contains(host, ":") {
		return "http://" + host
	}
	return "http://" + host + ":" + port
}

type tagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// ListModels returns the installed model references (e.g. "qwen3:4b"),
// sorted for deterministic output. It is the SSOT primitive behind
// `resource-ollama models list` — the single /api/tags reader callers depend
// on instead of opening their own HTTP path.
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	tags, err := c.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(tags))
	for name := range tags {
		if name = strings.TrimSpace(name); name != "" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out, nil
}

// ShowResponse captures the subset of POST /api/show the probe SSOT needs:
// the model's live capability set, its prompt template (to detect stub
// templates), and family/parameter metadata for reporting.
type ShowResponse struct {
	Template     string           `json:"template"`
	Capabilities []string         `json:"capabilities"`
	Parameters   string           `json:"parameters"`
	Details      ShowModelDetails `json:"details"`
	ModelInfo    map[string]any   `json:"model_info,omitempty"`
}

// ShowModelDetails is the nested details block of /api/show.
type ShowModelDetails struct {
	Family            string   `json:"family"`
	Families          []string `json:"families,omitempty"`
	Format            string   `json:"format,omitempty"`
	ParameterSize     string   `json:"parameter_size,omitempty"`
	QuantizationLevel string   `json:"quantization_level,omitempty"`
}

// ShowModel calls POST /api/show and returns the live model metadata. This is
// the only /api/show reader in the tree — every capability/template check
// flows through it (probe SSOT).
func (c *Client) ShowModel(ctx context.Context, model string) (ShowResponse, error) {
	body, err := json.Marshal(map[string]any{"name": model})
	if err != nil {
		return ShowResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/show", bytes.NewReader(body))
	if err != nil {
		return ShowResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return ShowResponse{}, fmt.Errorf("show %s: %w", model, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return ShowResponse{}, fmt.Errorf("show %s: HTTP %d: %s", model, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed ShowResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return ShowResponse{}, fmt.Errorf("decode show response: %w", err)
	}
	return parsed, nil
}

// ListTags returns the set of installed model references (e.g. "qwen3:4b").
func (c *Client) ListTags(ctx context.Context) (map[string]bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("list tags: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode tags: %w", err)
	}
	out := make(map[string]bool, len(parsed.Models))
	for _, m := range parsed.Models {
		out[m.Name] = true
	}
	return out, nil
}

type RunningModel struct {
	Name          string `json:"name"`
	Model         string `json:"model,omitempty"`
	Size          int64  `json:"size,omitempty"`
	SizeVRAM      int64  `json:"size_vram,omitempty"`
	Processor     string `json:"processor,omitempty"`
	Until         string `json:"until,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	Digest        string `json:"digest,omitempty"`
	DetailsFamily string `json:"details_family,omitempty"`
}

type psResponse struct {
	Models []struct {
		Name      string `json:"name"`
		Model     string `json:"model"`
		Size      int64  `json:"size"`
		SizeVRAM  int64  `json:"size_vram"`
		Processor string `json:"processor"`
		Until     string `json:"until"`
		ExpiresAt string `json:"expires_at"`
		Digest    string `json:"digest"`
		Details   struct {
			Family string `json:"family"`
		} `json:"details"`
	} `json:"models"`
}

// ListRunning returns models currently loaded by Ollama according to /api/ps.
func (c *Client) ListRunning(ctx context.Context) ([]RunningModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/ps", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list running models: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("list running models: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed psResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode running models: %w", err)
	}
	out := make([]RunningModel, 0, len(parsed.Models))
	for _, m := range parsed.Models {
		name := m.Name
		if name == "" {
			name = m.Model
		}
		out = append(out, RunningModel{
			Name:          name,
			Model:         m.Model,
			Size:          m.Size,
			SizeVRAM:      m.SizeVRAM,
			Processor:     m.Processor,
			Until:         m.Until,
			ExpiresAt:     m.ExpiresAt,
			Digest:        m.Digest,
			DetailsFamily: m.Details.Family,
		})
	}
	return out, nil
}

// PullProgress is emitted for each status line Ollama streams during a pull.
type PullProgress struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Pull streams /api/pull for a single model reference, forwarding each
// progress line to onProgress. Returns once Ollama reports "success" or a
// fatal error.
func (c *Client) Pull(ctx context.Context, modelRef string, onProgress func(PullProgress)) error {
	body, err := json.Marshal(map[string]any{
		"name":   modelRef,
		"stream": true,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/pull", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("pull %s: %w", modelRef, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("pull %s: HTTP %d: %s", modelRef, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var last PullProgress
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var prog PullProgress
		if err := json.Unmarshal(line, &prog); err != nil {
			return fmt.Errorf("pull %s: decode progress: %w", modelRef, err)
		}
		if onProgress != nil {
			onProgress(prog)
		}
		if prog.Error != "" {
			return fmt.Errorf("pull %s: %s", modelRef, prog.Error)
		}
		last = prog
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("pull %s: read stream: %w", modelRef, err)
	}
	if !strings.EqualFold(last.Status, "success") {
		return fmt.Errorf("pull %s: ended without success status (last=%q)", modelRef, last.Status)
	}
	return nil
}

// EmbedRequest mirrors the relevant subset of POST /api/embeddings.
type EmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// EmbedResponse is the float vector returned by Ollama's embedding endpoint.
// We accept either the legacy `embedding` (singular) or `embeddings[0]` shape
// so this client survives both Ollama 0.1.x and 0.5.x+.
type EmbedResponse struct {
	Embedding  []float64   `json:"embedding,omitempty"`
	Embeddings [][]float64 `json:"embeddings,omitempty"`
}

// Embed calls /api/embeddings and returns the resulting vector.
func (c *Client) Embed(ctx context.Context, model, input string) ([]float64, error) {
	body, err := json.Marshal(EmbedRequest{Model: model, Prompt: input})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("embed: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}
	if len(parsed.Embedding) > 0 {
		return parsed.Embedding, nil
	}
	if len(parsed.Embeddings) > 0 {
		return parsed.Embeddings[0], nil
	}
	return nil, fmt.Errorf("embed: response contained no embedding vector")
}

// Unload asks Ollama to evict a loaded model from VRAM immediately by issuing a
// generate with keep_alive=0 (the Ollama-sanctioned way to release a model's
// GPU memory now). This is the actuation behind the capacity broker's ollama
// degrade rung — freeing VRAM for a higher-priority interactive workload.
func (c *Client) Unload(ctx context.Context, model string) error {
	requestBody := struct {
		Model     string `json:"model"`
		KeepAlive int    `json:"keep_alive"`
		Stream    bool   `json:"stream"`
	}{Model: model, KeepAlive: 0, Stream: false}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("unload %s: %w", model, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("unload %s: HTTP %d: %s", model, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

// GenerateRequest mirrors the relevant subset of POST /api/generate. Stream is
// always false at this layer — callers wanting NDJSON should add a separate
// streaming entrypoint when needed.
type GenerateRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	Think       *bool    `json:"think,omitempty"`
	NumPredict  *int     `json:"num_predict,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
}

// GenerateResponse captures the buffered (stream=false) shape.
type GenerateResponse struct {
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	EvalCount int    `json:"eval_count,omitempty"`
}

// Generate calls /api/generate with stream=false and returns the full response.
func (c *Client) Generate(ctx context.Context, in GenerateRequest) (GenerateResponse, error) {
	options := map[string]any{}
	if in.NumPredict != nil {
		options["num_predict"] = *in.NumPredict
	}
	if in.Temperature != nil {
		options["temperature"] = *in.Temperature
	}
	requestBody := struct {
		Model   string         `json:"model"`
		Prompt  string         `json:"prompt"`
		Stream  bool           `json:"stream"`
		Think   *bool          `json:"think,omitempty"`
		Options map[string]any `json:"options,omitempty"`
	}{
		Model:   in.Model,
		Prompt:  in.Prompt,
		Stream:  false,
		Think:   in.Think,
		Options: options,
	}
	if len(options) == 0 {
		requestBody.Options = nil
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return GenerateResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return GenerateResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("generate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return GenerateResponse{}, fmt.Errorf("generate: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return GenerateResponse{}, fmt.Errorf("decode generate response: %w", err)
	}
	return parsed, nil
}

// ChatMessage is one Ollama /api/chat message.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatTool is one tool advertised to the model on a /api/chat request. Only
// the "function" tool type is used. Parameters is a raw JSON Schema object.
type ChatTool struct {
	Type     string           `json:"type"`
	Function ChatToolFunction `json:"function"`
}

// ChatToolFunction describes a callable function offered to the model.
type ChatToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ChatToolCall is one structured tool call the model emitted in its response.
// Its presence (vs. a tool call narrated as message text) is exactly what the
// tool-calling smoke asserts.
type ChatToolCall struct {
	Function struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments,omitempty"`
	} `json:"function"`
}

// ChatRequest mirrors the relevant subset of POST /api/chat. Stream is always
// false at this layer.
type ChatRequest struct {
	Model       string
	Messages    []ChatMessage
	NumPredict  *int
	Temperature *float64
	Think       *bool
	Tools       []ChatTool
}

// ChatResponse captures the buffered (stream=false) chat response shape,
// including any structured tool_calls the model emitted.
type ChatResponse struct {
	Message struct {
		Content   string         `json:"content"`
		ToolCalls []ChatToolCall `json:"tool_calls,omitempty"`
	} `json:"message"`
	DoneReason string `json:"done_reason,omitempty"`
	EvalCount  int    `json:"eval_count,omitempty"`
}

// Chat calls /api/chat with stream=false and returns the full response.
func (c *Client) Chat(ctx context.Context, in ChatRequest) (ChatResponse, error) {
	options := map[string]any{}
	if in.NumPredict != nil {
		options["num_predict"] = *in.NumPredict
	}
	if in.Temperature != nil {
		options["temperature"] = *in.Temperature
	}
	requestBody := struct {
		Model    string         `json:"model"`
		Messages []ChatMessage  `json:"messages"`
		Stream   bool           `json:"stream"`
		Think    *bool          `json:"think,omitempty"`
		Tools    []ChatTool     `json:"tools,omitempty"`
		Options  map[string]any `json:"options,omitempty"`
	}{
		Model:    in.Model,
		Messages: in.Messages,
		Stream:   false,
		Think:    in.Think,
		Tools:    in.Tools,
		Options:  options,
	}
	if len(options) == 0 {
		requestBody.Options = nil
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("chat: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return ChatResponse{}, fmt.Errorf("chat: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return ChatResponse{}, fmt.Errorf("decode chat response: %w", err)
	}
	return parsed, nil
}
