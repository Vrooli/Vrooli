package investigation

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/investigation"
	policyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/investigation/investigationconnect"
)

type handlers struct {
	client policyconnect.InvestigationPolicyServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: policyconnect.NewInvestigationPolicyServiceClient(httpClient, baseURL)}
}

func (h *handlers) getPolicy(ctx cliapp.RunContext) error {
	response, err := h.client.GetPolicy(context.Background(), connect.NewRequest(&policyv1.GetPolicyRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get investigation policy", err, nil)
	}
	return cliapp.RenderProtoList(ctx, response.Msg, cliapp.ListReport{Summary: []string{"Active investigation policy loaded."}})
}

func (h *handlers) putPolicy(ctx cliapp.RunContext) error {
	path := strings.TrimSpace(ctx.Flag("file"))
	if path == "" {
		return fmt.Errorf("--file is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read policy file: %w", err)
	}
	response, err := h.client.PutPolicy(context.Background(), connect.NewRequest(&policyv1.PutPolicyRequest{PolicyJson: string(raw), Active: true, ExpectedVersion: strings.TrimSpace(ctx.Flag("expected-version"))}))
	if err != nil {
		return cliapp.WrapAPIError("store investigation policy", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, response.Msg, cliapp.MutationReport{Result: []string{"Investigation policy stored: " + response.Msg.GetVersion()}})
}

func (h *handlers) preview(ctx cliapp.RunContext) error {
	path := strings.TrimSpace(ctx.Flag("file"))
	if path == "" {
		return fmt.Errorf("--file is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read observation file: %w", err)
	}
	response, err := h.client.PreviewTrigger(context.Background(), connect.NewRequest(&policyv1.PreviewTriggerRequest{ObservationJson: string(raw)}))
	if err != nil {
		return cliapp.WrapAPIError("preview investigation trigger", err, nil)
	}
	return cliapp.RenderProtoList(ctx, response.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Trigger eligible: %t.", response.Msg.GetEligible())}})
}

func (h *handlers) record(ctx cliapp.RunContext) error {
	path := strings.TrimSpace(ctx.Flag("file"))
	if path == "" {
		return fmt.Errorf("--file is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read observation file: %w", err)
	}
	response, err := h.client.RecordTrigger(context.Background(), connect.NewRequest(&policyv1.RecordTriggerRequest{ObservationJson: string(raw)}))
	if err != nil {
		return cliapp.WrapAPIError("record investigation trigger", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, response.Msg, cliapp.MutationReport{Result: []string{"Investigation trigger recorded; reused=" + fmt.Sprint(response.Msg.GetReused())}})
}

func (h *handlers) link(ctx cliapp.RunContext) error {
	fingerprint := strings.TrimSpace(ctx.Flag("fingerprint"))
	investigationID := strings.TrimSpace(ctx.Flag("investigation-id"))
	if fingerprint == "" || investigationID == "" {
		return fmt.Errorf("--fingerprint and --investigation-id are required")
	}
	response, err := h.client.LinkTrigger(context.Background(), connect.NewRequest(&policyv1.LinkTriggerRequest{IncidentFingerprint: fingerprint, InvestigationId: investigationID}))
	if err != nil {
		return cliapp.WrapAPIError("link investigation trigger", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, response.Msg, cliapp.MutationReport{Result: []string{"Investigation linked: " + investigationID}})
}

func (h *handlers) listIncidents(ctx cliapp.RunContext) error {
	limit := 50
	if raw := strings.TrimSpace(ctx.Flag("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("--limit must be an integer: %w", err)
		}
		limit = parsed
	}
	response, err := h.client.ListIncidents(context.Background(), connect.NewRequest(&policyv1.ListIncidentsRequest{
		ExecutionId: ctx.Flag("execution-id"),
		FamilyId:    ctx.Flag("family-id"),
		State:       ctx.Flag("state"),
		Limit:       int32(limit),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list investigation incidents", err, nil)
	}
	return cliapp.RenderProtoList(ctx, response.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d investigation incident(s).", len(response.Msg.GetIncidents()))}})
}

func (h *handlers) getIncident(ctx cliapp.RunContext) error {
	fingerprint := strings.TrimSpace(ctx.Flag("fingerprint"))
	if fingerprint == "" {
		return fmt.Errorf("--fingerprint is required")
	}
	response, err := h.client.GetIncident(context.Background(), connect.NewRequest(&policyv1.GetIncidentRequest{IncidentFingerprint: fingerprint}))
	if err != nil {
		return cliapp.WrapAPIError("get investigation incident", err, nil)
	}
	return cliapp.RenderProtoList(ctx, response.Msg, cliapp.ListReport{Summary: []string{"Investigation incident loaded."}})
}

func (h *handlers) listOccurrences(ctx cliapp.RunContext) error {
	limit := 50
	if raw := strings.TrimSpace(ctx.Flag("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("--limit must be an integer: %w", err)
		}
		limit = parsed
	}
	response, err := h.client.ListOccurrences(context.Background(), connect.NewRequest(&policyv1.ListOccurrencesRequest{
		ExecutionId:  ctx.Flag("execution-id"),
		EligibleOnly: ctx.BoolFlag("eligible-only"),
		Limit:        int32(limit),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list investigation trigger occurrences", err, nil)
	}
	return cliapp.RenderProtoList(ctx, response.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d trigger occurrence(s).", len(response.Msg.GetOccurrences()))}})
}

func (h *handlers) getBrief(ctx cliapp.RunContext) error {
	response, err := h.client.GetBrief(context.Background(), connect.NewRequest(&policyv1.GetBriefRequest{
		ExecutionId: ctx.Flag("execution-id"), PhaseId: ctx.Flag("phase-id"), ValidationOperationId: ctx.Flag("validation-operation-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get investigation brief", err, nil)
	}
	return cliapp.RenderProtoList(ctx, response.Msg, cliapp.ListReport{Summary: []string{"Authoritative investigation brief loaded."}})
}
