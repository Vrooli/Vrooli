package routing_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/routing"
)

type successfulMediaExecutor struct{}

func (successfulMediaExecutor) Execute(_ context.Context, _ *routingv1.SubmitMediaRequest) (*routing.MediaExecutionResult, error) {
	return &routing.MediaExecutionResult{
		RouteEvidence: &routingv1.RouteEvidence{EventId: "route-media-1", Status: "succeeded", SelectedProvider: "resource-openrouter"},
		Outputs:       []*routingv1.MediaOutput{{Reference: "blob://output-1", MediaType: "image/png", Bytes: 42, Checksum: "sha256:test"}},
		ActualCostUSD: 0.015,
		ResolvedModel: "resource-resolved-model",
		Seed:          "1234",
		Warnings:      []string{"deterministic fixture"},
	}, nil
}

func (successfulMediaExecutor) Cancel(context.Context, string) error {
	return errors.New("not running")
}

func TestMediaSubmissionPersistsReceiptIsIdempotentAndFailsTruthfullyWithoutExecutor(t *testing.T) { // [REQ:AIGW-MEDIA-EXECUTION]
	db := newSchemaDB(t)
	svc := routing.NewMediaService(db, nil)
	req := mediaRequest("media-idempotency-1")

	first, err := svc.Submit(context.Background(), req)
	require.NoError(t, err)
	require.NotEmpty(t, first.GetExecutionId())
	second, err := svc.Submit(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, first.GetExecutionId(), second.GetExecutionId())

	require.Eventually(t, func() bool {
		got, getErr := svc.Get(context.Background(), first.GetExecutionId())
		return getErr == nil && got.GetStatus() == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED
	}, time.Second, 10*time.Millisecond)
	got, err := svc.Get(context.Background(), first.GetExecutionId())
	require.NoError(t, err)
	require.Equal(t, "executor_unavailable", got.GetErrorCode())
	require.Empty(t, got.GetOutputs())
}

func TestMediaSubmissionRejectsNonMediaRequestKinds(t *testing.T) { // [REQ:AIGW-MEDIA-EXECUTION]
	db := newSchemaDB(t)
	svc := routing.NewMediaService(db, nil)
	req := mediaRequest("media-invalid-kind")
	req.Request.Kind = sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION
	_, err := svc.Submit(context.Background(), req)
	require.ErrorContains(t, err, "image_generation or video_generation")
}

func TestMediaSubmissionPersistsExecutorOutputAndAccounting(t *testing.T) { // [REQ:AIGW-MEDIA-EXECUTION]
	db := newSchemaDB(t)
	svc := routing.NewMediaService(db, successfulMediaExecutor{})
	exec, err := svc.Submit(context.Background(), mediaRequest("media-success-1"))
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		got, getErr := svc.Get(context.Background(), exec.GetExecutionId())
		return getErr == nil && (got.GetStatus() == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_SUCCEEDED || got.GetStatus() == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED)
	}, time.Second, 10*time.Millisecond)
	got, err := svc.Get(context.Background(), exec.GetExecutionId())
	require.NoError(t, err)
	require.Equal(t, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_SUCCEEDED, got.GetStatus(), "error=%s: %s", got.GetErrorCode(), got.GetErrorMessage())
	require.Len(t, got.GetOutputs(), 1)
	require.Equal(t, "blob://output-1", got.GetOutputs()[0].GetReference())
	require.InDelta(t, 0.015, got.GetActualCostUsd(), 1e-9)
	require.Equal(t, "resource-resolved-model", got.GetResolvedModel())
	require.Equal(t, "1234", got.GetSeed())
}

func TestMediaRecoveryResumesDurableRunningReceipt(t *testing.T) { // [REQ:AIGW-MEDIA-EXECUTION]
	db := newSchemaDB(t)
	initial := routing.NewMediaService(db, nil)
	exec, err := initial.Submit(context.Background(), mediaRequest("media-recover-1"))
	require.NoError(t, err)
	// Simulate an interrupted worker: the receipt and request are durable, but
	// no terminal result has been written when the server restarts.
	require.Eventually(t, func() bool {
		got, getErr := initial.Get(context.Background(), exec.GetExecutionId())
		return getErr == nil && got.GetStatus() == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED
	}, time.Second, 10*time.Millisecond)
	_, err = db.Exec(`UPDATE media_executions SET status = ?, started_at = ? WHERE execution_id = ?`, int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING), time.Now().UTC().Format(time.RFC3339Nano), exec.GetExecutionId())
	require.NoError(t, err)

	recovered := routing.NewMediaService(db, successfulMediaExecutor{})
	recovered.Recover(context.Background())
	require.Eventually(t, func() bool {
		got, getErr := recovered.Get(context.Background(), exec.GetExecutionId())
		return getErr == nil && got.GetStatus() == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_SUCCEEDED
	}, time.Second, 10*time.Millisecond)
}

func mediaRequest(key string) *routingv1.SubmitMediaRequest {
	return &routingv1.SubmitMediaRequest{
		Request: &sharedv1.GatewayRequest{
			Kind:         sharedv1.RequestKind_REQUEST_KIND_IMAGE_GENERATION,
			Role:         "media.generate",
			Profile:      sharedv1.Profile_PROFILE_LOCAL_FIRST,
			PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
			Scenario:     "fixture",
			Operation:    "generate",
		},
		Prompt:          "a red square",
		OutputCount:     1,
		IdempotencyKey:  key,
		OutputReference: filepath.Join(os.TempDir(), "ai-gateway-media-test-output.png"),
	}
}
