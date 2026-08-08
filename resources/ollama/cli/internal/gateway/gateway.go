// Package gateway provides the `resource-ollama gateway ...` subcommand group.
//
// It is the canonical entrypoint scenarios use to talk to the shared Ollama
// daemon — never raw HTTP. Every invocation acquires a host-wide cross-process
// semaphore (sized to OLLAMA_NUM_PARALLEL) before issuing the request, so the
// fleet of scenarios cannot collectively exceed the daemon's parallelism even
// when individual scenarios forget to bound their own concurrency.
package gateway

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil/hostsem"
)

const (
	defaultParallel  = 4
	defaultAcquire   = 60 * time.Second
	charsPerToken    = 4
	envNumParallel   = "OLLAMA_NUM_PARALLEL"
	envAcquireTO     = "OLLAMA_GATEWAY_ACQUIRE_TIMEOUT"
	envLockDir       = "OLLAMA_GATEWAY_LOCK_DIR"
	envRuntimeDir    = "VROOLI_RUNTIME_DIR"
	defaultLockChild = "vrooli/resources/ollama/sem"
)

// Client is the upstream-facing surface used by the gateway handlers. It is
// satisfied by *ensure.Client in production and by fakes in tests.
type Client interface {
	Embed(ctx context.Context, model, input string) ([]float64, error)
	Generate(ctx context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error)
	Chat(ctx context.Context, in ensure.ChatRequest) (ensure.ChatResponse, error)
}

// Handlers owns the runtime dependencies for the gateway subcommand group.
type Handlers struct {
	NewClient func() Client
	Sem       *hostsem.Semaphore
	GetEnv    func(string) string
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
}

// Default returns Handlers wired to the real upstream client and a flock-based
// host semaphore. The semaphore is created lazily so callers that only invoke
// `--help` don't pay any filesystem cost.
func Default() *Handlers {
	h := &Handlers{
		GetEnv: os.Getenv,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	h.NewClient = func() Client { return ensure.NewClient() }
	return h
}

// Commands returns the `gateway` subcommand group for registration.
func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "gateway",
		Description: "Issue throttled requests to the shared Ollama daemon",
		Subcommands: []cliapp.Command{
			{
				Name:        "embed",
				Description: "Compute an embedding vector for --input (or stdin) using --role or --model",
				Usage:       "resource-ollama gateway embed --role embedding.default [--json] [--input <text> | --input-stdin]",
				Run:         h.Embed,
			},
			{
				Name:        "generate",
				Description: "Generate a completion for --prompt (or stdin) using --role or --model",
				Usage:       "resource-ollama gateway generate --role chat.default [--json] [--max-tokens <n>] [--temperature <f>] [--prompt <text> | --prompt-stdin]",
				Run:         h.Generate,
			},
			{
				Name:        "chat",
				Description: "Generate a chat response for --prompt (or stdin) using --role or --model",
				Usage:       "resource-ollama gateway chat --role summarize.default --system <text> [--json] [--think=false] [--max-tokens <n>] [--temperature <f>] [--prompt <text> | --prompt-stdin]",
				Run:         h.Chat,
			},
		},
	}
}

// --- embed --------------------------------------------------------------------

func (h *Handlers) Embed(args []string) error {
	fs := flag.NewFlagSet("gateway embed", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	model := fs.String("model", "", "Ollama model reference (e.g. nomic-embed-text)")
	role := fs.String("role", "", "Ollama model role from model-policy.json (e.g. embedding.default)")
	input := fs.String("input", "", "Inline text to embed")
	fromStdin := fs.Bool("input-stdin", false, "Read input text from stdin")
	asJSON := fs.Bool("json", false, "Emit a single JSON object {\"embedding\":[...]} on stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	selected, err := h.resolveModelSelection(*model, *role, "embedding")
	if err != nil {
		return err
	}
	text, err := h.resolveInput(*input, *fromStdin, "input")
	if err != nil {
		return err
	}

	ctx, release, err := h.acquire(context.Background())
	if err != nil {
		return err
	}
	defer release()

	vec, err := h.NewClient().Embed(ctx, selected.Ref, text)
	if err != nil {
		return err
	}
	if *asJSON {
		return json.NewEncoder(h.Stdout).Encode(struct {
			Embedding []float64 `json:"embedding"`
		}{Embedding: vec})
	}
	parts := make([]string, len(vec))
	for i, f := range vec {
		parts[i] = strconv.FormatFloat(f, 'g', -1, 64)
	}
	_, err = fmt.Fprintln(h.Stdout, strings.Join(parts, " "))
	return err
}

// --- generate -----------------------------------------------------------------

func (h *Handlers) Generate(args []string) error {
	fs := flag.NewFlagSet("gateway generate", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	model := fs.String("model", "", "Ollama model reference")
	role := fs.String("role", "", "Ollama model role from model-policy.json (e.g. chat.default)")
	prompt := fs.String("prompt", "", "Inline prompt text")
	fromStdin := fs.Bool("prompt-stdin", false, "Read prompt from stdin")
	maxTokens := fs.Int("max-tokens", 0, "Maximum tokens to generate; omitted when <= 0")
	temperature := fs.Float64("temperature", -1, "Sampling temperature; omitted when < 0")
	format := fs.String("format", "", "Ollama JSON format (json or a JSON Schema object)")
	asJSON := fs.Bool("json", false, "Emit a single JSON object {\"response\":\"...\",\"eval_count\":0} on stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	selected, err := h.resolveModelSelection(*model, *role, "generate")
	if err != nil {
		return err
	}
	text, err := h.resolveInput(*prompt, *fromStdin, "prompt")
	if err != nil {
		return err
	}
	if err := validateContextWindow(text, *maxTokens, selected); err != nil {
		return err
	}
	var formatJSON json.RawMessage
	if strings.TrimSpace(*format) != "" {
		if strings.TrimSpace(*format) == "json" {
			formatJSON = json.RawMessage(`"json"`)
		} else if !json.Valid([]byte(*format)) {
			return fmt.Errorf("--format must be json or valid JSON Schema")
		} else {
			formatJSON = json.RawMessage(*format)
		}
	}

	ctx, release, err := h.acquire(context.Background())
	if err != nil {
		return err
	}
	defer release()

	// Gateway generation is the structured, visible-output path.  Disable
	// model thinking explicitly so thinking-capable models do not spend the
	// caller's output budget on hidden reasoning and return an empty response.
	think := false
	req := ensure.GenerateRequest{Model: selected.Ref, Prompt: text, Think: &think, Format: formatJSON}
	if *maxTokens > 0 {
		req.NumPredict = maxTokens
	}
	if *temperature >= 0 {
		req.Temperature = temperature
	}
	out, err := h.NewClient().Generate(ctx, req)
	if err != nil {
		return err
	}
	if *asJSON {
		return json.NewEncoder(h.Stdout).Encode(struct {
			Response  string `json:"response"`
			EvalCount int    `json:"eval_count"`
		}{Response: out.Response, EvalCount: out.EvalCount})
	}
	_, err = fmt.Fprint(h.Stdout, out.Response)
	return err
}

// --- chat ---------------------------------------------------------------------

func (h *Handlers) Chat(args []string) error {
	fs := flag.NewFlagSet("gateway chat", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	model := fs.String("model", "", "Ollama model reference")
	role := fs.String("role", "", "Ollama model role from model-policy.json (e.g. summarize.default)")
	system := fs.String("system", "", "System message content")
	prompt := fs.String("prompt", "", "Inline user prompt text")
	fromStdin := fs.Bool("prompt-stdin", false, "Read user prompt from stdin")
	maxTokens := fs.Int("max-tokens", 0, "Maximum tokens to generate; omitted when <= 0")
	temperature := fs.Float64("temperature", -1, "Sampling temperature; omitted when < 0")
	think := fs.Bool("think", false, "Allow model thinking output when supported")
	asJSON := fs.Bool("json", false, "Emit a single JSON object {\"response\":\"...\",\"done_reason\":\"...\",\"eval_count\":0} on stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	selected, err := h.resolveModelSelection(*model, *role, "chat")
	if err != nil {
		return err
	}
	text, err := h.resolveInput(*prompt, *fromStdin, "prompt")
	if err != nil {
		return err
	}
	if err := validateContextWindow(*system+"\n"+text, *maxTokens, selected); err != nil {
		return err
	}

	ctx, release, err := h.acquire(context.Background())
	if err != nil {
		return err
	}
	defer release()

	messages := make([]ensure.ChatMessage, 0, 2)
	if strings.TrimSpace(*system) != "" {
		messages = append(messages, ensure.ChatMessage{Role: "system", Content: *system})
	}
	messages = append(messages, ensure.ChatMessage{Role: "user", Content: text})
	req := ensure.ChatRequest{Model: selected.Ref, Messages: messages, Think: think}
	if *maxTokens > 0 {
		req.NumPredict = maxTokens
	}
	if *temperature >= 0 {
		req.Temperature = temperature
	}
	out, err := h.NewClient().Chat(ctx, req)
	if err != nil {
		return err
	}
	response := out.Message.Content
	if *asJSON {
		return json.NewEncoder(h.Stdout).Encode(struct {
			Response   string `json:"response"`
			DoneReason string `json:"done_reason"`
			EvalCount  int    `json:"eval_count"`
		}{Response: response, DoneReason: out.DoneReason, EvalCount: out.EvalCount})
	}
	_, err = fmt.Fprint(h.Stdout, response)
	return err
}

// --- shared -------------------------------------------------------------------

type selectedModel struct {
	Ref                 string
	Source              string
	ContextWindowTokens int
}

func (h *Handlers) resolveModelSelection(model, role, requiredCapability string) (selectedModel, error) {
	model = strings.TrimSpace(model)
	role = strings.TrimSpace(role)
	if model == "" && role == "" {
		return selectedModel{}, fmt.Errorf("--role or --model is required")
	}
	if model != "" && role != "" {
		return selectedModel{}, fmt.Errorf("--role and --model are mutually exclusive")
	}
	if model != "" {
		selected := selectedModel{Ref: model, Source: "model"}
		p, _, err := policy.LoadDefaultFile(h.GetEnv)
		if err != nil {
			return selected, nil
		}
		resolved, err := p.ResolveModel(model)
		if err != nil {
			return selected, nil
		}
		if !hasCapability(resolved.Capabilities, requiredCapability) {
			return selectedModel{}, fmt.Errorf("model %q does not declare %s capability", model, requiredCapability)
		}
		selected.ContextWindowTokens = resolved.ContextWindowTokens
		return selected, nil
	}

	p, _, err := policy.LoadDefaultFile(h.GetEnv)
	if err != nil {
		return selectedModel{}, err
	}
	resolution, err := p.Resolve(policy.ResolveRequest{
		ModelRoles: []policy.RoleRequest{{Role: role}},
	})
	if err != nil {
		return selectedModel{}, err
	}
	if len(resolution.Models) == 0 {
		return selectedModel{}, fmt.Errorf("role %q resolved no models", role)
	}
	ref := resolution.Models[0].Ref
	modelPolicy, ok := p.Models[ref]
	if !ok {
		return selectedModel{}, fmt.Errorf("role %q resolved unknown model %q", role, ref)
	}
	if !hasCapability(modelPolicy.Capabilities, requiredCapability) {
		return selectedModel{}, fmt.Errorf("role %q resolves to %q without %s capability", role, ref, requiredCapability)
	}
	return selectedModel{
		Ref:                 ref,
		Source:              "role",
		ContextWindowTokens: modelPolicy.ContextWindowTokens,
	}, nil
}

func validateContextWindow(prompt string, maxTokens int, selected selectedModel) error {
	if selected.ContextWindowTokens <= 0 || maxTokens <= 0 {
		return nil
	}
	estimatedPromptTokens := estimatePromptTokens(prompt)
	requestedTokens := estimatedPromptTokens + maxTokens
	if requestedTokens <= selected.ContextWindowTokens {
		return nil
	}
	return fmt.Errorf("request exceeds context window for %s: estimated prompt tokens %d + max_tokens %d = %d, context_window_tokens %d", selected.Ref, estimatedPromptTokens, maxTokens, requestedTokens, selected.ContextWindowTokens)
}

func estimatePromptTokens(text string) int {
	if text == "" {
		return 0
	}
	return (len([]rune(text)) + charsPerToken - 1) / charsPerToken
}

func hasCapability(capabilities []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, capability := range capabilities {
		if strings.TrimSpace(capability) == want {
			return true
		}
	}
	return false
}

func (h *Handlers) resolveInput(inline string, fromStdin bool, name string) (string, error) {
	if inline != "" && fromStdin {
		return "", fmt.Errorf("--%s and --%s-stdin are mutually exclusive", name, name)
	}
	if fromStdin {
		buf, err := io.ReadAll(h.Stdin)
		if err != nil {
			return "", fmt.Errorf("read %s from stdin: %w", name, err)
		}
		return string(buf), nil
	}
	if inline == "" {
		return "", fmt.Errorf("--%s or --%s-stdin is required", name, name)
	}
	return inline, nil
}

// acquire builds a context bounded by the gateway acquire timeout and waits
// for a slot in the host semaphore. The returned release MUST be called.
func (h *Handlers) acquire(parent context.Context) (context.Context, func(), error) {
	timeout := h.acquireTimeout()
	ctx, cancel := context.WithTimeout(parent, timeout)
	if h.Sem == nil {
		sem, err := hostsem.New(h.lockDir(), h.parallelism())
		if err != nil {
			cancel()
			return nil, nil, fmt.Errorf("init host semaphore: %w", err)
		}
		h.Sem = sem
	}
	release, err := h.Sem.Acquire(ctx)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("acquire host semaphore slot: %w", err)
	}
	return ctx, func() {
		release()
		cancel()
	}, nil
}

func (h *Handlers) parallelism() int {
	if v := strings.TrimSpace(h.GetEnv(envNumParallel)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultParallel
}

func (h *Handlers) acquireTimeout() time.Duration {
	if v := strings.TrimSpace(h.GetEnv(envAcquireTO)); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return defaultAcquire
}

func (h *Handlers) lockDir() string {
	if v := strings.TrimSpace(h.GetEnv(envLockDir)); v != "" {
		return v
	}
	if v := strings.TrimSpace(h.GetEnv(envRuntimeDir)); v != "" {
		return filepath.Join(v, "resources", "ollama", "sem")
	}
	if cache, err := os.UserCacheDir(); err == nil && cache != "" {
		return filepath.Join(cache, defaultLockChild)
	}
	return filepath.Join(os.TempDir(), defaultLockChild)
}
