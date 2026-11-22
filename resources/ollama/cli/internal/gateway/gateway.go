// Package gateway provides the `resource-ollama gateway ...` subcommand group.
//
// It is the canonical entrypoint scenarios use to talk to the shared Ollama
// daemon — never raw HTTP. Every invocation acquires a host-wide cross-process
// semaphore (sized to OLLAMA_NUM_PARALLEL) before issuing the request, so the
// fleet of scenarios cannot collectively exceed the daemon's parallelism even
// when individual scenarios forget to bound their own concurrency.
package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/runtimeprofile"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil/hostsem"
)

var (
	defaultParallel        = 1
	defaultAcquire         = tuning.ResourceHTTPTimeout()
	charsPerToken          = 4
	envNumParallel         = "OLLAMA_NUM_PARALLEL"
	envAcquireTO           = "OLLAMA_GATEWAY_ACQUIRE_TIMEOUT"
	envLockDir             = "OLLAMA_GATEWAY_LOCK_DIR"
	envRuntimeDir          = "VROOLI_RUNTIME_DIR"
	defaultLockChild       = "vrooli/resources/ollama/sem"
	defaultMaxImageBytes   = 10 * 1024 * 1024
	defaultMaxImages       = 4
	maxEmbedInputs         = 64
	maxEmbedInputBytes     = 1 << 20
	envMaxImageBytes       = "OLLAMA_GATEWAY_MAX_IMAGE_BYTES"
	envMaxImages           = "OLLAMA_GATEWAY_MAX_IMAGES"
)

// ImageInputError is a typed, caller-actionable image transport error.
type ImageInputError struct {
	Code    string
	Message string
}

func (e *ImageInputError) Error() string {
	return "image_input." + e.Code + ": " + e.Message
}

type imageInputEnvelope struct {
	Prompt string `json:"prompt"`
	Images []struct {
		MediaType string `json:"media_type"`
		DataB64   string `json:"data_b64"`
	} `json:"images"`
}

// Client is the upstream-facing surface used by the gateway handlers. It is
// satisfied by *ensure.Client in production and by fakes in tests.
type Client interface {
	EmbedBatch(ctx context.Context, model string, inputs []string, options map[string]any) (ensure.EmbedResponse, error)
	Generate(ctx context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error)
	Chat(ctx context.Context, in ensure.ChatRequest) (ensure.ChatResponse, error)
}

// workerAttestationClient is implemented by the production Ollama client. It
// lets an inference caller bind the returned text to the active worker Ollama
// reports, without making every gateway operation depend on inventory APIs.
type workerAttestationClient interface {
	ListModelInventory(ctx context.Context) ([]ensure.ModelInfo, error)
	ListRunning(ctx context.Context) ([]ensure.RunningModel, error)
}

type modelInventoryClient interface {
	ListModelInventory(ctx context.Context) ([]ensure.ModelInfo, error)
}

// WorkerAttestation is the resource-owned identity and processor snapshot for
// the model that served one completed generation request.
type WorkerAttestation struct {
	Model         string `json:"model"`
	Digest        string `json:"digest"`
	Processor     string `json:"processor"`
	SizeVRAMBytes int64  `json:"size_vram_bytes"`
	ObservedAt    string `json:"observed_at"`
}

// Handlers owns the runtime dependencies for the gateway subcommand group.
type Handlers struct {
	NewClient         func() Client
	NewLoopbackClient func() (Client, error)
	Sem               *hostsem.Semaphore
	GetEnv            func(string) string
	Stdin             io.Reader
	Stdout            io.Writer
	Stderr            io.Writer
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
	h.NewLoopbackClient = func() (Client, error) { return ensure.NewLoopbackClient() }
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
				Usage:       "resource-ollama gateway embed --role embedding.default [--require-loopback] [--expected-model <ref> --expected-artifact-digest <sha256>] [--expected-rights-decision-sha256 <sha256> --expected-rights-evaluated-at <timestamp> --expected-rights-receipt-sha256 <sha256> --expected-policy-file-sha256 <sha256>] [--json] [--cpu-only] [--input <text> | --input-stdin | --input-json-stdin]",
				Run:         h.Embed,
			},
			{
				Name:        "generate",
				Description: "Generate a completion for --prompt (or stdin) using --role or --model",
				Usage:       "resource-ollama gateway generate --role chat.default [--json] [--require-loopback] [--expected-model <ref> --expected-artifact-digest <sha256>] [--expected-rights-decision-sha256 <sha256> --expected-rights-evaluated-at <timestamp> --expected-rights-receipt-sha256 <sha256> --expected-policy-file-sha256 <sha256>] [--require-worker-digest <sha256>] [--one-shot --timeout 90s] [--cpu-only] [--max-tokens <n>] [--temperature <f>] [--context-budget <n>] [--prompt <text> | --prompt-stdin | --input-json-stdin]",
				Run:         h.Generate,
			},
			{
				Name:        "chat",
				Description: "Generate a chat response for --prompt (or stdin) using --role or --model",
				Usage:       "resource-ollama gateway chat --role summarize.default --system <text> [--require-loopback] [--expected-model <ref> --expected-artifact-digest <sha256>] [--expected-rights-decision-sha256 <sha256> --expected-rights-evaluated-at <timestamp> --expected-rights-receipt-sha256 <sha256> --expected-policy-file-sha256 <sha256>] [--json] [--think=false] [--max-tokens <n>] [--temperature <f>] [--context-budget <n>] [--prompt <text> | --prompt-stdin]",
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
	expectedModel := fs.String("expected-model", "", "Require role resolution to match this exact model reference")
	expectedArtifactDigest := fs.String("expected-artifact-digest", "", "Require current Ollama model contents to match this SHA-256 digest")
	expectedRightsDecision := fs.String("expected-rights-decision-sha256", "", "Require current resource rights facts/context to match this gateway decision SHA-256")
	expectedRightsEvaluatedAt := fs.String("expected-rights-evaluated-at", "", "Evaluation timestamp paired with --expected-rights-decision-sha256")
	expectedRightsReceipt := fs.String("expected-rights-receipt-sha256", "", "Require the current canonical rights receipt SHA-256")
	expectedPolicyFileSHA256 := fs.String("expected-policy-file-sha256", "", "Require the exact current Ollama model-policy file SHA-256")
	fs.Bool("require-loopback", false, "Compatibility assertion; gateway inference always requires verified loopback transport")
	input := fs.String("input", "", "Inline text to embed")
	fromStdin := fs.Bool("input-stdin", false, "Read input text from stdin")
	inputJSONStdin := fs.Bool("input-json-stdin", false, "Read {\"inputs\":[...]} from stdin for a bounded batch")
	cpuOnly := fs.Bool("cpu-only", false, "Run on the runtime profile's cpu-only variant (replaces a loaded GPU runner for this model)")
	asJSON := fs.Bool("json", false, "Emit a single JSON object {\"embedding\":[...]} on stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	selected, err := h.resolveModelSelection(*model, *role, "embedding")
	if err != nil {
		return err
	}
	if err := requireExpectedModel(*expectedModel, selected.Ref); err != nil {
		return err
	}
	if err := policy.CheckExpectedLocalInferenceEvidence(h.GetEnv, selected.Role, selected.Ref, *expectedRightsDecision, *expectedRightsEvaluatedAt, *expectedRightsReceipt, *expectedPolicyFileSHA256); err != nil {
		return err
	}
	inputs, err := h.resolveEmbedInputs(*input, *fromStdin, *inputJSONStdin)
	if err != nil {
		return err
	}
	if len(inputs) > 1 && !*asJSON {
		return fmt.Errorf("batch embedding requires --json")
	}
	profile := runtimeprofile.Current()
	options, err := profile.GatewayOptions(selected.Ref, variantFor(*cpuOnly))
	if err != nil {
		return err
	}
	inputLimit := profile.ContextTokens(selected.Ref)

	admissionStarted := time.Now()
	ctx, release, err := h.acquire(context.Background())
	if err != nil {
		return err
	}
	defer release()
	admissionWaitNS := time.Since(admissionStarted).Nanoseconds()

	client, err := h.newLoopbackClient()
	if err != nil {
		return err
	}
	// Re-read current rights after request preparation and semaphore acquisition,
	// directly before the provider call.
	if err := h.requireLocalRights(ctx, selected, client, *expectedArtifactDigest, *expectedRightsDecision, *expectedRightsEvaluatedAt, *expectedRightsReceipt, *expectedPolicyFileSHA256); err != nil {
		return err
	}
	providerStarted := time.Now()
	// Inputs longer than the runner context are truncated at the tail by
	// Ollama (runtime profile embedding overflow policy), so a long document
	// embeds its head instead of failing on every retry.
	runtimeResponse, err := client.EmbedBatch(ctx, selected.Ref, inputs, options)
	if err != nil {
		return err
	}
	vectors := runtimeResponse.Embeddings
	if len(vectors) != len(inputs) {
		return fmt.Errorf("embed: response vector count %d differs from input count %d", len(vectors), len(inputs))
	}
	providerDurationNS := time.Since(providerStarted).Nanoseconds()
	if *asJSON {
		out := struct {
			Embedding              []float64   `json:"embedding,omitempty"`
			Embeddings             [][]float64 `json:"embeddings,omitempty"`
			InputLimitTokens       int         `json:"input_limit_tokens"`
			Truncated              *bool       `json:"truncated,omitempty"`
			AdmissionWaitNS        int64       `json:"admission_wait_ns"`
			ProviderDurationNS     int64       `json:"provider_duration_ns"`
			RuntimeTotalDurationNS int64       `json:"runtime_total_duration_ns,omitempty"`
			RuntimeLoadDurationNS  int64       `json:"runtime_load_duration_ns,omitempty"`
			RuntimePromptEvalCount int         `json:"runtime_prompt_eval_count,omitempty"`
		}{InputLimitTokens: inputLimit, AdmissionWaitNS: admissionWaitNS, ProviderDurationNS: providerDurationNS}
		if len(vectors) == 1 {
			out.Embedding = vectors[0]
			truncated := runtimeResponse.PromptEvalCount >= inputLimit
			out.Truncated = &truncated
		} else {
			out.Embeddings = vectors
		}
		out.RuntimeTotalDurationNS = runtimeResponse.TotalDuration
		out.RuntimeLoadDurationNS = runtimeResponse.LoadDuration
		out.RuntimePromptEvalCount = runtimeResponse.PromptEvalCount
		return cliout.NewCompactEncoder(h.Stdout).Encode(out)
	}
	parts := make([]string, len(vectors[0]))
	for i, f := range vectors[0] {
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
	expectedModel := fs.String("expected-model", "", "Require role resolution to match this exact model reference")
	expectedArtifactDigest := fs.String("expected-artifact-digest", "", "Require current Ollama model contents to match this SHA-256 digest")
	expectedRightsDecision := fs.String("expected-rights-decision-sha256", "", "Require current resource rights facts/context to match this gateway decision SHA-256")
	expectedRightsEvaluatedAt := fs.String("expected-rights-evaluated-at", "", "Evaluation timestamp paired with --expected-rights-decision-sha256")
	expectedRightsReceipt := fs.String("expected-rights-receipt-sha256", "", "Require the current canonical rights receipt SHA-256")
	expectedPolicyFileSHA256 := fs.String("expected-policy-file-sha256", "", "Require the exact current Ollama model-policy file SHA-256")
	prompt := fs.String("prompt", "", "Inline prompt text")
	fromStdin := fs.Bool("prompt-stdin", false, "Read prompt from stdin")
	inputJSONStdin := fs.Bool("input-json-stdin", false, "Read {prompt,images:[{media_type,data_b64}]} from stdin")
	fs.Bool("require-loopback", false, "Compatibility assertion; gateway inference always requires verified loopback transport")
	requireWorkerDigest := fs.String("require-worker-digest", "", "Require the installed and active GPU worker to match this SHA-256 digest; include worker attestation in JSON output")
	cpuOnly := fs.Bool("cpu-only", false, "Force Ollama to run this request without GPU layers")
	maxTokens := fs.Int("max-tokens", 0, "Maximum tokens to generate; omitted when <= 0")
	temperature := fs.Float64("temperature", -1, "Sampling temperature; omitted when < 0")
	contextBudget := fs.Int("context-budget", 0, "Refuse the request unless estimated prompt plus max tokens fit in n tokens; n may not exceed the runner context, which the runtime profile owns")
	oneShot := fs.Bool("one-shot", false, "Require target model absent before generation, unload after, and verify it left /api/ps")
	requestTimeout := fs.Duration("timeout", 0, "Generation timeout (e.g. 90s); requires --one-shot")
	format := fs.String("format", "", "Ollama JSON format (json or a JSON Schema object)")
	asJSON := fs.Bool("json", false, "Emit a single JSON object {\"response\":\"...\",\"eval_count\":0} on stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var err error
	selected, err := h.resolveModelSelection(*model, *role, "generate")
	if err != nil {
		return err
	}
	if err := requireExpectedModel(*expectedModel, selected.Ref); err != nil {
		return err
	}
	if err := policy.CheckExpectedLocalInferenceEvidence(h.GetEnv, selected.Role, selected.Ref, *expectedRightsDecision, *expectedRightsEvaluatedAt, *expectedRightsReceipt, *expectedPolicyFileSHA256); err != nil {
		return err
	}
	client, err := h.newLoopbackClient()
	if err != nil {
		return err
	}
	if strings.TrimSpace(*requireWorkerDigest) != "" && !*asJSON {
		return fmt.Errorf("--require-worker-digest requires --json")
	}
	var text string
	var images []string
	if *inputJSONStdin {
		if *prompt != "" || *fromStdin {
			return fmt.Errorf("--input-json-stdin cannot be combined with --prompt or --prompt-stdin")
		}
		text, images, err = h.resolveImageInput(selected)
	} else {
		text, err = h.resolveInput(*prompt, *fromStdin, "prompt")
	}
	if err != nil {
		return err
	}
	budget := effectiveMaxTokens(*maxTokens, selected)
	window, err := admissionContext(*contextBudget, selected)
	if err != nil {
		return err
	}
	if err := validateContextWindow(text, budget, selected, window); err != nil {
		return err
	}
	if *requestTimeout < 0 {
		return fmt.Errorf("--timeout cannot be negative")
	}
	if *requestTimeout > 0 && !*oneShot {
		return fmt.Errorf("--timeout requires --one-shot so cleanup is verified")
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
	if *oneShot {
		attestor, ok := client.(workerAttestationClient)
		if !ok {
			return fmt.Errorf("one-shot worker inventory is unavailable")
		}
		running, listErr := attestor.ListRunning(ctx)
		if listErr != nil {
			return fmt.Errorf("check one-shot target before generation: %w", listErr)
		}
		if containsRunningModel(running, selected.Ref) {
			return fmt.Errorf("one-shot target %s is already loaded; refusing to change another owner's worker lifetime", selected.Ref)
		}
	}
	var attestor workerAttestationClient
	if expected := strings.TrimSpace(*requireWorkerDigest); expected != "" {
		var ok bool
		attestor, ok = client.(workerAttestationClient)
		if !ok {
			return fmt.Errorf("active worker attestation is unavailable")
		}
		if err := verifyInstalledDigest(ctx, attestor, selected.Ref, expected); err != nil {
			return err
		}
	}

	// Gateway generation is the structured, visible-output path.  Disable
	// model thinking explicitly so thinking-capable models do not spend the
	// caller's output budget on hidden reasoning and return an empty response.
	think := false
	req := ensure.GenerateRequest{Model: selected.Ref, Prompt: text, Images: images, Think: &think, Format: formatJSON}
	if req.RuntimeOptions, err = runtimeprofile.Current().GatewayOptions(selected.Ref, variantFor(*cpuOnly)); err != nil {
		return err
	}
	if budget > 0 {
		req.NumPredict = &budget
	}
	if *temperature >= 0 {
		req.Temperature = temperature
	}
	if *oneShot {
		keepAlive := 0
		req.KeepAlive = &keepAlive
	}
	requestCtx := ctx
	cancel := func() {}
	if *requestTimeout > 0 {
		requestParent := ctx
		if *oneShot {
			requestParent = context.WithoutCancel(ctx)
		}
		requestCtx, cancel = context.WithTimeout(requestParent, *requestTimeout)
	}
	if err := h.requireLocalRights(requestCtx, selected, client, *expectedArtifactDigest, *expectedRightsDecision, *expectedRightsEvaluatedAt, *expectedRightsReceipt, *expectedPolicyFileSHA256); err != nil {
		cancel()
		return err
	}
	out, generationErr := client.Generate(requestCtx, req)
	var worker *WorkerAttestation
	var attestationErr error
	if generationErr == nil {
		if expected := strings.TrimSpace(*requireWorkerDigest); expected != "" {
			worker, attestationErr = attestRunningWorker(requestCtx, attestor, selected.Ref, expected)
		}
	}
	cancel()
	cleanupVerified := false
	var cleanupErr error
	if *oneShot {
		cleanupVerified, cleanupErr = ensureOneShotWorkerGone(selected.Ref, client)
	}
	if generationErr != nil || attestationErr != nil || cleanupErr != nil {
		return errors.Join(generationErr, attestationErr, cleanupErr)
	}
	if *asJSON {
		return cliout.NewCompactEncoder(h.Stdout).Encode(struct {
			Response              string             `json:"response"`
			EvalCount             int                `json:"eval_count"`
			TotalDuration         int64              `json:"total_duration_ns,omitempty"`
			LoadDuration          int64              `json:"load_duration_ns,omitempty"`
			PromptEvalCount       int                `json:"prompt_eval_count,omitempty"`
			PromptEvalDuration    int64              `json:"prompt_eval_duration_ns,omitempty"`
			EvalDuration          int64              `json:"eval_duration_ns,omitempty"`
			Done                  *bool              `json:"done,omitempty"`
			DoneReason            *string            `json:"done_reason,omitempty"`
			Model                 *string            `json:"model,omitempty"`
			Worker                *WorkerAttestation `json:"worker,omitempty"`
			WorkerCleanupVerified bool               `json:"worker_cleanup_verified,omitempty"`
		}{Response: out.Response, EvalCount: out.EvalCount, TotalDuration: out.TotalDuration, LoadDuration: out.LoadDuration, PromptEvalCount: out.PromptEvalCount, PromptEvalDuration: out.PromptEvalDuration, EvalDuration: out.EvalDuration, Done: out.Done, DoneReason: out.DoneReason, Model: out.Model, Worker: worker, WorkerCleanupVerified: cleanupVerified})
	}
	_, err = fmt.Fprint(h.Stdout, out.Response)
	return err
}

func requireExpectedModel(expected, selected string) error {
	expected = strings.TrimSpace(expected)
	if expected != "" && expected != strings.TrimSpace(selected) {
		return fmt.Errorf("resolved model %q does not match expected model %q", selected, expected)
	}
	return nil
}

func (h *Handlers) newLoopbackClient() (Client, error) {
	if h.NewLoopbackClient == nil {
		return nil, fmt.Errorf("local-only endpoint verification is unavailable")
	}
	client, err := h.NewLoopbackClient()
	if err != nil {
		return nil, fmt.Errorf("verify loopback Ollama endpoint: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("verified loopback Ollama client is unavailable")
	}
	return client, nil
}

func containsRunningModel(running []ensure.RunningModel, model string) bool {
	for _, item := range running {
		ref := item.Name
		if ref == "" {
			ref = item.Model
		}
		if normalizeModelRef(ref) == normalizeModelRef(model) {
			return true
		}
	}
	return false
}

type unloadClient interface {
	Unload(context.Context, string) error
	ListRunning(context.Context) ([]ensure.RunningModel, error)
}

func ensureOneShotWorkerGone(model string, client Client) (bool, error) {
	unloader, ok := client.(unloadClient)
	if !ok {
		return false, fmt.Errorf("one-shot cleanup interface is unavailable")
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	running, err := unloader.ListRunning(cleanupCtx)
	if err != nil {
		return false, fmt.Errorf("verify one-shot worker release: %w", err)
	}
	if !containsRunningModel(running, model) {
		return true, nil
	}
	if err := unloader.Unload(cleanupCtx, model); err != nil {
		return false, fmt.Errorf("unload one-shot worker %s: %w", model, err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		running, err = unloader.ListRunning(cleanupCtx)
		if err != nil {
			return false, fmt.Errorf("verify one-shot worker release: %w", err)
		}
		if !containsRunningModel(running, model) {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, fmt.Errorf("one-shot worker %s remains loaded after unload", model)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func verifyInstalledDigest(ctx context.Context, client workerAttestationClient, model, expected string) error {
	installed, err := client.ListModelInventory(ctx)
	if err != nil {
		return fmt.Errorf("attest installed model before generation: %w", err)
	}
	for _, item := range installed {
		if normalizeModelRef(item.Name) != normalizeModelRef(model) {
			continue
		}
		if normalizeDigest(item.Digest) != normalizeDigest(expected) {
			return fmt.Errorf("installed model digest mismatch for %s", model)
		}
		return nil
	}
	return fmt.Errorf("attest installed model before generation: model %s is not installed", model)
}

func attestRunningWorker(ctx context.Context, client workerAttestationClient, model, expected string) (*WorkerAttestation, error) {
	running, err := client.ListRunning(ctx)
	if err != nil {
		return nil, fmt.Errorf("attest active Ollama worker after generation: %w", err)
	}
	for _, item := range running {
		ref := item.Name
		if ref == "" {
			ref = item.Model
		}
		if normalizeModelRef(ref) != normalizeModelRef(model) {
			continue
		}
		if normalizeDigest(item.Digest) != normalizeDigest(expected) {
			return nil, fmt.Errorf("active worker digest mismatch for %s", model)
		}
		processor := strings.ToLower(strings.TrimSpace(item.Processor))
		if strings.Contains(processor, "cpu") || item.SizeVRAM <= 0 {
			return nil, fmt.Errorf("active worker for %s is not verified GPU-resident", model)
		}
		if processor == "" {
			processor = "gpu (inferred from nonzero size_vram)"
		}
		return &WorkerAttestation{
			Model:         ref,
			Digest:        item.Digest,
			Processor:     processor,
			SizeVRAMBytes: item.SizeVRAM,
			ObservedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		}, nil
	}
	return nil, fmt.Errorf("active worker for %s was not found after generation", model)
}

func normalizeModelRef(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if !strings.Contains(value, ":") && value != "" {
		value += ":latest"
	}
	return value
}

func normalizeDigest(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.TrimPrefix(value, "sha256:")
}

// --- chat ---------------------------------------------------------------------

func (h *Handlers) Chat(args []string) error {
	fs := flag.NewFlagSet("gateway chat", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	model := fs.String("model", "", "Ollama model reference")
	role := fs.String("role", "", "Ollama model role from model-policy.json (e.g. summarize.default)")
	expectedModel := fs.String("expected-model", "", "Require role resolution to match this exact model reference")
	fs.Bool("require-loopback", false, "Compatibility assertion; gateway inference always requires verified loopback transport")
	expectedArtifactDigest := fs.String("expected-artifact-digest", "", "Require current Ollama model contents to match this SHA-256 digest")
	expectedRightsDecision := fs.String("expected-rights-decision-sha256", "", "Require current resource rights facts/context to match this gateway decision SHA-256")
	expectedRightsEvaluatedAt := fs.String("expected-rights-evaluated-at", "", "Evaluation timestamp paired with --expected-rights-decision-sha256")
	expectedRightsReceipt := fs.String("expected-rights-receipt-sha256", "", "Require the current canonical rights receipt SHA-256")
	expectedPolicyFileSHA256 := fs.String("expected-policy-file-sha256", "", "Require the exact current Ollama model-policy file SHA-256")
	system := fs.String("system", "", "System message content")
	prompt := fs.String("prompt", "", "Inline user prompt text")
	fromStdin := fs.Bool("prompt-stdin", false, "Read user prompt from stdin")
	maxTokens := fs.Int("max-tokens", 0, "Maximum tokens to generate; omitted when <= 0")
	temperature := fs.Float64("temperature", -1, "Sampling temperature; omitted when < 0")
	contextBudget := fs.Int("context-budget", 0, "Refuse the request unless estimated prompt plus max tokens fit in n tokens; n may not exceed the runner context, which the runtime profile owns")
	think := fs.Bool("think", false, "Allow model thinking output when supported")
	asJSON := fs.Bool("json", false, "Emit a single JSON object {\"response\":\"...\",\"done_reason\":\"...\",\"eval_count\":0} on stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	selected, err := h.resolveModelSelection(*model, *role, "chat")
	if err != nil {
		return err
	}
	if err := requireExpectedModel(*expectedModel, selected.Ref); err != nil {
		return err
	}
	if err := policy.CheckExpectedLocalInferenceEvidence(h.GetEnv, selected.Role, selected.Ref, *expectedRightsDecision, *expectedRightsEvaluatedAt, *expectedRightsReceipt, *expectedPolicyFileSHA256); err != nil {
		return err
	}
	text, err := h.resolveInput(*prompt, *fromStdin, "prompt")
	if err != nil {
		return err
	}
	budget := effectiveMaxTokens(*maxTokens, selected)
	window, err := admissionContext(*contextBudget, selected)
	if err != nil {
		return err
	}
	if err := validateContextWindow(*system+"\n"+text, budget, selected, window); err != nil {
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
	if req.RuntimeOptions, err = runtimeprofile.Current().GatewayOptions(selected.Ref, ""); err != nil {
		return err
	}
	if budget > 0 {
		req.NumPredict = &budget
	}
	if *temperature >= 0 {
		req.Temperature = temperature
	}
	client, err := h.newLoopbackClient()
	if err != nil {
		return err
	}
	if err := h.requireLocalRights(ctx, selected, client, *expectedArtifactDigest, *expectedRightsDecision, *expectedRightsEvaluatedAt, *expectedRightsReceipt, *expectedPolicyFileSHA256); err != nil {
		return err
	}
	out, err := client.Chat(ctx, req)
	if err != nil {
		return err
	}
	response := out.Message.Content
	if *asJSON {
		return cliout.NewCompactEncoder(h.Stdout).Encode(struct {
			Response           string `json:"response"`
			DoneReason         string `json:"done_reason"`
			EvalCount          int    `json:"eval_count"`
			TotalDuration      int64  `json:"total_duration_ns,omitempty"`
			LoadDuration       int64  `json:"load_duration_ns,omitempty"`
			PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
			PromptEvalDuration int64  `json:"prompt_eval_duration_ns,omitempty"`
			EvalDuration       int64  `json:"eval_duration_ns,omitempty"`
		}{Response: response, DoneReason: out.DoneReason, EvalCount: out.EvalCount, TotalDuration: out.TotalDuration, LoadDuration: out.LoadDuration, PromptEvalCount: out.PromptEvalCount, PromptEvalDuration: out.PromptEvalDuration, EvalDuration: out.EvalDuration})
	}
	_, err = fmt.Fprint(h.Stdout, response)
	return err
}

// --- shared -------------------------------------------------------------------

type selectedModel struct {
	Ref                 string
	Source              string
	Role                string
	ContextWindowTokens int
	InputModalities     []policy.Modality
	// RoleMaxTokens is the resolved role's declared output cap, or nil when the
	// role declares none (or when the caller bypassed roles with --model).
	RoleMaxTokens *int
}

// effectiveMaxTokens resolves the output cap: an explicit --max-tokens wins, an
// absent flag falls through to the role's declared cap, and absent both the
// request carries no num_predict at all and generation is bounded only by the
// context window. Resolving before validateContextWindow matters — the guard
// must reserve the budget that will actually be sent, not the flag alone.
func effectiveMaxTokens(flag int, selected selectedModel) int {
	if flag > 0 {
		return flag
	}
	if selected.RoleMaxTokens != nil {
		return *selected.RoleMaxTokens
	}
	return 0
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
		p, _, err := policy.LoadDefaultFile(h.GetEnv)
		if err != nil {
			return selectedModel{}, fmt.Errorf("load Ollama model policy for direct model %q: %w", model, err)
		}
		resolved, err := p.ResolveModelReference(model)
		if err != nil {
			return selectedModel{}, err
		}
		if !hasCapability(resolved.Capabilities, requiredCapability) {
			return selectedModel{}, fmt.Errorf("model %q does not declare %s capability", model, requiredCapability)
		}
		selected := selectedModel{Ref: resolved.Model, Source: "model", ContextWindowTokens: resolved.ContextWindowTokens}
		selected.InputModalities = append([]policy.Modality{}, resolved.Modalities.Input...)
		return selected, nil
	}

	p, _, err := policy.LoadDefaultFile(h.GetEnv)
	if err != nil {
		return selectedModel{}, err
	}
	context, err := policy.LoadDefaultRightsContext(h.GetEnv)
	if err != nil {
		return selectedModel{}, fmt.Errorf("load trusted rights context for role %q: %w", role, err)
	}
	resolved, err := p.ResolveRoleWithRightsFallback(role, sharedpolicy.RightsUseLocalInference, context, time.Now())
	if err != nil {
		return selectedModel{}, err
	}
	if !hasCapability(resolved.Capabilities, requiredCapability) {
		return selectedModel{}, fmt.Errorf("role %q resolves to %q without %s capability", role, resolved.Model, requiredCapability)
	}
	return selectedModel{
		Ref:                 resolved.Model,
		Source:              "role",
		Role:                role,
		ContextWindowTokens: resolved.ContextWindowTokens,
		InputModalities:     append([]policy.Modality{}, resolved.Modalities.Input...),
		RoleMaxTokens:       resolved.MaxTokens,
	}, nil
}

func (h *Handlers) requireLocalRights(ctx context.Context, selected selectedModel, client Client, expectedDigest, expectedRightsDecision, expectedRightsEvaluatedAt, expectedRightsReceipt, expectedPolicyFileSHA256 string) error {
	inventoryClient, ok := client.(modelInventoryClient)
	if !ok {
		return fmt.Errorf("current Ollama model inventory is unavailable; refusing local inference")
	}
	models, err := inventoryClient.ListModelInventory(ctx)
	if err != nil {
		return fmt.Errorf("read current Ollama model inventory: %w", err)
	}
	var digest string
	for _, model := range models {
		if model.Name != selected.Ref {
			continue
		}
		if digest != "" {
			return fmt.Errorf("current Ollama inventory contains duplicate model identity %q", selected.Ref)
		}
		digest = model.Digest
	}
	if strings.TrimSpace(expectedDigest) != "" && digest != strings.TrimSpace(expectedDigest) {
		return fmt.Errorf("local model %q changed from the planned artifact digest; refusing inference", selected.Ref)
	}
	if strings.TrimSpace(expectedRightsReceipt) != "" || strings.TrimSpace(expectedPolicyFileSHA256) != "" {
		return policy.AuthorizeLocalInferenceArtifactEvidence(h.GetEnv, selected.Role, selected.Ref, digest, expectedRightsDecision, expectedRightsEvaluatedAt, expectedRightsReceipt, expectedPolicyFileSHA256)
	}
	return policy.AuthorizeLocalInferenceArtifactDecision(h.GetEnv, selected.Role, selected.Ref, digest, expectedRightsDecision, expectedRightsEvaluatedAt)
}

func (h *Handlers) resolveImageInput(selected selectedModel) (string, []string, error) {
	maxBytes := h.maxImageBytes()
	maxImages := h.maxImages()
	limit := int64(maxBytes*2 + 64*1024)
	data, err := io.ReadAll(io.LimitReader(h.Stdin, limit))
	if err != nil {
		return "", nil, fmt.Errorf("read input JSON: %w", err)
	}
	if int64(len(data)) >= limit {
		return "", nil, &ImageInputError{Code: "request_too_large", Message: fmt.Sprintf("JSON envelope exceeds %d bytes", limit)}
	}
	var envelope imageInputEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", nil, &ImageInputError{Code: "invalid_json", Message: err.Error()}
	}
	if strings.TrimSpace(envelope.Prompt) == "" {
		return "", nil, &ImageInputError{Code: "prompt_required", Message: "prompt must not be empty"}
	}
	if len(envelope.Images) == 0 {
		return envelope.Prompt, nil, nil
	}
	if len(envelope.Images) > maxImages {
		return "", nil, &ImageInputError{Code: "too_many_images", Message: fmt.Sprintf("received %d images; maximum is %d", len(envelope.Images), maxImages)}
	}
	if len(selected.InputModalities) > 0 && !containsModality(selected.InputModalities, policy.ModalityImage) {
		return "", nil, &ImageInputError{Code: "capability_mismatch", Message: fmt.Sprintf("model %q does not declare image input modality", selected.Ref)}
	}
	images := make([]string, 0, len(envelope.Images))
	for i, image := range envelope.Images {
		mediaType := strings.ToLower(strings.TrimSpace(image.MediaType))
		if mediaType != "image/png" && mediaType != "image/jpeg" && mediaType != "image/webp" {
			return "", nil, &ImageInputError{Code: "unsupported_media_type", Message: fmt.Sprintf("image %d has unsupported media_type %q", i, image.MediaType)}
		}
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(image.DataB64))
		if err != nil {
			return "", nil, &ImageInputError{Code: "invalid_base64", Message: fmt.Sprintf("image %d: %v", i, err)}
		}
		if len(decoded) == 0 {
			return "", nil, &ImageInputError{Code: "empty_image", Message: fmt.Sprintf("image %d is empty", i)}
		}
		if len(decoded) > maxBytes {
			return "", nil, &ImageInputError{Code: "image_too_large", Message: fmt.Sprintf("image %d is %d bytes; maximum is %d", i, len(decoded), maxBytes)}
		}
		images = append(images, base64.StdEncoding.EncodeToString(decoded))
	}
	return envelope.Prompt, images, nil
}

func containsModality(values []policy.Modality, want policy.Modality) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (h *Handlers) maxImageBytes() int {
	if n, err := strconv.Atoi(strings.TrimSpace(h.GetEnv(envMaxImageBytes))); err == nil && n > 0 {
		return n
	}
	return defaultMaxImageBytes
}

func (h *Handlers) maxImages() int {
	if n, err := strconv.Atoi(strings.TrimSpace(h.GetEnv(envMaxImages))); err == nil && n > 0 {
		return n
	}
	return defaultMaxImages
}

// variantFor names the runtime profile variant a gateway flag selects.
func variantFor(cpuOnly bool) string {
	if cpuOnly {
		return runtimeprofile.VariantCPUOnly
	}
	return ""
}

// admissionContext returns the caller's context budget for admission checks,
// or nil to use the model policy window. A budget never reaches Ollama: the
// runner context is owned by the runtime profile so every caller shares one
// runner, and a budget above it would be silently truncated by the runner.
func admissionContext(budget int, selected selectedModel) (*int, error) {
	if budget < 0 {
		return nil, fmt.Errorf("--context-budget cannot be negative")
	}
	if budget == 0 {
		return nil, nil
	}
	runner := runtimeprofile.Current().ContextTokens(selected.Ref)
	if budget > runner {
		return nil, fmt.Errorf("--context-budget %d exceeds the %d-token runner context the Ollama runtime profile declares for %s", budget, runner, selected.Ref)
	}
	if selected.ContextWindowTokens > 0 && budget > selected.ContextWindowTokens {
		return nil, fmt.Errorf("--context-budget %d exceeds context window for %s (%d)", budget, selected.Ref, selected.ContextWindowTokens)
	}
	return &budget, nil
}

func validateContextWindow(prompt string, maxTokens int, selected selectedModel, numCtx *int) error {
	contextWindow := selected.ContextWindowTokens
	if numCtx != nil {
		contextWindow = *numCtx
	}
	if contextWindow <= 0 || maxTokens <= 0 {
		return nil
	}
	estimatedPromptTokens := estimatePromptTokens(prompt)
	requestedTokens := estimatedPromptTokens + maxTokens
	if requestedTokens <= contextWindow {
		return nil
	}
	return fmt.Errorf("request exceeds context window for %s: estimated prompt tokens %d + max_tokens %d = %d, context_window_tokens %d", selected.Ref, estimatedPromptTokens, maxTokens, requestedTokens, contextWindow)
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

func (h *Handlers) resolveEmbedInputs(inline string, fromStdin, fromJSONStdin bool) ([]string, error) {
	if fromJSONStdin {
		if inline != "" || fromStdin {
			return nil, fmt.Errorf("--input-json-stdin cannot be combined with --input or --input-stdin")
		}
		encoded, err := io.ReadAll(io.LimitReader(h.Stdin, int64(maxEmbedInputBytes+1)))
		if err != nil {
			return nil, fmt.Errorf("read embedding batch from stdin: %w", err)
		}
		if len(encoded) > maxEmbedInputBytes {
			return nil, fmt.Errorf("embedding batch exceeds the %d-byte input limit", maxEmbedInputBytes)
		}
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		decoder.DisallowUnknownFields()
		var request struct {
			Inputs []string `json:"inputs"`
		}
		if err := decoder.Decode(&request); err != nil {
			return nil, fmt.Errorf("decode embedding batch from stdin: %w", err)
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			return nil, fmt.Errorf("embedding batch stdin must contain one JSON object")
		}
		if err := validateEmbedInputCount(request.Inputs); err != nil {
			return nil, err
		}
		return request.Inputs, nil
	}
	text, err := h.resolveInput(inline, fromStdin, "input")
	if err != nil {
		return nil, err
	}
	inputs := []string{text}
	if err := validateEmbedInputCount(inputs); err != nil {
		return nil, err
	}
	return inputs, nil
}

func validateEmbedInputCount(inputs []string) error {
	if len(inputs) < 1 || len(inputs) > maxEmbedInputs {
		return fmt.Errorf("embedding input count must be between 1 and %d", maxEmbedInputs)
	}
	totalBytes := 0
	for _, input := range inputs {
		if strings.TrimSpace(input) == "" {
			return fmt.Errorf("embedding inputs must be non-empty")
		}
		totalBytes += len(input)
		if totalBytes > maxEmbedInputBytes {
			return fmt.Errorf("embedding batch exceeds the %d-byte input limit", maxEmbedInputBytes)
		}
	}
	return nil
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
