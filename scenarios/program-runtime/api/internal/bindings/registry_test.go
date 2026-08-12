package bindings

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
)

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
	require.Greater(t, recorder.rows[0].LatencyMS, int64(0))
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

func fixtureRegistry(t *testing.T, manifest string) *Registry {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	require.NoError(t, os.WriteFile(path, []byte(manifest), 0o644))
	r, err := LoadFiles(filepath.Join(repoRoot(t), "packages/proto/gen/descriptor/image.binpb"), []string{path})
	require.NoError(t, err)
	return r
}
