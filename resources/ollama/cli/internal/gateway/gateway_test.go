package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/runtimeprofile"

	"github.com/vrooli/cli-core/cliutil/hostsem"
)

type fakeClient struct {
	embed        func(ctx context.Context, model, input string) ([]float64, error)
	embedBatch   func(ctx context.Context, model string, inputs []string, options map[string]any) (ensure.EmbedResponse, error)
	generate     func(ctx context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error)
	chat         func(ctx context.Context, in ensure.ChatRequest) (ensure.ChatResponse, error)
	inventory    []ensure.ModelInfo
	running      []ensure.RunningModel
	inventoryErr error
	runningErr   error
	unload       func(context.Context, string) error
}

func TestRequireExpectedModelFailsClosedOnRoleReresolutionChangeAndAllowsDirectMatch(t *testing.T) {
	if err := requireExpectedModel("qwen3.5:9b", "qwen3.5:9b"); err != nil {
		t.Fatalf("exact direct model was rejected: %v", err)
	}
	if err := requireExpectedModel("qwen3.5:9b", "llama3.2:3b"); err == nil {
		t.Fatal("changed role resolution was accepted")
	}
	if err := requireExpectedModel("", "llama3.2:3b"); err != nil {
		t.Fatalf("unbound legacy direct selection changed behavior: %v", err)
	}
}

// EmbedBatch serves the gateway's only embedding path. Tests that care about a
// single vector set embed; it is applied per input.
func (f *fakeClient) EmbedBatch(ctx context.Context, model string, inputs []string, options map[string]any) (ensure.EmbedResponse, error) {
	if f.embedBatch != nil {
		return f.embedBatch(ctx, model, inputs, options)
	}
	if f.embed == nil {
		return ensure.EmbedResponse{}, errors.New("fake embed unavailable")
	}
	var out ensure.EmbedResponse
	for _, input := range inputs {
		vector, err := f.embed(ctx, model, input)
		if err != nil {
			return ensure.EmbedResponse{}, err
		}
		out.Embeddings = append(out.Embeddings, vector)
	}
	return out, nil
}

func (f *fakeClient) Generate(ctx context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
	return f.generate(ctx, in)
}

func (f *fakeClient) Chat(ctx context.Context, in ensure.ChatRequest) (ensure.ChatResponse, error) {
	return f.chat(ctx, in)
}

func (f *fakeClient) ListModelInventory(context.Context) ([]ensure.ModelInfo, error) {
	return f.inventory, f.inventoryErr
}

func (f *fakeClient) ListRunning(context.Context) ([]ensure.RunningModel, error) {
	return f.running, f.runningErr
}

func (f *fakeClient) Unload(ctx context.Context, model string) error {
	if f.unload == nil {
		return errors.New("fake unload unavailable")
	}
	return f.unload(ctx, model)
}

func newHandlers(t *testing.T, client *fakeClient, env map[string]string) (*Handlers, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	if env == nil {
		env = policyEnv(t)
	}
	if len(client.inventory) == 0 && env["OLLAMA_MODEL_POLICY_PATH"] != "" {
		p, _, err := policy.LoadDefaultFile(func(key string) string { return env[key] })
		if err == nil {
			for name, model := range p.Models {
				if model.RuntimeIdentity != nil {
					client.inventory = append(client.inventory, ensure.ModelInfo{Name: name, Digest: model.RuntimeIdentity.Digest})
				}
			}
		}
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	sem, err := hostsem.New(t.TempDir(), 4)
	if err != nil {
		t.Fatalf("hostsem.New: %v", err)
	}
	h := &Handlers{
		NewClient:         func() Client { return client },
		NewLoopbackClient: func() (Client, error) { return client, nil },
		Sem:               sem,
		GetEnv:            func(k string) string { return env[k] },
		Stdin:             strings.NewReader(""),
		Stdout:            stdout,
		Stderr:            stderr,
	}
	return h, stdout, stderr
}

func TestEmbedJSONOutput(t *testing.T) {
	client := &fakeClient{
		embed: func(_ context.Context, model, input string) ([]float64, error) {
			if model != "nomic-embed-text:latest" {
				t.Errorf("model = %q", model)
			}
			if input != "hello" {
				t.Errorf("input = %q", input)
			}
			return []float64{0.1, 0.2, 0.3}, nil
		},
	}
	h, stdout, _ := newHandlers(t, client, nil)
	if err := h.Embed([]string{"--model", "nomic-embed-text", "--input", "hello", "--json"}); err != nil {
		t.Fatalf("Embed: %v", err)
	}
	var got struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode stdout %q: %v", stdout.String(), err)
	}
	if len(got.Embedding) != 3 {
		t.Fatalf("embedding length = %d, want 3 (raw=%q)", len(got.Embedding), stdout.String())
	}
}

func TestEmbedRequireLoopbackUsesVerifiedClient(t *testing.T) {
	var embedded bool
	loopbackClient := &fakeClient{embed: func(context.Context, string, string) ([]float64, error) {
		embedded = true
		return []float64{1}, nil
	}}
	loopbackClient.inventory = []ensure.ModelInfo{{Name: "nomic-embed-text:latest", Digest: strings.Repeat("c", 64)}}
	h, _, _ := newHandlers(t, &fakeClient{}, policyEnv(t))
	h.NewLoopbackClient = func() (Client, error) { return loopbackClient, nil }
	if err := h.Embed([]string{"--model", "nomic-embed-text", "--input", "text", "--require-loopback"}); err != nil {
		t.Fatalf("verified local embedding: %v", err)
	}
	if !embedded {
		t.Fatal("verified loopback client was not used for embedding")
	}
}

func TestEmbedRequireLoopbackRefusesUnverifiableEndpointBeforeProvider(t *testing.T) {
	var normalClientUsed, embedded bool
	client := &fakeClient{embed: func(context.Context, string, string) ([]float64, error) {
		embedded = true
		return []float64{1}, nil
	}}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	h.NewClient = func() Client { normalClientUsed = true; return client }
	h.NewLoopbackClient = func() (Client, error) { return nil, errors.New("endpoint is not a verified loopback") }
	err := h.Embed([]string{"--model", "nomic-embed-text", "--input", "text", "--require-loopback"})
	if err == nil || !strings.Contains(err.Error(), "verified loopback") {
		t.Fatalf("unverifiable endpoint was accepted: %v", err)
	}
	if normalClientUsed || embedded {
		t.Fatalf("unverifiable endpoint used normal client=%t embedding=%t", normalClientUsed, embedded)
	}
}

func TestChatRequireLoopbackRefusesUnverifiableEndpointBeforeProvider(t *testing.T) {
	var normalClientUsed, chatted bool
	client := &fakeClient{chat: func(context.Context, ensure.ChatRequest) (ensure.ChatResponse, error) {
		chatted = true
		return ensure.ChatResponse{}, nil
	}}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	h.NewClient = func() Client { normalClientUsed = true; return client }
	h.NewLoopbackClient = func() (Client, error) { return nil, errors.New("endpoint is not a verified loopback") }
	err := h.Chat([]string{"--role", "chat.default", "--prompt", "text", "--require-loopback"})
	if err == nil || !strings.Contains(err.Error(), "verified loopback") {
		t.Fatalf("unverifiable endpoint was accepted: %v", err)
	}
	if normalClientUsed || chatted {
		t.Fatalf("unverifiable endpoint used normal client=%t chat=%t", normalClientUsed, chatted)
	}
}

func TestGatewayInferenceDefaultsToVerifiedLoopback(t *testing.T) {
	endpoints := []struct {
		name, baseURL, host string
	}{
		{name: "remote base url", baseURL: "http://192.0.2.44:11434"},
		{name: "remote host", host: "192.0.2.44:11434"},
	}
	operations := []struct {
		name string
		args []string
	}{
		{name: "embed", args: []string{"--model", "nomic-embed-text:latest", "--input", "hello"}},
		{name: "generate", args: []string{"--model", "qwen3.5:9b", "--prompt", "hello"}},
		{name: "chat", args: []string{"--model", "qwen3.5:4b", "--prompt", "hello"}},
	}
	for _, endpoint := range endpoints {
		for _, operation := range operations {
			t.Run(endpoint.name+"/"+operation.name, func(t *testing.T) {
				t.Setenv("OLLAMA_BASE_URL", endpoint.baseURL)
				t.Setenv("OLLAMA_HOST", endpoint.host)
				var normalClientConstructions, providerCalls atomic.Int32
				client := &fakeClient{
					embed: func(context.Context, string, string) ([]float64, error) {
						providerCalls.Add(1)
						return []float64{1}, nil
					},
					generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
						providerCalls.Add(1)
						return ensure.GenerateResponse{Response: "unexpected"}, nil
					},
					chat: func(context.Context, ensure.ChatRequest) (ensure.ChatResponse, error) {
						providerCalls.Add(1)
						return ensure.ChatResponse{}, nil
					},
				}
				h, _, _ := newHandlers(t, client, policyEnv(t))
				h.NewClient = func() Client {
					normalClientConstructions.Add(1)
					return client
				}
				h.NewLoopbackClient = func() (Client, error) {
					verified, err := ensure.NewLoopbackClient()
					return verified, err
				}
				var err error
				switch operation.name {
				case "embed":
					err = h.Embed(operation.args)
				case "generate":
					err = h.Generate(operation.args)
				case "chat":
					err = h.Chat(operation.args)
				}
				if err == nil || !strings.Contains(err.Error(), "loopback") {
					t.Fatalf("request without --require-loopback accepted remote endpoint: %v", err)
				}
				if normalClientConstructions.Load() != 0 || providerCalls.Load() != 0 {
					t.Fatalf("remote endpoint escaped guard: ordinary clients=%d provider calls=%d", normalClientConstructions.Load(), providerCalls.Load())
				}
			})
		}
	}
}

func TestRequireLoopbackFalseCannotOptOut(t *testing.T) {
	t.Setenv("OLLAMA_BASE_URL", "http://192.0.2.44:11434")
	t.Setenv("OLLAMA_HOST", "")
	var normalClientConstructions, providerCalls atomic.Int32
	client := &fakeClient{generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
		providerCalls.Add(1)
		return ensure.GenerateResponse{}, nil
	}}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	h.NewClient = func() Client {
		normalClientConstructions.Add(1)
		return client
	}
	h.NewLoopbackClient = func() (Client, error) {
		verified, err := ensure.NewLoopbackClient()
		return verified, err
	}
	err := h.Generate([]string{"--model", "qwen3.5:9b", "--prompt", "hello", "--require-loopback=false"})
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("--require-loopback=false bypassed local-only gateway transport: %v", err)
	}
	if normalClientConstructions.Load() != 0 || providerCalls.Load() != 0 {
		t.Fatalf("explicit false escaped guard: ordinary clients=%d provider calls=%d", normalClientConstructions.Load(), providerCalls.Load())
	}
}

func TestDirectAndRoleRequestsFailClosedWithoutQualifiedRights(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "direct generate", args: []string{"--model", "llama3.2:1b", "--prompt", "hello"}},
		{name: "direct embed", args: []string{"--model", "nomic-embed-text", "--input", "hello"}},
		{name: "role generate", args: []string{"--role", "chat.default", "--prompt", "hello"}},
		{name: "role embed", args: []string{"--role", "embedding.default", "--input", "hello"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := unqualifiedPolicyEnv(t)
			var constructors, providerCalls atomic.Int32
			client := &fakeClient{
				embed: func(context.Context, string, string) ([]float64, error) {
					providerCalls.Add(1)
					return []float64{1}, nil
				},
				generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
					providerCalls.Add(1)
					return ensure.GenerateResponse{Response: "unexpected"}, nil
				},
			}
			h, _, _ := newHandlers(t, client, env)
			h.NewClient = func() Client { constructors.Add(1); return client }
			var err error
			if strings.Contains(tc.name, "embed") {
				err = h.Embed(tc.args)
			} else {
				err = h.Generate(tc.args)
			}
			if err == nil || !strings.Contains(err.Error(), "rights decision unknown") {
				t.Fatalf("unqualified local route was not rejected as unknown: %v", err)
			}
			if constructors.Load() != 0 || providerCalls.Load() != 0 {
				t.Fatalf("unqualified request reached provider construction=%d calls=%d", constructors.Load(), providerCalls.Load())
			}
		})
	}
}

func TestDirectAndRoleChatFailClosedWithoutQualifiedRights(t *testing.T) {
	for _, selection := range []struct {
		name string
		args []string
	}{
		{name: "direct model", args: []string{"--model", "qwen3.5:4b", "--prompt", "hello"}},
		{name: "role", args: []string{"--role", "chat.small", "--prompt", "hello"}},
	} {
		t.Run(selection.name, func(t *testing.T) {
			env := unqualifiedPolicyEnv(t)
			var constructors, calls atomic.Int32
			client := &fakeClient{chat: func(context.Context, ensure.ChatRequest) (ensure.ChatResponse, error) {
				calls.Add(1)
				return ensure.ChatResponse{}, nil
			}}
			h, _, _ := newHandlers(t, client, env)
			h.NewClient = func() Client { constructors.Add(1); return client }
			err := h.Chat(selection.args)
			if err == nil || !strings.Contains(err.Error(), "rights decision unknown") {
				t.Fatalf("unqualified chat returned %v", err)
			}
			if constructors.Load() != 0 || calls.Load() != 0 {
				t.Fatalf("unqualified chat constructed=%d calls=%d", constructors.Load(), calls.Load())
			}
		})
	}
}

func TestDirectAndRoleRequestsProceedOnlyWithQualifiedCurrentEvidence(t *testing.T) {
	var embedCalls, generateCalls atomic.Int32
	client := &fakeClient{
		embed: func(_ context.Context, model, _ string) ([]float64, error) {
			if model != "nomic-embed-text:latest" {
				t.Errorf("resolved direct model = %q", model)
			}
			embedCalls.Add(1)
			return []float64{1}, nil
		},
		generate: func(_ context.Context, request ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			if request.Model != "qwen3.5:9b" {
				t.Errorf("resolved model = %q", request.Model)
			}
			generateCalls.Add(1)
			return ensure.GenerateResponse{Response: "ok"}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Embed([]string{"--model", "nomic-embed-text", "--input", "hello", "--json"}); err != nil {
		t.Fatalf("qualified direct embed: %v", err)
	}
	if err := h.Generate([]string{"--role", "chat.default", "--prompt", "hello", "--json"}); err != nil {
		t.Fatalf("qualified role generate: %v", err)
	}
	if embedCalls.Load() != 1 || generateCalls.Load() != 1 {
		t.Fatalf("provider calls embed=%d generate=%d", embedCalls.Load(), generateCalls.Load())
	}
}

func TestRoleSelectionFallsBackOnCurrentRightsAndRestoresPrimary(t *testing.T) {
	primary, fallback := "llama3.2:1b", "qwen3.5:9b"
	sourcePolicyPath := filepath.Join("..", "..", "..", "model-policy.json")
	sourcePolicyBytes, err := os.ReadFile(sourcePolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	var sourcePolicy map[string]any
	if err := json.Unmarshal(sourcePolicyBytes, &sourcePolicy); err != nil {
		t.Fatal(err)
	}
	actualRights := sourcePolicy["models"].(map[string]any)[primary].(map[string]any)["rights"]
	actualRightsFields := actualRights.(map[string]any)
	env := writeTestPolicy(t, true, func(document map[string]any) {
		roles := document["roles"].(map[string]any)
		role := roles["chat.default"].(map[string]any)
		role["model"] = primary
		role["fallbacks"] = []string{fallback}
		models := document["models"].(map[string]any)
		fallbackModel := models[fallback].(map[string]any)
		rights := fallbackModel["rights"].(sharedpolicy.ArtifactRights)
		rights.Classification = sharedpolicy.RightsUnrestricted
		rights.Conditions = nil
		fallbackModel["rights"] = rights
		primaryModel := models[primary].(map[string]any)
		primaryModel["rights"] = actualRights
		primaryModel["runtime_identity"] = sharedpolicy.RuntimeArtifactIdentity{
			Provider: sharedpolicy.RuntimeProviderOllama, Model: primary,
			ArtifactID: actualRightsFields["artifact_id"].(string), ArtifactRevision: actualRightsFields["revision"].(string),
			DigestKind: sharedpolicy.RuntimeDigestOllamaModelContents, Digest: strings.Repeat("c", 64),
			Source: sharedpolicy.RuntimeSourceOllamaTags, ObservedAt: "2026-11-10",
		}
	})
	canonicalContextPath := filepath.Join("..", "..", "..", "..", "..", "scenarios", "ai-gateway", "config", "model-rights-context.json")
	canonicalContext, err := os.ReadFile(canonicalContextPath)
	if err != nil {
		t.Fatal(err)
	}
	// This test proves license-bound local inference follows the context, so it
	// copies the canonical context without the operator's standing normal-use
	// answer (which would otherwise allow every license class).
	var strictDocument map[string]any
	if err := json.Unmarshal(canonicalContext, &strictDocument); err != nil {
		t.Fatal(err)
	}
	strictFields := strictDocument["context"].(map[string]any)
	delete(strictFields, "normal_use_licensing")
	delete(strictFields, "normal_use_licensing_evidence")
	strictContext, err := json.Marshal(strictDocument)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env["VROOLI_MODEL_RIGHTS_CONTEXT_PATH"], strictContext, 0o600); err != nil {
		t.Fatal(err)
	}
	var served []string
	client := &fakeClient{generate: func(_ context.Context, request ensure.GenerateRequest) (ensure.GenerateResponse, error) {
		served = append(served, request.Model)
		return ensure.GenerateResponse{Response: request.Model}, nil
	}}
	h, _, _ := newHandlers(t, client, env)
	args := []string{"--role", "chat.default", "--prompt", "hello", "--json"}
	if err := h.Generate(args); err != nil {
		t.Fatalf("primary allowed: %v", err)
	}
	setTestMonthlyActiveUsers(t, env, "700000001")
	if err := h.Generate(args); err != nil {
		t.Fatalf("rights-qualified fallback: %v", err)
	}
	setTestMonthlyActiveUsers(t, env, "0")
	// Selection is pinned before client construction. If trusted context crosses
	// the threshold before execution, the exact selected artifact must be
	// refused rather than silently swapped after planning.
	h.NewLoopbackClient = func() (Client, error) {
		setTestMonthlyActiveUsers(t, env, "700000001")
		return client, nil
	}
	if err := h.Generate(args); err == nil || !strings.Contains(err.Error(), "rights decision denied") {
		t.Fatalf("execution-time threshold crossing was not reported as a refusal: %v", err)
	}
	h.NewLoopbackClient = func() (Client, error) { return client, nil }
	setTestMonthlyActiveUsers(t, env, "0")
	if err := h.Generate(args); err != nil {
		t.Fatalf("restored primary: %v", err)
	}
	want := []string{primary, fallback, primary}
	if strings.Join(served, ",") != strings.Join(want, ",") {
		t.Fatalf("models served = %v, want primary/fallback/restored path %v (the execution-race request must not reach the provider)", served, want)
	}
}

func setTestMonthlyActiveUsers(t *testing.T, env map[string]string, users string) {
	t.Helper()
	path := env["VROOLI_MODEL_RIGHTS_CONTEXT_PATH"]
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	context := document["context"].(map[string]any)
	monthly := context["monthly_active_users"].(map[string]any)
	monthly["2024-08"] = users
	updated, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, updated, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLocalRightsAreReevaluatedAfterContextOrArtifactChanges(t *testing.T) {
	env := policyEnv(t)
	var providerCalls atomic.Int32
	client := &fakeClient{generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
		providerCalls.Add(1)
		return ensure.GenerateResponse{Response: "ok"}, nil
	}}
	h, _, _ := newHandlers(t, client, env)
	args := []string{"--model", "llama3.2:1b", "--prompt", "hello"}
	if err := h.Generate(args); err != nil {
		t.Fatalf("qualified request: %v", err)
	}

	contextPath := env["VROOLI_MODEL_RIGHTS_CONTEXT_PATH"]
	changedContext := `{"schema_version":"1","context":{"id":"test-operator-context","version":"test-v2","entity_scope":"different-entity","revenue_status":"pre_revenue","as_of":"2026-11-11","source":"operator_statement"}}`
	if err := os.WriteFile(contextPath, []byte(changedContext), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := h.Generate(args); err == nil || !strings.Contains(err.Error(), "rights decision unknown") {
		t.Fatalf("changed current context was not rejected: %v", err)
	}
	if providerCalls.Load() != 1 {
		t.Fatalf("changed context reached provider; calls=%d", providerCalls.Load())
	}

	if err := os.WriteFile(contextPath, []byte(`{"schema_version":"1","context":{"id":"test-operator-context","version":"test-v1","entity_scope":"test-entity","revenue_status":"pre_revenue","as_of":"2026-11-10","source":"operator_statement"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	modelPolicyPath := env["OLLAMA_MODEL_POLICY_PATH"]
	data, err := os.ReadFile(modelPolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	models := document["models"].(map[string]any)
	model := models["llama3.2:1b"].(map[string]any)
	delete(model, "rights")
	data, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPolicyPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := h.Generate(args); err == nil {
		t.Fatalf("changed current artifact rights were not rejected: %v", err)
	}
	if providerCalls.Load() != 1 {
		t.Fatalf("changed artifact rights reached provider; calls=%d", providerCalls.Load())
	}
}

func TestExpectedGatewayRightsDecisionIsRecheckedAtResourceBoundary(t *testing.T) {
	for _, verb := range []string{"embed", "generate", "chat"} {
		for _, change := range []string{"context version", "artifact policy revision"} {
			t.Run(verb+"/"+change, func(t *testing.T) {
				env := policyEnv(t)
				modelRef := "llama3.2:1b"
				input := "--prompt"
				if verb == "embed" {
					modelRef = "nomic-embed-text:latest"
					input = "--input"
				}
				decisionSHA, evaluatedAt := expectedRightsReceipt(t, env, modelRef)
				var providerCalls atomic.Int32
				client := &fakeClient{
					embed: func(context.Context, string, string) ([]float64, error) {
						providerCalls.Add(1)
						return []float64{1}, nil
					},
					generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
						providerCalls.Add(1)
						return ensure.GenerateResponse{Response: "ok"}, nil
					},
					chat: func(context.Context, ensure.ChatRequest) (ensure.ChatResponse, error) {
						providerCalls.Add(1)
						response := ensure.ChatResponse{}
						response.Message.Content = "ok"
						return response, nil
					},
				}
				h, _, _ := newHandlers(t, client, env)
				h.NewLoopbackClient = func() (Client, error) {
					if change == "context version" {
						contextPath := env["VROOLI_MODEL_RIGHTS_CONTEXT_PATH"]
						content, err := os.ReadFile(contextPath)
						if err != nil {
							return nil, err
						}
						updated := strings.Replace(string(content), "test-v1", "test-v2", 1)
						if err := os.WriteFile(contextPath, []byte(updated), 0o600); err != nil {
							return nil, err
						}
					} else if err := changeFixtureRightsRevision(env["OLLAMA_MODEL_POLICY_PATH"], modelRef); err != nil {
						return nil, err
					}
					return client, nil
				}
				args := []string{
					"--model", modelRef, input, "hello", "--expected-artifact-digest", strings.Repeat("c", 64),
					"--expected-rights-decision-sha256", decisionSHA, "--expected-rights-evaluated-at", evaluatedAt,
				}
				var err error
				switch verb {
				case "embed":
					err = h.Embed(append(args, "--json"))
				case "generate":
					err = h.Generate(append(args, "--json"))
				case "chat":
					args = []string{
						"--model", modelRef, "--system", "system", "--prompt", "hello", "--expected-artifact-digest", strings.Repeat("c", 64),
						"--expected-rights-decision-sha256", decisionSHA, "--expected-rights-evaluated-at", evaluatedAt,
					}
					err = h.Chat(args)
				}
				if err == nil || !strings.Contains(err.Error(), "differs from the expected decision") {
					t.Fatalf("expected stale gateway decision to be refused, got %v", err)
				}
				if providerCalls.Load() != 0 {
					t.Fatalf("stale rights decision reached provider; calls=%d", providerCalls.Load())
				}
			})
		}
	}
}

func expectedRightsReceipt(t *testing.T, env map[string]string, model string) (string, string) {
	t.Helper()
	getenv := func(key string) string { return env[key] }
	p, _, err := policy.LoadDefaultFile(getenv)
	if err != nil {
		t.Fatal(err)
	}
	currentContext, err := policy.LoadDefaultRightsContext(getenv)
	if err != nil {
		t.Fatal(err)
	}
	evaluatedAt := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	decision, err := p.EvaluateModelRights(model, sharedpolicy.RightsUseLocalInference, currentContext, evaluatedAt)
	if err != nil || decision.Status != sharedpolicy.RightsAllowed {
		t.Fatalf("fixture rights decision = %+v, err=%v", decision, err)
	}
	return decision.DecisionSHA256, decision.EvaluatedAt
}

func changeFixtureRightsRevision(path, modelRef string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return err
	}
	model, ok := document["models"].(map[string]any)[modelRef].(map[string]any)
	if !ok {
		return fmt.Errorf("model %q is missing", modelRef)
	}
	rights, ok := model["rights"].(map[string]any)
	if !ok {
		return fmt.Errorf("model %q has no rights fixture", modelRef)
	}
	identity, ok := model["runtime_identity"].(map[string]any)
	if !ok {
		return fmt.Errorf("model %q has no runtime identity fixture", modelRef)
	}
	rights["revision"] = "test-model-v2"
	identity["artifact_revision"] = "test-model-v2"
	updated, err := json.Marshal(document)
	if err != nil {
		return err
	}
	return os.WriteFile(path, updated, 0o600)
}

type mutatingInputReader struct {
	path    string
	content []byte
	fired   bool
}

func (r *mutatingInputReader) Read(p []byte) (int, error) {
	if !r.fired {
		r.fired = true
		if err := os.WriteFile(r.path, r.content, 0o600); err != nil {
			return 0, err
		}
		return copy(p, []byte("hello")), io.EOF
	}
	return 0, io.EOF
}

func TestGenerateRechecksRightsImmediatelyBeforeProviderAfterInputChange(t *testing.T) {
	for _, tc := range []struct {
		name   string
		args   []string
		mutate func(t *testing.T, env map[string]string) (path string, content []byte)
	}{
		{
			name: "trusted context changes",
			mutate: func(t *testing.T, env map[string]string) (string, []byte) {
				return env["VROOLI_MODEL_RIGHTS_CONTEXT_PATH"], []byte(`{"schema_version":"1","context":{"id":"test-operator-context","version":"test-v2","entity_scope":"different-entity","revenue_status":"pre_revenue","as_of":"2026-11-11","source":"operator_statement"}}`)
			},
		},
		{
			name: "catalog rights are removed",
			mutate: func(t *testing.T, env map[string]string) (string, []byte) {
				data, err := os.ReadFile(env["OLLAMA_MODEL_POLICY_PATH"])
				if err != nil {
					t.Fatal(err)
				}
				var document map[string]any
				if err := json.Unmarshal(data, &document); err != nil {
					t.Fatal(err)
				}
				delete(document["models"].(map[string]any)["llama3.2:1b"].(map[string]any), "rights")
				changed, err := json.Marshal(document)
				if err != nil {
					t.Fatal(err)
				}
				return env["OLLAMA_MODEL_POLICY_PATH"], changed
			},
		},
		{
			name: "role resolves to another exact model",
			args: []string{"--role", "chat.default", "--prompt-stdin"},
			mutate: func(t *testing.T, env map[string]string) (string, []byte) {
				data, err := os.ReadFile(env["OLLAMA_MODEL_POLICY_PATH"])
				if err != nil {
					t.Fatal(err)
				}
				var document map[string]any
				if err := json.Unmarshal(data, &document); err != nil {
					t.Fatal(err)
				}
				role := document["roles"].(map[string]any)["chat.default"].(map[string]any)
				role["model"] = "qwen3.5:4b"
				changed, err := json.Marshal(document)
				if err != nil {
					t.Fatal(err)
				}
				return env["OLLAMA_MODEL_POLICY_PATH"], changed
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := policyEnv(t)
			path, content := tc.mutate(t, env)
			var calls atomic.Int32
			client := &fakeClient{generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
				calls.Add(1)
				return ensure.GenerateResponse{Response: "must not escape"}, nil
			}}
			h, _, _ := newHandlers(t, client, env)
			h.Stdin = &mutatingInputReader{path: path, content: content}
			args := tc.args
			if len(args) == 0 {
				args = []string{"--model", "llama3.2:1b", "--prompt-stdin"}
			}
			err := h.Generate(args)
			if err == nil {
				t.Fatalf("stale rights were not rejected before provider call: %v", err)
			}
			if calls.Load() != 0 {
				t.Fatalf("provider was called after rights changed during request preparation: %d", calls.Load())
			}
		})
	}
}

func TestGenerateRequiresCurrentArtifactDigestMatchBeforeProvider(t *testing.T) {
	for _, tc := range []struct {
		name       string
		inventory  []ensure.ModelInfo
		wantErrMsg string
	}{
		{name: "mismatched digest", inventory: []ensure.ModelInfo{{Name: "llama3.2:1b", Digest: strings.Repeat("e", 64)}}, wantErrMsg: "runtime artifact identity binding"},
		{name: "missing digest", inventory: []ensure.ModelInfo{{Name: "llama3.2:1b"}}, wantErrMsg: "runtime artifact identity binding"},
		{name: "missing inventory entry", inventory: nil, wantErrMsg: "runtime artifact identity binding"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := policyEnv(t)
			var calls atomic.Int32
			client := &fakeClient{
				inventory: tc.inventory,
				generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
					calls.Add(1)
					return ensure.GenerateResponse{Response: "must not escape"}, nil
				},
			}
			h, _, _ := newHandlers(t, client, env)
			// Override auto-seeded fixture inventory with the deliberate mismatch.
			client.inventory = tc.inventory
			err := h.Generate([]string{"--model", "llama3.2:1b", "--prompt", "hello"})
			if err == nil || !strings.Contains(err.Error(), tc.wantErrMsg) {
				t.Fatalf("artifact mismatch was not rejected: %v", err)
			}
			if calls.Load() != 0 {
				t.Fatalf("provider called after artifact mismatch: %d", calls.Load())
			}
		})
	}
}

func TestGenerateRejectsPlannedArtifactDigestChangeBeforeProvider(t *testing.T) {
	var calls atomic.Int32
	client := &fakeClient{generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
		calls.Add(1)
		return ensure.GenerateResponse{Response: "must not escape"}, nil
	}}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	err := h.Generate([]string{
		"--model", "llama3.2:1b", "--expected-artifact-digest", strings.Repeat("e", 64), "--prompt", "hello",
	})
	if err == nil || !strings.Contains(err.Error(), "planned artifact digest") {
		t.Fatalf("changed planned artifact digest was accepted: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("provider called after planned artifact digest changed: %d", calls.Load())
	}
}

func TestEmbedRequiresModel(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeClient{}, nil)
	err := h.Embed([]string{"--input", "hi"})
	if err == nil || !strings.Contains(err.Error(), "--role or --model") {
		t.Fatalf("expected model selection error, got %v", err)
	}
}

func TestEmbedRoleResolvesModel(t *testing.T) {
	client := &fakeClient{
		embed: func(_ context.Context, model, input string) ([]float64, error) {
			if model != "nomic-embed-text:latest" {
				t.Errorf("model = %q", model)
			}
			if input != "hello" {
				t.Errorf("input = %q", input)
			}
			return []float64{0.1}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Embed([]string{"--role", "embedding.default", "--input", "hello", "--json"}); err != nil {
		t.Fatalf("Embed: %v", err)
	}
}

func TestEmbedBatchSendsOnlyTheProfileVariantAndAdmissionTimings(t *testing.T) {
	wantOptions, err := runtimeprofile.Current().VariantOptions(runtimeprofile.VariantCPUOnly)
	if err != nil {
		t.Fatal(err)
	}
	client := &fakeClient{embedBatch: func(_ context.Context, model string, inputs []string, options map[string]any) (ensure.EmbedResponse, error) {
		if model != "nomic-embed-text:latest" {
			t.Errorf("model = %q", model)
		}
		if !reflect.DeepEqual(inputs, []string{"first document", "second document"}) {
			t.Errorf("inputs = %#v", inputs)
		}
		if !reflect.DeepEqual(options, wantOptions) {
			t.Errorf("runtime options = %#v, want only the cpu-only variant %#v", options, wantOptions)
		}
		return ensure.EmbedResponse{Embeddings: [][]float64{{0.1, 0.2}, {0.3, 0.4}}, TotalDuration: 900, LoadDuration: 100, PromptEvalCount: 2}, nil
	}}
	h, stdout, _ := newHandlers(t, client, policyEnv(t))
	h.Stdin = strings.NewReader(`{"inputs":["first document","second document"]}`)
	if err := h.Embed([]string{"--role", "embedding.default", "--input-json-stdin", "--cpu-only", "--json"}); err != nil {
		t.Fatalf("Embed batch: %v", err)
	}
	var got struct {
		Embeddings             [][]float64 `json:"embeddings"`
		AdmissionWaitNS        int64       `json:"admission_wait_ns"`
		ProviderDurationNS     int64       `json:"provider_duration_ns"`
		RuntimeTotalDurationNS int64       `json:"runtime_total_duration_ns"`
		RuntimeLoadDurationNS  int64       `json:"runtime_load_duration_ns"`
		RuntimePromptEvalCount int         `json:"runtime_prompt_eval_count"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode stdout %q: %v", stdout.String(), err)
	}
	if len(got.Embeddings) != 2 || got.AdmissionWaitNS < 0 || got.ProviderDurationNS < 0 || got.RuntimeTotalDurationNS != 900 || got.RuntimeLoadDurationNS != 100 || got.RuntimePromptEvalCount != 2 {
		t.Fatalf("unexpected batch output: %+v", got)
	}
}

func TestEmbedBatchRejectsExcessInputsBeforeProvider(t *testing.T) {
	called := false
	client := &fakeClient{embedBatch: func(context.Context, string, []string, map[string]any) (ensure.EmbedResponse, error) {
		called = true
		return ensure.EmbedResponse{}, nil
	}}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	var inputs strings.Builder
	inputs.WriteString(`{"inputs":[`)
	for i := 0; i <= maxEmbedInputs; i++ {
		if i > 0 {
			inputs.WriteByte(',')
		}
		inputs.WriteString(`"doc"`)
	}
	inputs.WriteString(`]}`)
	h.Stdin = strings.NewReader(inputs.String())
	err := h.Embed([]string{"--role", "embedding.default", "--input-json-stdin", "--json"})
	if err == nil || !strings.Contains(err.Error(), "input count") {
		t.Fatalf("expected bounded input refusal, got %v", err)
	}
	if called {
		t.Fatal("provider called for an oversized batch")
	}
}

func TestEmbedRejectsGenerateRole(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeClient{}, policyEnv(t))
	err := h.Embed([]string{"--role", "chat.default", "--input", "hello"})
	if err == nil || !strings.Contains(err.Error(), "without embedding capability") {
		t.Fatalf("expected capability error, got %v", err)
	}
}

func TestEmbedRejectsRoleAndModelTogether(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeClient{}, policyEnv(t))
	err := h.Embed([]string{"--role", "embedding.default", "--model", "nomic-embed-text", "--input", "hello"})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutual exclusion error, got %v", err)
	}
}

func TestEmbedRejectsUnknownRole(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeClient{}, policyEnv(t))
	err := h.Embed([]string{"--role", "missing.role", "--input", "hello"})
	if err == nil || !strings.Contains(err.Error(), `unknown model role "missing.role"`) {
		t.Fatalf("expected unknown role error, got %v", err)
	}
}

func TestEmbedSurfacesUpstreamError(t *testing.T) {
	client := &fakeClient{
		embed: func(context.Context, string, string) ([]float64, error) {
			return nil, errors.New("HTTP 503: ollama unavailable")
		},
	}
	h, _, _ := newHandlers(t, client, nil)
	err := h.Embed([]string{"--model", "x", "--input", "y", "--json"})
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected upstream error to surface, got %v", err)
	}
}

func TestGenerateJSONOutput(t *testing.T) {
	client := &fakeClient{
		generate: func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			if in.Model != "llama3.2:1b" || in.Prompt != "hi" {
				t.Errorf("unexpected args: %+v", in)
			}
			if in.NumPredict == nil || *in.NumPredict != 123 {
				t.Errorf("num_predict = %v, want 123", in.NumPredict)
			}
			if in.Temperature == nil || *in.Temperature != 0.25 {
				t.Errorf("temperature = %v, want 0.25", in.Temperature)
			}
			if in.Think == nil || *in.Think {
				t.Errorf("think = %v, want false", in.Think)
			}
			if len(in.RuntimeOptions) != 0 {
				t.Errorf("generate carried runner load options %#v; the daemon profile owns them", in.RuntimeOptions)
			}
			return ensure.GenerateResponse{Response: "hello!", EvalCount: 7}, nil
		},
	}
	h, stdout, _ := newHandlers(t, client, nil)
	if err := h.Generate([]string{"--model", "llama3.2:1b", "--prompt", "hi", "--max-tokens", "123", "--temperature", "0.25", "--context-budget", "4096", "--json"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var got struct {
		Response  string `json:"response"`
		EvalCount int    `json:"eval_count"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v (raw=%q)", err, stdout.String())
	}
	if got.Response != "hello!" {
		t.Fatalf("response = %q", got.Response)
	}
	if got.EvalCount != 7 {
		t.Fatalf("eval_count = %d, want 7", got.EvalCount)
	}
}

func TestParallelismFallbackMatchesOllamaManifestBudget(t *testing.T) {
	h := &Handlers{GetEnv: func(string) string { return "" }}
	if got := h.parallelism(); got != 1 {
		t.Fatalf("parallelism fallback = %d, want manifest budget 1", got)
	}
}

func TestGenerateAttestsExpectedActiveGPUWorker(t *testing.T) {
	digest := strings.Repeat("c", 64)
	client := &fakeClient{
		inventory: []ensure.ModelInfo{{Name: "qwen3.5:4b", Digest: digest}},
		generate: func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			if in.Model != "qwen3.5:4b" {
				t.Fatalf("model = %q", in.Model)
			}
			return ensure.GenerateResponse{Response: "hello"}, nil
		},
		running: []ensure.RunningModel{{Name: "qwen3.5:4b", Digest: "sha256:" + digest, Processor: "GPU", SizeVRAM: 5_000_000_000}},
	}
	h, stdout, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Generate([]string{"--role", "chat.small", "--prompt", "hi", "--json", "--require-worker-digest", digest}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var got struct {
		Response string `json:"response"`
		Worker   struct {
			Model         string `json:"model"`
			Digest        string `json:"digest"`
			Processor     string `json:"processor"`
			SizeVRAMBytes int64  `json:"size_vram_bytes"`
			ObservedAt    string `json:"observed_at"`
		} `json:"worker"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v (%s)", err, stdout.String())
	}
	if got.Response != "hello" || got.Worker.Model != "qwen3.5:4b" || got.Worker.Digest != "sha256:"+digest || got.Worker.Processor != "gpu" || got.Worker.SizeVRAMBytes != 5_000_000_000 || got.Worker.ObservedAt == "" {
		t.Fatalf("response/worker attestation = %+v", got)
	}
}

func TestGenerateRefusesInstalledDigestMismatchBeforeInference(t *testing.T) {
	called := false
	client := &fakeClient{
		inventory: []ensure.ModelInfo{{Name: "qwen3.5:4b", Digest: strings.Repeat("c", 64)}},
		generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			called = true
			return ensure.GenerateResponse{Response: "must not run"}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	err := h.Generate([]string{"--role", "chat.small", "--prompt", "hi", "--json", "--require-worker-digest", "expected"})
	if err == nil || !strings.Contains(err.Error(), "installed model digest mismatch") {
		t.Fatalf("Generate error = %v, want installed digest mismatch", err)
	}
	if called {
		t.Fatal("inference ran before installed model identity was verified")
	}
}

func TestGenerateRefusesActiveWorkerDigestOrProcessorMismatch(t *testing.T) {
	digest := "2a654d98e6fba55d452b7043684e9b57a947e393bbffa62485a7aac05ee4eefd"
	for _, tc := range []struct {
		name     string
		running  ensure.RunningModel
		wantText string
	}{
		{name: "digest", running: ensure.RunningModel{Name: "qwen3.5:4b", Digest: "other", SizeVRAM: 5_000_000_000}, wantText: "active worker digest mismatch"},
		{name: "cpu", running: ensure.RunningModel{Name: "qwen3.5:4b", Digest: digest, Processor: "CPU", SizeVRAM: 0}, wantText: "not verified GPU-resident"},
		{name: "missing worker", wantText: "was not found after generation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeClient{
				inventory: []ensure.ModelInfo{{Name: "qwen3.5:4b", Digest: digest}},
				running:   []ensure.RunningModel{tc.running},
				generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
					return ensure.GenerateResponse{Response: "not accepted"}, nil
				},
			}
			env := policyEnvWithRuntimeDigest(t, "qwen3.5:4b", digest)
			h, stdout, _ := newHandlers(t, client, env)
			err := h.Generate([]string{"--role", "chat.small", "--prompt", "hi", "--json", "--require-worker-digest", digest})
			if err == nil || !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("Generate error = %v, want %q", err, tc.wantText)
			}
			if stdout.Len() != 0 {
				t.Fatalf("unattested output escaped: %s", stdout.String())
			}
		})
	}
}

func TestGenerateRequireLoopbackUsesVerifiedClient(t *testing.T) {
	var generated bool
	client := &fakeClient{generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
		generated = true
		return ensure.GenerateResponse{Response: "{}"}, nil
	}}
	client.inventory = []ensure.ModelInfo{{Name: "qwen3.5:9b", Digest: strings.Repeat("c", 64)}}
	h, _, _ := newHandlers(t, &fakeClient{}, policyEnv(t))
	h.NewLoopbackClient = func() (Client, error) { return client, nil }
	if err := h.Generate([]string{"--role", "chat.default", "--prompt", "hello", "--require-loopback"}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !generated {
		t.Fatal("verified loopback client was not used for generation")
	}
}

func TestGenerateRequireLoopbackRefusesBeforeNormalClient(t *testing.T) {
	var normalClientUsed bool
	h, _, _ := newHandlers(t, &fakeClient{}, policyEnv(t))
	h.NewClient = func() Client { normalClientUsed = true; return &fakeClient{} }
	h.NewLoopbackClient = func() (Client, error) { return nil, errors.New("local-only endpoint must use a literal loopback IP") }
	err := h.Generate([]string{"--role", "chat.default", "--prompt", "hello", "--require-loopback"})
	if err == nil || !strings.Contains(err.Error(), "literal loopback IP") {
		t.Fatalf("Generate() error = %v, want locality refusal", err)
	}
	if normalClientUsed {
		t.Fatal("normal client was used after loopback verification failed")
	}
}

func TestGenerateRoleResolvesModel(t *testing.T) {
	client := &fakeClient{
		generate: func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			if in.Model != "qwen3.5:9b" {
				t.Errorf("model = %q", in.Model)
			}
			if in.Prompt != "hi" {
				t.Errorf("prompt = %q", in.Prompt)
			}
			return ensure.GenerateResponse{Response: "hello!", EvalCount: 1}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Generate([]string{"--role", "chat.default", "--prompt", "hi", "--json"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
}

func TestGenerateRejectsExplicitContextWindowOverflow(t *testing.T) {
	var called bool
	client := &fakeClient{
		generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			called = true
			return ensure.GenerateResponse{}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	err := h.Generate([]string{"--role", "chat.default", "--prompt", "hi", "--max-tokens", "40000"})
	if err == nil || !strings.Contains(err.Error(), "exceeds context window") {
		t.Fatalf("expected context window error, got %v", err)
	}
	if called {
		t.Fatal("Generate called upstream after context window rejection")
	}
}

func TestGenerateOneShotPinsKeepAliveZeroAndVerifiesCleanup(t *testing.T) {
	var generated bool
	client := &fakeClient{}
	client.generate = func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
		generated = true
		if in.KeepAlive == nil || *in.KeepAlive != 0 {
			t.Fatalf("keep_alive = %v, want explicit zero", in.KeepAlive)
		}
		client.running = []ensure.RunningModel{{Name: "qwen3.5:9b"}}
		return ensure.GenerateResponse{Response: "bounded", EvalCount: 1}, nil
	}
	client.unload = func(_ context.Context, model string) error {
		if model != "qwen3.5:9b" {
			t.Fatalf("unload model = %q", model)
		}
		client.running = nil
		return nil
	}
	h, stdout, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Generate([]string{"--role", "chat.default", "--prompt", "hi", "--json", "--require-loopback", "--one-shot", "--timeout", "90s"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !generated {
		t.Fatal("generation was not called")
	}
	var got struct {
		Response              string `json:"response"`
		WorkerCleanupVerified bool   `json:"worker_cleanup_verified"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if got.Response != "bounded" || !got.WorkerCleanupVerified {
		t.Fatalf("one-shot result = %+v", got)
	}
}

func TestGenerateOneShotRefusesAlreadyLoadedTarget(t *testing.T) {
	called := false
	client := &fakeClient{
		running: []ensure.RunningModel{{Name: "qwen3.5:9b"}},
		generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			called = true
			return ensure.GenerateResponse{}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	err := h.Generate([]string{"--role", "chat.default", "--prompt", "hi", "--json", "--require-loopback", "--one-shot"})
	if err == nil || !strings.Contains(err.Error(), "already loaded") || called {
		t.Fatalf("Generate error=%v called=%v, want preflight refusal", err, called)
	}
}

func TestGenerateRejectsUncataloguedDirectModelBeforeProvider(t *testing.T) {
	var providerCalls atomic.Int32
	client := &fakeClient{
		generate: func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			providerCalls.Add(1)
			return ensure.GenerateResponse{Response: "ok"}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Generate([]string{"--model", "unlisted-model:latest", "--prompt", "hi"}); err == nil {
		t.Fatal("unlisted model bypassed the local rights catalog")
	}
	if providerCalls.Load() != 0 {
		t.Fatalf("unlisted model reached provider; calls=%d", providerCalls.Load())
	}
}

func TestGenerateInputJSONStdinForwardsImagesWithoutArgvData(t *testing.T) {
	client := &fakeClient{
		generate: func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			if in.Model != "qwen3.5:4b" || in.Prompt != "describe" {
				t.Errorf("unexpected request: %+v", in)
			}
			if len(in.Images) != 1 || in.Images[0] == "" {
				t.Fatalf("images = %v, want one decoded image", in.Images)
			}
			return ensure.GenerateResponse{Response: "seen"}, nil
		},
	}
	h, stdout, _ := newHandlers(t, client, nil)
	h.Stdin = strings.NewReader(`{"prompt":"describe","images":[{"media_type":"image/png","data_b64":"aGVsbG8="}]}`)
	if err := h.Generate([]string{"--model", "qwen3.5:4b", "--input-json-stdin", "--json"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(stdout.String(), `"response":"seen"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestGenerateCPUOnlyUsesTheProfileVariant(t *testing.T) {
	want, err := runtimeprofile.Current().VariantOptions(runtimeprofile.VariantCPUOnly)
	if err != nil {
		t.Fatal(err)
	}
	client := &fakeClient{
		generate: func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			if !reflect.DeepEqual(in.RuntimeOptions, want) {
				t.Fatalf("runtime options = %#v, want cpu-only variant %#v", in.RuntimeOptions, want)
			}
			return ensure.GenerateResponse{Response: "cpu"}, nil
		},
	}
	h, _, _ := newHandlers(t, client, nil)
	if err := h.Generate([]string{"--model", "qwen3-vl:2b", "--prompt", "describe", "--cpu-only"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
}

func TestGatewayNoLongerAcceptsRunnerLoadFlags(t *testing.T) {
	called := false
	client := &fakeClient{generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
		called = true
		return ensure.GenerateResponse{}, nil
	}}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	for _, flagName := range []string{"--num-ctx", "--num-thread", "--num-batch"} {
		if err := h.Generate([]string{"--model", "qwen3-vl:2b", "--prompt", "describe", flagName, "256"}); err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
			t.Fatalf("%s error = %v, want an undefined flag", flagName, err)
		}
	}
	if called {
		t.Fatal("upstream called with a removed runner load flag")
	}
}

func TestGenerateRejectsContextBudgetAboveRunnerContext(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeClient{}, policyEnv(t))
	runner := runtimeprofile.Current().ContextTokens("llama3.2:1b")
	err := h.Generate([]string{"--model", "llama3.2:1b", "--prompt", "describe", "--context-budget", strconv.Itoa(runner + 1)})
	if err == nil || !strings.Contains(err.Error(), "runner context") {
		t.Fatalf("expected runner context refusal, got %v", err)
	}
}

func TestGenerateContextBudgetRefusesOversizedPromptWithoutLoadOptions(t *testing.T) {
	called := false
	h, _, _ := newHandlers(t, &fakeClient{generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
		called = true
		return ensure.GenerateResponse{}, nil
	}}, policyEnv(t))
	prompt := strings.Repeat("abcd", 1000)
	err := h.Generate([]string{"--role", "vision.default", "--prompt", prompt, "--max-tokens", "64", "--context-budget", "512"})
	if err == nil || !strings.Contains(err.Error(), "exceeds context window") {
		t.Fatalf("expected budget refusal, got %v", err)
	}
	if called {
		t.Fatal("upstream called for a prompt over the caller's context budget")
	}
}

func TestGenerateInputJSONStdinRejectsRoleWithoutImageModality(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeClient{}, nil)
	h.Stdin = strings.NewReader(`{"prompt":"describe","images":[{"media_type":"image/png","data_b64":"aGVsbG8="}]}`)
	_, _, err := h.resolveImageInput(selectedModel{Ref: "text-only", InputModalities: []policy.Modality{policy.ModalityText}})
	if err == nil || !strings.Contains(err.Error(), "image_input.capability_mismatch") {
		t.Fatalf("expected typed capability mismatch, got %v", err)
	}
}

func TestChatRoleResolvesModelAndForwardsControls(t *testing.T) {
	env := policyEnv(t)
	p, _, err := policy.LoadDefaultFile(func(key string) string { return env[key] })
	if err != nil {
		t.Fatal(err)
	}
	wantModel := p.Roles["summarize.default"].Model
	if wantModel == "" {
		t.Fatal("policy has no summarize.default model")
	}
	client := &fakeClient{
		chat: func(_ context.Context, in ensure.ChatRequest) (ensure.ChatResponse, error) {
			if in.Model != wantModel {
				t.Errorf("model = %q, want policy summarize.default %q", in.Model, wantModel)
			}
			if len(in.Messages) != 2 {
				t.Fatalf("messages len = %d, want 2", len(in.Messages))
			}
			if in.Messages[0].Role != "system" || in.Messages[0].Content != "be concise" {
				t.Errorf("system message = %+v", in.Messages[0])
			}
			if in.Messages[1].Role != "user" || in.Messages[1].Content != "hello" {
				t.Errorf("user message = %+v", in.Messages[1])
			}
			if in.NumPredict == nil || *in.NumPredict != 123 {
				t.Errorf("num_predict = %v, want 123", in.NumPredict)
			}
			if in.Temperature == nil || *in.Temperature != 0.25 {
				t.Errorf("temperature = %v, want 0.25", in.Temperature)
			}
			if in.Think == nil || *in.Think != false {
				t.Errorf("think = %v, want false", in.Think)
			}
			var out ensure.ChatResponse
			out.Message.Content = "summary"
			out.DoneReason = "stop"
			out.EvalCount = 7
			return out, nil
		},
	}
	h, stdout, _ := newHandlers(t, client, env)
	if err := h.Chat([]string{"--role", "summarize.default", "--system", "be concise", "--prompt", "hello", "--max-tokens", "123", "--temperature", "0.25", "--json"}); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	var got struct {
		Response   string `json:"response"`
		DoneReason string `json:"done_reason"`
		EvalCount  int    `json:"eval_count"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v (raw=%q)", err, stdout.String())
	}
	if got.Response != "summary" || got.DoneReason != "stop" || got.EvalCount != 7 {
		t.Fatalf("unexpected JSON: %+v", got)
	}
}

func TestChatRejectsEmbeddingRole(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeClient{}, policyEnv(t))
	err := h.Chat([]string{"--role", "embedding.default", "--prompt", "hello"})
	if err == nil || !strings.Contains(err.Error(), "without chat capability") {
		t.Fatalf("expected capability error, got %v", err)
	}
}

func TestEmbedStdinInput(t *testing.T) {
	client := &fakeClient{
		embed: func(_ context.Context, _, input string) ([]float64, error) {
			if input != "from-stdin" {
				t.Errorf("input = %q", input)
			}
			return []float64{1}, nil
		},
	}
	h, _, _ := newHandlers(t, client, nil)
	h.Stdin = strings.NewReader("from-stdin")
	if err := h.Embed([]string{"--model", "m", "--input-stdin", "--json"}); err != nil {
		t.Fatalf("Embed: %v", err)
	}
}

func TestSemaphoreSerializesConcurrentInvocations(t *testing.T) {
	const slots = 2
	const callers = 8

	sem, err := hostsem.New(t.TempDir(), slots)
	if err != nil {
		t.Fatalf("hostsem: %v", err)
	}

	var (
		current int32
		peak    int32
	)
	client := &fakeClient{
		inventory: []ensure.ModelInfo{{Name: "m", Digest: strings.Repeat("c", 64)}},
		embed: func(ctx context.Context, _, _ string) ([]float64, error) {
			n := atomic.AddInt32(&current, 1)
			for {
				p := atomic.LoadInt32(&peak)
				if n <= p || atomic.CompareAndSwapInt32(&peak, p, n) {
					break
				}
			}
			time.Sleep(15 * time.Millisecond)
			atomic.AddInt32(&current, -1)
			return []float64{0}, nil
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			env := policyEnv(t)
			h := &Handlers{
				NewClient:         func() Client { return client },
				NewLoopbackClient: func() (Client, error) { return client, nil },
				Sem:               sem,
				GetEnv:            func(k string) string { return env[k] },
				Stdin:             strings.NewReader(""),
				Stdout:            io.Discard,
				Stderr:            io.Discard,
			}
			if err := h.Embed([]string{"--model", "m", "--input", fmt.Sprintf("p%d", i), "--json"}); err != nil {
				t.Errorf("Embed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if got := atomic.LoadInt32(&peak); got > slots {
		t.Fatalf("peak in-flight = %d, want <= %d", got, slots)
	}
}

func TestAcquireRespectsContextDeadline(t *testing.T) {
	sem, err := hostsem.New(t.TempDir(), 1)
	if err != nil {
		t.Fatalf("hostsem: %v", err)
	}
	hold, err := sem.Acquire(context.Background())
	if err != nil {
		t.Fatalf("priming: %v", err)
	}
	defer hold()

	h := &Handlers{
		NewClient: func() Client { return &fakeClient{} },
		Sem:       sem,
		GetEnv: func(k string) string {
			if k == envAcquireTO {
				return "30ms"
			}
			return ""
		},
		Stdin:  strings.NewReader(""),
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	env := policyEnv(t)
	h.GetEnv = func(k string) string {
		if value := env[k]; value != "" {
			return value
		}
		if k == envAcquireTO {
			return "30ms"
		}
		return ""
	}
	err = h.Embed([]string{"--model", "m", "--input", "x", "--json"})
	if err == nil || !strings.Contains(err.Error(), "acquire host semaphore") {
		t.Fatalf("expected acquire timeout error, got %v", err)
	}
}

func policyEnv(t *testing.T) map[string]string {
	t.Helper()
	return writeTestPolicy(t, true, nil)
}

func policyEnvWithRuntimeDigest(t *testing.T, modelRef, digest string) map[string]string {
	t.Helper()
	return writeTestPolicy(t, true, func(document map[string]any) {
		models, ok := document["models"].(map[string]any)
		if !ok {
			t.Fatal("policy has no models object")
		}
		entry, ok := models[modelRef].(map[string]any)
		if !ok {
			t.Fatalf("policy has no model %q", modelRef)
		}
		identity, ok := entry["runtime_identity"].(sharedpolicy.RuntimeArtifactIdentity)
		if !ok {
			t.Fatalf("model %q has no runtime identity", modelRef)
		}
		identity.Digest = digest
		entry["runtime_identity"] = identity
	})
}

func unqualifiedPolicyEnv(t *testing.T) map[string]string {
	t.Helper()
	return writeTestPolicy(t, false, func(document map[string]any) {
		models, ok := document["models"].(map[string]any)
		if !ok {
			t.Fatal("policy has no models object")
		}
		for _, raw := range models {
			entry, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			delete(entry, "rights")
			delete(entry, "runtime_identity")
		}
	})
}

// policyEnvWithRoleMaxTokens copies the shipped policy and declares an output
// cap on one role, so the role-owned cap can be exercised without waiting for a
// role that ships one.
func policyEnvWithRoleMaxTokens(t *testing.T, role string, maxTokens int) map[string]string {
	t.Helper()
	return writeTestPolicy(t, true, func(document map[string]any) {
		roles, ok := document["roles"].(map[string]any)
		if !ok {
			t.Fatal("policy has no roles object")
		}
		entry, ok := roles[role].(map[string]any)
		if !ok {
			t.Fatalf("policy has no role %q", role)
		}
		entry["max_tokens"] = maxTokens
	})
}

func writeTestPolicy(t *testing.T, qualify bool, mutate func(map[string]any)) map[string]string {
	t.Helper()
	source, err := filepath.Abs(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatalf("policy path: %v", err)
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read policy: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("parse policy: %v", err)
	}
	if qualify {
		models, ok := document["models"].(map[string]any)
		if !ok {
			t.Fatal("policy has no models object")
		}
		for ref, raw := range models {
			entry, ok := raw.(map[string]any)
			if !ok {
				t.Fatalf("model %q has invalid test fixture shape", ref)
			}
			entry["rights"] = sharedpolicy.ArtifactRights{
				ArtifactID:     "test-artifact:" + ref,
				Revision:       "test-model-v1",
				SHA256:         strings.Repeat("a", 64),
				LicenseID:      "LicenseRef-TestFixture",
				Classification: sharedpolicy.RightsConditional,
				Evidence: sharedpolicy.RightsEvidence{
					URL: "https://example.invalid/test-license", Revision: "test-license-v1",
					SHA256: strings.Repeat("b", 64), ObservedAt: "2026-11-10",
				},
				Conditions: []sharedpolicy.RightsCondition{{
					ID: "test-revenue-condition", Kind: "annual_revenue", Comparator: "lt",
					Threshold: 10000000, Unit: "USD", EntityScope: "test-entity",
					Interpretation: "test fixture only",
				}},
			}
			entry["runtime_identity"] = sharedpolicy.RuntimeArtifactIdentity{
				Provider: "ollama", Model: ref, ArtifactID: "test-artifact:" + ref,
				ArtifactRevision: "test-model-v1", DigestKind: "ollama_model_contents",
				Digest: strings.Repeat("c", 64), Source: sharedpolicy.RuntimeSourceOllamaTags, ObservedAt: "2026-11-10",
			}
		}
		for _, ref := range []string{"m", "x", "requested:model", "local-test-model:latest"} {
			template, ok := models["llama3.2:1b"].(map[string]any)
			if !ok {
				t.Fatal("policy has no fixture model template")
			}
			models[ref] = map[string]any{
				"family": "test-fixture", "tags": []any{"test"},
				"capabilities":          []any{"embedding", "generate", "chat"},
				"modalities":            map[string]any{"input": []any{"text", "image"}, "output": []any{"text", "vector"}},
				"context_window_tokens": 8192, "embedding_dimensions": 3,
				"disk_size_gb_estimate": 1, "ram_gb_estimate": 1, "vram_gb_estimate": 1,
				"default_eligible": true, "use_case_notes": "test-only fixture", "caveats": []any{},
				"provenance": template["provenance"],
				"rights": sharedpolicy.ArtifactRights{
					ArtifactID: "test-artifact:" + ref, Revision: "test-model-v1", SHA256: strings.Repeat("a", 64),
					LicenseID: "LicenseRef-TestFixture", Classification: sharedpolicy.RightsConditional,
					Evidence:   sharedpolicy.RightsEvidence{URL: "https://example.invalid/test-license", Revision: "test-license-v1", SHA256: strings.Repeat("b", 64), ObservedAt: "2026-11-10"},
					Conditions: []sharedpolicy.RightsCondition{{ID: "test-revenue-condition", Kind: "annual_revenue", Comparator: "lt", Threshold: 10000000, Unit: "USD", EntityScope: "test-entity", Interpretation: "test fixture only"}},
				},
				"runtime_identity": sharedpolicy.RuntimeArtifactIdentity{
					Provider: "ollama", Model: ref, ArtifactID: "test-artifact:" + ref,
					ArtifactRevision: "test-model-v1", DigestKind: "ollama_model_contents",
					Digest: strings.Repeat("c", 64), Source: sharedpolicy.RuntimeSourceOllamaTags, ObservedAt: "2026-11-10",
				},
			}
		}
	}
	if mutate != nil {
		mutate(document)
	}
	path := filepath.Join(t.TempDir(), "model-policy.json")
	rightsContext := sharedpolicy.RightsContext{
		ID: "test-operator-context", Version: "test-v1", EntityScope: "test-entity",
		RevenueStatus: "pre_revenue", AsOf: "2026-11-10", Source: "operator_statement",
	}
	rightsContextPath := filepath.Join(filepath.Dir(path), "model-rights-context.json")
	rightsContextBytes, err := json.Marshal(map[string]any{"schema_version": "1", "context": rightsContext})
	if err != nil {
		t.Fatalf("encode test rights context: %v", err)
	}
	if err := os.WriteFile(rightsContextPath, rightsContextBytes, 0o600); err != nil {
		t.Fatalf("write test rights context: %v", err)
	}
	if _, ok := document["roles"].(map[string]any); !ok {
		t.Fatal("policy has no roles object")
	}
	patched, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode policy: %v", err)
	}
	if err := os.WriteFile(path, patched, 0o600); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	return map[string]string{"OLLAMA_MODEL_POLICY_PATH": path, "VROOLI_MODEL_RIGHTS_CONTEXT_PATH": rightsContextPath}
}

func testRightsContext(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "model-rights-context.json")
	content := `{"schema_version":"1","context":{"id":"test-operator-context","version":"test-v1","entity_scope":"test-entity","revenue_status":"pre_revenue","as_of":"2026-11-10","source":"operator_statement"}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test rights context: %v", err)
	}
	return path
}

// A direct model exception has no role-owned cap, so a caller who sends
// nothing keeps the explicit uncapped behavior. Normal role traffic is
// bounded by the shipped role policy defaults.
func TestGenerateOmitsNumPredictWhenNeitherFlagNorRoleDeclaresCap(t *testing.T) {
	var seen ensure.GenerateRequest
	client := &fakeClient{
		generate: func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			seen = in
			return ensure.GenerateResponse{Response: "ok"}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Generate([]string{"--model", "local-test-model:latest", "--prompt", "hi"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if seen.NumPredict != nil {
		t.Fatalf("num_predict = %d, want omitted", *seen.NumPredict)
	}
}

func TestGenerateAppliesRoleMaxTokensWhenFlagAbsent(t *testing.T) {
	var seen ensure.GenerateRequest
	client := &fakeClient{
		generate: func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			seen = in
			return ensure.GenerateResponse{Response: "ok"}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnvWithRoleMaxTokens(t, "chat.default", 1234))
	if err := h.Generate([]string{"--role", "chat.default", "--prompt", "hi"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if seen.NumPredict == nil || *seen.NumPredict != 1234 {
		t.Fatalf("num_predict = %v, want 1234", seen.NumPredict)
	}
}

func TestGenerateExplicitMaxTokensOverridesRoleCap(t *testing.T) {
	var seen ensure.GenerateRequest
	client := &fakeClient{
		generate: func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			seen = in
			return ensure.GenerateResponse{Response: "ok"}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnvWithRoleMaxTokens(t, "chat.default", 1234))
	if err := h.Generate([]string{"--role", "chat.default", "--prompt", "hi", "--max-tokens", "77"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if seen.NumPredict == nil || *seen.NumPredict != 77 {
		t.Fatalf("num_predict = %v, want 77", seen.NumPredict)
	}
}

// The context guard must reserve the budget that will actually be sent. A role
// cap large enough to overflow the window is refused by name, not silently
// truncated by the server sliding its context window.
func TestGenerateRoleMaxTokensReachesContextWindowGuard(t *testing.T) {
	var called bool
	client := &fakeClient{
		generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			called = true
			return ensure.GenerateResponse{}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnvWithRoleMaxTokens(t, "chat.default", 40000))
	err := h.Generate([]string{"--role", "chat.default", "--prompt", "hi"})
	if err == nil || !strings.Contains(err.Error(), "exceeds context window") {
		t.Fatalf("expected context window error, got %v", err)
	}
	if called {
		t.Fatal("Generate called upstream after context window rejection")
	}
}

func TestChatAppliesRoleMaxTokensWhenFlagAbsent(t *testing.T) {
	var seen ensure.ChatRequest
	client := &fakeClient{
		chat: func(_ context.Context, in ensure.ChatRequest) (ensure.ChatResponse, error) {
			seen = in
			var response ensure.ChatResponse
			response.Message.Content = "ok"
			return response, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnvWithRoleMaxTokens(t, "summarize.default", 4321))
	if err := h.Chat([]string{"--role", "summarize.default", "--prompt", "hi"}); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if seen.NumPredict == nil || *seen.NumPredict != 4321 {
		t.Fatalf("num_predict = %v, want 4321", seen.NumPredict)
	}
}

// Generate JSON must preserve optional provider completion metadata rather than
// replacing missing values with a success-shaped default or an inferred model.
func TestGenerateJSONPreservesOptionalCompletionMetadata(t *testing.T) {
	for _, tc := range []struct {
		name     string
		metadata string
		want     map[string]string
		wantErr  bool
	}{
		{name: "stop", metadata: `,"done":true,"done_reason":"stop","model":"provider:echo"`, want: map[string]string{"done": "true", "done_reason": `"stop"`, "model": `"provider:echo"`}},
		{name: "length", metadata: `,"done":true,"done_reason":"length","model":"provider:echo"`, want: map[string]string{"done": "true", "done_reason": `"length"`, "model": `"provider:echo"`}},
		{name: "reported_false", metadata: `,"done":false,"done_reason":"provider_pause","model":" raw echo "`, want: map[string]string{"done": "false", "done_reason": `"provider_pause"`, "model": `" raw echo "`}},
		{name: "missing", want: map[string]string{}},
		{name: "null_unknown", metadata: `,"done":null,"done_reason":null,"model":null`, want: map[string]string{}},
		{name: "reported_empty", metadata: `,"done":false,"done_reason":"","model":""`, want: map[string]string{"done": "false", "done_reason": `""`, "model": `""`}},
		{name: "malformed_done", metadata: `,"done":"yes"`, wantErr: true},
		{name: "malformed_reason", metadata: `,"done_reason":123`, wantErr: true},
		{name: "malformed_model", metadata: `,"model":{"name":"untrusted"}`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"response":"visible unchanged","thinking":"synthetic private field must not be exported","eval_count":7,"total_duration":100,"load_duration":20,"prompt_eval_count":3,"prompt_eval_duration":30,"eval_duration":40` + tc.metadata + `}`
			client := &fakeClient{generate: func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
				if in.NumPredict == nil || *in.NumPredict != 500 || in.Think == nil || *in.Think {
					t.Errorf("changed requested cap/visible-output controls: %+v", in)
				}
				var response ensure.GenerateResponse
				err := json.Unmarshal([]byte(body), &response)
				return response, err
			}}
			h, stdout, _ := newHandlers(t, client, nil)
			err := h.Generate([]string{"--model", "requested:model", "--prompt", "literal input", "--max-tokens", "500", "--json"})
			if tc.wantErr {
				if err == nil {
					t.Fatal("malformed reported metadata must not produce successful CLI JSON")
				}
				if stdout.Len() != 0 {
					t.Fatalf("failed request published output: %q", stdout.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(stdout.Bytes(), &fields); err != nil {
				t.Fatal(err)
			}
			if string(fields["response"]) != `"visible unchanged"` || string(fields["eval_count"]) != "7" || string(fields["total_duration_ns"]) != "100" || string(fields["load_duration_ns"]) != "20" || string(fields["prompt_eval_count"]) != "3" || string(fields["prompt_eval_duration_ns"]) != "30" || string(fields["eval_duration_ns"]) != "40" {
				t.Fatalf("visible text/counts/durations changed: %s", stdout.Bytes())
			}
			if _, exists := fields["thinking"]; exists {
				t.Fatal("private thinking field was exported")
			}
			for _, key := range []string{"done", "done_reason", "model"} {
				want, present := tc.want[key]
				value, exists := fields[key]
				if exists != present || (present && string(value) != want) {
					t.Errorf("%s = %s (present=%t), want %s (present=%t)", key, value, exists, want, present)
				}
			}
		})
	}
}

func TestEmbedLongInputReachesProviderAndReportsTruncation(t *testing.T) {
	limit := runtimeprofile.Current().ContextTokens("nomic-embed-text:latest")
	long := strings.Repeat("token ", limit*4)
	client := &fakeClient{embedBatch: func(_ context.Context, _ string, inputs []string, options map[string]any) (ensure.EmbedResponse, error) {
		if len(inputs) != 1 || inputs[0] != long {
			t.Errorf("gateway altered or split the input before the provider")
		}
		if len(options) != 0 {
			t.Errorf("default embed sent runner load options %#v", options)
		}
		return ensure.EmbedResponse{Embeddings: [][]float64{{0.6, 0.8}}, PromptEvalCount: limit}, nil
	}}
	h, stdout, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Embed([]string{"--role", "embedding.default", "--input", long, "--json"}); err != nil {
		t.Fatalf("long input must embed its head instead of failing: %v", err)
	}
	var got struct {
		Embedding        []float64 `json:"embedding"`
		InputLimitTokens int       `json:"input_limit_tokens"`
		Truncated        *bool     `json:"truncated"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode %q: %v", stdout.String(), err)
	}
	if got.InputLimitTokens != limit || got.Truncated == nil || !*got.Truncated || len(got.Embedding) != 2 {
		t.Fatalf("output = %+v, want truncated=true at limit %d", got, limit)
	}
}

func TestEmbedShortInputIsNotReportedTruncated(t *testing.T) {
	client := &fakeClient{embedBatch: func(context.Context, string, []string, map[string]any) (ensure.EmbedResponse, error) {
		return ensure.EmbedResponse{Embeddings: [][]float64{{1}}, PromptEvalCount: 3}, nil
	}}
	h, stdout, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Embed([]string{"--role", "embedding.default", "--input", "short text", "--json"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"truncated":false`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestChatSendsTheDeclaredRunnerContextForTheResolvedModel(t *testing.T) {
	client := &fakeClient{chat: func(_ context.Context, in ensure.ChatRequest) (ensure.ChatResponse, error) {
		want := runtimeprofile.Current().RequestOptions(in.Model)
		if !reflect.DeepEqual(in.RuntimeOptions, want) {
			t.Errorf("runtime options for %s = %#v, want the profile's %#v", in.Model, in.RuntimeOptions, want)
		}
		if len(want) == 0 {
			t.Errorf("fixture model %s declares no request options; pick one that does", in.Model)
		}
		return ensure.ChatResponse{}, nil
	}}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Chat([]string{"--model", "qwen3.5:9b", "--prompt", "hi"}); err != nil {
		t.Fatal(err)
	}
}
