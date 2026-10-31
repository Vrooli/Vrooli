package bindings

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/demand"
	"github.com/vrooli/api-core/discovery"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protodesc"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
)

type invocationDemandFixture struct {
	held                        bool
	acquired, released, started int
	start                       func() error
	releaseErr                  error
}

func (f *invocationDemandFixture) Acquire(_ context.Context, req demand.AcquireRequest) (demand.Lease, error) {
	f.acquired++
	f.held = true
	return demand.Lease{LeaseID: req.LeaseID, Status: "active", ExpiresAt: time.Now().Add(req.TTL)}, nil
}

func (f *invocationDemandFixture) Renew(_ context.Context, id string, ttl time.Duration) (demand.Lease, error) {
	return demand.Lease{LeaseID: id, Status: "active", ExpiresAt: time.Now().Add(ttl)}, nil
}

func (f *invocationDemandFixture) Release(ctx context.Context, _ string, _ string) (demand.Lease, error) {
	if err := ctx.Err(); err != nil {
		return demand.Lease{}, err
	}
	f.released++
	f.held = false
	if f.releaseErr != nil {
		return demand.Lease{}, f.releaseErr
	}
	return demand.Lease{LeaseID: "invocation-lease", Status: "released"}, nil
}

func (f *invocationDemandFixture) StartScenario(context.Context, string, string) error {
	f.started++
	return f.start()
}

type bindingRoundTripper func(*http.Request) (*http.Response, error)

func (f bindingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestExecuteHoldsDemandThroughStartupAndInvocation(t *testing.T) {
	for _, mode := range []string{"running", "stopped", "starting", "start-failure", "resolve-failure", "remote-failure", "cancelled", "cleanup-failure"} {
		t.Run(mode, func(t *testing.T) {
			registry := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
			binding := registry.bindings[0]
			binding.Scenario = "fixture-target"
			fixture := &invocationDemandFixture{}
			if mode == "cleanup-failure" {
				fixture.releaseErr = errors.New("control plane unavailable during cleanup")
			}
			fixture.start = func() error {
				require.True(t, fixture.held, "startup must be protected by demand")
				if mode == "start-failure" {
					return errors.New("startup failed")
				}
				return nil
			}
			registry.SetDemandLeaseClient(fixture)
			resolveAttempts := 0
			registry.SetReachabilityResolver(func(context.Context, string) (string, error) {
				resolveAttempts++
				if mode == "starting" && fixture.started > 0 && resolveAttempts < 4 {
					return "", &discovery.Error{Kind: discovery.ErrScenarioNotRunning}
				}
				if mode == "resolve-failure" && fixture.started > 0 {
					return "", &discovery.Error{Kind: discovery.ErrCommandFailed}
				}
				if (mode == "stopped" || mode == "start-failure" || mode == "resolve-failure") && fixture.started == 0 {
					return "", &discovery.Error{Kind: discovery.ErrScenarioNotRunning}
				}
				return "http://fixture.invalid", nil
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			invoked := false
			client := &http.Client{Transport: bindingRoundTripper(func(req *http.Request) (*http.Response, error) {
				invoked = true
				require.True(t, fixture.held, "request must retain its demand lease")
				if mode == "cancelled" {
					cancel()
					return nil, ctx.Err()
				}
				status := http.StatusOK
				if mode == "remote-failure" {
					status = http.StatusServiceUnavailable
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header), Request: req}, nil
			})}
			_, err := registry.Execute(ctx, binding.GetId(), map[string]any{}, nil, false, InvocationMetadata{SessionID: "fixture-session"}, client)
			if mode == "running" || mode == "stopped" || mode == "starting" || mode == "cleanup-failure" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, mode != "start-failure" && mode != "resolve-failure", invoked)
			require.Equal(t, 1, fixture.acquired)
			require.Equal(t, 1, fixture.released, "all exit paths must release exactly once, even after cancellation")
			require.False(t, fixture.held)
		})
	}
}

func TestInferenceUsageDetailsPreserveMarginalChargePresence(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		value                  map[string]any
		present, chargePresent bool
		wantCost               int64
	}{
		{name: "fully measured", value: map[string]any{"usage": map[string]any{"input_tokens": float64(3), "output_tokens": float64(5), "cost_micros": float64(7)}}, present: true, chargePresent: true, wantCost: 7},
		{name: "explicit zero", value: map[string]any{"usage": map[string]any{"cost_micros": float64(0)}}, present: true, chargePresent: true},
		{name: "null cost", value: map[string]any{"usage": map[string]any{"cost_micros": nil}}, present: true},
		{name: "malformed cost", value: map[string]any{"usage": map[string]any{"cost_micros": "not-a-number"}}, present: true},
		{name: "negative cost", value: map[string]any{"usage": map[string]any{"cost_micros": float64(-1)}}, present: true},
		{name: "fractional cost", value: map[string]any{"usage": map[string]any{"cost_micros": float64(1.5)}}, present: true},
		{name: "unpriced", value: map[string]any{"usage": map[string]any{"input_tokens": float64(3), "output_tokens": float64(5)}}, present: true},
		{name: "missing usage", value: map[string]any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, cost, present, chargePresent := InferenceUsageDetails(tc.value)
			require.Equal(t, tc.present, present)
			require.Equal(t, tc.chargePresent, chargePresent)
			require.Equal(t, tc.wantCost, cost)
		})
	}
}

func TestResponseProjectionUsesDescriptorAndDeclaredPrimary(t *testing.T) {
	file := &descriptorpb.FileDescriptorProto{
		Name: protoString("fixture.proto"), Package: protoString("fixture"), Syntax: protoString("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: protoString("Response"), Field: []*descriptorpb.FieldDescriptorProto{
				{Name: protoString("ranked"), JsonName: protoString("ranked"), Number: protoInt32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				{Name: protoString("corpora_searched"), JsonName: protoString("corporaSearched"), Number: protoInt32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				{Name: protoString("latency_ms"), JsonName: protoString("latencyMs"), Number: protoInt32(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum()},
			}},
		},
	}
	fd, err := protodesc.NewFile(file, nil)
	require.NoError(t, err)
	message := fd.Messages().Get(0)

	ambiguous := responseProjectionFor(message, "")
	require.Empty(t, ambiguous.rowsField)
	require.Equal(t, []string{"corporaSearched", "ranked"}, ambiguous.candidateFields)
	require.Equal(t, []string{"corporaSearched", "latencyMs", "ranked"}, ambiguous.metaFields)

	declared := responseProjectionFor(message, "ranked")
	require.Equal(t, "ranked", declared.rowsField)
	require.Empty(t, declared.candidateFields)
	require.Equal(t, []string{"corporaSearched", "latencyMs"}, declared.metaFields)
}

func TestResolveMethodAcceptsAllScenarioOwnedDescriptorRoots(t *testing.T) {
	for _, source := range []string{
		"fixture/v1/service.proto",
		"schemas/fixture/v1/service.proto",
		"packages/proto/schemas/fixture/v1/service.proto",
		"scenarios/fixture/api/service.proto",
	} {
		t.Run(source, func(t *testing.T) {
			got, err := resolveMethod("fixture", "FixtureService.List", map[string][]methodInfo{
				"FixtureService.List": {{service: "fixture.v1.FixtureService", source: source}},
			}, nil)
			require.NoError(t, err)
			require.Equal(t, source, got.source)
		})
	}
}

func TestLiveDeviceStudioManifestBindsSchemaOwnedMethods(t *testing.T) {
	registry, err := Load(repoRoot(t))
	require.NoError(t, err)
	bindings := registry.List("device-studio", "")
	ids := make(map[string]bool, len(bindings))
	for _, binding := range bindings {
		ids[binding.GetId()] = true
	}
	for _, id := range []string{
		"device-studio/shots/create",
		"device-studio/shots/save",
		"device-studio/shots/get",
		"device-studio/renders/submit",
		"device-studio/renders/get",
	} {
		require.True(t, ids[id], "expected governed binding %s; registered Device Studio bindings: %v", id, ids)
	}
}

func protoString(value string) *string { return &value }
func protoInt32(value int32) *int32    { return &value }

func TestLiveBindingsHaveDescriptorDrivenResponseProjection(t *testing.T) {
	registry, err := Load(repoRoot(t))
	require.NoError(t, err)
	for _, binding := range registry.List("", "") {
		if binding.GetRowsField() == "" {
			// Scalar-only responses are represented as one raw row; ambiguous
			// repeated responses carry their candidates and fail closed.
			if len(binding.GetRowFieldCandidates()) == 0 {
				continue
			}
			require.Greater(t, len(binding.GetRowFieldCandidates()), 1, "binding %s has an invalid candidate projection", binding.GetId())
			continue
		}
		require.NotContains(t, binding.GetRowFieldCandidates(), binding.GetRowsField(), "binding %s has an unresolved selected field", binding.GetId())
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := repocontract.ResolveRepoRoot()
	require.NoError(t, err)
	return root
}

func TestProjectsManifestBoundMethods(t *testing.T) {
	// [REQ:PRT-P0-001]
	r, err := Load(repoRoot(t))
	require.NoError(t, err)
	bound, unbound := r.Count()
	if bound == 0 || unbound == 0 {
		t.Fatalf("registry snapshot = bound %d, unbound %d; expected both projections", bound, unbound)
	}
	var found bool
	for _, b := range r.List("program-runtime", "") {
		if b.GetService() != "" && b.GetMethod() != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("program-runtime manifest did not produce a typed callable")
	}
}

func TestAIGatewayExecuteBindingSerializesExactCallerVRAM(t *testing.T) {
	const requiredBytes = int64(6 * 1024 * 1024 * 1024)
	var received *routingv1.ExecuteRouteRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("read binding request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var decoded routingv1.ExecuteRouteRequest
		if err := protojson.Unmarshal(body, &decoded); err != nil {
			t.Errorf("decode binding request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		received = &decoded
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	registry, err := Load(repoRoot(t))
	require.NoError(t, err)
	registry.SetReachabilityResolver(func(context.Context, string) (string, error) {
		return server.URL, nil
	})
	_, err = registry.Execute(context.Background(), "ai-gateway/routing/execute", map[string]any{
		"role":                "classify.fast",
		"input-text":          "select the best flow",
		"profile":             "PROFILE_LOCAL_ONLY",
		"privacy-class":       "PRIVACY_CLASS_INTERNAL",
		"timeout-ms":          "15000",
		"max-cost-usd":        "0",
		"max-output-tokens":   "16",
		"required-vram-bytes": "6442450944",
		"metadata":            map[string]any{"required_vram_bytes": "1000000"},
	}, nil, true, InvocationMetadata{ProgramID: "typed-vram-binding-test"}, server.Client())
	require.NoError(t, err)
	require.NotNil(t, received)
	require.Equal(t, requiredBytes, received.GetRequest().GetRequiredVramBytes(), "Program Runtime must serialize the exact typed value to the owner RPC")
	require.Equal(t, "1000000", received.GetRequest().GetMetadata()["required_vram_bytes"], "the compatibility metadata value must survive the boundary")
}

func TestProjectManifestBindsControlPlaneMethods(t *testing.T) {
	r, err := Load(repoRoot(t))
	require.NoError(t, err)
	bindings := r.List("vrooli", "scenario")
	require.Len(t, bindings, 8)
	want := map[string]struct{}{
		"ListScenarios":     {},
		"GetScenarioStatus": {}, "GetScenarioFreshness": {}, "GetScenarioLogs": {},
		"StartScenario": {}, "StopScenario": {}, "RestartScenario": {}, "SetupScenario": {},
	}
	for _, binding := range bindings {
		if binding.GetMethod() == "GetScenarioFreshness" {
			require.Equal(t, "checks", binding.GetRowsField())
			require.Empty(t, binding.GetRowFieldCandidates())
		}
		delete(want, binding.GetMethod())
	}
	if len(want) != 0 {
		t.Fatalf("project manifest missing methods: %v", want)
	}
}

func TestProjectControlPlaneBindingsAreOwnedBySharedCLIContracts(t *testing.T) {
	r, err := Load(repoRoot(t))
	require.NoError(t, err)
	doctor := r.Doctor("vrooli")
	require.Equal(t, int32(8), doctor.GetBindings())
	require.Zero(t, doctor.GetMisroutes())
	require.Zero(t, doctor.GetFieldCollisions())
	require.Zero(t, doctor.GetControlFlagsBound())
	require.Zero(t, doctor.GetRequiredFieldsUnpopulated())
	require.Empty(t, doctor.GetIssues())
}

func TestResolveTargetURLDefaultUsesProjectPort(t *testing.T) {
	t.Setenv("VROOLI_API_PORT", "18092")
	t.Setenv("VROOLI_API_HOST", "localhost")
	got, err := ResolveTargetURLDefault(context.Background(), "vrooli")
	require.NoError(t, err)
	require.Equal(t, "http://localhost:18092", got)
}

func TestRegistryRefreshesOnManifestGenerationChange(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	descriptorPath := filepath.Join(dir, "image.binpb")
	raw, err := os.ReadFile(filepath.Join(root, "packages", "proto", "gen", "descriptor", "image.binpb"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(descriptorPath, raw, 0o644))
	manifestPath := filepath.Join(dir, "manifest.json")
	manifest, err := os.ReadFile(filepath.Join(root, "scenarios", "program-runtime", "cli", "manifest.json"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifestPath, manifest, 0o644))
	source, err := descriptorimage.New(descriptorimage.Config{DescriptorPath: descriptorPath, ManifestPaths: []string{manifestPath}})
	require.NoError(t, err)
	registry, err := LoadFromSource(source)
	require.NoError(t, err)
	_, firstGeneration, _, _, _ := registry.SnapshotMetadata()
	time.Sleep(2 * time.Millisecond)
	require.NoError(t, os.WriteFile(manifestPath, append(manifest, '\n'), 0o644))
	registry.Count()
	_, secondGeneration, _, _, _ := registry.SnapshotMetadata()
	if secondGeneration <= firstGeneration {
		t.Fatalf("registry generation did not advance: first=%d second=%d", firstGeneration, secondGeneration)
	}
}

func TestNewManifestNeedsNoScenarioCode(t *testing.T) {
	// [REQ:PRT-P0-001]
	root := repoRoot(t)
	fixture := filepath.Join(t.TempDir(), "manifest.json")
	data := []byte(`{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	require.NoError(t, os.WriteFile(fixture, data, 0o644))
	r, err := LoadFiles(filepath.Join(root, "packages/proto/gen/descriptor/image.binpb"), []string{fixture})
	require.NoError(t, err)
	if got := len(r.List("program-runtime", "records")); got != 1 {
		t.Fatalf("fixture produced %d bindings, want 1", got)
	}
}

func TestNestedManifestCommandsRetainTheirInvocationPath(t *testing.T) {
	manifest := `{"name":"program-runtime","groups":[{"name":"bindings","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}],"groups":[{"name":"compatibility","flat":true,"commands":[{"name":"describe","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"DescribeBinding"},"governance":{"effect":"read","run_eligible":true}}]},{"name":"audit","commands":[{"name":"doctor","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}]}`
	r := fixtureRegistry(t, manifest)

	require.NotNil(t, r.byID["program-runtime/bindings/list"])
	require.NotNil(t, r.byID["program-runtime/bindings/describe"], "flat child commands dispatch at the parent path")
	require.NotNil(t, r.byID["program-runtime/bindings/audit/doctor"], "non-flat child commands retain the complete invocation path")
	require.Equal(t, "bindings", r.byID["program-runtime/bindings/audit/doctor"].GetGroup())
	require.Equal(t, "audit/doctor", r.byID["program-runtime/bindings/audit/doctor"].GetCommand())
}

func TestMalformedManifestIsSkippedAndReported(t *testing.T) {
	root := repoRoot(t)
	valid := filepath.Join(t.TempDir(), "valid", "manifest.json")
	malformed := filepath.Join(t.TempDir(), "broken", "manifest.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(valid), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(malformed), 0o755))
	require.NoError(t, os.WriteFile(valid, []byte(`{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`), 0o644))
	require.NoError(t, os.WriteFile(malformed, []byte(`{"name":"broken","groups":[]}`), 0o644))

	r, err := LoadFiles(filepath.Join(root, "packages/proto/gen/descriptor/image.binpb"), []string{valid, malformed})
	require.NoError(t, err)
	require.Len(t, r.List("program-runtime", "records"), 1)
	doctor := r.Doctor("")
	require.Equal(t, int32(1), doctor.GetSkippedManifestCount())
	require.Len(t, doctor.GetSkippedManifests(), 1)
	require.Equal(t, "broken", doctor.GetSkippedManifests()[0].GetScenario())
	require.Contains(t, doctor.GetSkippedManifests()[0].GetParseError(), "at least one group")

	malformedUnbound := r.Unbound("broken")
	require.Len(t, malformedUnbound, 1)
	require.Equal(t, bindingsv1.UnboundReason_UNBOUND_REASON_MALFORMED_MANIFEST, malformedUnbound[0].GetReason())
}

func TestMissingDescriptorImageRemainsFatal(t *testing.T) {
	_, err := LoadFiles(filepath.Join(t.TempDir(), "missing-image.binpb"), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "read descriptor image")
}

func TestDoctorReportsSemanticBindingCounts(t *testing.T) {
	root := repoRoot(t)
	fixture := filepath.Join(t.TempDir(), "manifest.json")
	manifest := `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"create","flags":[{"name":"title","bind":{"field":"id"}},{"name":"body","bind":{"field":"id"}},{"name":"json","bind":{"field":"id"}}],"binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"DescribeBinding"},"governance":{"effect":"write","run_eligible":true}}]}]}`
	require.NoError(t, os.WriteFile(fixture, []byte(manifest), 0o644))
	r, err := LoadFiles(filepath.Join(root, "packages/proto/gen/descriptor/image.binpb"), []string{fixture})
	require.NoError(t, err)
	doctor := r.Doctor("program-runtime")
	require.Equal(t, int32(1), doctor.GetFieldCollisions())
	require.Equal(t, int32(1), doctor.GetControlFlagsBound())
	require.Equal(t, int32(0), doctor.GetRequiredFieldsUnpopulated())
	require.Equal(t, int32(0), doctor.GetBindsWhereRenameSuffices())
}

func TestDoctorReportsManifestCeilingAndReachability(t *testing.T) {
	registry := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	registry.SetReachabilityResolver(func(_ context.Context, scenario string) (string, error) {
		if scenario == "program-runtime" {
			return "http://127.0.0.1:19001", nil
		}
		return "", errors.New("not running")
	})
	doctor := registry.Doctor("program-runtime")
	require.Equal(t, int32(1), doctor.GetManifestScenarios())
	require.Equal(t, int32(1), doctor.GetTotalScenarios())
	require.Equal(t, []string{"program-runtime"}, doctor.GetReachableScenarios())
	require.Empty(t, doctor.GetUnreachableScenarios())
}

func TestReachabilityReprobesAfterTTL(t *testing.T) {
	registry := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	clock := time.Date(2026, time.August, 14, 3, 0, 0, 0, time.UTC)
	registry.SetClock(func() time.Time { return clock })
	probes := 0
	registry.SetReachabilityResolver(func(context.Context, string) (string, error) {
		probes++
		return "http://127.0.0.1:19001", nil
	})
	first := registry.ReachabilityCheckedAt(context.Background(), "program-runtime")
	require.Equal(t, 1, probes)
	clock = clock.Add(reachabilityTTL - time.Nanosecond)
	require.Equal(t, first, registry.ReachabilityCheckedAt(context.Background(), "program-runtime"))
	require.Equal(t, 1, probes)
	clock = clock.Add(2 * time.Nanosecond)
	second := registry.ReachabilityCheckedAt(context.Background(), "program-runtime")
	require.Equal(t, 2, probes)
	require.True(t, second.After(first))
}

func TestTransportInvalidationOnlyRemovesFailingScenario(t *testing.T) {
	registry := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	now := time.Date(2026, time.August, 14, 3, 0, 0, 0, time.UTC)
	registry.SetClock(func() time.Time { return now })
	registry.reachabilityCache["program-runtime"] = reachabilityStatus{reachable: true, reason: "cached"}
	registry.reachabilityCache["other"] = reachabilityStatus{reachable: true, reason: "cached"}
	registry.reachabilityAt["program-runtime"] = now
	registry.reachabilityAt["other"] = now
	registry.invalidateReachability("program-runtime")
	_, ok := registry.reachabilityCache["program-runtime"]
	require.False(t, ok)
	_, ok = registry.reachabilityCache["other"]
	require.True(t, ok)
}

type sweepRecorder struct{ rows []Invocation }

func (r *sweepRecorder) RecordInvocation(_ context.Context, invocation Invocation) error {
	r.rows = append(r.rows, invocation)
	return nil
}

func (r *sweepRecorder) ListInvocations(context.Context, time.Time, string, string) ([]Invocation, error) {
	return r.rows, nil
}

func TestSweepRefusesDestructiveEffect(t *testing.T) {
	registry := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"ops","commands":[{"name":"delete","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"destructive","run_eligible":true}}]}]}`)
	response := registry.Sweep(context.Background(), "program-runtime", "read", true)
	require.Len(t, response.GetResults(), 1)
	require.Equal(t, "not read-effect", response.GetResults()[0].GetSkippedReason())
	require.Equal(t, int32(1), response.GetSkipped())
}

func TestSweepRecordsOperatorInvocationWithLatency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"bindings":[]}`))
	}))
	defer server.Close()
	registry := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	recorder := &sweepRecorder{}
	registry.SetInvocationRecorder(recorder)
	registry.SetReachabilityResolver(func(context.Context, string) (string, error) { return server.URL, nil })
	response := registry.Sweep(context.Background(), "program-runtime", "read", false)
	require.Equal(t, int32(1), response.GetAttempted())
	require.Equal(t, int32(1), response.GetSucceeded())
	require.Len(t, recorder.rows, 1)
	require.Equal(t, "PROVENANCE_OPERATOR", recorder.rows[0].Provenance)
	require.Equal(t, "sweep", recorder.rows[0].ProgramID)
	require.Equal(t, "synthetic", recorder.rows[0].Origin)
	require.Greater(t, recorder.rows[0].LatencyMS, int64(0))
	registry.SetExerciseReader(fakeExerciseReader{})
	conditions, err := registry.Conditions(context.Background(), "", "", time.Hour)
	require.NoError(t, err)
	require.Equal(t, int32(1), conditions.GetLedgerExercise().GetInstrumentedBindings(), "operator sweep must count as ledger exercise")
}

func TestListAnnotatesAndFiltersUnreachableBindings(t *testing.T) {
	registry := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	registry.SetReachabilityResolver(func(_ context.Context, _ string) (string, error) {
		return "", errors.New("scenario API is not running")
	})
	all := registry.ListContext(context.Background(), "program-runtime", "", false)
	require.Len(t, all, 1)
	require.False(t, all[0].GetReachable())
	require.Equal(t, "scenario API is not running", all[0].GetReachabilityReason())
	require.Empty(t, registry.ListContext(context.Background(), "program-runtime", "", true))
	require.NotZero(t, registry.ReachabilityCheckedAt(context.Background(), "program-runtime"))
}

func TestRejectsUnknownFieldBeforeDispatch(t *testing.T) {
	// [REQ:PRT-P0-002]
	r := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	err := r.ValidateArguments("program-runtime/records/list", map[string]any{"not_a_field": "x"})
	if err == nil || !strings.Contains(err.Error(), "not_a_field") {
		t.Fatalf("unknown field error = %v, want offending field", err)
	}
}

func TestUnknownArgumentNamesCandidateFieldsOnce(t *testing.T) {
	r := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	err := r.ValidateArguments("program-runtime/records/list", map[string]any{"not_a_field": "x"})
	require.Error(t, err)
	require.Equal(t, 1, strings.Count(err.Error(), `argument "not_a_field"`), err.Error())
	require.Contains(t, err.Error(), "candidate fields:")
	require.NotEqual(t, "candidate fields:", err.Error()[strings.Index(err.Error(), "candidate fields:"):])
}

func TestErrorNamesOffendingField(t *testing.T) {
	// [REQ:PRT-P0-002]
	r := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	err := r.ValidateArguments("program-runtime/records/list", map[string]any{"limit": "not-an-int"})
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("mistyped field error = %v, want offending field", err)
	}
}

func TestRejectsMistypedAndMissingRequiredFields(t *testing.T) { // [REQ:PRT-P0-002]
	r := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","flags":[{"name":"limit","required":true}],"binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	if err := r.ValidateArguments("program-runtime/records/list", map[string]any{"limit": "not-an-int"}); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("mistyped field error = %v", err)
	}
	if err := r.ValidateArguments("program-runtime/records/list", nil); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("missing field error = %v", err)
	}
}

func TestEveryUnboundCapabilityCarriesAReason(t *testing.T) { // [REQ:PRT-P1-007]
	r := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"local","binding":{"kind":"local"},"governance":{"effect":"read","run_eligible":true}}]}],"omitted":[{"service":"MissingService","method":"List","reason":"not promoted"}]}`)
	for _, capability := range r.Unbound("program-runtime") {
		if capability.GetReason() == bindingsv1.UnboundReason_UNBOUND_REASON_UNSPECIFIED {
			t.Fatalf("unbound capability lacks reason: %v", capability)
		}
	}
}

func TestRunIneligibleCommandGeneratesNoBinding(t *testing.T) {
	// [REQ:PRT-P0-005]
	r := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"private","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":false}}]}]}`)
	if got := len(r.List("program-runtime", "")); got != 0 {
		t.Fatalf("run-ineligible command produced %d bindings", got)
	}
	unbound := r.Unbound("program-runtime")
	if len(unbound) != 1 || unbound[0].GetReason().String() == "UNBOUND_REASON_UNSPECIFIED" {
		t.Fatalf("run-ineligible command unbound record = %+v", unbound)
	}
}

func TestExecutePropagatesTestModeToTargetScenario(t *testing.T) {
	registry := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	binding := registry.bindings[0]
	binding.Scenario = "fixture-target"
	registry.SetReachabilityResolver(func(context.Context, string) (string, error) { return "http://fixture.invalid", nil })
	client := &http.Client{Transport: bindingRoundTripper(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, apihttp.TestModeValue, req.Header.Get(apihttp.TestModeHeader))
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header), Request: req}, nil
	})}
	_, err := registry.Execute(database.WithTestMode(context.Background()), binding.GetId(), map[string]any{}, nil, false, InvocationMetadata{}, client)
	require.NoError(t, err)
}

func fixtureRegistry(t *testing.T, manifest string) *Registry {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	require.NoError(t, os.WriteFile(path, []byte(manifest), 0o644))
	r, err := LoadFiles(filepath.Join(repoRoot(t), "packages/proto/gen/descriptor/image.binpb"), []string{path})
	require.NoError(t, err)
	return r
}

func TestProgramArgumentsRejectClientOnlyControls(t *testing.T) {
	registry, err := Load(repoRoot(t))
	require.NoError(t, err)
	for _, tc := range []struct{ id, arg string }{
		{"vrooli/scenario/logs", "follow"},
		{"vrooli/scenario/logs", "force-follow"},
		{"vrooli/scenario/logs", "clean"},
		{"vrooli/scenario/start", "open"},
		{"vrooli/scenario/start", "node"},
		{"vrooli/scenario/status", "instance"},
	} {
		_, err := registry.canonicalArguments(tc.id, map[string]any{"name": "fixture", tc.arg: true})
		require.ErrorContains(t, err, "client-side only", tc.arg)
	}
}

// [REQ:LV-10] Service availability does not change code compatibility; governance does.
func TestLearningContractDigestTracksGovernanceNotReachability(t *testing.T) {
	registry := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	binding := registry.bindings[0]
	original := registry.ContractDigest(binding.Id)
	require.NotEmpty(t, original)
	binding.Reachable = !binding.Reachable
	binding.ReachabilityReason = "temporarily unavailable"
	require.Equal(t, original, registry.ContractDigest(binding.Id))
	binding.Effect = "write"
	require.NotEqual(t, original, registry.ContractDigest(binding.Id))
}

func TestRepeatedBindingArgumentExplainsArrayRequirement(t *testing.T) {
	r := fixtureRegistry(t, `{"name":"web-console","groups":[{"name":"session","commands":[{"name":"native-usage","flags":[{"name":"threads","bind":{"field":"thread_ids"}}],"binding":{"kind":"connect-rpc","service":"SessionsService","method":"ReadNativeUsage"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	err := r.ValidateArguments("web-console/session/native-usage", map[string]any{"threads": "thread-1"})
	require.ErrorContains(t, err, `argument "threads" expects an array`)
	require.NoError(t, r.ValidateArguments("web-console/session/native-usage", map[string]any{"threads": []any{"thread-1"}}))
}

func TestOwnerMeasureProjectionValidation(t *testing.T) {
	root := repoRoot(t)
	descriptor := filepath.Join(root, "packages/proto/gen/descriptor/image.binpb")
	artifact, err := os.ReadFile(filepath.Join(root, "scenarios/agent-manager/.vrooli/generated/runtime-measures.json"))
	require.NoError(t, err)
	source, err := os.ReadFile(filepath.Join(root, "scenarios/agent-manager/api/internal/measures/measures.go"))
	require.NoError(t, err)
	for _, tc := range []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{"valid", func(s string) string { return s }, ""},
		{"unknown field", func(s string) string { return strings.Replace(s, `"version": 1`, `"version": 1, "unknown": true`, 1) }, "unknown field"},
		{"duplicate key", func(s string) string { return strings.Replace(s, `"version": 1`, `"version": 1, "version": 1`, 1) }, "duplicate"},
		{"wrong owner", func(s string) string {
			return strings.Replace(s, `"scenario": "agent-manager"`, `"scenario": "other"`, 1)
		}, "owner"},
		{"invalid permission", func(s string) string { return strings.Replace(s, `"network:internal"`, `"network:external"`, 1) }, "invalid governed"},
		{"unknown method", func(s string) string {
			return strings.Replace(s, `"method": "ToolFriction"`, `"method": "DoesNotExist"`, 1)
		}, "resolve"},
		{"invalid field", func(s string) string {
			return strings.Replace(s, `"primary_field": "summary"`, `"primary_field": "not_a_field"`, 1)
		}, "primary field"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "agent-manager")
			projection := filepath.Join(dir, ".vrooli/generated/runtime-measures.json")
			sourcePath := filepath.Join(dir, "api/internal/measures/measures.go")
			require.NoError(t, os.MkdirAll(filepath.Dir(projection), 0o755))
			require.NoError(t, os.MkdirAll(filepath.Dir(sourcePath), 0o755))
			require.NoError(t, os.WriteFile(projection, []byte(tc.mutate(string(artifact))), 0o644))
			require.NoError(t, os.WriteFile(sourcePath, source, 0o644))
			r, err := LoadFiles(descriptor, []string{projection})
			require.NoError(t, err)
			binding, ok := r.Binding("agent-manager/measures/tool-friction")
			if tc.want == "" {
				require.True(t, ok)
				require.Equal(t, []string{"network:internal"}, binding.Permissions)
				require.Equal(t, "ToolFriction", binding.Method)
				require.Zero(t, r.SkippedManifestCount())
			} else {
				require.False(t, ok)
				require.Equal(t, 1, r.SkippedManifestCount())
				require.Contains(t, r.skipped[0].ParseError, tc.want)
			}
		})
	}
}

func TestOwnerMeasureProjectionSourceDrift(t *testing.T) {
	root := repoRoot(t)
	dir := filepath.Join(t.TempDir(), "agent-manager")
	projection := filepath.Join(dir, ".vrooli/generated/runtime-measures.json")
	sourcePath := filepath.Join(dir, "api/internal/measures/measures.go")
	require.NoError(t, os.MkdirAll(filepath.Dir(projection), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(sourcePath), 0o755))
	artifact, err := os.ReadFile(filepath.Join(root, "scenarios/agent-manager/.vrooli/generated/runtime-measures.json"))
	require.NoError(t, err)
	source, err := os.ReadFile(filepath.Join(root, "scenarios/agent-manager/api/internal/measures/measures.go"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(projection, artifact, 0o644))
	require.NoError(t, os.WriteFile(sourcePath, source, 0o644))
	descriptor := filepath.Join(root, "packages/proto/gen/descriptor/image.binpb")
	watcher, err := descriptorimage.New(descriptorimage.Config{DescriptorPath: descriptor, ManifestPaths: []string{projection, sourcePath}})
	require.NoError(t, err)
	r, err := LoadFromSource(watcher)
	require.NoError(t, err)
	_, ok := r.Binding("agent-manager/measures/tool-friction")
	require.True(t, ok)
	require.NoError(t, os.WriteFile(sourcePath, append(source, '\n'), 0o644))
	_, ok = r.Binding("agent-manager/measures/tool-friction")
	require.False(t, ok)
	require.Contains(t, r.Doctor("agent-manager").SkippedManifests[0].ParseError, "stale")
	require.NoError(t, os.Remove(sourcePath))
	require.Contains(t, r.Doctor("agent-manager").SkippedManifests[0].ParseError, "unavailable")
}

func TestOwnerMeasureProjectionCollisionPreservesCLIBinding(t *testing.T) {
	root := repoRoot(t)
	dir := filepath.Join(t.TempDir(), "agent-manager")
	projection := filepath.Join(dir, ".vrooli/generated/runtime-measures.json")
	sourcePath := filepath.Join(dir, "api/internal/measures/measures.go")
	manifest := filepath.Join(dir, "cli/manifest.json")
	for _, path := range []string{projection, sourcePath, manifest} {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	}
	for _, pair := range [][2]string{{"scenarios/agent-manager/.vrooli/generated/runtime-measures.json", projection}, {"scenarios/agent-manager/api/internal/measures/measures.go", sourcePath}} {
		data, err := os.ReadFile(filepath.Join(root, pair[0]))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(pair[1], data, 0o644))
	}
	data := `{"name":"agent-manager","groups":[{"name":"measures","commands":[{"name":"tool-friction","binding":{"kind":"connect-rpc","service":"agent_manager.v1.measures.MeasuresService","method":"ToolFriction"},"governance":{"effect":"read","run_eligible":true}}]}]}`
	require.NoError(t, os.WriteFile(manifest, []byte(data), 0o644))
	r, err := LoadFiles(filepath.Join(root, "packages/proto/gen/descriptor/image.binpb"), []string{manifest, projection})
	require.NoError(t, err)
	_, ok := r.Binding("agent-manager/measures/tool-friction")
	require.True(t, ok)
	require.Equal(t, 1, r.SkippedManifestCount())
	require.Contains(t, r.skipped[0].ParseError, "conflicts")
	_, ok = r.Binding("agent-manager/measures/run-volume")
	require.False(t, ok)
}
