package ingest

import (
	"context"
	"errors"
	"log"
	"net/http"

	"money-ledger/internal/ingest"
	"money-ledger/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	store  *ingest.Store
	logger *log.Logger
}

func NewService(db *database.RoutedDB, logger *log.Logger) *Service {
	return &Service{store: ingest.NewStore(db, nil), logger: logger}
}

func Module(db *database.RoutedDB, _ interface{}, logger *log.Logger) module.Module {
	s := NewService(db, logger)
	return module.Module{Name: "ingest", Mount: func(r *mux.Router) { p, h := ingestconnect.NewIngestServiceHandler(s); r.PathPrefix(p).Handler(h) }, Endpoints: Endpoints}
}

func invalid(err error) error  { return connect.NewError(connect.CodeInvalidArgument, err) }
func internal(err error) error { return connect.NewError(connect.CodeInternal, err) }
func notFound(err error) error { return connect.NewError(connect.CodeNotFound, err) }

func (s *Service) RegisterAdapter(ctx context.Context, req *connect.Request[ingestpb.RegisterAdapterRequest]) (*connect.Response[ingestpb.RegisterAdapterResponse], error) {
	a, err := s.store.RegisterAdapter(ctx, req.Msg.Adapter)
	if err != nil {
		return nil, invalid(err)
	}
	return connect.NewResponse(&ingestpb.RegisterAdapterResponse{Adapter: a}), nil
}

func (s *Service) ListAdapters(ctx context.Context, _ *connect.Request[ingestpb.ListAdaptersRequest]) (*connect.Response[ingestpb.ListAdaptersResponse], error) {
	a, err := s.store.ListAdapters(ctx)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&ingestpb.ListAdaptersResponse{Adapters: a}), nil
}

func (s *Service) IngestEvent(ctx context.Context, req *connect.Request[ingestpb.IngestEventRequest]) (*connect.Response[ingestpb.IngestEventResponse], error) {
	p, dup, receipt, err := s.store.IngestEvent(ctx, req.Msg.Event)
	if err != nil {
		return nil, invalid(err)
	}
	return connect.NewResponse(&ingestpb.IngestEventResponse{Posting: p, Duplicate: dup, Receipt: receipt}), nil
}

func (s *Service) RunAdapter(ctx context.Context, req *connect.Request[ingestpb.RunAdapterRequest]) (*connect.Response[ingestpb.RunAdapterResponse], error) {
	r, a, err := s.store.RunAdapter(ctx, req.Msg.AdapterId, req.Msg.From, req.Msg.To)
	if err != nil && r == nil {
		return nil, notFound(err)
	}
	return connect.NewResponse(&ingestpb.RunAdapterResponse{Receipt: r, Availability: a}), nil
}

func (s *Service) ImportFile(ctx context.Context, req *connect.Request[ingestpb.ImportFileRequest]) (*connect.Response[ingestpb.ImportFileResponse], error) {
	r, err := s.store.ImportFile(ctx, req.Msg.AdapterId, req.Msg.Csv, req.Msg.From, req.Msg.To)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, invalid(err)
	}
	return connect.NewResponse(&ingestpb.ImportFileResponse{Receipt: r}), nil
}

func (s *Service) ImportOperatorInputs(ctx context.Context, req *connect.Request[ingestpb.OperatorImportRequest]) (*connect.Response[ingestpb.OperatorImportResponse], error) {
	mode := ingest.OperatorSourceModeOperator
	if req.Msg.SourceMode == ingestpb.SourceMode_SOURCE_MODE_FIXTURE {
		mode = ingest.OperatorSourceModeFixture
	}
	var report *ingest.OperatorImportReport
	var err error
	if len(req.Msg.SourceJson) > 0 {
		report, err = s.store.ImportOperatorInputsJSON(ctx, req.Msg.SourceJson, req.Msg.Apply, req.Msg.AdapterId, req.Msg.BookId, req.Msg.AccountId)
	} else {
		report, err = s.store.ImportOperatorInputsSource(ctx, req.Msg.SourcePath, mode, req.Msg.Apply, req.Msg.AdapterId, req.Msg.BookId, req.Msg.AccountId)
	}
	if err != nil && report == nil {
		return nil, invalid(err)
	}
	response := &ingestpb.OperatorImportResponse{}
	if report != nil {
		response.Read, response.Written, response.Findings, response.Applied = int32(report.Read), int32(report.Written), int32(report.Findings), report.Applied
		for _, field := range report.Fields {
			out := &ingestpb.OperatorInputField{Path: field.Path, Status: field.Status, Written: field.Written, Unit: field.Unit, WindowDays: int32(field.WindowDays), Kind: field.Kind, Reason: field.Reason}
			if field.ObservedAt != nil {
				out.ObservedAt = timestamppb.New(*field.ObservedAt)
			}
			response.Fields = append(response.Fields, out)
		}
	}
	if err != nil {
		return connect.NewResponse(response), invalid(err)
	}
	return connect.NewResponse(response), nil
}

func (s *Service) OperatorInputStatus(ctx context.Context, req *connect.Request[ingestpb.OperatorInputStatusRequest]) (*connect.Response[ingestpb.OperatorInputStatusResponse], error) {
	fields, err := s.store.OperatorInputStatus(ctx, req.Msg.BookId)
	if err != nil {
		return nil, internal(err)
	}
	response := &ingestpb.OperatorInputStatusResponse{}
	for _, field := range fields {
		out := &ingestpb.OperatorInputField{Path: field.Path, Status: field.Status, Written: field.Written, Unit: field.Unit, WindowDays: int32(field.WindowDays), Kind: field.Kind, Reason: field.Reason}
		if field.ObservedAt != nil {
			out.ObservedAt = timestamppb.New(*field.ObservedAt)
		}
		response.Fields = append(response.Fields, out)
	}
	return connect.NewResponse(response), nil
}

var Endpoints = []module.EndpointDescriptor{
	ep("ingest_register", "/vrooli.money_ledger.v1.ingest.IngestService/RegisterAdapter", "Register a money adapter"),
	ep("ingest_list", "/vrooli.money_ledger.v1.ingest.IngestService/ListAdapters", "List money adapters"),
	ep("ingest_event", "/vrooli.money_ledger.v1.ingest.IngestService/IngestEvent", "Admit one typed money event"),
	ep("ingest_run", "/vrooli.money_ledger.v1.ingest.IngestService/RunAdapter", "Run an adapter and report availability"),
	ep("ingest_file", "/vrooli.money_ledger.v1.ingest.IngestService/ImportFile", "Import a CSV through the adapter door"),
	ep("ingest_operator_inputs", "/vrooli.money_ledger.v1.ingest.IngestService/ImportOperatorInputs", "Rehearse or apply operator financial inputs"),
	ep("ingest_operator_status", "/vrooli.money_ledger.v1.ingest.IngestService/OperatorInputStatus", "Read operator input status and age"),
}

func ep(key, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: key, Method: http.MethodPost, Path: path, Summary: summary}
}
