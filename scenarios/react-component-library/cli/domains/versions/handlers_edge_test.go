package versions

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
)

func TestCleanupScopeRejectsInvalidOrConflictingFilters(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags map[string]string
		want  string
	}{
		{"negative age", map[string]string{"older-than-days": "-1"}, "non-negative integer"},
		{"nonnumeric age", map[string]string{"older-than-days": "soon"}, "non-negative integer"},
		{"conflicting filters", map[string]string{"component-id": "Button", "library-id": "button"}, "use only one"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "component-id"}, {Name: "library-id"}, {Name: "older-than-days"}}}, Flags: tc.flags})
			_, err := cleanupScope(ctx)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want %q", err, tc.want)
			}
		})
	}
}

func TestCleanupRenderingSkipsIncompleteItemsAndShowsScenarioReferences(t *testing.T) {
	got := renderCleanupItems([]*versionsv1.CleanupItem{
		nil,
		{},
		{Version: &versionsv1.RetireCandidate{LibraryId: "button", Version: "1.0.0"}, Eligible: true, AgeDays: 12, AdoptionCount: 2, DependencyCount: 1, Reason: "unused", References: []*versionsv1.VersionReference{{OwnerLibraryId: "screen", OwnerVersion: "2.0.0", OwnerPath: "src/page.tsx", OwnerScenario: "storefront", AdoptionId: "adopt-1", Kind: "import", ImportSpecifier: "@ui/button", Evidence: "source"}}},
	})
	if len(got) != 2 || !strings.Contains(got[0], "eligible age=12d") || !strings.Contains(got[1], "storefront adoption=adopt-1") || !strings.Contains(got[1], `imports "@ui/button"`) {
		t.Fatalf("unexpected cleanup rendering: %#v", got)
	}
}

func TestMaterializeRequiresCompleteTargetUnlessAll(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "component-id"}, {Name: "version"}}, Flags: []cliapp.Flag{{Name: "all"}, {Name: "into"}}}})
	if _, err := (&handlers{}).materializeCall(ctx); err == nil || !strings.Contains(err.Error(), "provide <component-id> <version>") {
		t.Fatalf("error=%v", err)
	}
}

func TestLifecycleReportsRenderMaterializationPresenceAndDoctorDetails(t *testing.T) {
	h := &handlers{}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{})

	materialized := h.materializeReport(ctx, &versionsv1.MaterializeVersionResponse{Versions: []*versionsv1.MaterializedVersion{
		{LibraryId: "button", Version: "1.2.0", Directory: "components/button/1.2.0"},
		{LibraryId: "card", Version: "2.0.0", Directory: "components/card/2.0.0", AlreadyPresent: true},
	}})
	if !strings.Contains(strings.Join(materialized.Summary, "\n"), "Materialized 2 version(s)") || !strings.Contains(strings.Join(materialized.Results, "\n"), "already-present=true") {
		t.Fatalf("unexpected materialization report: %#v", materialized)
	}

	presence := h.reconcilePresenceReport(ctx, &versionsv1.ReconcilePresenceResponse{
		Evict:       []*versionsv1.RetireCandidate{{LibraryId: "button", Version: "1.0.0", Status: "stale"}, nil},
		Materialize: []*versionsv1.RetireCandidate{{LibraryId: "card", Version: "2.0.0", Status: "missing"}},
		Unchanged:   []*versionsv1.RetireCandidate{{LibraryId: "input", Version: "1.1.0", Status: "current"}}, Applied: true,
	})
	joinedPresence := strings.Join(append(presence.Summary, presence.Results...), "\n")
	for _, want := range []string{"evict=2 materialize=1 unchanged=1 applied=true", "evict button@1.0.0 status=stale", "materialize card@2.0.0 status=missing", "unchanged input@1.1.0 status=current"} {
		if !strings.Contains(joinedPresence, want) {
			t.Errorf("presence report missing %q: %s", want, joinedPresence)
		}
	}

	doctor := h.doctorReport(ctx, &versionsv1.DoctorResponse{Issues: []*versionsv1.DoctorIssue{{
		LibraryId: "button", Version: "1.2.0", Path: "manifest.json", ExpectedSha256: "expected", ActualSha256: "actual", Reason: "hash mismatch",
	}}})
	if !strings.Contains(strings.Join(doctor.Summary, "\n"), "found 1 issue") || !strings.Contains(strings.Join(doctor.Results, "\n"), "expected=expected actual=actual (hash mismatch)") {
		t.Fatalf("unexpected doctor report: %#v", doctor)
	}
}
