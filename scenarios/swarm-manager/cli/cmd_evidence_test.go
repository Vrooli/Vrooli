package main

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	clitest "github.com/vrooli/cli-core/cliapptest"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

type stubEvidenceHandler struct {
	apiconnect.UnimplementedEvidenceServiceHandler
	listRun    func(*apipb.EvidenceListRunRequest) (*apipb.EvidenceListResponse, error)
	listEntity func(*apipb.EvidenceListEntityRequest) (*apipb.EvidenceListResponse, error)
	reconcile  func(*apipb.EvidenceReconcileRequest) (*apipb.EvidenceReconcileResponse, error)
	verify     func(*apipb.EvidenceOperatorVerificationRequest) (*apipb.EvidenceRecord, error)
}

func (s *stubEvidenceHandler) ListRun(_ context.Context, request *connect.Request[apipb.EvidenceListRunRequest]) (*connect.Response[apipb.EvidenceListResponse], error) {
	result, err := s.listRun(request.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *stubEvidenceHandler) ListEntity(_ context.Context, request *connect.Request[apipb.EvidenceListEntityRequest]) (*connect.Response[apipb.EvidenceListResponse], error) {
	result, err := s.listEntity(request.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *stubEvidenceHandler) Reconcile(_ context.Context, request *connect.Request[apipb.EvidenceReconcileRequest]) (*connect.Response[apipb.EvidenceReconcileResponse], error) {
	result, err := s.reconcile(request.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *stubEvidenceHandler) RecordOperatorVerification(_ context.Context, request *connect.Request[apipb.EvidenceOperatorVerificationRequest]) (*connect.Response[apipb.EvidenceRecord], error) {
	result, err := s.verify(request.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func newEvidenceTestApp(t *testing.T, stub apiconnect.EvidenceServiceHandler) *App {
	t.Helper()
	path, handler := apiconnect.NewEvidenceServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})
	server := clitest.NewAPIServer(t, mux)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	_ = server
	return app
}

func TestEvidenceCommandsUseGeneratedContract(t *testing.T) {
	var runID, entityKind, entityID, actor, reason string
	app := newEvidenceTestApp(t, &stubEvidenceHandler{
		listRun: func(request *apipb.EvidenceListRunRequest) (*apipb.EvidenceListResponse, error) {
			runID = request.GetRunId()
			return &apipb.EvidenceListResponse{Records: []*apipb.EvidenceRecord{{RunId: runID, SubjectKind: "plan", SubjectId: "plan-1", Action: "created", Confidence: "authoritative", Verification: "verified"}}}, nil
		},
		listEntity: func(request *apipb.EvidenceListEntityRequest) (*apipb.EvidenceListResponse, error) {
			entityKind, entityID = request.GetSubjectKind(), request.GetSubjectId()
			return &apipb.EvidenceListResponse{}, nil
		},
		reconcile: func(request *apipb.EvidenceReconcileRequest) (*apipb.EvidenceReconcileResponse, error) {
			return &apipb.EvidenceReconcileResponse{RunId: request.GetRunId(), Status: "reconciled"}, nil
		},
		verify: func(request *apipb.EvidenceOperatorVerificationRequest) (*apipb.EvidenceRecord, error) {
			actor, reason = request.GetActor(), request.GetReason()
			return &apipb.EvidenceRecord{Confidence: "operator_verified"}, nil
		},
	})
	if err := app.Run([]string{"evidence", "run", "--run-id", "run-1"}); err != nil || runID != "run-1" {
		t.Fatalf("run err=%v id=%q", err, runID)
	}
	if err := app.Run([]string{"evidence", "entity", "--kind", "plan", "--id", "plan-1"}); err != nil || entityKind != "plan" || entityID != "plan-1" {
		t.Fatalf("entity err=%v kind=%q id=%q", err, entityKind, entityID)
	}
	if err := app.Run([]string{"evidence", "reconcile", "--run-id", "run-1"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"evidence", "verify", "--owner-kind", "agent_session", "--owner-id", "session-1", "--event-id", "operator-1", "--run-id", "run-1", "--subject-kind", "plan", "--subject-id", "plan-1", "--action", "approved", "--actor", "matt", "--reason", "checked"}); err != nil || actor != "matt" || reason != "checked" {
		t.Fatalf("verify err=%v actor=%q reason=%q", err, actor, reason)
	}
}

func TestEvidenceCommandsRequireIdentifiers(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"evidence", "run"}, {"evidence", "entity"}, {"evidence", "reconcile"}, {"evidence", "verify"}} {
		if err := app.Run(args); err == nil {
			t.Fatal("evidence command accepted missing required identifiers")
		}
	}
}
