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
	Provider           string // cloud default provider (openrouter)
	ChatModel          string // cloud default chat model
	CompletionModel    string // cloud default completion model
	OllamaDefaultModel string // local model declared in the provider block
	NumCtx             int    // per-model num_ctx for the local coder
	LocalRole          string // ollama policy role backing local coding (code.local)
	LegacyTargets      []string
}

// DefaultDefaults returns the built-in defaults, honoring the same env
// overrides the bash defaults.sh recognized.
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
	return Defaults{
		Provider:           "openrouter",
		ChatModel:          "deepseek/deepseek-v4-flash",
		CompletionModel:    "deepseek/deepseek-v4-flash",
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
	// InstalledModels returns the installed model refs, or an error when the
	// daemon/SSOT is unreachable (treated as "Ollama not reachable").
	InstalledModels(ctx context.Context) ([]string, error)
	// LocalRole resolves the local coding role's model + clamped sampling.
	LocalRole(ctx context.Context, role string) (RoleResolution, error)
}

// EnsureOptions configures Ensure.
type EnsureOptions struct {
	ConfigPath     string
	Defaults       Defaults
	HaveOpenRouter bool
	Resolver       Resolver
	Logf           func(format string, args ...any)
}

// Ensure is the Go port of opencode::ensure_config: it decides the provider
// (OpenRouter vs local Ollama self-heal), renders opencode.json preserving the
// permission map and unknown keys, and writes only on a real change. Returns
// whether the file changed.
func Ensure(ctx context.Context, opts EnsureOptions) (bool, error) {
	logf := opts.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	d := opts.Defaults

	// Reachability + local model/sampling come from the SSOT.
	installed, listErr := opts.Resolver.InstalledModels(ctx)
	ollamaReachable := listErr == nil && len(installed) > 0

	provider := d.Provider
	chatModel := d.ChatModel
	completionModel := d.CompletionModel
	ollamaBlockModel := d.OllamaDefaultModel
	useOllama := false

	var sampling Sampling
	if ollamaReachable {
		if rr, err := opts.Resolver.LocalRole(ctx, d.LocalRole); err == nil {
			sampling = rr.Sampling
			if rr.Model != "" && containsModel(installed, rr.Model) {
				ollamaBlockModel = rr.Model
			}
		}
	}

	// Use Ollama as the ACTIVE model only when there is no usable cloud key.
	if !opts.HaveOpenRouter && ollamaReachable {
		useOllama = true
		provider = ollamaProviderID
		chatModel = ollamaBlockModel
		completionModel = ollamaBlockModel
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
		in.Repoint = true // a fresh file pins the chosen model
	} else {
		in.MigrateLegacy = true
		in.LegacyTargets = d.LegacyTargets
		in.LegacyChat = "openrouter/" + d.ChatModel
		in.LegacySmall = "openrouter/" + d.CompletionModel
		// Self-heal repoint only the cloud default or an empty model onto the
		// reachable local provider; an operator-pinned model is left alone.
		if useOllama && (currentProvider == "" || currentProvider == "openrouter") {
			in.Repoint = true
		}
	}

	// Provider block: write/refresh whenever Ollama is reachable; otherwise
	// migrate an existing stale block in place.
	if ollamaReachable {
		in.Ollama = &OllamaProvider{
			BaseURL:    ollamaBaseURL(os.Getenv) + "/api",
			ChatModel:  ollamaBlockModel,
			SmallModel: ollamaBlockModel,
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
	case in.Repoint && useOllama:
		logf("Self-healed OpenCode model -> %s/%s (no OpenRouter key; local Ollama reachable)", provider, chatModel)
	default:
		logf("Updated OpenCode config at %s", opts.ConfigPath)
	}

	// Loud warning when the active provider needs a key we can't resolve and
	// there is no local fallback — otherwise the failure is silent until a run.
	if !useOllama && currentProvider == "openrouter" && !opts.HaveOpenRouter && !ollamaReachable {
		logf("WARNING: OpenCode model %q uses OpenRouter but no OPENROUTER_API_KEY is resolvable and no local Ollama is reachable — runs will fail. Set the key (resource-vault) or start Ollama.", currentModel)
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

// normalize re-parses and re-marshals so a comparison ignores incidental
// whitespace differences between the on-disk file and a fresh render.
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

// ollamaBaseURL normalizes OLLAMA_HOST into scheme://host:port, mirroring the
// daemon-side resolution (bare host, host:port, or full URL; default
// localhost:11434).
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

// --- default Resolver (execs the resource-ollama SSOT) ------------------------

// ExecResolver implements Resolver by shelling out to `resource-ollama`.
type ExecResolver struct {
	Command string // defaults to "resource-ollama"
}

func (r ExecResolver) bin() string {
	if strings.TrimSpace(r.Command) != "" {
		return r.Command
	}
	return "resource-ollama"
}

// InstalledModels runs `resource-ollama models list --json`.
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

// LocalRole runs `resource-ollama policy resolve --role <role> --json`.
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
