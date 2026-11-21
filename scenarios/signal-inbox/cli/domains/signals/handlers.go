package signals

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/shared"
	signalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/signals"
	signalsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/signals/signals_v1connect"
)

type handlers struct {
	client signalsconnect.SignalsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	client, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: signalsconnect.NewSignalsServiceClient(client, base)}
}

func (h *handlers) captureCall(ctx cliapp.OperationContext) (*signalsv1.CaptureSignalResponse, error) {
	req := &signalsv1.CaptureSignalRequest{CaptureNote: ctx.Flag("note"), Tags: parseTags(ctx.Flag("tags"))}
	switch {
	case strings.TrimSpace(ctx.Flag("url")) != "":
		req.Source = &signalsv1.CaptureSignalRequest_Url{Url: ctx.Flag("url")}
	case strings.TrimSpace(ctx.Flag("text")) != "":
		req.Source = &signalsv1.CaptureSignalRequest_Text{Text: ctx.Flag("text")}
	case strings.TrimSpace(ctx.Flag("image-payload-ref")) != "":
		req.Source = &signalsv1.CaptureSignalRequest_ImagePayloadRef{ImagePayloadRef: ctx.Flag("image-payload-ref")}
	}
	resp, err := h.client.CaptureSignal(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("capture signal", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Signal == nil {
		return nil, fmt.Errorf("server returned no signal")
	}
	return resp.Msg, nil
}

func parseTags(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if tag := strings.TrimSpace(part); tag != "" {
			result = append(result, tag)
		}
	}
	return result
}

func (h *handlers) captureReport(_ cliapp.OperationContext, resp *signalsv1.CaptureSignalResponse) cliapp.MutationReport {
	prefix := "Captured"
	if resp.Duplicate {
		prefix = "Reused existing"
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("%s signal %s.", prefix, resp.Signal.Id)}, Changes: []string{formatSignal(resp.Signal)}, NextCommand: []string{"`signals list` — view the full corpus"}}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*signalsv1.GetSignalResponse, error) {
	resp, err := h.client.GetSignal(context.Background(), connect.NewRequest(&signalsv1.GetSignalRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get signal", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Signal == nil {
		return nil, fmt.Errorf("server returned no signal")
	}
	return resp.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, resp *signalsv1.GetSignalResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Fetched signal."}, ResultsHeading: "Signal", Results: []string{formatSignal(resp.Signal)}}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*signalsv1.ListSignalsResponse, error) {
	var pageSize uint64
	if raw := strings.TrimSpace(ctx.Flag("page-size")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("page-size: %w", err)
		}
		pageSize = parsed
	}
	resp, err := h.client.ListSignals(context.Background(), connect.NewRequest(&signalsv1.ListSignalsRequest{PageSize: uint32(pageSize)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list signals", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no signals response")
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, resp *signalsv1.ListSignalsResponse) cliapp.ListReport {
	rows := make([]string, 0, len(resp.Signals))
	for _, signal := range resp.Signals {
		rows = append(rows, formatSignal(signal))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d signal(s).", len(rows))}, ResultsHeading: "Signals", Results: rows, RetrievalHints: []string{"`signals capture --url <url>` — capture a URL", "`signals get <id>` — read one signal"}}
}

func formatSignal(signal *sharedv1.Signal) string {
	if signal == nil {
		return "(nil)"
	}
	captured := ""
	if signal.CapturedAt != nil {
		captured = signal.CapturedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — kind=%s captured=%s needs_attention=%t", signal.Id, signal.SourceKind.String(), captured, signal.NeedsAttention)
}
