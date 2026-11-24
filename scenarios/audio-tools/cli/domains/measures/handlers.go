package measures

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/measures"
	measuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/measures/measures_v1connect"
	sharedmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

type handlers struct {
	client         measuresconnect.MeasuresServiceClient
	requestContext func() (context.Context, context.CancelFunc)
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client: measuresconnect.NewMeasuresServiceClient(httpClient, baseURL),
		requestContext: func() (context.Context, context.CancelFunc) {
			return signal.NotifyContext(context.Background(), os.Interrupt)
		},
	}
}

type countResult struct {
	Count  int64  `json:"count"`
	Unit   string `json:"unit"`
	Window string `json:"window"`
}

func (h *handlers) count(method, unit string) cliapp.PrimitiveHandler {
	return cliapp.ActionCustomRenderers(func(ctx cliapp.OperationContext) (countResult, error) {
		requestCtx, cancel := h.requestContext()
		defer cancel()
		window := strings.TrimSpace(ctx.Flag("window"))
		if window == "" {
			window = "this_week"
		}
		reqWindow, err := timeWindow(window)
		if err != nil {
			return countResult{}, err
		}
		result := countResult{Unit: unit, Window: window}
		switch method {
		case "CountBYOKCredentials":
			resp, err := h.client.CountBYOKCredentials(requestCtx, connect.NewRequest(&measuresv1.CountBYOKCredentialsRequest{Window: reqWindow}))
			if err != nil {
				return countResult{}, cliapp.WrapAPIError("count BYOK credentials", err, nil)
			}
			result.Count = resp.Msg.GetCount()
		case "CountCorpusClips":
			resp, err := h.client.CountCorpusClips(requestCtx, connect.NewRequest(&measuresv1.CountCorpusClipsRequest{Window: reqWindow}))
			if err != nil {
				return countResult{}, cliapp.WrapAPIError("count corpus clips", err, nil)
			}
			result.Count = resp.Msg.GetCount()
		case "CountExperimentRuns":
			resp, err := h.client.CountExperimentRuns(requestCtx, connect.NewRequest(&measuresv1.CountExperimentRunsRequest{Window: reqWindow}))
			if err != nil {
				return countResult{}, cliapp.WrapAPIError("count experiment runs", err, nil)
			}
			result.Count = resp.Msg.GetCount()
		case "CountExperiments":
			resp, err := h.client.CountExperiments(requestCtx, connect.NewRequest(&measuresv1.CountExperimentsRequest{Window: reqWindow}))
			if err != nil {
				return countResult{}, cliapp.WrapAPIError("count experiments", err, nil)
			}
			result.Count = resp.Msg.GetCount()
		case "CountSpeakerProfiles":
			resp, err := h.client.CountSpeakerProfiles(requestCtx, connect.NewRequest(&measuresv1.CountSpeakerProfilesRequest{Window: reqWindow}))
			if err != nil {
				return countResult{}, cliapp.WrapAPIError("count speaker profiles", err, nil)
			}
			result.Count = resp.Msg.GetCount()
		case "CountWakewordTemplates":
			resp, err := h.client.CountWakewordTemplates(requestCtx, connect.NewRequest(&measuresv1.CountWakewordTemplatesRequest{Window: reqWindow}))
			if err != nil {
				return countResult{}, cliapp.WrapAPIError("count wakeword templates", err, nil)
			}
			result.Count = resp.Msg.GetCount()
		case "CountDuplexAccountingOperations":
			resp, err := h.client.CountDuplexAccountingOperations(requestCtx, connect.NewRequest(&measuresv1.CountDuplexAccountingOperationsRequest{Window: reqWindow}))
			if err != nil {
				return countResult{}, cliapp.WrapAPIError("count duplex accounting operations", err, nil)
			}
			result.Count = resp.Msg.GetCount()
		default:
			return countResult{}, fmt.Errorf("unsupported measure RPC %q", method)
		}
		return result, nil
	}, renderCount, renderCountJSON)
}

func timeWindow(token string) (*sharedmeasuresv1.TimeWindow, error) {
	values := map[string]sharedmeasuresv1.TimeWindowToken{
		"this_week":    sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK,
		"last_7d":      sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D,
		"last_30d":     sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_30D,
		"this_month":   sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_MONTH,
		"last_month":   sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_MONTH,
		"this_quarter": sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_QUARTER,
	}
	value, ok := values[token]
	if !ok {
		return nil, fmt.Errorf("unknown time window %q", token)
	}
	return &sharedmeasuresv1.TimeWindow{Window: &sharedmeasuresv1.TimeWindow_Token{Token: value}}, nil
}

func renderCount(ctx cliapp.RunContext, result countResult) error {
	_, err := fmt.Fprintf(ctx.Stdout(), "%d %s created (%s)\n", result.Count, result.Unit, result.Window)
	return err
}

func renderCountJSON(ctx cliapp.RunContext, result countResult) error {
	return json.NewEncoder(ctx.Stdout()).Encode(result)
}
