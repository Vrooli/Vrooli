package measures

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/cli-core/cliapp"
	aigwmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/measures"
	aigwmeasuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/measures/measures_v1connect"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

// measuresTimeout bounds a route measure (a single SQL aggregate, fast).
const measuresTimeout = 30 * time.Second

// defaultWindow matches the manifest measures' `window` default so the command
// and any search-hub-routed question resolve the identical range when the user
// omits --window.
const defaultWindow = "this_week"

type handlers struct {
	client aigwmeasuresconnect.MeasuresServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, measuresTimeout)
	return &handlers{client: aigwmeasuresconnect.NewMeasuresServiceClient(httpClient, baseURL)}
}

// scalar runs one route measure: resolve the window, call the typed RPC via
// invoke, and render the scalar. The per-command closures (register.go) supply
// only the RPC call and the formatted value, so the window/error/render
// scaffold lives here once.
func (h *handlers) scalar(ctx cliapp.RunContext, label string, invoke func(context.Context, *measuresv1.TimeWindow) (proto.Message, string, error)) error {
	window, tw, err := resolveWindow(ctx)
	if err != nil {
		return err
	}
	msg, value, err := invoke(context.Background(), tw)
	if err != nil {
		return cliapp.WrapAPIError("measures "+label, err, nil)
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s (%s).", value, window)},
		ResultsHeading: "Route measure",
		Results:        []string{fmt.Sprintf("%s (%s)", value, window)},
	})
}

func req(tw *measuresv1.TimeWindow) *connect.Request[aigwmeasuresv1.RouteMeasureRequest] {
	return connect.NewRequest(&aigwmeasuresv1.RouteMeasureRequest{Window: tw})
}

func formatRate(v float64) string { return strconv.FormatFloat(v, 'f', 4, 64) }

func resolveWindow(ctx cliapp.RunContext) (string, *measuresv1.TimeWindow, error) {
	window := strings.TrimSpace(ctx.Flag("window"))
	if window == "" {
		window = defaultWindow
	}
	tok, err := timeWindowToken(window)
	if err != nil {
		return "", nil, err
	}
	return window, &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: tok}}, nil
}

func timeWindowToken(window string) (measuresv1.TimeWindowToken, error) {
	name := "TIME_WINDOW_TOKEN_" + strings.ToUpper(window)
	v, ok := measuresv1.TimeWindowToken_value[name]
	if !ok || measuresv1.TimeWindowToken(v) == measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_UNSPECIFIED {
		return 0, fmt.Errorf("unknown time window %q (use one of: this_week, last_7d, last_30d, this_month, last_month, this_quarter)", window)
	}
	return measuresv1.TimeWindowToken(v), nil
}
