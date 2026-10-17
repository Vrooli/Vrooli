package credentialsvc

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/credentials"
	"scenario-to-cloud/domain"

	"google.golang.org/protobuf/encoding/protojson"
)

// [REQ:STC-P0-032] The proto projection carries every lifecycle field
// (receipts with nested detail values, handoff, break-glass window,
// unreached) and a lifecycle refusal carries its HTTP status and operation.
func TestOperationProtoCarriesLifecycleStanding(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	resume := now.Add(time.Hour)
	rotation := &domain.CredentialRotation{
		ID: "op-1", DeploymentID: "dep-1", BindingID: "cb_1", Kind: domain.CredentialOperationBreakGlass, FromVersion: 1, ToVersion: 2, State: domain.RotationPendingOperatorInput,
		Consumers:            []domain.CredentialConsumerProgress{{Consumer: "scenario:app", State: domain.ConsumerUnreachable, Version: 2, Reason: "no acknowledgement recorded", UpdatedAt: now}},
		Unreached:            []string{"scenario:app"},
		PendingOperatorInput: &domain.OperatorHandoff{Reference: "op-1/revoke-predecessor", Provider: "mailer", Instruction: "revoke in console", ResumeWith: "POST …/resume", RequestedAt: now},
		ResumeAfter:          &resume,
		BreakGlass:           &domain.BreakGlassWindow{Scope: "incident", Operator: "operator:alice", IssuedAt: now, ExpiresAt: resume, PredecessorRef: "cb_1@1", ConfirmationRef: "confirmed"},
		Receipts:             []domain.CredentialReceipt{{Step: "verify", State: "consumers_updated", Outcome: "incomplete", At: now, Details: map[string]any{"version": int64(2), "unreached": []string{"scenario:app"}, "created_at": now, "nested": map[string]any{"ok": true}}, Limitations: []string{"incomplete"}}},
		Error:                &domain.CredentialOperationError{Code: "x", Message: "y"},
		CreatedAt:            now, UpdatedAt: now,
	}
	msg := OperationProto(rotation)
	raw, err := protojson.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{`"pendingOperatorInput"`, `"breakGlass"`, `"unreached":["scenario:app"]`, `"resumeAfter"`, `"limitations":["incomplete"]`, `"nested":{"ok":true}`, `"unreached":["scenario:app"]`, `"2026-09-09T12:00:00Z"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("proto JSON lacks %s: %s", want, text)
		}
	}
	if msg.GetReceipts()[0].GetDetails().GetFields()["version"].GetNumberValue() != 2 {
		t.Fatalf("receipt version detail = %v", msg.GetReceipts()[0].GetDetails())
	}

	typed := OperationError(apierrors.New(credentials.CodeRevocationIncomplete, "unreached"), rotation)
	if typed.HTTPStatus != 202 {
		t.Fatalf("status = %d", typed.HTTPStatus)
	}
	if _, ok := typed.Details["operation"]; !ok {
		t.Fatal("operation missing from refusal details")
	}
	untyped := OperationError(errors.New("boom"), nil)
	if untyped.Code != apierrors.CodeInternal {
		t.Fatalf("untyped error code = %s", untyped.Code)
	}
	if OperationError(nil, rotation) != nil {
		t.Fatal("nil error mapped to a refusal")
	}
	expires := now.Add(time.Hour)
	view := BindingViewProto(credentials.BindingView{Binding: domain.CredentialBinding{ID: "cb_1", Descriptor: domain.CredentialDescriptor{LogicalID: "fixture/store", Field: "Pass_Word"}, PreviousVersion: &domain.CredentialVersion{Number: 1}, Version: domain.CredentialVersion{ExpiresAt: &expires}, RecoveryKeyRef: "fixture/backup:recovery-key"}, LifecycleState: "renewal_due", LifecycleDetail: "renew soon", NextAction: "rotate", Acks: []domain.CredentialAck{{BindingID: "cb_1", Consumer: "scenario:app", Version: 1, VerifiedAt: now}}})
	if view.GetBinding().GetDescriptor_().GetField() != "Pass_Word" || view.GetBinding().GetPreviousVersion().GetNumber() != 1 || view.GetBinding().GetRecoveryKeyRef() == "" || len(view.GetAcks()) != 1 {
		t.Fatalf("binding view = %v", view)
	}
	if view.GetLifecycleState() != "renewal_due" || view.GetNextAction() != "rotate" || view.GetBinding().GetVersion().GetExpiresAt() == nil {
		t.Fatalf("lifecycle metadata = %v", view)
	}
	if raw, _ := json.Marshal(rotation); strings.Contains(string(raw), `"value"`) {
		t.Fatal("rotation JSON has a value field")
	}
}
