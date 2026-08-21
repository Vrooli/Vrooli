package facts

import (
	"context"
	"fmt"
	"strings"
	"time"

	"code-facts/internal/indexcontrol"
	"connectrpc.com/connect"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
)

type IndexRuntime interface {
	Reconcile(context.Context, string) (indexcontrol.Job, error)
	StartShadow(context.Context, string) (indexcontrol.Job, error)
	Cancel(context.Context, string) error
	Promote(context.Context, string) error
	Rollback(context.Context, string) error
	Cleanup(context.Context) error
}

type RuntimeIndexController struct {
	StatusReader indexcontrol.StatusReader
	Runtime      IndexRuntime
}

func NewRuntimeIndexController(reader indexcontrol.StatusReader, runtime IndexRuntime) *RuntimeIndexController {
	return &RuntimeIndexController{StatusReader: reader, Runtime: runtime}
}

func (controller *RuntimeIndexController) Status(ctx context.Context) (*factsv1.IndexStatus, error) {
	if controller == nil || controller.StatusReader == nil {
		return nil, fmt.Errorf("index status is not configured")
	}
	status, err := controller.StatusReader.Status(ctx)
	if err != nil {
		return nil, err
	}
	result := &factsv1.IndexStatus{
		ActiveGeneration: status.ActiveGeneration, PreviousGeneration: status.PreviousGeneration,
		State: status.State, SourceFiles: status.SourceFiles, SearchDocuments: status.SearchDocuments,
		SemanticCards: status.SemanticCards, GraphFacts: status.GraphFacts, StorageBytes: status.StorageBytes,
		LastReconcileAtUnix: status.LastReconcileAt.Unix(), LastReconcileOutcome: status.LastReconcileOutcome,
		DescriptorDigest: status.DescriptorDigest, SourceDigest: status.SourceDigest,
		DegradedStages: append([]string(nil), status.Degraded...),
	}
	for _, job := range status.ActiveJobs {
		result.ActiveJobs = append(result.ActiveJobs, protoIndexJob(job))
	}
	return result, nil
}

func (controller *RuntimeIndexController) Reconcile(ctx context.Context, generation string) (*factsv1.IndexControlResponse, error) {
	if err := controller.ready(); err != nil {
		return nil, err
	}
	job, err := controller.Runtime.Reconcile(ctx, generation)
	return controlResponse("reconciliation started", job, controller, ctx, err)
}

func (controller *RuntimeIndexController) Reindex(ctx context.Context, generation string) (*factsv1.IndexControlResponse, error) {
	if err := controller.ready(); err != nil {
		return nil, err
	}
	job, err := controller.Runtime.StartShadow(ctx, generation)
	return controlResponse("shadow reindex started", job, controller, ctx, err)
}

func (controller *RuntimeIndexController) Cancel(ctx context.Context, id string) (*factsv1.IndexControlResponse, error) {
	if err := controller.ready(); err != nil {
		return nil, err
	}
	if err := controller.Runtime.Cancel(ctx, id); err != nil {
		return nil, err
	}
	status, _ := controller.Status(ctx)
	return &factsv1.IndexControlResponse{Status: status, Message: "cancellation requested"}, nil
}

func (controller *RuntimeIndexController) Promote(ctx context.Context, generation string) (*factsv1.IndexControlResponse, error) {
	if err := controller.ready(); err != nil {
		return nil, err
	}
	if err := controller.Runtime.Promote(ctx, generation); err != nil {
		return nil, err
	}
	status, _ := controller.Status(ctx)
	return &factsv1.IndexControlResponse{Status: status, Message: "generation promoted"}, nil
}

func (controller *RuntimeIndexController) Rollback(ctx context.Context, generation string) (*factsv1.IndexControlResponse, error) {
	if err := controller.ready(); err != nil {
		return nil, err
	}
	if err := controller.Runtime.Rollback(ctx, generation); err != nil {
		return nil, err
	}
	status, _ := controller.Status(ctx)
	return &factsv1.IndexControlResponse{Status: status, Message: "generation rolled back"}, nil
}

func (controller *RuntimeIndexController) Cleanup(ctx context.Context, dryRun bool) (*factsv1.IndexControlResponse, error) {
	if err := controller.ready(); err != nil {
		return nil, err
	}
	message := "cleanup preview completed"
	if !dryRun {
		if err := controller.Runtime.Cleanup(ctx); err != nil {
			return nil, err
		}
		message = "bounded cleanup batch completed"
	}
	status, _ := controller.Status(ctx)
	return &factsv1.IndexControlResponse{Status: status, Message: message}, nil
}

func (controller *RuntimeIndexController) ready() error {
	if controller == nil || controller.Runtime == nil {
		return fmt.Errorf("index mutation runtime is not configured")
	}
	return nil
}

func controlResponse(message string, job indexcontrol.Job, controller *RuntimeIndexController, ctx context.Context, err error) (*factsv1.IndexControlResponse, error) {
	if err != nil {
		return nil, err
	}
	status, _ := controller.Status(ctx)
	return &factsv1.IndexControlResponse{Job: protoIndexJob(job), Status: status, Message: message}, nil
}

func protoIndexJob(job indexcontrol.Job) *factsv1.IndexJob {
	return &factsv1.IndexJob{
		Id: job.ID, Kind: job.Kind, State: protoJobState(job.State), Generation: job.Generation,
		Processed: job.Progress, Total: job.Total, Cursor: job.Cursor, Error: job.Error,
		CreatedAtUnix: job.CreatedAt.Unix(), UpdatedAtUnix: job.UpdatedAt.Unix(),
		CancellationRequested: job.CancellationRequested,
	}
}

func protoJobState(state string) factsv1.IndexJobState {
	return map[string]factsv1.IndexJobState{
		"queued":                 factsv1.IndexJobState_INDEX_JOB_STATE_QUEUED,
		"running":                factsv1.IndexJobState_INDEX_JOB_STATE_RUNNING,
		"cancellation_requested": factsv1.IndexJobState_INDEX_JOB_STATE_CANCELLATION_REQUESTED,
		"succeeded":              factsv1.IndexJobState_INDEX_JOB_STATE_SUCCEEDED,
		"failed":                 factsv1.IndexJobState_INDEX_JOB_STATE_FAILED,
		"cancelled":              factsv1.IndexJobState_INDEX_JOB_STATE_CANCELLED,
		"interrupted":            factsv1.IndexJobState_INDEX_JOB_STATE_INTERRUPTED,
	}[state]
}

// SearchControlHandler adapts Code Facts' richer generation controls to the
// provider-neutral Search Hub control contract. It contains no index logic: it
// translates authentication, job state, cancellation, and the optional
// generation actions onto the same controller used by the native API and CLI.
type SearchControlHandler struct {
	Controller IndexController
	Authorizer IndexAuthorizer
	Jobs       indexcontrol.JobStore
	Now        func() time.Time
}

func (handler *SearchControlHandler) authorize(ctx context.Context, operation, token string) error {
	if handler == nil || handler.Authorizer == nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("search control authorization is not configured"))
	}
	if err := handler.Authorizer.AuthorizeIndexControl(ctx, operation, token); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return nil
}

func (handler *SearchControlHandler) Reindex(ctx context.Context, req *connect.Request[controlv1.ReindexRequest]) (*connect.Response[controlv1.ReindexResponse], error) {
	if err := handler.authorize(ctx, "reindex", req.Msg.GetControlToken()); err != nil {
		return nil, err
	}
	status, err := handler.Controller.Status(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if req.Msg.GetDryRun() {
		return connect.NewResponse(&controlv1.ReindexResponse{PlannedUpserts: boundedInt32(status.GetSearchDocuments()), DryRun: true}), nil
	}
	action := strings.ToLower(strings.TrimSpace(req.Msg.GetAction()))
	var response *factsv1.IndexControlResponse
	switch action {
	case "", "reindex", "shadow":
		generation := strings.TrimSpace(req.Msg.GetShadowCollection())
		if generation == "" {
			now := time.Now
			if handler.Now != nil {
				now = handler.Now
			}
			generation = fmt.Sprintf("shadow-%d", now().UTC().UnixNano())
		}
		response, err = handler.Controller.Reindex(ctx, generation)
	case "reconcile":
		response, err = handler.Controller.Reconcile(ctx, status.GetActiveGeneration())
	case "promote":
		response, err = handler.Controller.Promote(ctx, req.Msg.GetShadowCollection())
	case "rollback":
		response, err = handler.Controller.Rollback(ctx, req.Msg.GetRollbackCollection())
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported reindex action %q", action))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	jobID := ""
	if response.GetJob() != nil {
		jobID = response.GetJob().GetId()
	}
	return connect.NewResponse(&controlv1.ReindexResponse{JobId: jobID, PlannedUpserts: boundedInt32(status.GetSearchDocuments())}), nil
}

func (handler *SearchControlHandler) ReindexStatus(ctx context.Context, req *connect.Request[controlv1.ReindexStatusRequest]) (*connect.Response[controlv1.ReindexStatusResponse], error) {
	if err := handler.authorize(ctx, "reindex-status", req.Msg.GetControlToken()); err != nil {
		return nil, err
	}
	if handler.Jobs == nil || strings.TrimSpace(req.Msg.GetJobId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("job_id and job store are required"))
	}
	job, err := handler.Jobs.Get(ctx, req.Msg.GetJobId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&controlv1.ReindexStatusResponse{
		JobId: job.ID, State: sharedJobState(job.State), Processed: boundedInt32(job.Progress), Total: boundedInt32(job.Total), Error: job.Error,
	}), nil
}

func (handler *SearchControlHandler) ReindexCancel(ctx context.Context, req *connect.Request[controlv1.ReindexCancelRequest]) (*connect.Response[controlv1.ReindexCancelResponse], error) {
	if err := handler.authorize(ctx, "reindex-cancel", req.Msg.GetControlToken()); err != nil {
		return nil, err
	}
	if handler.Jobs == nil || strings.TrimSpace(req.Msg.GetJobId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("job_id and job store are required"))
	}
	job, err := handler.Jobs.Get(ctx, req.Msg.GetJobId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if job.State == "succeeded" || job.State == "failed" || job.State == "cancelled" {
		return connect.NewResponse(&controlv1.ReindexCancelResponse{JobId: job.ID, Cancelled: false}), nil
	}
	if _, err := handler.Controller.Cancel(ctx, job.ID); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&controlv1.ReindexCancelResponse{JobId: job.ID, Cancelled: true}), nil
}

func (*SearchControlHandler) WriteConfig(context.Context, *connect.Request[controlv1.WriteConfigRequest]) (*connect.Response[controlv1.WriteConfigResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("Code Facts tuning is pinned; config_endpoint is intentionally absent"))
}

func (*SearchControlHandler) WriteCorpus(context.Context, *connect.Request[controlv1.WriteCorpusRequest]) (*connect.Response[controlv1.WriteCorpusResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("Code Facts corpus write-back is not advertised"))
}

func sharedJobState(state string) string {
	if state == "queued" || state == "interrupted" || state == "cancellation_requested" {
		return "pending"
	}
	return state
}

func boundedInt32(value int64) int32 {
	if value > int64(^uint32(0)>>1) {
		return int32(^uint32(0) >> 1)
	}
	if value < 0 {
		return 0
	}
	return int32(value)
}
