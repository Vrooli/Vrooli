package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
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
		// Placement is a host fact; keep the fixture hermetic. Tests that need a
		// specific device or admission verdict inject their own seams.
		Residents:     func(context.Context) ([]upstream.ResidentModel, error) { return nil, nil },
		DaemonDevice:  func(context.Context) (string, error) { return "cuda", nil },
		CapacityAdmit: func(context.Context, string, int64) (bool, string, error) { return true, "", nil },
	}, &stdout, &stderr
}

// requireModelPlacement is the policy boundary that keeps a model that cannot
// fit from paying a load. It must refuse with a retryable typed reason, never
// run the slow load, and never fail a request just because a probe was
// unavailable.
func TestRequireModelPlacement(t *testing.T) {
	p := shippedModelPolicy(t)
	winnow := policy.Resolution{Role: "decide.local.accurate", Locality: policy.LocalityLocal, Model: "winnow:e4b"}
	kev := policy.Resolution{Role: "decide.local.kev", Locality: policy.LocalityLocal, Model: "kev:4b"}
	layaEN := policy.Resolution{Role: "decide.local", Locality: policy.LocalityLocal, Model: "laya:en"}
	layaRouter := policy.Resolution{Role: "decide.local", Locality: policy.LocalityLocal, Model: "laya"}
	gib := int64(1 << 30)

	idle := func(context.Context) ([]upstream.ResidentModel, error) { return nil, nil }
	cuda := func(context.Context) (string, error) { return "cuda", nil }
	admit := func(ok bool, reason string) func(context.Context, string, int64) (bool, string, error) {
		return func(context.Context, string, int64) (bool, string, error) { return ok, reason, nil }
	}
	resident := func(models ...upstream.ResidentModel) func(context.Context) ([]upstream.ResidentModel, error) {
		return func(context.Context) ([]upstream.ResidentModel, error) { return models, nil }
	}

	cases := []struct {
		name       string
		resolution policy.Resolution
		h          *Handlers
		wantCode   string
	}{
		{"a CPU-resident GPU-only model is refused before inference", winnow,
			&Handlers{Residents: resident(upstream.ResidentModel{Name: "winnow:e4b", Device: "cpu"})}, upstream.CodeInsufficientVRAM},
		{"a GPU-resident GPU-only model proceeds without asking the broker", winnow,
			&Handlers{Residents: resident(upstream.ResidentModel{Name: "winnow:e4b", Device: "cuda:0", SizeVRAM: 9 * gib}),
				CapacityAdmit: func(context.Context, string, int64) (bool, string, error) {
					t.Fatal("broker asked for a resident model")
					return true, "", nil
				}}, ""},
		{"a GPU-only model on a CPU daemon is refused", winnow,
			&Handlers{Residents: idle, DaemonDevice: func(context.Context) (string, error) { return "cpu", nil }}, upstream.CodeInsufficientVRAM},
		{"a GPU-only model the broker will not admit is refused with the reason", winnow,
			&Handlers{Residents: idle, DaemonDevice: cuda, CapacityAdmit: admit(false, "insufficient vram: floor exceeds available")}, upstream.CodeInsufficientVRAM},
		{"a GPU-only model the broker admits proceeds", winnow,
			&Handlers{Residents: idle, DaemonDevice: cuda, CapacityAdmit: admit(true, "")}, ""},
		{"unavailable probes fail open", winnow,
			&Handlers{
				Residents:     func(context.Context) ([]upstream.ResidentModel, error) { return nil, errors.New("offline") },
				DaemonDevice:  func(context.Context) (string, error) { return "", errors.New("offline") },
				CapacityAdmit: func(context.Context, string, int64) (bool, string, error) { return false, "", errors.New("offline") },
			}, ""},
		{"an any-device model that does not fit the accelerator is refused", layaEN,
			&Handlers{Residents: idle, DaemonDevice: cuda, CapacityAdmit: admit(false, "insufficient vram")}, upstream.CodeInsufficientVRAM},
		{"an any-device model on the CPU daemon needs host memory, not VRAM", layaEN,
			&Handlers{Residents: idle, DaemonDevice: func(context.Context) (string, error) { return "cpu", nil },
				CapacityAdmit: func(_ context.Context, kind string, _ int64) (bool, string, error) {
					return kind != "ram", "swap pressure", nil
				}}, upstream.CodeInsufficientMemory},
		{"an any-device model on the CPU daemon proceeds when memory fits", layaEN,
			&Handlers{Residents: idle, DaemonDevice: func(context.Context) (string, error) { return "cpu", nil },
				CapacityAdmit: func(_ context.Context, kind string, _ int64) (bool, string, error) { return kind == "ram", "", nil }}, ""},
		{"a CPU-resident any-device model is allowed", layaEN,
			&Handlers{Residents: resident(upstream.ResidentModel{Name: "laya:en", Device: "cpu"})}, ""},
		{"a router is resident when one of its targets is", layaRouter,
			&Handlers{Residents: resident(upstream.ResidentModel{Name: "laya:multilingual", Device: "cuda:0", SizeVRAM: gib}),
				CapacityAdmit: func(context.Context, string, int64) (bool, string, error) {
					t.Fatal("broker asked for a resident router target")
					return true, "", nil
				}}, ""},
		{"a router that does not fit is refused", layaRouter,
			&Handlers{Residents: idle, DaemonDevice: cuda, CapacityAdmit: admit(false, "insufficient vram")}, upstream.CodeInsufficientVRAM},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := requirePlacement(tc.h, p, tc.resolution)
			if tc.wantCode == "" {
				if err != nil {
					t.Fatalf("requireModelPlacement = %v, want nil", err)
				}
				return
			}
			var providerErr *upstream.ProviderError
			if !errors.As(err, &providerErr) || providerErr.Code != tc.wantCode {
				t.Fatalf("err = %v, want typed %s", err, tc.wantCode)
			}
		})
	}

	t.Run("resident models are counted in the ask and idle ones unloaded largest first until it fits", func(t *testing.T) {
		need := p.Models["kev:4b"].Placement.VRAMBytes
		var asked []int64
		var unloaded []string
		residents := []upstream.ResidentModel{
			{Name: "laya:en", Device: "cuda:0", SizeVRAM: 1 * gib},
			{Name: "winnow:e4b", Device: "cuda:0", SizeVRAM: 9 * gib},
		}
		h := &Handlers{
			Residents:    func(context.Context) ([]upstream.ResidentModel, error) { return residents, nil },
			DaemonDevice: cuda,
			CapacityAdmit: func(_ context.Context, kind string, bytes int64) (bool, string, error) {
				asked = append(asked, bytes)
				// Room for the model and the small resident, not for both residents.
				return bytes <= need+gib, "insufficient vram", nil
			},
			Unload: func(_ context.Context, name string) error { unloaded = append(unloaded, name); return nil },
		}
		if err := requirePlacement(h, p, kev); err != nil {
			t.Fatalf("kev must fit once winnow is unloaded: %v", err)
		}
		if len(unloaded) != 1 || unloaded[0] != "winnow:e4b" {
			t.Fatalf("unloaded = %v, want only the largest resident", unloaded)
		}
		if len(asked) != 2 || asked[0] != need+10*gib || asked[1] != need+1*gib {
			t.Fatalf("asked = %v, want the footprint plus every resident, then plus the one that remains", asked)
		}
	})

	t.Run("a resident that cannot be unloaded refuses the load and names why", func(t *testing.T) {
		h := &Handlers{
			Residents:     resident(upstream.ResidentModel{Name: "winnow:e4b", Device: "cuda:0", SizeVRAM: 9 * gib}),
			DaemonDevice:  cuda,
			CapacityAdmit: admit(false, "insufficient vram"),
			Unload:        func(context.Context, string) error { return errors.New("daemon refused keep_alive 0") },
		}
		err := requirePlacement(h, p, kev)
		var providerErr *upstream.ProviderError
		if !errors.As(err, &providerErr) || providerErr.Code != upstream.CodeInsufficientVRAM ||
			!strings.Contains(providerErr.Message, "winnow:e4b") || !strings.Contains(providerErr.Message, "daemon refused keep_alive 0") {
			t.Fatalf("err = %v, want insufficient_vram naming the resident and the unload failure", err)
		}
	})

	t.Run("a model that cannot fit even alone is refused after unloading what it could", func(t *testing.T) {
		var unloaded []string
		h := &Handlers{
			Residents:     resident(upstream.ResidentModel{Name: "laya:en", Device: "cuda:0", SizeVRAM: gib}),
			DaemonDevice:  cuda,
			CapacityAdmit: admit(false, "other claims hold the rest"),
			Unload:        func(_ context.Context, name string) error { unloaded = append(unloaded, name); return nil },
		}
		err := requirePlacement(h, p, kev)
		var providerErr *upstream.ProviderError
		if !errors.As(err, &providerErr) || providerErr.Code != upstream.CodeInsufficientVRAM || !strings.Contains(providerErr.Message, "other claims hold the rest") {
			t.Fatalf("err = %v, want insufficient_vram carrying the broker's reason", err)
		}
		if len(unloaded) != 1 {
			t.Fatalf("unloaded = %v, want the one resident released before giving up", unloaded)
		}
	})
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
		rights["classification"] != string(sharedpolicy.RightsBehavioralRestrictions) || rights["sha256"] != shipped.Models["laya"].Digest {
		t.Fatalf("resolve = %v", resolved)
	}

	// A raw local command cannot use the pinned manifest alone as rights
	// authority; a model whose rights are unknown must fail before the
	// daemon's /api/decide endpoint.
	var tagsCalls, decideCalls int
	observed := observedDaemon(t, map[string]string{
		"laya:latest":       shipped.Models["laya"].Digest,
		"laya:en":           shipped.Models["laya:en"].Digest,
		"laya:multilingual": shipped.Models["laya:multilingual"].Digest,
	}, func() { tagsCalls++ }, func() { decideCalls++ })
	h, _, stderr := handlers(t, map[string]string{"OLLAYA_URL": observed.URL}, `{"state":"x","questions":{"q":{"type":"noul"}}}`)
	unknownContext := sharedpolicy.RightsContext{ID: "fixture", Version: "v1", RevenueStatus: "pre_revenue", AsOf: "2026-11-11", Source: "test_fixture"}
	h.RightsContext = func(context.Context) (sharedpolicy.RightsContext, error) { return unknownContext, nil }
	err := h.Decide(localDecisionArgs(t, shippedModelPolicyPath(t), "decide.local.accurate", "winnow:e4b", shipped.Models["winnow:e4b"].Digest, unknownContext))
	if err == nil || !strings.Contains(stderr.String(), "not currently allowed (unknown") || tagsCalls != 0 || decideCalls != 0 {
		t.Fatalf("err=%v stderr=%q tagsCalls=%d decideCalls=%d; want shipped unknown rights refusal before model/provider endpoint", err, stderr.String(), tagsCalls, decideCalls)
	}
}

// Verified Apache-2.0 facts bound to the pinned manifest admit local use under
// a trusted context, and every unresolved artifact stays an explicit unknown
// that the evaluator refuses.
func TestShippedRightsAdmitVerifiedModelsAndRefuseUnknownOnes(t *testing.T) {
	shipped := shippedModelPolicy(t)
	contextValue := fixtureRightsContextValue()
	at := time.Date(2026, 11, 11, 20, 0, 0, 0, time.UTC)
	for name, model := range shipped.Models {
		if model.Rights == nil {
			t.Fatalf("%s has no rights facts", name)
		}
		decision := sharedpolicy.EvaluateRights(model.Rights, sharedpolicy.RightsUseLocalInference, contextValue, at)
		switch model.Rights.Classification {
		case sharedpolicy.RightsBehavioralRestrictions:
			if decision.Status != sharedpolicy.RightsAllowed || model.Rights.SHA256 != model.Digest || model.Rights.LicenseID != "Apache-2.0" {
				t.Errorf("%s decision = %+v rights = %+v; verified Apache facts bound to the manifest must admit local use", name, decision, model.Rights)
			}
		case sharedpolicy.RightsUnverified:
			if decision.Status != sharedpolicy.RightsUnknown || model.Rights.UnverifiedReason == "" {
				t.Errorf("%s decision = %+v; an unresolved artifact must stay unknown with a reason", name, decision)
			}
		default:
			t.Errorf("%s classification %q is not one the policy ships", name, model.Rights.Classification)
		}
	}
	for _, name := range []string{"laya", "laya:en", "laya:multilingual", "laya:typed-decisions"} {
		if shipped.Models[name].Rights.Classification != sharedpolicy.RightsBehavioralRestrictions {
			t.Errorf("%s must be classified from its fetched Apache-2.0 license text", name)
		}
	}
	winnow := shipped.Models["winnow:e4b"].Rights
	if winnow.Classification != sharedpolicy.RightsUnverified {
		t.Fatalf("Winnow rights = %+v; the unresolved Gemma/Apache source conflict must stay an explicit unknown", winnow)
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

// Contract C5: `policy roles` is where callers read each local role's limits
// and placement, so they can size digests and batches from data. A router
// reports the bounds every one of its targets honours; hosted roles report none.
func TestRolesExposeModelLimitsAndPlacement(t *testing.T) {
	h, stdout, _ := handlers(t, nil, "")
	if err := h.Roles([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Roles []struct {
			Role      string
			Locality  string
			Limits    *policy.ModelLimits
			Placement *policy.ModelPlacement
		}
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	shipped := shippedModelPolicy(t)
	byRole := map[string]int{}
	for index, role := range report.Roles {
		byRole[role.Role] = index
		if role.Locality == policy.LocalityRemote {
			if role.Limits != nil || role.Placement != nil {
				t.Errorf("hosted role %s must not claim local limits or placement", role.Role)
			}
			continue
		}
		if role.Limits == nil || role.Placement == nil {
			t.Errorf("local role %s reports no limits or placement", role.Role)
		}
	}
	kev := report.Roles[byRole["decide.local.kev"]]
	if kev.Limits.ContextTokens != shipped.Models["kev:4b"].Limits.ContextTokens || kev.Placement.Device != policy.PlacementDeviceGPU || kev.Placement.KeepAlive == "" {
		t.Fatalf("decide.local.kev = %+v / %+v, want the policy's declared bounds and residency", kev.Limits, kev.Placement)
	}
	fast := report.Roles[byRole["decide.local"]]
	if fast.Limits.ContextTokens != shipped.EffectiveLimits("laya").ContextTokens {
		t.Fatalf("decide.local limits = %+v, want the router's effective bounds", fast.Limits)
	}
}

func TestResolveCarriesLimitsAndPlacementForLocalRoles(t *testing.T) {
	shipped := shippedModelPolicy(t)
	server := daemon(t, map[string]string{"kev:4b": shipped.Models["kev:4b"].Digest})
	h, stdout, _ := handlers(t, map[string]string{"OLLAYA_URL": server.URL}, "")
	if err := h.Resolve([]string{"--role", "decide.local.kev", "--json"}); err != nil {
		t.Fatal(err)
	}
	var resolved struct {
		Limits    policy.ModelLimits
		Placement policy.ModelPlacement
	}
	if err := json.Unmarshal(stdout.Bytes(), &resolved); err != nil {
		t.Fatal(err)
	}
	if resolved.Limits != *shipped.Models["kev:4b"].Limits || resolved.Placement != *shipped.Models["kev:4b"].Placement {
		t.Fatalf("resolve = %+v, want the pinned model's declared bounds", resolved)
	}
}

// A role whose model cannot fit right now must not resolve, so the gateway
// skips it before spending rights or load work on it.
func TestResolveRefusesARoleThatCannotFitWithoutPayingALoad(t *testing.T) {
	shipped := shippedModelPolicy(t)
	server := daemon(t, map[string]string{"kev:4b": shipped.Models["kev:4b"].Digest})
	h, _, stderr := handlers(t, map[string]string{"OLLAYA_URL": server.URL}, "")
	h.CapacityAdmit = func(context.Context, string, int64) (bool, string, error) { return false, "insufficient vram", nil }
	if err := h.Resolve([]string{"--role", "decide.local.kev", "--json"}); err == nil {
		t.Fatal("a role that cannot fit must not resolve")
	}
	if !strings.Contains(stderr.String(), upstream.ProviderErrorMarker+`{"code":"insufficient_vram"`) {
		t.Fatalf("stderr = %q, want the typed insufficient_vram marker", stderr.String())
	}
}

// Resolving a role must never evict a warm model: rights may still refuse the
// request, and the eviction belongs to the dispatch that will use the room.
func TestResolveTimePlacementCheckNeverUnloadsAResident(t *testing.T) {
	p := shippedModelPolicy(t)
	gib := int64(1 << 30)
	need := p.Models["kev:4b"].Placement.VRAMBytes
	var asked []int64
	h := &Handlers{
		Residents: func(context.Context) ([]upstream.ResidentModel, error) {
			return []upstream.ResidentModel{{Name: "winnow:e4b", Device: "cuda:0", SizeVRAM: 9 * gib}}, nil
		},
		DaemonDevice: func(context.Context) (string, error) { return "cuda", nil },
		CapacityAdmit: func(_ context.Context, _ string, bytes int64) (bool, string, error) {
			asked = append(asked, bytes)
			return bytes <= need+gib, "insufficient vram", nil
		},
		Unload: func(context.Context, string) error { t.Fatal("a resolve-time check must not unload"); return nil },
	}
	kev := policy.Resolution{Role: "decide.local.kev", Locality: policy.LocalityLocal, Model: "kev:4b"}
	if err := h.checkModelPlacement(context.Background(), p, kev); err != nil {
		t.Fatalf("kev fits once winnow is released, so the check must pass: %v", err)
	}
	if want := p.Models["kev:4b"].Placement.VRAMBytes; len(asked) != 1 || asked[0] != want {
		t.Fatalf("asked = %v, want the footprint alone (%d): the broker already excludes Ollaya's own holdings", asked, want)
	}
}

// vanishingEntry is a directory entry the daemon renamed away between the
// directory listing and the inspection, as a finishing download does.
type vanishingEntry struct{ fs.DirEntry }

func (vanishingEntry) Info() (fs.FileInfo, error) {
	return nil, &fs.PathError{Op: "lstat", Path: "x", Err: fs.ErrNotExist}
}

func TestModelStoreSampleToleratesEntriesThatVanishMidWalk(t *testing.T) {
	if size, err := storedEntryBytes("blobs/x.partial", vanishingEntry{}); err != nil || size != 0 {
		t.Fatalf("a renamed-away temp file = %d, %v; want zero bytes and no error, because an aborted sample cancels the pull it accounts for", size, err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "blob"), make([]byte, 1234), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := modelDirectoryBytes(root); err != nil || got != 1234 {
		t.Fatalf("modelDirectoryBytes = %d, %v; want the regular file's bytes", got, err)
	}
	if err := os.Symlink("/etc/hostname", filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := modelDirectoryBytes(root); err == nil {
		t.Fatal("a symlink in the store must still be refused")
	}
	if _, err := modelDirectoryBytes(filepath.Join(root, "missing")); err == nil {
		t.Fatal("a missing store root must still be an error")
	}
}
