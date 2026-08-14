package ingest

import (
	"context"
	"errors"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	"money-ledger/internal/ingest"
	"money-ledger/internal/module"
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

var Endpoints = []module.EndpointDescriptor{
	ep("ingest_register", "/vrooli.money_ledger.v1.ingest.IngestService/RegisterAdapter", "Register a money adapter"),
	ep("ingest_list", "/vrooli.money_ledger.v1.ingest.IngestService/ListAdapters", "List money adapters"),
	ep("ingest_event", "/vrooli.money_ledger.v1.ingest.IngestService/IngestEvent", "Admit one typed money event"),
	ep("ingest_run", "/vrooli.money_ledger.v1.ingest.IngestService/RunAdapter", "Run an adapter and report availability"),
	ep("ingest_file", "/vrooli.money_ledger.v1.ingest.IngestService/ImportFile", "Import a CSV through the adapter door"),
}

func ep(key, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: key, Method: http.MethodPost, Path: path, Summary: summary}
}
