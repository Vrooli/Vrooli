package evidence

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"swarm-manager/internal/identity"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	"google.golang.org/protobuf/types/known/structpb"
)

type stubReconciler struct{ runID string }

func (s *stubReconciler) Reconcile(_ context.Context, runID string) error {
	s.runID = runID
	return nil
}

func TestConnectServiceRejectsAgentOperatorAttestation(t *testing.T) {
	service, _ := newEvidenceService(t, &stubOwnerIndex{}, &stubOwnerIndex{})
	connectService := NewConnectService(service, &stubReconciler{})
	ctx := identity.NewContext(context.Background(), identity.Provenance{Type: identity.TypeAgent, RunID: "run-42"})
	_, err := connectService.RecordOperatorVerification(ctx, connect.NewRequest(&apipb.EvidenceOperatorVerificationRequest{OwnerKind: string(OwnerAgentSession), OwnerId: "session-1", EventId: "operator-1", RunId: "run-42", SubjectKind: "plan", SubjectId: "plan-42", Action: "approved", Actor: "agent", Reason: "self report"}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("agent operator verification code = %v, want %v", connect.CodeOf(err), connect.CodePermissionDenied)
	}
}

func TestEvidenceConnectErrorPreservesImmutableSourceConflict(t *testing.T) {
	if got := connect.CodeOf(evidenceConnectError(ErrSourceConflict)); got != connect.CodeAlreadyExists {
		t.Fatalf("source conflict code = %v, want %v", got, connect.CodeAlreadyExists)
	}
}

func TestConnectServiceQueriesReconcilesAndRecordsOperatorEvidence(t *testing.T) {
	service, _ := newEvidenceService(t, &stubOwnerIndex{}, &stubOwnerIndex{})
	owner := Owner{Kind: OwnerAgentSession, ID: "session-1"}
	if _, err := service.IngestForOwner(context.Background(), owner, verifiedObservation()); err != nil {
		t.Fatal(err)
	}
	reconciler := &stubReconciler{}
	router := mux.NewRouter()
	RegisterConnectService(router, service, reconciler)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := apiconnect.NewEvidenceServiceClient(server.Client(), server.URL)

	byRun, err := client.ListRun(context.Background(), connect.NewRequest(&apipb.EvidenceListRunRequest{RunId: "run-42"}))
	if err != nil || len(byRun.Msg.GetRecords()) != 1 || byRun.Msg.GetRecords()[0].GetSubjectId() != "plan-42" {
		t.Fatalf("ListRun = %+v, %v", byRun, err)
	}
	if _, err := client.Reconcile(context.Background(), connect.NewRequest(&apipb.EvidenceReconcileRequest{RunId: "run-42"})); err != nil || reconciler.runID != "run-42" {
		t.Fatalf("Reconcile = %v, run=%q", err, reconciler.runID)
	}
	metadata, err := structpb.NewStruct(map[string]any{"plan_id": "plan-42"})
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.RecordOperatorVerification(context.Background(), connect.NewRequest(&apipb.EvidenceOperatorVerificationRequest{OwnerKind: string(owner.Kind), OwnerId: owner.ID, EventId: "operator-1", RunId: "run-42", SubjectKind: "plan", SubjectId: "plan-42", Action: "approved", Actor: "matt", Reason: "audited", Metadata: metadata}))
	if err != nil || created.Msg.GetConfidence() != string(ConfidenceOperator) || created.Msg.GetMetadata().GetFields()["operator_actor"].GetStringValue() != "matt" {
		t.Fatalf("RecordOperatorVerification = %+v, %v", created, err)
	}
}
