package learning

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning/learningv1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

const GroupName = "learning"

type handlers struct{ client rpc.LearningServiceClient }

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	http, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	h := &handlers{rpc.NewLearningServiceClient(http, base)}
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"LearningService.RecordAttempt":     cliapp.ProtoMutation(h.record, h.recordReport),
		"LearningService.RecordObservation": cliapp.ProtoMutation(h.observe, h.observeReport),
		"LearningService.MeasureLearning":   cliapp.ProtoList(h.measure, h.measureReport),
	})
}

func (h *handlers) record(ctx cliapp.OperationContext) (*pb.RecordAttemptResponse, error) {
	a := &pb.Attempt{}
	if err := protojson.Unmarshal([]byte(ctx.Flag("attempt")), a); err != nil {
		return nil, fmt.Errorf("attempt must be an Attempt JSON object: %w", err)
	}
	res, err := h.client.RecordAttempt(context.Background(), connect.NewRequest(&pb.RecordAttemptRequest{Scope: ctx.Flag("scope"), Attempt: a}))
	if err != nil {
		return nil, cliapp.WrapAPIError("record learning", err, nil)
	}
	return res.Msg, nil
}

func (h *handlers) recordReport(_ cliapp.OperationContext, r *pb.RecordAttemptResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Recorded learning attempt " + r.EntryId}, Changes: []string{fmt.Sprintf("Existing immutable attempt: %t", r.Existing)}}
}

func (h *handlers) observe(ctx cliapp.OperationContext) (*pb.RecordObservationResponse, error) {
	o := &pb.Observation{}
	if err := protojson.Unmarshal([]byte(ctx.Flag("observation")), o); err != nil {
		return nil, fmt.Errorf("observation must be an Observation JSON object: %w", err)
	}
	res, err := h.client.RecordObservation(context.Background(), connect.NewRequest(&pb.RecordObservationRequest{Scope: ctx.Flag("scope"), Observation: o}))
	if err != nil {
		return nil, cliapp.WrapAPIError("record learning observation", err, nil)
	}
	return res.Msg, nil
}

func (h *handlers) observeReport(_ cliapp.OperationContext, r *pb.RecordObservationResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Recorded learning observation " + r.EntryId}, Changes: []string{fmt.Sprintf("Existing observation: %t", r.Existing)}}
}

func (h *handlers) measure(ctx cliapp.OperationContext) (*pb.MeasureLearningResponse, error) {
	res, err := h.client.MeasureLearning(context.Background(), connect.NewRequest(&pb.MeasureLearningRequest{Scope: ctx.Flag("scope"), From: ctx.Flag("from"), To: ctx.Flag("to"), Operation: ctx.Flag("operation"), ContextKey: ctx.Flag("context-key")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("measure learning", err, nil)
	}
	return res.Msg, nil
}

func (h *handlers) measureReport(_ cliapp.OperationContext, r *pb.MeasureLearningResponse) cliapp.ListReport {
	lines := []string{fmt.Sprintf("Eligible attempts: %d; test excluded: %d; legacy: %d; invalid: %d; scan capped: %t", r.EligibleAttempts, r.ExcludedTestAttempts, r.LegacyTaskRecords, r.InvalidRecords, r.Truncated)}
	for _, c := range r.Cohorts {
		speed := "unknown"
		if c.MedianAttemptsToSuccess != nil && c.MedianSecondsToSuccess != nil {
			speed = fmt.Sprintf("%.1f attempts / %.1fs", *c.MedianAttemptsToSuccess, *c.MedianSecondsToSuccess)
		}
		lines = append(lines, fmt.Sprintf("%s [%s]: %d attempts; %d completed / %d unresolved tasks; success median %s; recurring fingerprints %d; advice supported/contradicted/unknown %d/%d/%d", c.Operation, c.ContextKey, c.Attempts, c.CompletedTasks, c.UnresolvedTasks, speed, c.RecurringFailureFingerprints, c.SupportedAdvice, c.ContradictedAdvice, c.UnassessedAdvice))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Learning window %s to %s; reliable=%t %s", r.From, r.To, r.Reliable, r.Reason)}, ResultsHeading: "Observed learning outcomes", Results: lines, RetrievalHints: []string{r.Interpretation}}
}
