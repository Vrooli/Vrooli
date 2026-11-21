package measures

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	mhmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/measures"
	mhmeasuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/measures/measures_v1connect"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

// measuresTimeout bounds a measure count (a single SQL aggregate, fast).
const measuresTimeout = 30 * time.Second

// defaultWindowToken matches the manifest measures' `window` default so the
// command and the search-hub-routed question resolve the identical range when
// the user omits --window.
const defaultWindowToken = "this_week"

type handlers struct {
	core   *cliapp.ScenarioApp
	client mhmeasuresconnect.MeasuresServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, measuresTimeout)
	return &handlers{
		core:   core,
		client: mhmeasuresconnect.NewMeasuresServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) failed(ctx cliapp.RunContext) error {
	window, tw, err := resolveWindow(ctx)
	if err != nil {
		return err
	}
	resp, err := h.client.CountFailedValidations(context.Background(), connect.NewRequest(&mhmeasuresv1.CountFailedValidationsRequest{Window: tw}))
	if err != nil {
		return cliapp.WrapAPIError("measures failed", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no count response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d scenario(s) failed measures validation (%s).", resp.Msg.GetCount(), window)},
		ResultsHeading: "Failed validations",
		Results:        []string{fmt.Sprintf("%d (%s)", resp.Msg.GetCount(), window)},
		RetrievalHints: []string{
			"`measures-health measures coverage` — the passing count",
			"`measures-health measures failed --window last_30d` — widen the window",
		},
	})
}

func (h *handlers) coverage(ctx cliapp.RunContext) error {
	window, tw, err := resolveWindow(ctx)
	if err != nil {
		return err
	}
	resp, err := h.client.CountValidationCoverage(context.Background(), connect.NewRequest(&mhmeasuresv1.CountValidationCoverageRequest{Window: tw}))
	if err != nil {
		return cliapp.WrapAPIError("measures coverage", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no count response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d scenario(s) passed measures validation (%s).", resp.Msg.GetCount(), window)},
		ResultsHeading: "Passing validations",
		Results:        []string{fmt.Sprintf("%d (%s)", resp.Msg.GetCount(), window)},
		RetrievalHints: []string{
			"`measures-health measures failed` — the failing count",
		},
	})
}

// resolveWindow reads --window (defaulting to this_week) and maps it to the
// shared canonical TimeWindow proto.
func resolveWindow(ctx cliapp.RunContext) (string, *measuresv1.TimeWindow, error) {
	window := strings.TrimSpace(ctx.Flag("window"))
	if window == "" {
		window = defaultWindowToken
	}
	token, err := timeWindowToken(window)
	if err != nil {
		return "", nil, err
	}
	return window, &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: token}}, nil
}

// timeWindowToken maps a lowercase canonical token to the generated proto enum.
// Unknown tokens are a usage error, never a silent fallback — a wrong window
// would silently answer the wrong question.
func timeWindowToken(token string) (measuresv1.TimeWindowToken, error) {
	name := "TIME_WINDOW_TOKEN_" + strings.ToUpper(token)
	v, ok := measuresv1.TimeWindowToken_value[name]
	if !ok || measuresv1.TimeWindowToken(v) == measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_UNSPECIFIED {
		return 0, fmt.Errorf("unknown time window %q (use one of: this_week, last_7d, last_30d, this_month, last_month, this_quarter)", token)
	}
	return measuresv1.TimeWindowToken(v), nil
}
