package triage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	triagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/triage"
	triageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/triage/triage_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type handlers struct {
	client triageconnect.TriageServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	client, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: triageconnect.NewTriageServiceClient(client, base)}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*triagev1.GetTriageResponse, error) {
	response, err := h.client.GetTriage(context.Background(), connect.NewRequest(&triagev1.GetTriageRequest{SignalId: ctx.Positional("signal-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get triage", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, response *triagev1.GetTriageResponse) cliapp.ListReport {
	if response.Triage == nil {
		return cliapp.ListReport{Summary: []string{"No triage record."}}
	}
	rows := []string{fmt.Sprintf("disposition=%s revisit_at=%s", response.Triage.Disposition.GetState(), response.Triage.Disposition.GetRevisitAt().AsTime().Format(time.RFC3339))}
	for _, annotation := range response.Triage.Annotations {
		rows = append(rows, formatAnnotation(annotation))
	}
	return cliapp.ListReport{Summary: []string{"Fetched triage record."}, ResultsHeading: "Triage", Results: rows}
}

func (h *handlers) setCall(ctx cliapp.OperationContext) (*triagev1.SetDispositionResponse, error) {
	state, ok := map[string]triagev1.DispositionState{"new": triagev1.DispositionState_DISPOSITION_STATE_NEW, "triaged": triagev1.DispositionState_DISPOSITION_STATE_TRIAGED, "routed": triagev1.DispositionState_DISPOSITION_STATE_ROUTED, "done": triagev1.DispositionState_DISPOSITION_STATE_DONE, "dropped": triagev1.DispositionState_DISPOSITION_STATE_DROPPED}[strings.ToLower(ctx.Flag("state"))]
	if !ok {
		return nil, fmt.Errorf("state must be new, triaged, routed, done, or dropped")
	}
	req := &triagev1.SetDispositionRequest{SignalId: ctx.Positional("signal-id"), State: state}
	if raw := ctx.Flag("revisit-at"); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, fmt.Errorf("parse revisit-at as RFC3339: %w", err)
		}
		req.RevisitAt = timestamppb.New(value)
	}
	response, err := h.client.SetDisposition(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("set disposition", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) setReport(_ cliapp.OperationContext, response *triagev1.SetDispositionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded disposition=%s", response.Disposition.GetState())}}
}

func (h *handlers) annotateCall(ctx cliapp.OperationContext) (*triagev1.AddAnnotationResponse, error) {
	author, ok := map[string]triagev1.AnnotationAuthor{"operator": triagev1.AnnotationAuthor_ANNOTATION_AUTHOR_OPERATOR, "agent": triagev1.AnnotationAuthor_ANNOTATION_AUTHOR_AGENT, "system": triagev1.AnnotationAuthor_ANNOTATION_AUTHOR_SYSTEM}[strings.ToLower(ctx.Flag("author"))]
	if !ok {
		return nil, fmt.Errorf("author must be operator, agent, or system")
	}
	req := &triagev1.AddAnnotationRequest{SignalId: ctx.Positional("signal-id"), Author: author, Body: ctx.Flag("body")}
	kind, target := ctx.Flag("outcome-kind"), ctx.Flag("outcome-target-id")
	if kind != "" || target != "" {
		value, ok := map[string]triagev1.OutcomeKind{"scenario": triagev1.OutcomeKind_OUTCOME_KIND_SCENARIO, "backlog": triagev1.OutcomeKind_OUTCOME_KIND_BACKLOG, "idea_pipeline": triagev1.OutcomeKind_OUTCOME_KIND_IDEA_PIPELINE, "knowledge_topic": triagev1.OutcomeKind_OUTCOME_KIND_KNOWLEDGE_TOPIC}[kind]
		if !ok || target == "" {
			return nil, fmt.Errorf("outcome-kind (scenario, backlog, idea_pipeline, knowledge_topic) and outcome-target-id must be supplied together")
		}
		req.Outcome = &triagev1.OutcomeLink{Kind: value, TargetId: target}
	}
	response, err := h.client.AddAnnotation(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("add annotation", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) annotateReport(_ cliapp.OperationContext, response *triagev1.AddAnnotationResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Appended " + formatAnnotation(response.Annotation)}}
}

func formatAnnotation(annotation *triagev1.Annotation) string {
	if annotation == nil {
		return "(nil)"
	}
	return fmt.Sprintf("annotation=%s author=%s body=%q outcome=%s:%s", annotation.Id, annotation.Author, annotation.Body, annotation.Outcome.GetKind(), annotation.Outcome.GetTargetId())
}
