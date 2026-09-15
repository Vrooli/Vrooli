package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Defaults carries the static config defaults formerly in config/defaults.sh.
type Defaults struct {
	Provider           string // active provider (empty = auto-detect)
	CloudRole          string // openrouter policy role backing the cloud default (code.default)
	ChatModel          string // cloud default chat model (resolved from CloudRole when empty)
	CompletionModel    string // cloud default completion model (resolved from CloudRole when empty)
	GoDefaultModel     string // opencode-go default model when Go subscription is active
	OllamaDefaultModel string // local model declared in the provider block
	NumCtx             int    // per-model num_ctx for the local coder
	LocalRole          string // ollama policy role backing local coding (code.local)
	LegacyTargets      []string
}

// DefaultDefaults returns the built-in defaults, honoring the same env
// overrides as the legacy shell configuration.
func DefaultDefaults(getenv func(string) string) Defaults {
	if getenv == nil {
		getenv = os.Getenv
	}
	numCtx := 16384
	if v := strings.TrimSpace(getenv("OPENCODE_OLLAMA_NUM_CTX")); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			numCtx = n
		}
	}
	localModel := strings.TrimSpace(getenv("OPENCODE_OLLAMA_DEFAULT_MODEL"))
	if localModel == "" {
		localModel = "gemma4:12b"
	}
	cloudRole := strings.TrimSpace(getenv("OPENCODE_DEFAULT_CHAT_ROLE"))
	if cloudRole == "" {
		cloudRole = "code.default"
	}
	goModel := strings.TrimSpace(getenv("OPENCODE_GO_DEFAULT_MODEL"))
	if goModel == "" {
		goModel = "deepseek-v4.1-flash"
	}
	return Defaults{
		Provider:           "",
		CloudRole:          cloudRole,
		GoDefaultModel:     goModel,
		OllamaDefaultModel: localModel,
		NumCtx:             numCtx,
		LocalRole:          "code.local",
		LegacyTargets:      []string{"openrouter/qwen3-coder", "openrouter/qwen/qwen3-coder"},
	}
}

// RoleResolution is the local-role view the config writer needs from the SSOT.
type RoleResolution struct {
	Model    string
	Sampling Sampling
}

// Resolver is the injectable seam over the Ollama probe SSOT
// (`resource-ollama`). The config writer NEVER probes the daemon directly.
type Resolver interface {
	InstalledModels(ctx context.Context) ([]string, error)
	LocalRole(ctx context.Context, role string) (RoleResolution, error)
}

// EnsureOptions configures Ensure.
type EnsureOptions struct {
	ConfigPath      string
	Defaults        Defaults
	HaveOpenCodeGo  bool
	HaveOpenRouter  bool
	Resolver        Resolver
	Logf            func(format string, args ...any)
}

// Ensure decides the provider (OpenCode Go → OpenRouter → local Ollama),
// renders opencode.json preserving the permission map and unknown keys,
// and writes only on a real change. Returns whether the file changed.
func Ensure(ctx context.Context, opts EnsureOptions) (bool, error) {
	logf := opts.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	d := opts.Defaults

	installed, listErr := opts.Resolver.InstalledModels(ctx)
	ollamaReachable := listErr == nil && len(installed) > 0

	var (
		provider       string
		chatModel      string
		completionModel string
		useOllama      bool
		useOpenCodeGo  bool
		useOpenRouter  bool
	)

	// Tier 1: OpenCode Go subscription (cheapest, subscription-billed)
	if opts.HaveOpenCodeGo {
		useOpenCodeGo = true
		provider = goProviderID
		chatModel = d.GoDefaultModel
		completionModel = d.GoDefaultModel
		logf("Selected OpenCode Go subscription (provider=%s, model=%s)", provider, chatModel)
	}

	// Tier 2: OpenRouter metered API (fallback when no Go key)
	if !useOpenCodeGo && opts.HaveOpenRouter {
		useOpenRouter = true
		if d.ChatModel == "" || d.CompletionModel == "" {
			role := strings.TrimSpace(d.CloudRole)
			if role == "" {
				role = "code.default"
			}
			model, err := resolveCloudModel(ctx, role)
			if err != nil {
				return false, fmt.Errorf("resolve OpenRouter cloud default model (role %q): %w", role, err)
			}
			if d.ChatModel == "" {
				d.ChatModel = model
			}
			if d.CompletionModel == "" {
				d.CompletionModel = model
			}
		}
		provider = "openrouter"
		chatModel = d.ChatModel
		completionModel = d.CompletionModel
		logf("Selected OpenRouter API (provider=%s, model=%s)", provider, chatModel)
	}

	// Tier 3: Local Ollama (fallback when no cloud credentials)
	if !useOpenCodeGo && !useOpenRouter && ollamaReachable {
		useOllama = true
		provider = ollamaProviderID
		chatModel = d.OllamaDefaultModel
		completionModel = d.OllamaDefaultModel
		logf("Self-healed to local Ollama (provider=%s, model=%s)", provider, chatModel)
	}

	var sampling Sampling
	if ollamaReachable {
		if rr, err := opts.Resolver.LocalRole(ctx, d.LocalRole); err == nil {
			sampling = rr.Sampling
			if rr.Model != "" && containsModel(installed, rr.Model) {
				if useOllama {
					chatModel = rr.Model
					completionModel = rr.Model
				}
			}
		}
	}

	existing, readErr := os.ReadFile(opts.ConfigPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return false, fmt.Errorf("read %s: %w", opts.ConfigPath, readErr)
	}
	freshFile := os.IsNotExist(readErr) || len(bytes.TrimSpace(existing)) == 0

	in := Inputs{
		Provider:        provider,
		ChatModel:       chatModel,
		CompletionModel: completionModel,
	}

	currentModel := currentModelOf(existing)
	currentProvider := ""
	if i := strings.Index(currentModel, "/"); i >= 0 {
		currentProvider = currentModel[:i]
	}

	if freshFile {
		in.Repoint = true
	} else if useOllama && (currentProvider == "" || currentProvider == "openrouter" || currentProvider == goProviderID) {
		in.Repoint = true
	} else if useOpenCodeGo && currentProvider != goProviderID {
		in.Repoint = true
	} else if useOpenRouter && currentProvider != "openrouter" {
		in.Repoint = true
	}

	if useOpenCodeGo {
		in.Go = &GoProvider{
			BaseURL:    goBaseURL,
			ChatModel:  chatModel,
			SmallModel: completionModel,
		}
	}
	if ollamaReachable {
		in.Ollama = &OllamaProvider{
			BaseURL:    ollamaBaseURL(os.Getenv) + "/api",
			ChatModel:  chatModel,
			SmallModel: completionModel,
			NumCtx:     d.NumCtx,
			Sampling:   sampling,
		}
	} else if !freshFile && hasOllamaBlock(existing) {
		in.Ollama = &OllamaProvider{
			BaseURL:    ollamaBaseURL(os.Getenv) + "/api",
			ChatModel:  d.OllamaDefaultModel,
			SmallModel: d.OllamaDefaultModel,
			NumCtx:     d.NumCtx,
			Sampling:   sampling,
		}
	}

	rendered, err := Render(existing, in)
	if err != nil {
		return false, err
	}

	if !freshFile && bytes.Equal(normalize(existing), normalize(rendered)) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(opts.ConfigPath), 0o755); err != nil {
		return false, fmt.Errorf("mkdir config dir: %w", err)
	}
	if err := os.WriteFile(opts.ConfigPath, rendered, 0o644); err != nil {
		return false, fmt.Errorf("write %s: %w", opts.ConfigPath, err)
	}

	switch {
	case freshFile:
		logf("Created OpenCode config at %s (provider=%s)", opts.ConfigPath, provider)
	case useOpenCodeGo:
		logf("Switched to OpenCode Go subscription (provider=%s, model=%s)", provider, chatModel)
	case useOllama:
		logf("Self-healed OpenCode model -> %s/%s (no cloud key; local Ollama reachable)", provider, chatModel)
	default:
		logf("Updated OpenCode config at %s", opts.ConfigPath)
	}

	if provider == "" {
		logf("WARNING: No usable provider credentials found. Set OPENCODE_GO_KEY or OPENROUTER_API_KEY, or ensure Ollama is running.")
	}
	return true, nil
}

// --- helpers ------------------------------------------------------------------

func currentModelOf(existing []byte) string {
	if len(bytes.TrimSpace(existing)) == 0 {
		return ""
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(existing, &top); err != nil {
		return ""
	}
	return rawString(top[keyModel])
}

func hasOllamaBlock(existing []byte) bool {
	var top struct {
		Provider struct {
			Ollama json.RawMessage `json:"ollama"`
		} `json:"provider"`
	}
	if err := json.Unmarshal(existing, &top); err != nil {
		return false
	}
	return len(top.Provider.Ollama) > 0
}

func containsModel(installed []string, model string) bool {
	for _, m := range installed {
		if m == model || m == model+":latest" {
			return true
		}
	}
	return false
}

func normalize(data []byte) []byte {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return data
	}
	out, err := json.Marshal(v)
	if err != nil {
		return data
	}
	return out
}

func ollamaBaseURL(getenv func(string) string) string {
	if getenv == nil {
		getenv = os.Getenv
	}
	raw := strings.TrimSpace(getenv("OLLAMA_HOST"))
	if raw == "" {
		raw = "localhost:11434"
	}
	scheme := "http://"
	switch {
	case strings.HasPrefix(raw, "https://"):
		scheme = "https://"
		raw = strings.TrimPrefix(raw, "https://")
	case strings.HasPrefix(raw, "http://"):
		raw = strings.TrimPrefix(raw, "http://")
	}
	hostport := raw
	if i := strings.Index(hostport, "/"); i >= 0 {
		hostport = hostport[:i]
	}
	if !strings.Contains(hostport, ":") {
		hostport += ":11434"
	}
	return scheme + hostport
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// --- cloud model role resolver (execs the resource-openrouter SSOT) -----------

var resolveCloudModel = execResolveCloudModel

func execResolveCloudModel(ctx context.Context, role string) (string, error) {
	out, err := exec.CommandContext(ctx, "resource-openrouter", "policy", "resolve", "--role", role, "--field", "model").Output()
	if err != nil {
		return "", fmt.Errorf("exec resource-openrouter policy resolve --role %s --field model: %w", role, err)
	}
	model := strings.TrimSpace(string(out))
	if model == "" {
		return "", fmt.Errorf("resource-openrouter resolved an empty model for role %q", role)
	}
	return model, nil
}

// --- default Resolver (execs the resource-ollama SSOT) ------------------------

type ExecResolver struct {
	Command string
}

func (r ExecResolver) bin() string {
	if strings.TrimSpace(r.Command) != "" {
		return r.Command
	}
	return "resource-ollama"
}

func (r ExecResolver) InstalledModels(ctx context.Context) ([]string, error) {
	out, err := exec.CommandContext(ctx, r.bin(), "models", "list", "--json").Output()
	if err != nil {
		return nil, err
	}
	var payload struct {
		Models []string `json:"models"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &payload); err != nil {
		return nil, err
	}
	return payload.Models, nil
}

func (r ExecResolver) LocalRole(ctx context.Context, role string) (RoleResolution, error) {
	out, err := exec.CommandContext(ctx, r.bin(), "policy", "resolve", "--role", role, "--json").Output()
	if err != nil {
		return RoleResolution{}, err
	}
	var payload struct {
		Model    string `json:"model"`
		Sampling *struct {
			Temperature    float64 `json:"temperature"`
			TopP           float64 `json:"top_p"`
			TopK           int     `json:"top_k"`
			HasTemperature bool    `json:"has_temperature"`
			HasTopP        bool    `json:"has_top_p"`
			HasTopK        bool    `json:"has_top_k"`
		} `json:"sampling"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &payload); err != nil {
		return RoleResolution{}, err
	}
	rr := RoleResolution{Model: payload.Model}
	if payload.Sampling != nil {
		if payload.Sampling.HasTemperature {
			t := payload.Sampling.Temperature
			rr.Sampling.Temperature = &t
		}
		if payload.Sampling.HasTopP {
			p := payload.Sampling.TopP
			rr.Sampling.TopP = &p
		}
		if payload.Sampling.HasTopK {
			k := payload.Sampling.TopK
			rr.Sampling.TopK = &k
		}
	}
	return rr, nil
}