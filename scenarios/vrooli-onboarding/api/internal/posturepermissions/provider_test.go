package posturepermissions

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/operatorsession"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

type enrollmentReader struct{ value operatorsession.Enrollment }

func (r *enrollmentReader) ReadEnrollment() (operatorsession.Enrollment, error) { return r.value, nil }

func TestUpdateControlUsesTypedOwnerCASAndRefusesUnsupportedValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), operatorstate.StateFile)
	store := operatorstate.New(operatorstate.Config{RepoRoot: filepath.Dir(path), StatePath: func(context.Context) (string, error) { return path, nil }})
	owner := operatorstateapi.New(store, nil)
	registry, err := permissions.NewRegistry(NewUpdateControl(owner), NewSessionMode(&enrollmentReader{value: testEnrollment()}), NewTrustPosture(owner))
	if err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-a", Verified: true, Scopes: []string{"vrooli-onboarding:write"}})
	readReq := permissions.ReadRequest{ActorID: "operator-a", Owner: "vrooli-onboarding"}
	initial, err := registry.Read(ctx, updateControlID, readReq)
	if err != nil {
		t.Fatal(err)
	}
	if initial.Revision == "" {
		t.Fatal("update-control revision is required")
	}
	trust, err := registry.Read(ctx, trustPostureID, readReq)
	if err != nil || string(trust.Value) != `{"value":"personal","supported_values":["personal","shared","hosted"]}` {
		t.Fatalf("trust-posture owner projection=%s err=%v", trust.Value, err)
	}
	if _, err := registry.Write(ctx, trustPostureID, permissions.WriteRequest{ActorID: "operator-a", Owner: "vrooli-onboarding", ExpectedRevision: trust.Revision, IdempotencyKey: "trust-posture-write", Value: []byte(`{"value":"shared"}`)}); err == nil {
		t.Fatal("trust posture write must remain unsupported through onboarding")
	}
	descriptors, err := registry.Inventory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	foundTrust := false
	for _, descriptor := range descriptors {
		if descriptor.ID == trustPostureID {
			foundTrust = true
			if descriptor.CanWrite {
				t.Fatal("trust posture descriptor must be read-only")
			}
		}
	}
	if !foundTrust {
		t.Fatal("typed trust-posture descriptor is missing")
	}
	if _, err := registry.Read(ctx, trustPostureID, permissions.ReadRequest{ActorID: "operator-a", Owner: "some-other-owner"}); err == nil {
		t.Fatal("cross-owner trust posture read was accepted")
	}

	value, _ := json.Marshal(map[string]string{"value": "guard"})
	request := permissions.WriteRequest{ActorID: "operator-a", Owner: "vrooli-onboarding", ExpectedRevision: initial.Revision, IdempotencyKey: "posture-1", Value: value}
	receipt, err := registry.Write(ctx, updateControlID, request)
	if err != nil || receipt.ResultRevision == initial.Revision {
		t.Fatalf("typed update-control write receipt=%+v err=%v", receipt, err)
	}
	current, err := owner.Get(ctx)
	if err != nil || current.UpdateControl != "guard" {
		t.Fatalf("stored update_control=%q err=%v", current.UpdateControl, err)
	}
	if _, exists := current.RawFields["update_control"]; exists {
		t.Fatal("typed update_control remained duplicated as opaque raw state")
	}

	request.IdempotencyKey = "posture-own"
	request.ExpectedRevision = receipt.ResultRevision
	request.Value = []byte(`{"value":"own"}`)
	if _, err := registry.Write(ctx, updateControlID, request); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("reserved own mode write error=%v, want FailedPrecondition", err)
	}
	unchanged, err := owner.Get(ctx)
	if err != nil || unchanged.UpdateControl != "guard" {
		t.Fatalf("unsupported value changed owner state: %+v err=%v", unchanged, err)
	}

	request.IdempotencyKey = "posture-stale"
	request.ExpectedRevision = initial.Revision
	request.Value = []byte(`{"value":"observe"}`)
	if _, err := registry.Write(ctx, updateControlID, request); !errors.Is(err, operatorstate.ErrRevisionConflict) {
		t.Fatalf("stale update-control write error=%v, want revision conflict", err)
	}
	if _, err := registry.Read(ctx, updateControlID, permissions.ReadRequest{ActorID: "operator-a", Owner: "another-owner"}); err == nil {
		t.Fatal("cross-owner read was accepted")
	}
}

func TestSessionModeIsEnrollmentOwnedReadOnlyMetadata(t *testing.T) {
	reader := &enrollmentReader{value: testEnrollment()}
	registry, err := permissions.NewRegistry(NewSessionMode(reader))
	if err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-a", Verified: true})
	readReq := permissions.ReadRequest{ActorID: "operator-a", Owner: "api-core/operatorsession"}
	first, err := registry.Read(ctx, sessionModeID, readReq)
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision == "" || string(first.Value) != `{"value":"personal","supported_values":["personal","shared","hosted"]}` {
		t.Fatalf("enrollment mode projection=%s revision=%q", first.Value, first.Revision)
	}
	reader.value.Mode = operatorsession.ModeShared
	second, err := registry.Read(ctx, sessionModeID, readReq)
	if err != nil || second.Revision == first.Revision {
		t.Fatalf("enrollment revision did not bind mode change: first=%q second=%q err=%v", first.Revision, second.Revision, err)
	}
	if _, err := registry.Write(ctx, sessionModeID, permissions.WriteRequest{ActorID: "operator-a", Owner: "api-core/operatorsession", ExpectedRevision: first.Revision, IdempotencyKey: "unsupported", Value: []byte(`{"value":"hosted"}`)}); err == nil {
		t.Fatal("session_mode onboarding write was accepted")
	}
	if _, err := registry.Read(ctx, sessionModeID, permissions.ReadRequest{ActorID: "different-operator", Owner: "api-core/operatorsession"}); err == nil {
		t.Fatal("cross-owner enrollment read was accepted")
	}
}

func TestEnrollmentMetadataReaderNeverNeedsPrivateKey(t *testing.T) {
	dir := t.TempDir()
	store, err := operatorsession.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	enrollment := testEnrollment()
	data, _ := json.Marshal(map[string]any{"enrollment": enrollment})
	if err := os.WriteFile(filepath.Join(dir, "enrollment.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := store.ReadEnrollment()
	if err != nil || !reflect.DeepEqual(got, enrollment) {
		t.Fatalf("metadata read=%+v err=%v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "private.key")); !os.IsNotExist(err) {
		t.Fatalf("test expected no private key file, stat err=%v", err)
	}
}

func testEnrollment() operatorsession.Enrollment {
	return operatorsession.Enrollment{OperatorID: "operator-a", IdentityProvider: operatorsession.IdentityProviderAuthenticator, Mode: operatorsession.ModePersonal, Reference: "opaque-enrollment-ref", EnrolledAt: time.Unix(1_700_000_000, 0).UTC()}
}
