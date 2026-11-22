// Package intake exposes the restartable article-admission service.
package intake

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	intakev1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/intake"
	intakeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/intake/intake_v1connect"

	"content-desk/internal/artifacts"
	"content-desk/internal/intake"
	"content-desk/internal/module"
)

type handler struct{ service intake.Service }

var _ intakeconnect.ArticleIntakeServiceHandler = handler{}

func sourceFrom(in *intakev1.Source) intake.Source {
	if in == nil {
		return intake.Source{}
	}
	return intake.Source{ArtifactID: in.ArtifactId, RevisionID: in.RevisionId, Producer: in.Producer, Intent: in.ImportIntent}
}

func itemsFrom(raw []string) ([]intake.Item, error) {
	items := make([]intake.Item, 0, len(raw))
	for i, text := range raw {
		var item intake.Item
		if err := json.Unmarshal([]byte(text), &item); err != nil {
			return nil, fmt.Errorf("raw_items_json[%d]: %w", i, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func resultResponse[T any](value T) (string, error) {
	b, err := json.Marshal(value)
	return string(b), err
}

func (h handler) Preview(ctx context.Context, req *connect.Request[intakev1.PreviewRequest]) (*connect.Response[intakev1.PreviewResponse], error) {
	items, err := itemsFrom(req.Msg.RawItemsJson)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	value, err := h.service.Preview(ctx, intake.Request{Source: sourceFrom(req.Msg.Source), Items: items})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	body, err := resultResponse(value)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&intakev1.PreviewResponse{ResultJson: body}), nil
}

func (h handler) Admit(ctx context.Context, req *connect.Request[intakev1.AdmitRequest]) (*connect.Response[intakev1.AdmitResponse], error) {
	items, err := itemsFrom(req.Msg.RawItemsJson)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	value, err := h.service.Admit(ctx, intake.Request{Source: sourceFrom(req.Msg.Source), IdempotencyKey: req.Msg.IdempotencyKey, Items: items})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	body, err := resultResponse(value)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&intakev1.AdmitResponse{ResultJson: body}), nil
}

func (h handler) List(ctx context.Context, req *connect.Request[intakev1.ListRequest]) (*connect.Response[intakev1.ListResponse], error) {
	value, err := h.service.List(ctx, req.Msg.BatchId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	body, err := resultResponse(value)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&intakev1.ListResponse{ResultJson: body}), nil
}

func (h handler) Reconcile(ctx context.Context, req *connect.Request[intakev1.ReconcileRequest]) (*connect.Response[intakev1.ReconcileResponse], error) {
	items, err := itemsFrom(req.Msg.RawItemsJson)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	value, err := h.service.Reconcile(ctx, req.Msg.BatchId, intake.Request{Source: sourceFrom(req.Msg.Source), Items: items})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	body, err := resultResponse(value)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&intakev1.ReconcileResponse{ResultJson: body}), nil
}

func (h handler) Promote(ctx context.Context, req *connect.Request[intakev1.PromoteRequest]) (*connect.Response[intakev1.PromoteResponse], error) {
	value, err := h.service.Promote(ctx, intake.PromoteRequest{
		BatchID: req.Msg.BatchId, ExternalID: req.Msg.ExternalId,
		ExpectedReceiptID: req.Msg.ExpectedReceiptId, VariantIDs: req.Msg.VariantIds,
		Channel: req.Msg.Channel, Format: req.Msg.Format, PostTypeID: req.Msg.PostTypeId,
		CampaignID: req.Msg.CampaignId, ProjectID: req.Msg.ProjectId,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	body, err := resultResponse(value)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&intakev1.PromoteResponse{ResultJson: body}), nil
}

func Module(db *database.RoutedDB) module.Module {
	path, h := intakeconnect.NewArticleIntakeServiceHandler(handler{service: intake.NewServiceWithDrafts(db, artifacts.NewSQLiteRepository(db))})
	return module.Module{Name: "intake", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return intake.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "intake_preview", Path: intakeconnect.ArticleIntakeServicePreviewProcedure, Method: "POST", Summary: "Assess structured article admission without writing", Category: "intake"},
	{ID: "intake_admit", Path: intakeconnect.ArticleIntakeServiceAdmitProcedure, Method: "POST", Summary: "Admit exact structured source items with immutable receipts", Category: "intake"},
	{ID: "intake_list", Path: intakeconnect.ArticleIntakeServiceListProcedure, Method: "POST", Summary: "Read admitted article intake and receipts", Category: "intake"},
	{ID: "intake_reconcile", Path: intakeconnect.ArticleIntakeServiceReconcileProcedure, Method: "POST", Summary: "Reconcile unresolved source items only", Category: "intake"},
	{ID: "intake_promote", Path: intakeconnect.ArticleIntakeServicePromoteProcedure, Method: "POST", Summary: "Create requested drafts from an exact admitted article receipt and named variants", Category: "intake"},
}
