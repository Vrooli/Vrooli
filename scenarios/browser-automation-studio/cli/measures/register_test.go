package measures

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	basmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/measures"
)

func TestMeasureJSONIncludesZeroScalar(t *testing.T) {
	var out bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		JSON:   true,
		Stdout: &out,
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window"}}},
	})
	call := func(context.Context, *connect.Request[basmeasuresv1.MeasureRequest]) (*connect.Response[basmeasuresv1.RateResponse], error) {
		return connect.NewResponse(&basmeasuresv1.RateResponse{}), nil
	}
	format := func(response *basmeasuresv1.RateResponse) string { return "0.0000" }

	if err := measure(ctx, call, format); err != nil {
		t.Fatalf("measure: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "{\n  \"rate\": 0\n}" {
		t.Fatalf("zero rate JSON = %q, want rate field present", got)
	}
}
