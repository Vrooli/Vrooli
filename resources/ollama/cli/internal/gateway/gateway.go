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
	selectedModel, err := h.resolveModel(*model, *role, "embedding")
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

	vec, err := h.NewClient().Embed(ctx, selectedModel, text)
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
	asJSON := fs.Bool("json", false, "Emit a single JSON object {\"response\":\"...\",\"eval_count\":0} on stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	selectedModel, err := h.resolveModel(*model, *role, "generate")
	if err != nil {
		return err
	}
	text, err := h.resolveInput(*prompt, *fromStdin, "prompt")
	if err != nil {
		return err
	}

	ctx, release, err := h.acquire(context.Background())
	if err != nil {
		return err
	}
	defer release()

	req := ensure.GenerateRequest{Model: selectedModel, Prompt: text}
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

// --- shared -------------------------------------------------------------------

func (h *Handlers) resolveModel(model, role, requiredCapability string) (string, error) {
	model = strings.TrimSpace(model)
	role = strings.TrimSpace(role)
	if model == "" && role == "" {
		return "", fmt.Errorf("--role or --model is required")
	}
	if model != "" && role != "" {
		return "", fmt.Errorf("--role and --model are mutually exclusive")
	}
	if model != "" {
		return model, nil
	}

	p, _, err := policy.LoadDefaultFile(h.GetEnv)
	if err != nil {
		return "", err
	}
	resolution, err := p.Resolve(policy.ResolveRequest{
		ModelRoles: []policy.RoleRequest{{Role: role}},
	})
	if err != nil {
		return "", err
	}
	if len(resolution.Models) == 0 {
		return "", fmt.Errorf("role %q resolved no models", role)
	}
	ref := resolution.Models[0].Ref
	modelPolicy, ok := p.Models[ref]
	if !ok {
		return "", fmt.Errorf("role %q resolved unknown model %q", role, ref)
	}
	if !hasCapability(modelPolicy.Capabilities, requiredCapability) {
		return "", fmt.Errorf("role %q resolves to %q without %s capability", role, ref, requiredCapability)
	}
	return ref, nil
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
