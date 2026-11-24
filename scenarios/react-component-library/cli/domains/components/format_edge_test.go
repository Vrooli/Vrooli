package components

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
	componenttestsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests"
)

func TestSweepReportSkipsNilRowsAndSurfacesErrors(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "resume"}}}, BoolFlags: map[string]bool{"resume": true}})
	report := (&handlers{}).sweepReport(ctx, &componenttestsv1.SweepComponentTestsResponse{
		Planned: 3, Started: 1, Skipped: 2, Passed: 1, Failed: 1, Blocked: 1, Complete: false,
		Reports: []*componenttestsv1.ComponentTestReport{nil, {Id: "report-1", RootLibraryId: "button", RootVersion: "1.0.0", Verdict: "passed"}},
		Errors:  []string{"renderer unavailable"},
	})
	if !strings.Contains(report.Summary[0], "resume") || !strings.Contains(report.Summary[0], "planned=3 started=1 skipped=2 passed=1 failed=1 blocked=1") {
		t.Fatalf("summary: %#v", report.Summary)
	}
	if len(report.Results) != 2 || !strings.Contains(report.Results[0], "report-1\tbutton@1.0.0\tpassed") || report.Results[1] != "renderer unavailable" {
		t.Fatalf("results: %#v", report.Results)
	}
}

func TestFormattingHelpersCoverUnknownAndEmptyValues(t *testing.T) {
	if formatDesignAffinity(componentsv1.DesignAffinity(99)) != "unspecified" {
		t.Fatal("unknown affinity should remain explicit")
	}
	if formatStyleFitKind(componentsv1.StyleFitVerdictKind(99)) != "unspecified" {
		t.Fatal("unknown fit kind should remain explicit")
	}
	if formatStyleFitVerdict(nil) != "(nil)" || formatStyleFitVerdict(&componentsv1.ValidateStyleFitResponse{}) != "unspecified scenario= style= component= affinity=none" {
		t.Fatal("style fit empty rendering changed")
	}
	if formatStory(nil) != "(nil)" || formatComponent(nil) != "(nil)" {
		t.Fatal("nil records should render explicitly")
	}
}
