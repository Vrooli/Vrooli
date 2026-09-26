package reconcile

import (
	"react-component-library/internal/availability"
	"strings"
	"testing"
)

func TestVerdictVocabularyIsClosed(t *testing.T) {
	want := []Verdict{"matches", "drifted", "missing", "extra", "unverifiable", "resolved-local"}
	if len(VerdictVocabulary) != len(want) {
		t.Fatalf("got %d verdicts", len(VerdictVocabulary))
	}
	for i := range want {
		if VerdictVocabulary[i] != want[i] {
			t.Fatalf("verdict[%d] = %q", i, VerdictVocabulary[i])
		}
	}
}

func TestFindingCarriesAllFiveTypedFields(t *testing.T) {
	v := Verify("/repo/scenarios", "demo", "page", []Result{{Region: "r", Required: true, Reason: "none"}, {FilePath: "ui/src/Extra.tsx", Extra: true}}, nil)
	for _, row := range v.Regions {
		f := row.Finding
		if f.CatalogID == "" || f.Scope == "" || f.Owner == "" || f.Severity == "" {
			t.Fatalf("incomplete finding: %#v", f)
		}
		if f.Blocking {
			t.Fatal("phase-four findings must remain advisory")
		}
	}
}

func TestExitCodeFollowsVerdicts(t *testing.T) {
	passing := Verify("/repo/scenarios", "demo", "page", []Result{{Region: "r", FilePath: "x", Proven: true, Provenance: ProvenanceCustom}, {Region: "optional"}}, nil)
	if !passing.Passes {
		t.Fatal("optional unverifiable region should remain advisory when another source is proved")
	}
	failing := Verify("/repo/scenarios", "demo", "page", []Result{{Region: "r", Required: true, Reason: "none"}}, nil)
	if failing.Passes {
		t.Fatal("required missing region should fail")
	}
}

func TestCoverageRequiresExactProvenImplementation(t *testing.T) {
	baseline := Result{Region: "body", Required: true, Proven: true, FilePath: "ui/src/Body.tsx", Provenance: ProvenanceAdoptedUnmodified, ObservedAsset: "components.body", ObservedVersion: "1.0.0"}
	fills := map[string]Fill{"body": {Asset: "components.body", Version: "1.0.0"}}
	for _, tc := range []struct {
		name           string
		edit           func(*Result)
		asset, version string
		want           Verdict
	}{
		{name: "exact", want: VerdictMatches},
		{name: "custom", edit: func(r *Result) { r.Provenance = ProvenanceCustom }, want: VerdictDrifted},
		{name: "unknown", edit: func(r *Result) { r.Provenance = ProvenanceUnknown }, want: VerdictUnverifiable},
		{name: "heuristic", edit: func(r *Result) { r.Heuristic = true }, want: VerdictUnverifiable},
		{name: "wrong observed asset", edit: func(r *Result) { r.ObservedAsset = "components.other" }, want: VerdictDrifted},
		{name: "wrong observed version", edit: func(r *Result) { r.ObservedVersion = "2.0.0" }, want: VerdictDrifted},
		{name: "wrong build asset", asset: "components.other", want: VerdictUnverifiable},
		{name: "wrong build version", version: "2.0.0", want: VerdictUnverifiable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := baseline
			if tc.edit != nil {
				tc.edit(&row)
			}
			asset, version := tc.asset, tc.version
			if asset == "" {
				asset = "components.body"
			}
			if version == "" {
				version = "1.0.0"
			}
			evidence := map[string]availability.Result{"body": {CatalogID: asset, Version: version, State: availability.Built}}
			got := Verify("/scenarios", "demo", "page", []Result{row}, fills, evidence)
			if got.Regions[0].Verdict != tc.want || got.Passes != (tc.want == VerdictMatches) {
				t.Fatalf("unexpected verdict %+v", got)
			}
			if (got.Coverage.Built == 1) != (tc.want == VerdictMatches) {
				t.Fatalf("unearned coverage %+v", got.Coverage)
			}
		})
	}
}
func TestCoverageDenominatorIncludesMissingAndRejectsDuplicateRegions(t *testing.T) {
	evidence := map[string]availability.Result{"body": {CatalogID: "components.body", Version: "1.0.0", State: availability.Built}}
	row := Result{Region: "body", Required: true, Proven: true, FilePath: "Body.tsx", Provenance: ProvenanceAdoptedUnmodified, ObservedAsset: "components.body", ObservedVersion: "1.0.0"}
	fills := map[string]Fill{"body": {Asset: "components.body", Version: "1.0.0"}}
	got := Verify("/scenarios", "demo", "page", []Result{row, {Region: "footer", Required: true}, {Extra: true, FilePath: "Unrelated.tsx"}}, fills, evidence)
	if got.Coverage.Total != 2 || got.Coverage.Built != 1 || got.Coverage.Missing != 1 || got.Coverage.BuiltPercent != 50 || got.Passes {
		t.Fatalf("missing region hidden: %+v", got)
	}
	duplicate := Verify("/scenarios", "demo", "page", []Result{row, row}, fills, evidence)
	if duplicate.Coverage.Total != 1 || duplicate.Coverage.Built != 0 || duplicate.Passes {
		t.Fatalf("duplicate earned coverage: %+v", duplicate)
	}
	empty := Verify("/scenarios", "demo", "page", nil, nil)
	if empty.Coverage.Status != "not_applicable" || empty.Coverage.BuiltPercent != 0 {
		t.Fatalf("empty denominator: %+v", empty)
	}
}

func TestCoverageSeparatesDeclaredLibraryAndResolvedLocal(t *testing.T) {
	joins := []Result{
		{Region: "library", LibraryAsset: "experience-surface", LibraryVersion: "1.0.2"},
		{Region: "local", LocalComponent: "Results", Proven: true, FilePath: "ui/src/Results.tsx", Provenance: ProvenanceCustom},
	}
	got := Verify("/scenarios", "demo", "page", joins, nil)
	if got.Coverage.LibraryBacked != 1 || got.Coverage.Local != 1 || got.Coverage.Total != 2 || got.Coverage.Missing != 1 {
		t.Fatalf("declarations or resolved local counted incorrectly: %+v", got.Coverage)
	}
	if got.Regions[0].LibraryAsset != "experience-surface" || got.Regions[0].LibraryVersion != "1.0.2" || got.Regions[0].SelectedAsset != "" {
		t.Fatalf("declared intent lost or conflated with sketch selection: %+v", got.Regions[0])
	}
}

func TestResolvedCustomCodeHasItsOwnVerdictAndReason(t *testing.T) {
	got := Verify("/scenarios", "demo", "page", []Result{{Region: "controls", Required: true, Proven: true, JoinRule: "binding-testid", FilePath: "ui/src/Controls.tsx", Provenance: ProvenanceCustom}}, nil)
	if got.Regions[0].Verdict != VerdictResolvedLocal || !got.Passes || got.Coverage.Resolved != 1 || got.Coverage.ResolvedLocal != 1 || got.Coverage.Missing != 0 || got.Coverage.Built != 0 {
		t.Fatalf("custom code was discarded or credited as a library build: %+v", got)
	}
	for _, text := range []string{"CUSTOM", "ui/src/Controls.tsx"} {
		if !strings.Contains(got.Regions[0].Reason, text) {
			t.Fatalf("reason omits %q: %s", text, got.Regions[0].Reason)
		}
	}
}

func TestZeroResolvedRegionsNeverPass(t *testing.T) {
	for _, joins := range [][]Result{nil, {{Region: "optional"}}, {{Region: "guessed", FilePath: "ui/src/Guessed.tsx", Heuristic: true, Provenance: ProvenanceCustom}}} {
		got := Verify("/scenarios", "demo", "page", joins, nil)
		if got.Passes || got.Coverage.Resolved != 0 {
			t.Fatalf("zero-resolution false pass: %+v", got)
		}
	}
}

func TestEveryFindingHasACompleteReason(t *testing.T) {
	joins := []Result{
		{Region: "local", Proven: true, FilePath: "Local.tsx", Provenance: ProvenanceCustom},
		{Region: "unknown", Proven: true, FilePath: "Unknown.tsx", Provenance: ProvenanceUnknown},
		{Region: "missing"}, {Region: "extra", Extra: true, FilePath: "Extra.tsx"},
	}
	got := Verify("/scenarios", "demo", "page", joins, nil)
	for _, row := range got.Regions {
		if strings.TrimSpace(row.Reason) == "" || strings.HasSuffix(strings.TrimSpace(row.Finding.Message), ":") {
			t.Fatalf("empty finding explanation: %+v", row)
		}
	}
}
