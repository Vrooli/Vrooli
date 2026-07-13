package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"swarm-manager/internal/identity"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	"google.golang.org/protobuf/types/known/structpb"
)

// Reconciler refreshes all configured producer adapters for one verified run.
type Reconciler interface {
	Reconcile(context.Context, string) error
}

// ConnectService exposes the ledger through the generated EvidenceService
// contract. It is intentionally owner-neutral: callers query facts by run or
// subject while the ledger retains resolved owner linkage.
type ConnectService struct {
	service    *Service
	reconciler Reconciler
}

func NewConnectService(service *Service, reconciler Reconciler) *ConnectService {
	return &ConnectService{service: service, reconciler: reconciler}
}

func RegisterConnectService(router *mux.Router, service *Service, reconciler Reconciler) {
	path, handler := apiconnect.NewEvidenceServiceHandler(NewConnectService(service, reconciler))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

var _ apiconnect.EvidenceServiceHandler = (*ConnectService)(nil)

func (s *ConnectService) ListRun(ctx context.Context, request *connect.Request[apipb.EvidenceListRunRequest]) (*connect.Response[apipb.EvidenceListResponse], error) {
	records, err := s.service.ListByRun(ctx, request.Msg.GetRunId())
	if err != nil {
		return nil, evidenceConnectError(err)
	}
	return connect.NewResponse(&apipb.EvidenceListResponse{Records: recordsToProto(records)}), nil
}

func (s *ConnectService) ListEntity(ctx context.Context, request *connect.Request[apipb.EvidenceListEntityRequest]) (*connect.Response[apipb.EvidenceListResponse], error) {
	records, err := s.service.ListByEntity(ctx, Subject{Kind: request.Msg.GetSubjectKind(), ID: request.Msg.GetSubjectId()})
	if err != nil {
		return nil, evidenceConnectError(err)
	}
	return connect.NewResponse(&apipb.EvidenceListResponse{Records: recordsToProto(records)}), nil
}

func (s *ConnectService) Reconcile(ctx context.Context, request *connect.Request[apipb.EvidenceReconcileRequest]) (*connect.Response[apipb.EvidenceReconcileResponse], error) {
	if s.reconciler == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("evidence reconciliation is unavailable"))
	}
	runID := strings.TrimSpace(request.Msg.GetRunId())
	if runID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("run_id is required"))
	}
	if err := s.reconciler.Reconcile(ctx, runID); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&apipb.EvidenceReconcileResponse{RunId: runID, Status: "reconciled"}), nil
}

func (s *ConnectService) RecordOperatorVerification(ctx context.Context, request *connect.Request[apipb.EvidenceOperatorVerificationRequest]) (*connect.Response[apipb.EvidenceRecord], error) {
	if identity.FromContext(ctx).IsAgent() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("verified agents cannot attest operator evidence"))
	}
	metadata := map[string]string{}
	if request.Msg.GetMetadata() != nil {
		for key, value := range request.Msg.GetMetadata().GetFields() {
			if _, ok := value.GetKind().(*structpb.Value_StringValue); !ok {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("metadata %q must be a string", key))
			}
			metadata[key] = value.GetStringValue()
		}
	}
	result, err := s.service.RecordOperatorVerified(ctx, Owner{Kind: OwnerKind(request.Msg.GetOwnerKind()), ID: request.Msg.GetOwnerId(), Round: int(request.Msg.GetOwnerRound())}, request.Msg.GetEventId(), request.Msg.GetRunId(), Subject{Kind: request.Msg.GetSubjectKind(), ID: request.Msg.GetSubjectId()}, request.Msg.GetAction(), request.Msg.GetActor(), request.Msg.GetReason(), metadata)
	if err != nil {
		return nil, evidenceConnectError(err)
	}
	records, err := s.service.ListByOwner(ctx, *result.Owner)
	if err != nil {
		return nil, evidenceConnectError(err)
	}
	for _, record := range records {
		if record.Observation.SourceSystem == "swarm-manager.operator" && record.Observation.SourceEventID == strings.TrimSpace(request.Msg.GetEventId()) {
			return connect.NewResponse(recordToProto(record)), nil
		}
	}
	return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("operator evidence was committed but cannot be read"))
}

func evidenceConnectError(err error) *connect.Error {
	if errors.Is(err, ErrSourceConflict) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "exceeds") {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func recordsToProto(records []Record) []*apipb.EvidenceRecord {
	result := make([]*apipb.EvidenceRecord, 0, len(records))
	for _, record := range records {
		result = append(result, recordToProto(record))
	}
	return result
}

func recordToProto(record Record) *apipb.EvidenceRecord {
	metadata, _ := structpb.NewStruct(stringMapAny(record.Observation.Metadata))
	return &apipb.EvidenceRecord{OwnerKind: string(record.Owner.Kind), OwnerId: record.Owner.ID, OwnerRound: int32(record.Owner.Round), SourceSystem: record.Observation.SourceSystem, SourceEventId: record.Observation.SourceEventID, RunId: record.Observation.RunID, SubjectKind: record.Observation.Subject.Kind, SubjectId: record.Observation.Subject.ID, Action: record.Observation.Action, Confidence: string(record.Observation.Confidence), Verification: string(record.Observation.Verification), ContentDigest: record.Observation.ContentDigest, Metadata: metadata, ObservedAt: record.Observation.ObservedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), LinkedAt: record.LinkedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")}
}

func stringMapAny(input map[string]string) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
