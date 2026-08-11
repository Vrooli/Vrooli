package bindings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/repo-contract-go"
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

func TestNewManifestNeedsNoScenarioCode(t *testing.T) {
	// [REQ:PRT-P0-001]
	root := repoRoot(t)
	fixture := filepath.Join(t.TempDir(), "manifest.json")
	data := []byte(`{"name":"document-manager","groups":[{"name":"notes","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"NotesService","method":"ListNotes"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	require.NoError(t, os.WriteFile(fixture, data, 0o644))
	r, err := LoadFiles(filepath.Join(root, "packages/proto/gen/descriptor/image.binpb"), []string{fixture})
	require.NoError(t, err)
	if got := len(r.List("document-manager", "notes")); got != 1 {
		t.Fatalf("fixture produced %d bindings, want 1", got)
	}
}

func TestMalformedManifestIsSkippedAndReported(t *testing.T) {
	root := repoRoot(t)
	valid := filepath.Join(t.TempDir(), "valid", "manifest.json")
	malformed := filepath.Join(t.TempDir(), "broken", "manifest.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(valid), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(malformed), 0o755))
	require.NoError(t, os.WriteFile(valid, []byte(`{"name":"document-manager","groups":[{"name":"notes","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"NotesService","method":"ListNotes"},"governance":{"effect":"read","run_eligible":true}}]}]}`), 0o644))
	require.NoError(t, os.WriteFile(malformed, []byte(`{"name":"broken","groups":[]}`), 0o644))

	r, err := LoadFiles(filepath.Join(root, "packages/proto/gen/descriptor/image.binpb"), []string{valid, malformed})
	require.NoError(t, err)
	require.Len(t, r.List("document-manager", "notes"), 1)
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
	manifest := `{"name":"document-manager","groups":[{"name":"notes","commands":[{"name":"create","flags":[{"name":"title","bind":{"field":"title"}},{"name":"body","bind":{"field":"title"}},{"name":"json","bind":{"field":"title"}}],"binding":{"kind":"connect-rpc","service":"NotesService","method":"CreateNote"},"governance":{"effect":"write","run_eligible":true}}]}]}`
	require.NoError(t, os.WriteFile(fixture, []byte(manifest), 0o644))
	r, err := LoadFiles(filepath.Join(root, "packages/proto/gen/descriptor/image.binpb"), []string{fixture})
	require.NoError(t, err)
	doctor := r.Doctor("document-manager")
	require.Equal(t, int32(1), doctor.GetFieldCollisions())
	require.Equal(t, int32(1), doctor.GetControlFlagsBound())
	require.Equal(t, int32(0), doctor.GetRequiredFieldsUnpopulated())
	require.Equal(t, int32(2), doctor.GetBindsWhereRenameSuffices())
}

func TestRejectsUnknownFieldBeforeDispatch(t *testing.T) {
	// [REQ:PRT-P0-002]
	r := fixtureRegistry(t, `{"name":"document-manager","groups":[{"name":"notes","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"NotesService","method":"ListNotes"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	err := r.ValidateArguments("document-manager/notes/list", map[string]any{"not_a_field": "x"})
	if err == nil || !strings.Contains(err.Error(), "not_a_field") {
		t.Fatalf("unknown field error = %v, want offending field", err)
	}
}

func TestErrorNamesOffendingField(t *testing.T) {
	// [REQ:PRT-P0-002]
	r := fixtureRegistry(t, `{"name":"document-manager","groups":[{"name":"notes","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"NotesService","method":"ListNotes"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	err := r.ValidateArguments("document-manager/notes/list", map[string]any{"limit": "not-an-int"})
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("mistyped field error = %v, want offending field", err)
	}
}

func TestRejectsMistypedAndMissingRequiredFields(t *testing.T) { // [REQ:PRT-P0-002]
	r := fixtureRegistry(t, `{"name":"document-manager","groups":[{"name":"notes","commands":[{"name":"list","flags":[{"name":"limit","required":true}],"binding":{"kind":"connect-rpc","service":"NotesService","method":"ListNotes"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	if err := r.ValidateArguments("document-manager/notes/list", map[string]any{"limit": "not-an-int"}); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("mistyped field error = %v", err)
	}
	if err := r.ValidateArguments("document-manager/notes/list", nil); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("missing field error = %v", err)
	}
}

func TestEveryUnboundCapabilityCarriesAReason(t *testing.T) { // [REQ:PRT-P1-007]
	r := fixtureRegistry(t, `{"name":"document-manager","groups":[{"name":"notes","commands":[{"name":"local","binding":{"kind":"local"},"governance":{"effect":"read","run_eligible":true}}]}],"omitted":[{"service":"MissingService","method":"List","reason":"not promoted"}]}`)
	for _, capability := range r.Unbound("document-manager") {
		if capability.GetReason() == bindingsv1.UnboundReason_UNBOUND_REASON_UNSPECIFIED {
			t.Fatalf("unbound capability lacks reason: %v", capability)
		}
	}
}

func TestRunIneligibleCommandGeneratesNoBinding(t *testing.T) {
	// [REQ:PRT-P0-005]
	r := fixtureRegistry(t, `{"name":"document-manager","groups":[{"name":"notes","commands":[{"name":"private","binding":{"kind":"connect-rpc","service":"NotesService","method":"ListNotes"},"governance":{"effect":"read","run_eligible":false}}]}]}`)
	if got := len(r.List("document-manager", "")); got != 0 {
		t.Fatalf("run-ineligible command produced %d bindings", got)
	}
	unbound := r.Unbound("document-manager")
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
