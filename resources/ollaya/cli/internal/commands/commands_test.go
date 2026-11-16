package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

// daemon fakes the Ollaya HTTP API with the models it holds. A non-empty
// servedDigest makes pulls store that digest instead of the requested one, the
// way a registry would if the pin were ignored.
func daemon(t *testing.T, installed map[string]string, servedDigest ...string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			var models []map[string]string
			for name, digest := range installed {
				models = append(models, map[string]string{"name": name, "digest": digest})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"models": models})
		case "/api/pull":
			var body struct{ Model string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			name, digest, _ := strings.Cut(body.Model, "@sha256:")
			if len(servedDigest) > 0 {
				digest = servedDigest[0]
			}
			installed[upstream.InstalledName(name)] = digest
			_, _ = w.Write([]byte(`{"status":"success"}`))
		case "/api/decide":
			_, _ = io.Copy(io.Discard, r.Body)
			_, _ = w.Write([]byte(`{"model":"laya:en","answers":{"q":{"type":"noul","noul":0.9}},"usage":{"input_tokens":12,"output_tokens":0},"state_truncated":false,"eval_duration":1000000}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func handlers(t *testing.T, env map[string]string, stdin string) (*Handlers, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	values := map[string]string{"OLLAYA_MODEL_POLICY_PATH": filepath.Join(filepath.Dir(file), "..", "..", "..", "model-policy.json")}
	for key, value := range env {
		values[key] = value
	}
	var stdout, stderr bytes.Buffer
	return &Handlers{
		GetEnv: func(key string) string { return values[key] }, Stdin: strings.NewReader(stdin), Stdout: &stdout, Stderr: &stderr, HTTP: http.DefaultClient,
		Now: func() time.Time { return time.Date(2026, 11, 11, 20, 0, 0, 0, time.UTC) },
	}, &stdout, &stderr
}

func TestRolesReportLocalityAndUnconfiguredHostedRoles(t *testing.T) {
	h, stdout, _ := handlers(t, map[string]string{"EDENAI_API_KEY": "k"}, "")
	if err := h.Roles([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Roles []struct {
			Role, Locality, Status string
			Capabilities           []string
		}
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	status := map[string]string{}
	for _, role := range report.Roles {
		status[role.Role] = role.Locality + "/" + role.Status
		if len(role.Capabilities) != 1 || role.Capabilities[0] != "decide" {
			t.Fatalf("%s capabilities = %v, want [decide]", role.Role, role.Capabilities)
		}
	}
	want := map[string]string{"decide.local": "local/available", "decide.jev.edenai": "remote/available", "decide.jev.typesafe": "remote/unconfigured", "decide.clef.cloudflare": "remote/unconfigured"}
	for role, expected := range want {
		if status[role] != expected {
			t.Fatalf("%s = %s, want %s", role, status[role], expected)
		}
	}
}

func TestResolveFailsTypedWhenTheRoleCannotServe(t *testing.T) {
	h, _, stderr := handlers(t, nil, "")
	if err := h.Resolve([]string{"--role", "decide.jev.typesafe", "--json"}); err == nil {
		t.Fatal("hosted role without its key must not resolve")
	}
	if !strings.Contains(stderr.String(), upstream.ProviderErrorMarker+`{"code":"unconfigured"`) {
		t.Fatalf("stderr = %q, want the unconfigured marker", stderr.String())
	}

	shipped := shippedModelPolicy(t)
	server := daemon(t, map[string]string{"laya:latest": shipped.Models["laya"].Digest})
	h, _, stderr = handlers(t, map[string]string{"OLLAYA_URL": server.URL}, "")
	if err := h.Resolve([]string{"--role", "decide.local", "--json"}); err == nil || !strings.Contains(stderr.String(), "laya:en is not pulled") {
		t.Fatalf("err=%v stderr=%q, want a router missing its targets to be unconfigured", err, stderr.String())
	}
}

func TestResolveAndDecideLocalRole(t *testing.T) {
	shipped := shippedModelPolicy(t)
	server := daemon(t, map[string]string{
		"laya:latest":       shipped.Models["laya"].Digest,
		"laya:en":           shipped.Models["laya:en"].Digest,
		"laya:multilingual": shipped.Models["laya:multilingual"].Digest,
	})
	h, stdout, _ := handlers(t, map[string]string{"OLLAYA_URL": server.URL}, "")
	if err := h.Resolve([]string{"--role", "decide.local", "--json"}); err != nil {
		t.Fatal(err)
	}
	var resolved map[string]any
	_ = json.Unmarshal(stdout.Bytes(), &resolved)
	identity, _ := resolved["runtime_identity"].(map[string]any)
	rights, _ := resolved["rights"].(map[string]any)
	if resolved["role"] != "decide.local" || resolved["model"] != "laya" || resolved["locality"] != "local" ||
		resolved["schema_version"] != shipped.SchemaVersion || resolved["policy_sha256"] != shipped.LoadedSHA256 ||
		resolved["expected_artifact_digest"] != shipped.Models["laya"].Digest || identity["digest"] != shipped.Models["laya"].Digest ||
		rights["classification"] != string(sharedpolicy.RightsUnverified) || rights["sha256"] != shipped.Models["laya"].Digest {
		t.Fatalf("resolve = %v", resolved)
	}

	// A raw local command cannot use the pinned manifest alone as rights
	// authority; it must fail before the daemon's /api/decide endpoint.
	var tagsCalls, decideCalls int
	observed := observedDaemon(t, map[string]string{
		"laya:latest":       shipped.Models["laya"].Digest,
		"laya:en":           shipped.Models["laya:en"].Digest,
		"laya:multilingual": shipped.Models["laya:multilingual"].Digest,
	}, func() { tagsCalls++ }, func() { decideCalls++ })
	h, _, stderr := handlers(t, map[string]string{"OLLAYA_URL": observed.URL}, `{"state":"x","questions":{"q":{"type":"noul"}}}`)
	unknownContext := sharedpolicy.RightsContext{ID: "fixture", Version: "v1", RevenueStatus: "pre_revenue", AsOf: "2026-11-11", Source: "test_fixture"}
	h.RightsContext = func(context.Context) (sharedpolicy.RightsContext, error) { return unknownContext, nil }
	err := h.Decide(localDecisionArgs(t, shippedModelPolicyPath(t), "decide.local", "laya", shipped.Models["laya"].Digest, unknownContext))
	if err == nil || !strings.Contains(stderr.String(), "not currently allowed (unknown") || tagsCalls != 0 || decideCalls != 0 {
		t.Fatalf("err=%v stderr=%q tagsCalls=%d decideCalls=%d; want shipped unknown rights refusal before model/provider endpoint", err, stderr.String(), tagsCalls, decideCalls)
	}
}

func TestShippedLayaRightsRecordOnlyVerifiedProvenanceAndRemainsUnknown(t *testing.T) {
	shipped := shippedModelPolicy(t)
	contextValue := fixtureRightsContextValue()
	for _, name := range []string{"laya", "laya:en", "laya:multilingual"} {
		model := shipped.Models[name]
		if model.Rights == nil || model.Rights.Classification != sharedpolicy.RightsUnverified ||
			model.Rights.SHA256 != model.Digest || model.Rights.LicenseID != "Apache-2.0" {
			t.Fatalf("%s rights facts = %+v; want digest-bound, provenance-only Apache facts classified unknown", name, model.Rights)
		}
		decision := sharedpolicy.EvaluateRights(model.Rights, sharedpolicy.RightsUseLocalInference, contextValue, time.Date(2026, 11, 11, 20, 0, 0, 0, time.UTC))
		if decision.Status != sharedpolicy.RightsUnknown {
			t.Fatalf("%s decision = %+v; source facts alone must not authorize use", name, decision)
		}
	}
	winnow := shipped.Models["winnow:e4b"].Rights
	if winnow == nil || winnow.Classification != sharedpolicy.RightsUnverified || winnow.SHA256 != shipped.Models["winnow:e4b"].Digest || winnow.UnverifiedReason == "" {
		t.Fatalf("Winnow rights = %+v; unresolved Gemma/Apache source conflict must be recorded as an explicit unknown disposition", winnow)
	}
}

func TestLocalGatewayUnknownContextAndManifestDriftRefuseBeforeDecision(t *testing.T) {
	for _, tc := range []struct {
		name            string
		context         sharedpolicy.RightsContext
		installedDigest string
		contextSource   func() sharedpolicy.RightsContext
	}{
		{
			name:    "missing entity scope",
			context: sharedpolicy.RightsContext{ID: "fixture", Version: "v1", RevenueStatus: "pre_revenue", AsOf: "2026-11-11", Source: "test_fixture"},
			contextSource: func() sharedpolicy.RightsContext {
				return sharedpolicy.RightsContext{ID: "fixture", Version: "v1", RevenueStatus: "pre_revenue", AsOf: "2026-11-11", Source: "test_fixture"}
			},
		},
		{
			name:    "installed manifest differs from pin",
			context: fixtureRightsContextValue(), installedDigest: strings.Repeat("f", 64),
			contextSource: func() sharedpolicy.RightsContext { return fixtureRightsContextValue() },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model, digest, _ := fixtureRightsModel(t, "winnow:e4b", tc.context)
			installed := digest
			if tc.installedDigest != "" {
				installed = tc.installedDigest
			}
			var tagsCalls, decideCalls int
			server := observedDaemon(t, map[string]string{"winnow:e4b": installed}, func() { tagsCalls++ }, func() { decideCalls++ })
			h, _, stderr := handlers(t, map[string]string{"OLLAYA_URL": server.URL, "OLLAYA_MODEL_POLICY_PATH": model}, `{"state":"x","questions":{"q":{"type":"noul"}}}`)
			h.RightsContext = func(context.Context) (sharedpolicy.RightsContext, error) { return tc.contextSource(), nil }
			args := localDecisionArgs(t, model, "decide.local.accurate", "winnow:e4b", digest, tc.context)
			err := h.Decide(args)
			if err == nil || decideCalls != 0 {
				t.Fatalf("err=%v decideCalls=%d; want refusal before /api/decide", err, decideCalls)
			}
			if tc.name == "missing entity scope" && tagsCalls != 0 {
				t.Fatalf("tagsCalls=%d; unknown rights must refuse before any model endpoint", tagsCalls)
			}
			if !strings.Contains(stderr.String(), "local decision") {
				t.Fatalf("stderr=%q; want typed local-decision refusal", stderr.String())
			}
		})
	}
}

func TestLocalGatewayAllowsOnlyCurrentRightsAndPinnedManifest(t *testing.T) {
	contextValue := fixtureRightsContextValue()
	modelPath, digest, _ := fixtureRightsModel(t, "winnow:e4b", contextValue)
	var tagsCalls, decideCalls int
	server := observedDaemon(t, map[string]string{"winnow:e4b": digest}, func() { tagsCalls++ }, func() { decideCalls++ })
	h, stdout, _ := handlers(t, map[string]string{"OLLAYA_URL": server.URL, "OLLAYA_MODEL_POLICY_PATH": modelPath}, `{"state":"x","questions":{"q":{"type":"noul"}}}`)
	contextReads := 0
	h.RightsContext = func(context.Context) (sharedpolicy.RightsContext, error) {
		contextReads++
		return contextValue, nil
	}
	if err := h.Decide(localDecisionArgs(t, modelPath, "decide.local.accurate", "winnow:e4b", digest, contextValue)); err != nil {
		t.Fatal(err)
	}
	var result upstream.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Upstream != "local" || result.Model != "winnow:e4b" || tagsCalls != 2 || decideCalls != 1 || contextReads != 2 {
		t.Fatalf("result=%+v tags=%d decide=%d contextReads=%d; want double context/manifest recheck then one local call", result, tagsCalls, decideCalls, contextReads)
	}
}

func TestLocalGatewayRejectsContextChangeBetweenResourceChecks(t *testing.T) {
	contextValue := fixtureRightsContextValue()
	modelPath, digest, _ := fixtureRightsModel(t, "winnow:e4b", contextValue)
	var tagsCalls, decideCalls int
	server := observedDaemon(t, map[string]string{"winnow:e4b": digest}, func() { tagsCalls++ }, func() { decideCalls++ })
	h, _, stderr := handlers(t, map[string]string{"OLLAYA_URL": server.URL, "OLLAYA_MODEL_POLICY_PATH": modelPath}, `{"state":"x","questions":{"q":{"type":"noul"}}}`)
	contextReads := 0
	h.RightsContext = func(context.Context) (sharedpolicy.RightsContext, error) {
		contextReads++
		value := contextValue
		if contextReads > 1 {
			value.Version = "fixture-v2"
		}
		return value, nil
	}
	err := h.Decide(localDecisionArgs(t, modelPath, "decide.local.accurate", "winnow:e4b", digest, contextValue))
	if err == nil || decideCalls != 0 || tagsCalls != 1 || !strings.Contains(stderr.String(), "rights receipt changed before execution") {
		t.Fatalf("err=%v tags=%d decide=%d stderr=%q; want changed context refused before provider", err, tagsCalls, decideCalls, stderr.String())
	}
}

func TestLocalGatewayRejectsManifestChangeBetweenResourceChecks(t *testing.T) {
	contextValue := fixtureRightsContextValue()
	modelPath, digest, _ := fixtureRightsModel(t, "winnow:e4b", contextValue)
	var tagsCalls, decideCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			tagsCalls++
			observed := digest
			if tagsCalls > 1 {
				observed = strings.Repeat("f", 64)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]string{{"name": "winnow:e4b", "digest": observed}}})
		case "/api/decide":
			decideCalls++
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	h, _, stderr := handlers(t, map[string]string{"OLLAYA_URL": server.URL, "OLLAYA_MODEL_POLICY_PATH": modelPath}, `{"state":"x","questions":{"q":{"type":"noul"}}}`)
	h.RightsContext = func(context.Context) (sharedpolicy.RightsContext, error) { return contextValue, nil }
	err := h.Decide(localDecisionArgs(t, modelPath, "decide.local.accurate", "winnow:e4b", digest, contextValue))
	if err == nil || decideCalls != 0 || tagsCalls != 2 || !strings.Contains(stderr.String(), "manifest identity changed") {
		t.Fatalf("err=%v tags=%d decide=%d stderr=%q; want changed manifest refused before provider", err, tagsCalls, decideCalls, stderr.String())
	}
}

func TestLocalGatewayRejectsPolicyFileRevisionChangeBeforeDecision(t *testing.T) {
	contextValue := fixtureRightsContextValue()
	modelPath, digest, _ := fixtureRightsModel(t, "winnow:e4b", contextValue)
	var tagsCalls, decideCalls int
	server := observedDaemon(t, map[string]string{"winnow:e4b": digest}, func() { tagsCalls++ }, func() { decideCalls++ })
	defer server.Close()
	h, _, stderr := handlers(t, map[string]string{"OLLAYA_URL": server.URL, "OLLAYA_MODEL_POLICY_PATH": modelPath}, `{"state":"x","questions":{"q":{"type":"noul"}}}`)
	reads := 0
	h.RightsContext = func(context.Context) (sharedpolicy.RightsContext, error) {
		reads++
		if reads == 1 {
			contents, err := os.ReadFile(modelPath)
			if err != nil {
				return sharedpolicy.RightsContext{}, err
			}
			if err := os.WriteFile(modelPath, append(contents, ' '), 0o600); err != nil {
				return sharedpolicy.RightsContext{}, err
			}
		}
		return contextValue, nil
	}
	err := h.Decide(localDecisionArgs(t, modelPath, "decide.local.accurate", "winnow:e4b", digest, contextValue))
	if err == nil || tagsCalls != 1 || decideCalls != 0 || !strings.Contains(stderr.String(), "policy changed") {
		t.Fatalf("err=%v tags=%d decide=%d stderr=%q; want changed policy file refused before decision", err, tagsCalls, decideCalls, stderr.String())
	}
}

func observedDaemon(t *testing.T, installed map[string]string, onTags, onDecide func()) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			if onTags != nil {
				onTags()
			}
			var models []map[string]string
			for name, digest := range installed {
				models = append(models, map[string]string{"name": name, "digest": digest})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"models": models})
		case "/api/decide":
			if onDecide != nil {
				onDecide()
			}
			_, _ = io.Copy(io.Discard, r.Body)
			_, _ = w.Write([]byte(`{"model":"winnow:e4b","answers":{"q":{"type":"noul","noul":0.9}},"usage":{"input_tokens":12,"output_tokens":0},"state_truncated":false}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func shippedModelPolicy(t *testing.T) policy.Policy {
	t.Helper()
	p, err := policy.Load(shippedModelPolicyPath(t))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func shippedModelPolicyPath(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "model-policy.json")
}

func fixtureRightsContextValue() sharedpolicy.RightsContext {
	return sharedpolicy.RightsContext{ID: "fixture-context", Version: "fixture-v1", EntityScope: "fixture-entity", RevenueStatus: "pre_revenue", AsOf: "2026-11-11", Source: "test_fixture"}
}

func fixtureRightsModel(t *testing.T, modelName string, contextValue sharedpolicy.RightsContext) (string, string, sharedpolicy.RightsDecision) {
	t.Helper()
	p := shippedModelPolicy(t)
	model, ok := p.Models[modelName]
	if !ok {
		t.Fatalf("fixture model %s missing from policy", modelName)
	}
	rights := &sharedpolicy.ArtifactRights{
		ArtifactID: "ollaya/" + modelName, Revision: model.Digest, SHA256: model.Digest,
		LicenseID: "LicenseRef-TestFixture", Classification: sharedpolicy.RightsUnrestricted,
		Evidence: sharedpolicy.RightsEvidence{URL: "https://example.invalid/test-license", Revision: "fixture-license-v1", SHA256: strings.Repeat("a", 64), ObservedAt: "2026-11-11"},
	}
	model.Rights = rights
	p.Models[modelName] = model
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "model-policy.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	evaluatedAt := time.Date(2026, 11, 11, 20, 0, 0, 0, time.UTC)
	decision := sharedpolicy.EvaluateRights(rights, sharedpolicy.RightsUseLocalInference, contextValue, evaluatedAt)
	return path, model.Digest, decision
}

func localDecisionArgs(t *testing.T, policyPath, role, model, digest string, contextValue sharedpolicy.RightsContext) []string {
	t.Helper()
	p, err := policy.Load(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	modelPolicy, ok := p.Models[model]
	if !ok {
		t.Fatalf("model %q is not in fixture policy", model)
	}
	at := time.Date(2026, 11, 11, 20, 0, 0, 0, time.UTC)
	identity := &sharedpolicy.RuntimeArtifactIdentity{
		Provider: sharedpolicy.RuntimeProviderOllaya, Model: model,
		ArtifactID: "ollaya/" + model, ArtifactRevision: modelPolicy.Digest,
		DigestKind: sharedpolicy.RuntimeDigestOllayaManifest, Digest: digest,
		Source: sharedpolicy.RuntimeSourceOllayaTags, ObservedAt: "2026-11-11",
	}
	receipt := sharedpolicy.NewRightsDecisionReceipt(sharedpolicy.RightsDecisionReceiptExpectation{
		Role: role, Provider: sharedpolicy.RuntimeProviderOllaya, Model: model,
		Use: sharedpolicy.RightsUseLocalInference, PolicySchemaVersion: p.SchemaVersion,
		ArtifactRights: modelPolicy.Rights, RuntimeIdentity: identity, Context: contextValue,
	}, at)
	return []string{
		"--role", role, "--input-stdin", "--json", "--expected-model", model,
		"--expected-artifact-digest", digest,
		"--expected-rights-decision-sha256", receipt.Decision.DecisionSHA256,
		"--expected-rights-evaluated-at", receipt.Decision.EvaluatedAt,
		"--expected-rights-receipt-sha256", receipt.ReceiptSHA256,
		"--expected-policy-file-sha256", p.LoadedSHA256,
	}
}

func TestPullVerifiesEveryPinnedDigest(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/pull" {
			calls++
		}
		if r.URL.Path == "/api/tags" {
			_, _ = w.Write([]byte(`{"models":[]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	h, _, _ := handlers(t, map[string]string{"OLLAYA_URL": server.URL}, "")
	h.PrepareModelPull = func(context.Context) (string, *upstream.Client, func(context.Context) error, func(), error) {
		return t.TempDir(), &upstream.Client{HTTP: http.DefaultClient, GetEnv: func(key string) string {
			if key == "OLLAYA_URL" {
				return server.URL
			}
			return ""
		}}, func(context.Context) error { return nil }, func() {}, nil
	}
	h.ModelDiskAdmit = func(context.Context, []string, policy.Policy) (*modelDiskLease, func() error, error) {
		return nil, nil, errors.New("fixture capacity refusal")
	}
	if err := h.PullModels([]string{"--model", "laya:en"}); err == nil || !strings.Contains(err.Error(), "fixture capacity refusal") {
		t.Fatalf("err = %v, want capacity refusal", err)
	}
	if calls != 0 {
		t.Fatalf("daemon mutation endpoint calls = %d; refused command must stop before pull", calls)
	}
}
