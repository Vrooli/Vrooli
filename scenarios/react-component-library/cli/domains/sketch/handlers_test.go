package sketch

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
)

func TestParseTargetRequiresExactlyTwoNonemptyParts(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  *sharedv1.SketchTarget
	}{
		{name: "valid", value: "shop/home", want: &sharedv1.SketchTarget{Scenario: "shop", Page: "home"}},
		{name: "empty scenario", value: "/home"},
		{name: "empty page", value: "shop/"},
		{name: "missing slash", value: "home"},
		{name: "too many parts", value: "shop/home/mobile"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTarget(tt.value)
			if tt.want == nil {
				require.ErrorContains(t, err, "expected <scenario>/<page>")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestVerifyReportIncludesCoverageAndOutcome(t *testing.T) {
	core := cliapptest.NewTestApp(t, http.NewServeMux())
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "target"}},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"target": "shop/home"}})
	resp := &sketchv1.VerifySketchResponse{Passes: true, Coverage: &sketchv1.Coverage{
		Resolved: 3, Total: 4, ResolvedLocal: 1, BuiltPercent: 75, LibraryBacked: 2, Local: 2,
	}}
	report := verifyReport(ctx, resp)
	require.Contains(t, report.Summary, "Sketch verification: shop/home")
	require.Contains(t, report.Summary, "Proven source bindings: 3/4 (1 custom)")
	require.Contains(t, report.Summary, "Built coverage: 75.00%")
	require.Contains(t, report.Summary, "Declared regions: 2 library-backed, 2 local")
	require.Contains(t, report.Summary, "Passes: true")
	require.NoError(t, verifyOutcome(resp))
	require.EqualError(t, verifyOutcome(&sketchv1.VerifySketchResponse{}), "sketch verification found drifted or required missing regions")
	var coded interface{ ExitCode() int }
	require.ErrorAs(t, verifyOutcome(&sketchv1.VerifySketchResponse{}), &coded)
	require.Equal(t, 2, coded.ExitCode())
}

func TestExitErrorCarriesMessageAndCode(t *testing.T) {
	err := exitError{message: "failed", code: 7}
	require.Equal(t, "failed", err.Error())
	require.Equal(t, 7, err.ExitCode())
}
