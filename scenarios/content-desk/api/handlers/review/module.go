// Package review mounts the review-run Connect surface.
package review

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	internalartifacts "content-desk/internal/artifacts"
	"content-desk/internal/module"
	internalreview "content-desk/internal/review"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/review"
	reviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/review/review_v1connect"
)

type handler struct{ service internalreview.Service }

type artifactRevisionSource struct{ repo internalartifacts.Repository }

func (s artifactRevisionSource) CurrentRevision(ctx context.Context, draftID string) (internalreview.RevisionSnapshot, error) {
	current, err := s.repo.GetCurrentRevision(ctx, draftID)
	if err != nil {
		return internalreview.RevisionSnapshot{}, err
	}
	return internalreview.RevisionSnapshot{DraftID: current.DraftID, Body: current.Body, RevisionID: current.RevisionID}, nil
}

func (s artifactRevisionSource) CurrentEditorialSnapshotTx(ctx context.Context, tx *sql.Tx, draftID string) (internalreview.EditorialSnapshot, error) {
	var snap internalreview.EditorialSnapshot
	var latestBody string
	var actorKind, capacity, revisionCreatedAt string
	err := tx.QueryRowContext(ctx, `SELECT d.id,d.body,COALESCE(r.id,''),COALESCE(r.body,''),COALESCE(sl.channel,''),COALESCE(sl.format,''),d.campaign_id,COALESCE(pr.project_id,''),d.post_type_id,d.lane,d.sku,d.scenario_name,COALESCE(r.actor_kind,''),COALESCE(r.capacity,''),COALESCE(r.created_at,'') FROM drafts d LEFT JOIN draft_slots sl ON sl.draft_id=d.id LEFT JOIN draft_project_refs pr ON pr.draft_id=d.id LEFT JOIN draft_revisions r ON r.id=(SELECT id FROM draft_revisions WHERE draft_id=d.id ORDER BY created_at DESC,id DESC LIMIT 1) WHERE d.id=?`, draftID).Scan(&snap.DraftID, &snap.Body, &snap.RevisionID, &latestBody, &snap.Context.Channel, &snap.Context.Format, &snap.Context.CampaignID, &snap.Context.ProjectID, &snap.Context.PostTypeID, &snap.Context.Lane, &snap.Context.SKU, &snap.Context.ScenarioName, &actorKind, &capacity, &revisionCreatedAt)
	if err != nil {
		return internalreview.EditorialSnapshot{}, err
	}
	if latestBody != snap.Body {
		snap.RevisionID = ""
	}
	if snap.RevisionID != "" {
		snap.RevisionProvenance = &internalreview.RevisionProvenance{ActorKind: actorKind, Capacity: capacity, CreatedAt: revisionCreatedAt}
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,asset_id,role,aspect_ratio,alt_text,position FROM draft_attachments WHERE draft_id=? ORDER BY position,id`, draftID)
	if err != nil {
		return internalreview.EditorialSnapshot{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var a internalreview.EditorialAttachment
		if err := rows.Scan(&a.ID, &a.AssetID, &a.Role, &a.AspectRatio, &a.AltText, &a.Position); err != nil {
			return internalreview.EditorialSnapshot{}, err
		}
		snap.Context.Attachments = append(snap.Context.Attachments, a)
	}
	if err := rows.Err(); err != nil {
		return internalreview.EditorialSnapshot{}, err
	}
	return snap, nil
}

var _ reviewconnect.ReviewServiceHandler = handler{}

func (h handler) ListReviewRuns(ctx context.Context, _ *connect.Request[reviewv1.ListReviewRunsRequest]) (*connect.Response[reviewv1.ListReviewRunsResponse], error) {
	runs, err := h.service.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &reviewv1.ListReviewRunsResponse{}
	for _, run := range runs {
		response.ReviewRuns = append(response.ReviewRuns, reviewRunMessage(run))
	}
	return connect.NewResponse(response), nil
}

func (h handler) ListRequiredReviewModes(ctx context.Context, request *connect.Request[reviewv1.ListRequiredReviewModesRequest]) (*connect.Response[reviewv1.ListRequiredReviewModesResponse], error) {
	modes, err := h.service.ListRequiredModes(ctx, request.Msg.PostTypeId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&reviewv1.ListRequiredReviewModesResponse{Modes: modes}), nil
}

func (h handler) RecordReviewRun(ctx context.Context, request *connect.Request[reviewv1.RecordReviewRunRequest]) (*connect.Response[reviewv1.RecordReviewRunResponse], error) {
	if strings.TrimSpace(request.Msg.ExpectedBody) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("exact expected body is required for a revision-bound review"))
	}
	verdicts := make([]internalreview.Verdict, 0, len(request.Msg.Verdicts))
	for _, verdict := range request.Msg.Verdicts {
		if verdict == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("review verdict cannot be empty"))
		}
		verdicts = append(verdicts, internalreview.Verdict{Mode: verdict.Mode, Passed: verdict.Passed, Evidence: verdict.Evidence, Finding: verdict.Finding})
	}
	run, err := h.service.RecordExact(ctx, request.Msg.DraftId, request.Msg.ExpectedRevisionId, request.Msg.ExpectedBody, verdicts)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&reviewv1.RecordReviewRunResponse{ReviewRun: reviewRunMessage(run)}), nil
}

func (h handler) CreateEditorialPackage(ctx context.Context, request *connect.Request[reviewv1.CreateEditorialPackageRequest]) (*connect.Response[reviewv1.CreateEditorialPackageResponse], error) {
	item, err := h.service.CreateEditorialPackage(ctx, request.Msg.DraftId, request.Msg.ExpectedRevisionId, request.Msg.ExpectedBody, request.Msg.IdempotencyKey)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&reviewv1.CreateEditorialPackageResponse{EditorialPackage: editorialPackageMessage(item)}), nil
}

func (h handler) GetEditorialPackage(ctx context.Context, request *connect.Request[reviewv1.GetEditorialPackageRequest]) (*connect.Response[reviewv1.GetEditorialPackageResponse], error) {
	item, err := h.service.GetEditorialPackage(ctx, request.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&reviewv1.GetEditorialPackageResponse{EditorialPackage: editorialPackageMessage(item)}), nil
}

func reviewRunMessage(run internalreview.Run) *reviewv1.ReviewRun {
	return &reviewv1.ReviewRun{Id: run.ID, DraftId: run.DraftID, Outcome: run.Outcome, SourceRevisionId: run.SourceRevisionID}
}

func editorialPackageMessage(item internalreview.EditorialPackage) *reviewv1.EditorialPackage {
	return &reviewv1.EditorialPackage{Id: item.ID, DraftId: item.DraftID, SourceRevisionId: item.SourceRevisionID, Scope: item.Scope, Body: item.Body, BodySha256: item.BodySHA256, PackageSha256: item.PackageSHA256, CreatedAt: item.CreatedAt.Format(time.RFC3339Nano), ManifestJson: item.ManifestJSON}
}

func Module(db *database.RoutedDB) module.Module {
	path, h := reviewconnect.NewReviewServiceHandler(handler{service: internalreview.NewService(db, artifactRevisionSource{repo: internalartifacts.NewSQLiteRepository(db)})})
	return module.Module{Name: "review", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internalreview.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "review_list", Path: reviewconnect.ReviewServiceListReviewRunsProcedure, Method: "POST", Summary: "List review runs", Category: "review"},
	{ID: "review_modes", Path: reviewconnect.ReviewServiceListRequiredReviewModesProcedure, Method: "POST", Summary: "Read the required review modes for a native post type", Category: "review"},
	{ID: "review_record", Path: reviewconnect.ReviewServiceRecordReviewRunProcedure, Method: "POST", Summary: "Record exact-revision-bound craft and policy verdicts", Category: "review"},
	{ID: "review_package_create", Path: reviewconnect.ReviewServiceCreateEditorialPackageProcedure, Method: "POST", Summary: "Create immutable destination-unbound wording snapshot with revision guard", Category: "review"},
	{ID: "review_package_get", Path: reviewconnect.ReviewServiceGetEditorialPackageProcedure, Method: "POST", Summary: "Read and digest-check an immutable editorial wording package", Category: "review"},
}
