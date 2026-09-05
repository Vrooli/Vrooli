package measures

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	basmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/measures"
	basmeasuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/measures/measuresv1connect"
	sharedmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	"google.golang.org/protobuf/proto"
)

const GroupName = "measures"

// Register exposes the descriptor-backed BAS measure commands declared in the
// manifest. The API and the CLI therefore share the same Connect contract.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := basmeasuresv1.File_browser_automation_studio_v1_measures_measures_proto.Services().ByName("MeasuresService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: MeasuresService descriptor not found", GroupName)
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := basmeasuresconnect.NewMeasuresServiceClient(httpClient, baseURL)
	bindings := map[string]func(cliapp.RunContext) error{
		"MeasuresService.ExecutionSuccessRate": func(ctx cliapp.RunContext) error {
			return measure(ctx, client.ExecutionSuccessRate, func(r *basmeasuresv1.RateResponse) string { return fmt.Sprintf("%.4f", r.GetRate()) })
		},
		"MeasuresService.ExecutionDurationP95": func(ctx cliapp.RunContext) error {
			return measure(ctx, client.ExecutionDurationP95, func(r *basmeasuresv1.DurationResponse) string { return fmt.Sprintf("%.0f ms", r.GetDurationMs()) })
		},
		"MeasuresService.StepFailureRate": func(ctx cliapp.RunContext) error {
			return measure(ctx, client.StepFailureRate, func(r *basmeasuresv1.RateResponse) string { return fmt.Sprintf("%.4f", r.GetRate()) })
		},
		"MeasuresService.SelectorFailureRate": func(ctx cliapp.RunContext) error {
			return measure(ctx, client.SelectorFailureRate, func(r *basmeasuresv1.RateResponse) string { return fmt.Sprintf("%.4f", r.GetRate()) })
		},
	}
	// The manifest uses the fully-qualified service spelling; LoadFromManifest
	// matches the generated handler map by short service name.
	normalized := []byte(strings.ReplaceAll(string(manifest), string(svc.FullName()), string(svc.Name())))
	group, err := cliapp.LoadFromManifest(normalized, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: %w", GroupName, err)
	}
	return group, nil
}

type measureCall[T any] func(context.Context, *connect.Request[basmeasuresv1.MeasureRequest]) (*connect.Response[T], error)

func measure[T any](ctx cliapp.RunContext, call measureCall[T], format func(*T) string) error {
	window := ctx.Flag("window")
	if window == "" {
		window = "this_week"
	}
	token, ok := sharedmeasuresv1.TimeWindowToken_value["TIME_WINDOW_TOKEN_"+strings.ToUpper(window)]
	if !ok {
		return fmt.Errorf("unknown time window %q", window)
	}
	resp, err := call(context.Background(), connect.NewRequest(&basmeasuresv1.MeasureRequest{Window: &sharedmeasuresv1.TimeWindow{Window: &sharedmeasuresv1.TimeWindow_Token{Token: sharedmeasuresv1.TimeWindowToken(token)}}}))
	if err != nil {
		return cliapp.WrapAPIError("measure "+ctx.Positional("subcommand"), err, nil)
	}
	msg, ok := any(resp.Msg).(proto.Message)
	if !ok {
		return fmt.Errorf("measure response is not a protobuf message")
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{Summary: []string{format(resp.Msg)}, ResultsHeading: "Measure", Results: []string{format(resp.Msg)}})
}
