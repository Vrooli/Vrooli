package offers

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	"offer-desk/internal/catalog"
	"offer-desk/internal/module"
)

type Service struct {
	store    *catalog.Store
	logger   *log.Logger
	clock    schedule.Clock
	journal  ledgerconnect.JournalServiceClient
	position ledgerconnect.PositionServiceClient
	bookID   string
}

func NewService(db *database.RoutedDB, logger *log.Logger, clock schedule.Clock) *Service {
	if clock == nil {
		clock = schedule.System()
	}
	s := &Service{store: catalog.NewStore(db, clock.Now), logger: logger, clock: clock, bookID: os.Getenv("MONEY_LEDGER_BOOK_ID")}
	if base := os.Getenv("MONEY_LEDGER_API_URL"); base != "" {
		hc := &http.Client{Timeout: 700 * time.Millisecond}
		s.journal = ledgerconnect.NewJournalServiceClient(hc, base)
		s.position = ledgerconnect.NewPositionServiceClient(hc, base)
	}
	return s
}

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	s := NewService(db, logger, clock)
	s.startScheduler()
	return module.Module{Name: "offers", Mount: func(r *mux.Router) {
		p, h := offersconnect.NewCatalogServiceHandler(s)
		r.PathPrefix(p).Handler(h)
		p, h = offersconnect.NewGatesServiceHandler(s)
		r.PathPrefix(p).Handler(h)
		p, h = offersconnect.NewBoardServiceHandler(s)
		r.PathPrefix(p).Handler(h)
	}, Endpoints: Endpoints}
}

func (s *Service) startScheduler() {
	interval := time.Minute
	if raw := os.Getenv("OFFER_EVALUATION_INTERVAL"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			interval = parsed
		}
	}
	ticker := s.clock.NewTicker(interval)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer ticker.Stop()
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C():
				if _, err := s.store.Evaluate(ctx, false); err != nil && s.logger != nil {
					s.logger.Printf("scheduled offer evaluation failed: %v", err)
				}
			}
		}
	}()
}
func invalid(err error) error  { return connect.NewError(connect.CodeInvalidArgument, err) }
func internal(err error) error { return connect.NewError(connect.CodeInternal, err) }
func (s *Service) CreateNode(ctx context.Context, r *connect.Request[offerspb.CreateNodeRequest]) (*connect.Response[offerspb.CreateNodeResponse], error) {
	n, e := s.store.CreateNode(ctx, r.Msg.Kind, r.Msg.Name, r.Msg.Status, r.Msg.TriggerId, r.Msg.ActualAccountId)
	if e != nil {
		return nil, invalid(e)
	}
	return connect.NewResponse(&offerspb.CreateNodeResponse{Node: n}), nil
}

func (s *Service) ListNodes(ctx context.Context, r *connect.Request[offerspb.ListNodesRequest]) (*connect.Response[offerspb.ListNodesResponse], error) {
	n, e := s.store.ListNodes(ctx, r.Msg.Kind, r.Msg.Status)
	if e != nil {
		return nil, internal(e)
	}
	return connect.NewResponse(&offerspb.ListNodesResponse{Nodes: n}), nil
}

func (s *Service) Transition(ctx context.Context, r *connect.Request[offerspb.TransitionRequest]) (*connect.Response[offerspb.TransitionResponse], error) {
	n, e := s.store.Transition(ctx, r.Msg.NodeId, r.Msg.Status, r.Msg.Actor)
	if e != nil {
		return nil, invalid(e)
	}
	return connect.NewResponse(&offerspb.TransitionResponse{Node: n}), nil
}

func (s *Service) CreateEdge(ctx context.Context, r *connect.Request[offerspb.CreateEdgeRequest]) (*connect.Response[offerspb.CreateEdgeResponse], error) {
	e, err := s.store.CreateEdge(ctx, r.Msg.Edge)
	if err != nil {
		return nil, invalid(err)
	}
	return connect.NewResponse(&offerspb.CreateEdgeResponse{Edge: e}), nil
}

func (s *Service) ListEdges(ctx context.Context, r *connect.Request[offerspb.ListEdgesRequest]) (*connect.Response[offerspb.ListEdgesResponse], error) {
	edges, err := s.store.ListEdges(ctx, r.Msg.NodeId)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&offerspb.ListEdgesResponse{Edges: edges}), nil
}

func (s *Service) DeclareTrigger(ctx context.Context, r *connect.Request[offerspb.DeclareTriggerRequest]) (*connect.Response[offerspb.DeclareTriggerResponse], error) {
	t, e := s.store.AddTrigger(ctx, r.Msg.Trigger)
	if e != nil {
		return nil, invalid(e)
	}
	return connect.NewResponse(&offerspb.DeclareTriggerResponse{Trigger: t}), nil
}

func (s *Service) AddFact(ctx context.Context, r *connect.Request[offerspb.AddFactRequest]) (*connect.Response[offerspb.AddFactResponse], error) {
	f, e := s.store.AddFact(ctx, r.Msg.Fact)
	if e != nil {
		return nil, invalid(e)
	}
	return connect.NewResponse(&offerspb.AddFactResponse{Fact: f}), nil
}

func (s *Service) Evaluate(ctx context.Context, r *connect.Request[offerspb.EvaluateRequest]) (*connect.Response[offerspb.EvaluateResponse], error) {
	e, err := s.store.Evaluate(ctx, r.Msg.DryRun)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&offerspb.EvaluateResponse{Evaluations: e}), nil
}

func (s *Service) Promote(ctx context.Context, r *connect.Request[offerspb.PromoteRequest]) (*connect.Response[offerspb.PromoteResponse], error) {
	if r.Msg.Role == "operator" {
		if _, err := s.store.Transition(ctx, r.Msg.NodeId, offerspb.Status_ACTIVE, r.Msg.Actor); err != nil {
			return nil, invalid(err)
		}
	}
	p, e := s.store.Proposal(ctx, r.Msg.NodeId, r.Msg.Actor, offerspb.Status_ACTIVE, "operator promotion requested")
	if e != nil {
		return nil, internal(e)
	}
	return connect.NewResponse(&offerspb.PromoteResponse{Proposal: p}), nil
}

func (s *Service) GetBoard(ctx context.Context, _ *connect.Request[offerspb.ProjectionRequest]) (*connect.Response[offerspb.BoardResponse], error) {
	nodes, e := s.store.ListNodes(ctx, offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_STATUS_UNSPECIFIED)
	if e != nil {
		return nil, internal(e)
	}
	out := make([]*offerspb.BoardEntry, 0, len(nodes))
	response := &offerspb.BoardResponse{Entries: out}
	if s.position == nil || s.bookID == "" {
		response.Availability = append(response.Availability, &offerspb.Availability{Source: "money-ledger", Reason: "actuals unavailable: MONEY_LEDGER_API_URL and MONEY_LEDGER_BOOK_ID are not configured"})
	} else {
		deadline, cancel := context.WithTimeout(ctx, 700*time.Millisecond)
		defer cancel()
		position, err := s.position.GetPosition(deadline, connect.NewRequest(&ledgerpb.PositionRequest{BookId: s.bookID}))
		if err != nil {
			response.Availability = append(response.Availability, &offerspb.Availability{Source: "money-ledger.position", Reason: err.Error()})
		} else {
			response.Position = position.Msg
		}
	}
	for _, n := range nodes {
		reason := "active and earning nothing"
		if n.Status == offerspb.Status_TRIGGER_MET {
			reason = "trigger fired"
		}
		if n.Status == offerspb.Status_CANDIDATE {
			reason = "blocked: trigger not met"
		}
		entry := &offerspb.BoardEntry{NodeId: n.Id, Title: n.Name, RankReason: reason, Status: n.Status}
		if s.journal != nil && n.ActualAccountId != "" {
			deadline, cancel := context.WithTimeout(ctx, 700*time.Millisecond)
			postings, err := s.journal.ListPostings(deadline, connect.NewRequest(&ledgerpb.ListPostingsRequest{AccountId: n.ActualAccountId, Limit: 500}))
			cancel()
			if err != nil {
				entry.Availability = append(entry.Availability, &offerspb.Availability{Source: "money-ledger.actuals", Reason: err.Error()})
			} else {
				for _, p := range postings.Msg.Postings {
					entry.ActualMinor += p.Event.AmountMinor
				}
				entry.ActualsAvailable = true
			}
		} else {
			entry.Availability = append(entry.Availability, &offerspb.Availability{Source: "money-ledger.actuals", Reason: "no ledger account mapping"})
		}
		out = append(out, entry)
	}
	response.Entries = out
	return connect.NewResponse(response), nil
}
func Schema() string { return (&catalog.Store{}).Schema() }
func ep(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: http.MethodPost, Summary: summary, Category: "offers"}
}

var Endpoints = []module.EndpointDescriptor{
	ep("catalog_create", "/vrooli.offer_desk.v1.offers.CatalogService/CreateNode", "Create a typed offer-graph node"), ep("catalog_list", "/vrooli.offer_desk.v1.offers.CatalogService/ListNodes", "List offer-graph nodes"), ep("catalog_transition", "/vrooli.offer_desk.v1.offers.CatalogService/Transition", "Transition a node through the enforced lifecycle"), ep("catalog_edge", "/vrooli.offer_desk.v1.offers.CatalogService/CreateEdge", "Create a typed graph edge"), ep("catalog_edges", "/vrooli.offer_desk.v1.offers.CatalogService/ListEdges", "List typed graph edges"),
	ep("gates_trigger", "/vrooli.offer_desk.v1.offers.GatesService/DeclareTrigger", "Declare a machine-evaluable trigger"), ep("gates_fact", "/vrooli.offer_desk.v1.offers.GatesService/AddFact", "Record an observed fact"), ep("gates_evaluate", "/vrooli.offer_desk.v1.offers.GatesService/Evaluate", "Evaluate candidate triggers"), ep("gates_promote", "/vrooli.offer_desk.v1.offers.GatesService/Promote", "Create an operator promotion proposal"), ep("board_show", "/vrooli.offer_desk.v1.offers.BoardService/GetBoard", "Read the ranked offer board"),
}
