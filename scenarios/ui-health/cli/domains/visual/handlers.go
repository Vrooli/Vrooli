package visual

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/cli-core/cliapp"
	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth/visualhealth_v1connect"
)

type handlers struct {
	client visualconnect.VisualHealthServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: visualconnect.NewVisualHealthServiceClient(httpClient, baseURL)}
}

func (h *handlers) analyzeArtifacts(ctx cliapp.RunContext) error {
	var req visualpb.AnalyzeArtifactsRequest
	if err := readProtoJSON(ctx.Flag("request"), &req); err != nil {
		return err
	}
	resp, err := h.client.AnalyzeArtifacts(context.Background(), connect.NewRequest(&req))
	if err != nil {
		return cliapp.WrapAPIError("visual analyze-artifacts", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Visual verdict: %s (%d finding(s)).", resp.Msg.GetVerdict(), len(resp.Msg.GetFindings()))},
		ResultsHeading: "Findings",
		Results:        findingLines(resp.Msg.GetFindings()),
	})
}

func (h *handlers) compareArtifacts(ctx cliapp.RunContext) error {
	var req visualpb.CompareArtifactsRequest
	if err := readProtoJSON(ctx.Flag("request"), &req); err != nil {
		return err
	}
	resp, err := h.client.CompareArtifacts(context.Background(), connect.NewRequest(&req))
	if err != nil {
		return cliapp.WrapAPIError("visual compare-artifacts", err, nil)
	}
	lines := make([]string, 0, len(resp.Msg.GetDeltas()))
	for _, d := range resp.Msg.GetDeltas() {
		lines = append(lines, fmt.Sprintf("%s %s changed=%.4f", d.GetPage(), d.GetStatus(), d.GetChangedFraction()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Compared %d visual artifact(s).", len(resp.Msg.GetDeltas()))},
		ResultsHeading: "Deltas",
		Results:        lines,
	})
}

func (h *handlers) rules(ctx cliapp.RunContext) error {
	resp, err := h.client.ListRules(context.Background(), connect.NewRequest(&visualpb.ListRulesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("visual rules", err, nil)
	}
	lines := make([]string, 0, len(resp.Msg.GetRules()))
	for _, r := range resp.Msg.GetRules() {
		lines = append(lines, fmt.Sprintf("%s %s %s", r.GetId(), r.GetCategory(), r.GetSeverity()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d visual-health rule(s).", len(resp.Msg.GetRules()))},
		ResultsHeading: "Rules",
		Results:        lines,
	})
}

func readProtoJSON(path string, msg proto.Message) error {
	if path == "" {
		return fmt.Errorf("--request is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read request %q: %w", path, err)
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, msg); err != nil {
		return fmt.Errorf("parse request %q: %w", path, err)
	}
	return nil
}

func findingLines(findings []*visualpb.VisualFinding) []string {
	lines := make([]string, 0, len(findings))
	for _, f := range findings {
		lines = append(lines, fmt.Sprintf("%s %s %s", f.GetCode(), f.GetSeverity(), f.GetMessage()))
	}
	return lines
}
