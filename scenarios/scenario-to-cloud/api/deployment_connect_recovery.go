package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	deploymentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments"
	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/deploymentsvc"
	"scenario-to-cloud/domain"
)

// StartRecovery delegates to the canonical recovery admission path. The
// generated method is human-only at the authorization boundary; this adapter
// does not grant the Deployment Manager service principal recovery authority.
func (s *Server) StartRecovery(ctx context.Context, request *deploymentsv1.StartDeploymentRecoveryRequest) (*domain.RecoveryOperation, error) {
	if request == nil || request.GetAction() != "rollback" || request.GetDeploymentId() == "" || request.GetExpectedCurrentBundleSha256() == "" || request.GetLastGoodReleaseDigest() == "" || request.GetReviewRef() == "" || request.GetPreviewRef() == "" || request.GetDataCompatibility() != "compatible" || request.GetIdempotencyKey() == "" {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "rollback requires deployment, current bundle, last-good release, review, preview, compatible data state, and stable idempotency key")
	}
	if denied := s.authz.AuthorizeTarget(ctx, request.GetDeploymentId()); denied != nil {
		return nil, denied
	}
	body, err := json.Marshal(cloudRecoveryHTTPRequest{
		Action: "rollback", ExpectedBundleSHA: request.GetExpectedCurrentBundleSha256(),
		DataCompatibility: request.GetDataCompatibility(), IdempotencyKey: request.GetIdempotencyKey(),
		ReviewRef: request.GetReviewRef(), ReleaseDigest: request.GetLastGoodReleaseDigest(),
		PreviewRef: request.GetPreviewRef(),
	})
	if err != nil {
		return nil, apierrors.Internal("Failed to encode recovery request", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/"+request.GetDeploymentId()+"/recovery", bytes.NewReader(body)).WithContext(ctx)
	r = mux.SetURLVars(r, map[string]string{"id": request.GetDeploymentId()})
	w := httptest.NewRecorder()
	s.handleRecoverDeployment(w, r)
	if w.Code < 200 || w.Code >= 300 {
		var envelope apierrors.Envelope
		if json.Unmarshal(w.Body.Bytes(), &envelope) == nil && envelope.Error.Code != "" {
			return nil, apierrors.New(envelope.Error.Code, envelope.Error.Message)
		}
		return nil, apierrors.New(apierrors.CodeInternal, fmt.Sprintf("Recovery admission failed with status %d", w.Code))
	}
	var response struct {
		Receipt domain.CloudRecoveryReceipt `json:"receipt"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || response.Receipt.OperationID == "" {
		return nil, apierrors.New(apierrors.CodeReceiptInvalid, "Recovery admission returned no durable operation identity")
	}
	return s.GetRecovery(ctx, request.GetDeploymentId(), response.Receipt.OperationID)
}

// PreviewRecovery reuses the canonical REST recovery preview implementation.
// The typed Connect route is target-bound and human-only; this method forwards
// the authenticated request context and forces dry_run before calling it.
func (s *Server) PreviewRecovery(ctx context.Context, request *deploymentsv1.PreviewDeploymentRecoveryRequest) (*domain.CloudRecoveryPreview, error) {
	if request == nil || request.GetAction() != "rollback" || request.GetDeploymentId() == "" || request.GetExpectedCurrentBundleSha256() == "" || request.GetExpectedCurrentReleaseDigest() == "" || request.GetLastGoodReleaseDigest() == "" || request.GetReviewRef() == "" || request.GetDataCompatibility() != "compatible" || request.GetIdempotencyKey() == "" {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "rollback preview requires deployment, current bundle and release, last-good release, review, compatible data declaration, and stable idempotency key")
	}
	if denied := s.authz.AuthorizeTarget(ctx, request.GetDeploymentId()); denied != nil {
		return nil, denied
	}
	body, err := json.Marshal(cloudRecoveryHTTPRequest{
		Action: "rollback", ExpectedBundleSHA: request.GetExpectedCurrentBundleSha256(),
		ExpectedCurrentReleaseDigest: request.GetExpectedCurrentReleaseDigest(),
		DataCompatibility:            request.GetDataCompatibility(), IdempotencyKey: request.GetIdempotencyKey(),
		ReviewRef: request.GetReviewRef(), ReleaseDigest: request.GetLastGoodReleaseDigest(), DryRun: true,
	})
	if err != nil {
		return nil, apierrors.Internal("Failed to encode recovery preview request", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/"+request.GetDeploymentId()+"/recovery", bytes.NewReader(body)).WithContext(ctx)
	r = mux.SetURLVars(r, map[string]string{"id": request.GetDeploymentId()})
	w := httptest.NewRecorder()
	s.handleRecoverDeployment(w, r)
	if w.Code < 200 || w.Code >= 300 {
		var envelope apierrors.Envelope
		if json.Unmarshal(w.Body.Bytes(), &envelope) == nil && envelope.Error.Code != "" {
			return nil, apierrors.New(envelope.Error.Code, envelope.Error.Message)
		}
		return nil, apierrors.New(apierrors.CodeInternal, fmt.Sprintf("Recovery preview failed with status %d", w.Code))
	}
	var response struct {
		Receipt                   domain.CloudRecoveryReceipt `json:"receipt"`
		CurrentBundleSHA256       string                      `json:"current_bundle_sha256"`
		CurrentReleaseDigest      string                      `json:"current_release_digest"`
		DataCompatibility         string                      `json:"data_compatibility"`
		DataCompatibilityDeclared bool                        `json:"data_compatibility_declared"`
		IdempotencyKey            string                      `json:"idempotency_key"`
		ImpactSummary             string                      `json:"impact_summary"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		return nil, apierrors.New(apierrors.CodeReceiptInvalid, "Recovery preview returned invalid JSON")
	}
	receipt := response.Receipt
	if receipt.DeploymentID != request.GetDeploymentId() || receipt.Action != "rollback" || receipt.Outcome != "preview" || !receipt.DryRun || receipt.Health != "unknown" ||
		normalizeRecoverySHA(response.CurrentBundleSHA256) != normalizeRecoverySHA(request.GetExpectedCurrentBundleSha256()) ||
		normalizeRecoverySHA(response.CurrentReleaseDigest) != normalizeRecoverySHA(request.GetExpectedCurrentReleaseDigest()) ||
		normalizeRecoverySHA(receipt.ReleaseDigest) != normalizeRecoverySHA(request.GetLastGoodReleaseDigest()) || receipt.ReviewKey != request.GetReviewRef() ||
		response.DataCompatibility != "compatible" || !response.DataCompatibilityDeclared || response.IdempotencyKey != request.GetIdempotencyKey() ||
		!domain.IsSHA256Hex(normalizeRecoverySHA(receipt.BundleSHA256)) || receipt.PreviewRef == "" || response.ImpactSummary == "" ||
		(receipt.RouteKind != domain.RecoveryRouteBundlePipeline && receipt.RouteKind != domain.RecoveryRouteCloudTargetVerb) {
		return nil, apierrors.New(apierrors.CodeReceiptInvalid, "Recovery owner preview is incomplete or does not match the requested deployment identity")
	}
	return &domain.CloudRecoveryPreview{
		DeploymentID: receipt.DeploymentID, Action: receipt.Action,
		CurrentBundleSHA256: response.CurrentBundleSHA256, CurrentReleaseDigest: response.CurrentReleaseDigest,
		LastGoodReleaseDigest: receipt.ReleaseDigest, RepairBundleSHA256: receipt.BundleSHA256,
		ReviewRef: receipt.ReviewKey, DataCompatibility: response.DataCompatibility,
		RouteKind: receipt.RouteKind, ImpactSummary: response.ImpactSummary, PreviewRef: receipt.PreviewRef,
		IdempotencyKey: response.IdempotencyKey, DataCompatibilityDeclared: response.DataCompatibilityDeclared,
	}, nil
}

func (s *Server) GetRecovery(ctx context.Context, deploymentID, operationID string) (*domain.RecoveryOperation, error) {
	if denied := s.authz.AuthorizeTarget(ctx, deploymentID); denied != nil {
		return nil, denied
	}
	op, err := s.repo.GetCloudRecoveryOperation(ctx, operationID)
	if err != nil {
		return nil, apierrors.Internal("Failed to read recovery operation", err)
	}
	if op == nil || op.DeploymentID != deploymentID {
		return nil, apierrors.New("recovery_not_found", "Recovery operation was not found for this deployment")
	}
	return op, nil
}

func (s *Server) FindRecovery(ctx context.Context, deploymentID, idempotencyKey string) (*domain.RecoveryOperation, error) {
	if denied := s.authz.AuthorizeTarget(ctx, deploymentID); denied != nil {
		return nil, denied
	}
	op, err := s.repo.GetCloudRecoveryOperationByKey(ctx, deploymentID, idempotencyKey)
	if err != nil {
		return nil, apierrors.Internal("Failed to reconcile recovery operation", err)
	}
	if op == nil {
		return nil, apierrors.New("recovery_not_found", "Recovery operation was not found for this idempotency key")
	}
	return op, nil
}

var _ deploymentsvc.RecoveryOwner = (*Server)(nil)
