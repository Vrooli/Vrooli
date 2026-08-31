package offers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"offer-desk/internal/catalog"
	"offer-desk/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/schedule"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	store    *catalog.Store
	logger   *log.Logger
	clock    schedule.Clock
	journal  ledgerconnect.JournalServiceClient
	position ledgerconnect.PositionServiceClient
	bookID   string
	bookName string
	books    ledgerconnect.BooksServiceClient
	interval time.Duration
}

func NewService(db *database.RoutedDB, logger *log.Logger, clock schedule.Clock) *Service {
	if clock == nil {
		clock = schedule.System()
	}
	s := &Service{store: catalog.NewStore(db, clock.Now), logger: logger, clock: clock, bookID: os.Getenv("MONEY_LEDGER_BOOK_ID")}
	base := os.Getenv("MONEY_LEDGER_API_URL")
	if base == "" {
		// The declared scenario dependency is the default wiring. An explicit
		// environment URL remains an operator override for isolated deployments.
		if resolved, err := discovery.ResolveScenarioURLDefault(context.Background(), "money-ledger"); err == nil {
			base = resolved
		}
	}
	if base != "" {
		hc := &http.Client{Timeout: 700 * time.Millisecond}
		s.journal = ledgerconnect.NewJournalServiceClient(hc, base)
		s.position = ledgerconnect.NewPositionServiceClient(hc, base)
		s.books = ledgerconnect.NewBooksServiceClient(hc, base)
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
		p, h = offersconnect.NewReleaseLadderServiceHandler(s)
		r.PathPrefix(p).Handler(h)
		p, h = offersconnect.NewSpaceServiceHandler(s)
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
	s.interval = interval
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
				evaluations, err := s.store.Evaluate(ctx, false)
				if err != nil {
					s.recordEvaluation(ctx, "failed", 0, err.Error())
					if s.logger != nil {
						s.logger.Printf("scheduled offer evaluation failed: %v", err)
					}
				} else {
					s.recordEvaluation(ctx, "succeeded", len(evaluations), "")
				}
			}
		}
	}()
}
func invalid(err error) error  { return connect.NewError(connect.CodeInvalidArgument, err) }
func internal(err error) error { return connect.NewError(connect.CodeInternal, err) }

// boardRank orders the board by how much operator attention a row deserves. The
// page's declared first priority is "which triggers fired since I last looked",
// so a fired trigger must be reachable without scrolling past retired drills.
//
// Ranking is deliberately separate from rankReason: the reason explains a row,
// the rank places it, and conflating them made the board explain rows it never
// ordered. Lower is more urgent.
func boardRank(status offerspb.Status, actualsAvailable bool, actualMinor int64) int {
	switch status {
	case offerspb.Status_TRIGGER_MET:
		// A condition the operator declared has been satisfied. This is the only
		// row class that represents a decision the system is waiting on.
		return 0
	case offerspb.Status_PROPOSED:
		// An agent has proposed a promotion; only an operator can complete it.
		return 1
	case offerspb.Status_ACTIVE, offerspb.Status_SHIPPED:
		// Earning nothing while active outranks earning something. An unavailable
		// actuals read is NOT treated as zero: it sorts below a confirmed zero,
		// because missing evidence must never be promoted to a business finding.
		if actualsAvailable && actualMinor == 0 {
			return 2
		}
		if !actualsAvailable {
			return 3
		}
		return 4
	case offerspb.Status_CANDIDATE:
		return 5
	case offerspb.Status_IDEA:
		return 6
	case offerspb.Status_RETIRED:
		return 7
	default:
		return 8
	}
}

func rankReason(status offerspb.Status, actualsAvailable bool, actualMinor int64, unavailableSource string) string {
	if !actualsAvailable {
		source := strings.TrimSpace(unavailableSource)
		if source == "" {
			source = "money-ledger.actuals"
		}
		switch status {
		case offerspb.Status_ACTIVE:
			return fmt.Sprintf("active; earnings unknown — %s unavailable", source)
		case offerspb.Status_SHIPPED:
			return fmt.Sprintf("shipped; earnings unknown — %s unavailable", source)
		}
	}
	switch status {
	case offerspb.Status_STATUS_UNSPECIFIED:
		return "status not set"
	case offerspb.Status_IDEA:
		return "captured, not planned against"
	case offerspb.Status_CANDIDATE:
		return "blocked: trigger not met"
	case offerspb.Status_TRIGGER_MET:
		return "trigger fired"
	case offerspb.Status_PROPOSED:
		return "awaiting operator decision"
	case offerspb.Status_ACTIVE:
		if actualMinor == 0 {
			return "active and earning nothing"
		}
		return "active and earning"
	case offerspb.Status_SHIPPED:
		if actualMinor == 0 {
			return "shipped and earning nothing"
		}
		return "shipped and earning"
	case offerspb.Status_RETIRED:
		return "retired"
	default:
		return fmt.Sprintf("unknown status: %d", status)
	}
}

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

func (s *Service) SetReleaseRank(ctx context.Context, r *connect.Request[offerspb.SetReleaseRankRequest]) (*connect.Response[offerspb.SetReleaseRankResponse], error) {
	n, prior, err := s.store.SetReleaseRank(ctx, r.Msg.NodeId, r.Msg.ReleaseRank, r.Msg.Actor)
	if err != nil {
		return nil, invalid(err)
	}
	return connect.NewResponse(&offerspb.SetReleaseRankResponse{Node: n, PriorReleaseRank: prior}), nil
}

func (s *Service) GetReleaseLadder(ctx context.Context, r *connect.Request[offerspb.ReleaseLadderRequest]) (*connect.Response[offerspb.ReleaseLadderResponse], error) {
	ladder, err := s.store.ReleaseLadder(ctx, r.Msg.IncludeRetired)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(ladder), nil
}

func (s *Service) GetPrerequisites(ctx context.Context, r *connect.Request[offerspb.PrerequisiteWalkRequest]) (*connect.Response[offerspb.PrerequisiteWalkResponse], error) {
	result, err := s.store.Prerequisites(ctx, r.Msg.StreamNodeId)
	if err != nil {
		return nil, invalid(err)
	}
	return connect.NewResponse(result), nil
}

func (s *Service) ListEdges(ctx context.Context, r *connect.Request[offerspb.ListEdgesRequest]) (*connect.Response[offerspb.ListEdgesResponse], error) {
	edges, err := s.store.ListEdges(ctx, r.Msg.NodeId)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&offerspb.ListEdgesResponse{Edges: edges}), nil
}

func (s *Service) ImportCatalog(ctx context.Context, r *connect.Request[offerspb.ImportCatalogRequest]) (*connect.Response[offerspb.ImportCatalogResponse], error) {
	report, err := s.store.ImportCatalog(ctx, r.Msg.SourcePath, r.Msg.SourceMode, r.Msg.Apply, r.Msg.Actor)
	if err != nil && report == nil {
		return nil, invalid(err)
	}
	response := &offerspb.ImportCatalogResponse{}
	if report != nil {
		response.Applied = report.Applied
		for _, file := range report.Files {
			response.Files = append(response.Files, &offerspb.ImportFileReport{Path: file.Path, Read: int32(file.Read), Written: int32(file.Written), Findings: int32(file.Findings), Cardinality: file.Cardinality, NodeKind: file.NodeKind})
		}
		for _, status := range report.StatusMap {
			response.StatusMap = append(response.StatusMap, &offerspb.StatusMapEntry{Path: status.Path, Status: status.Status, Recognized: status.Recognized, Line: int32(status.Line)})
		}
		for _, finding := range report.Findings {
			response.Findings = append(response.Findings, &offerspb.ImportFinding{Path: finding.Path, Reason: finding.Reason, Blocking: finding.Blocking, Line: int32(finding.Line)})
		}
		response.TotalFindings = int32(len(report.Findings))
	}
	if err != nil {
		return connect.NewResponse(response), invalid(err)
	}
	return connect.NewResponse(response), nil
}

func (s *Service) MergeNodes(ctx context.Context, r *connect.Request[offerspb.MergeNodesRequest]) (*connect.Response[offerspb.MergeNodesResponse], error) {
	response, err := s.store.MergeNodes(ctx, r.Msg)
	if err != nil {
		return nil, invalid(err)
	}
	return connect.NewResponse(response), nil
}

func (s *Service) MapAccount(ctx context.Context, r *connect.Request[offerspb.MapAccountRequest]) (*connect.Response[offerspb.MapAccountResponse], error) {
	node, prior, err := s.store.MapAccount(ctx, r.Msg)
	if err != nil {
		return nil, invalid(err)
	}
	return connect.NewResponse(&offerspb.MapAccountResponse{Node: node, PriorAccountId: prior}), nil
}

func (s *Service) VerifyCatalog(ctx context.Context, r *connect.Request[offerspb.VerifyCatalogRequest]) (*connect.Response[offerspb.VerifyCatalogResponse], error) {
	report, err := s.store.VerifyCatalog(ctx, r.Msg.SourcePath, r.Msg.SourceMode)
	if err != nil {
		return nil, invalid(err)
	}
	response := &offerspb.VerifyCatalogResponse{
		TotalDrift:          int32(report.TotalDrift),
		Reconciled:          report.Reconciled,
		Comparable:          report.Comparable,
		NotComparableReason: report.NotComparableReason,
	}
	for _, file := range report.Files {
		response.Files = append(response.Files, &offerspb.VerifyFileReport{Path: file.Path, Expected: int32(file.Expected), Live: int32(file.Live)})
	}
	response.DuplicateIdentities = append(response.DuplicateIdentities, report.DuplicateIdentities...)
	response.OrphanEdgeIds = append(response.OrphanEdgeIds, report.OrphanEdgeIds...)
	response.ExtraNodeIds = append(response.ExtraNodeIds, report.ExtraNodeIds...)
	return connect.NewResponse(response), nil
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
	} else if _, err := s.store.Transition(ctx, r.Msg.NodeId, offerspb.Status_PROPOSED, r.Msg.Actor); err != nil {
		return nil, invalid(err)
	}
	p, e := s.store.Proposal(ctx, r.Msg.NodeId, r.Msg.Actor, offerspb.Status_ACTIVE, "operator promotion requested")
	if e != nil {
		return nil, internal(e)
	}
	return connect.NewResponse(&offerspb.PromoteResponse{Proposal: p}), nil
}

func (s *Service) ListProposals(ctx context.Context, r *connect.Request[offerspb.ListProposalsRequest]) (*connect.Response[offerspb.ListProposalsResponse], error) {
	proposals, err := s.store.ListProposals(ctx, r.Msg.NodeId, r.Msg.Status)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&offerspb.ListProposalsResponse{Proposals: proposals}), nil
}

func (s *Service) GetBoard(ctx context.Context, _ *connect.Request[offerspb.ProjectionRequest]) (*connect.Response[offerspb.BoardResponse], error) {
	nodes, e := s.store.ListNodes(ctx, offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_STATUS_UNSPECIFIED)
	if e != nil {
		return nil, internal(e)
	}
	out := make([]*offerspb.BoardEntry, 0, len(nodes))
	response := &offerspb.BoardResponse{Entries: out, Evaluation: &offerspb.EvaluationCondition{LastResult: offerspb.EvaluationResult_EVALUATION_NOT_RUN, Degraded: true, Reason: "evaluation has not run"}}
	if s.position == nil {
		response.Availability = append(response.Availability, &offerspb.Availability{Source: "money-ledger", Reason: "actuals unavailable: declared money-ledger dependency could not be resolved"})
	} else {
		if s.books != nil {
			books, bookErr := s.books.ListBooks(ctx, connect.NewRequest(&ledgerpb.ListBooksRequest{}))
			if bookErr == nil {
				if s.bookID == "" && len(books.Msg.Books) > 0 {
					s.bookID = books.Msg.Books[0].Id
				}
				for _, book := range books.Msg.Books {
					if book.Id == s.bookID {
						s.bookName = book.Name
						break
					}
				}
			}
		}
		if s.bookID == "" {
			response.Availability = append(response.Availability, &offerspb.Availability{Source: "money-ledger", Reason: "actuals unavailable: no ledger book is configured or available"})
		}
		if s.bookID != "" {
			deadline, cancel := context.WithTimeout(ctx, 700*time.Millisecond)
			defer cancel()
			position, err := s.position.GetPosition(deadline, connect.NewRequest(&ledgerpb.PositionRequest{BookId: s.bookID}))
			if err != nil {
				response.Availability = append(response.Availability, &offerspb.Availability{Source: "money-ledger.position", Reason: err.Error()})
			} else {
				response.Position = position.Msg
				response.PostureSource = "money-ledger.position"
				if s.bookName != "" {
					response.PostureSource += ": " + s.bookName
				}
				for _, input := range position.Msg.Inputs {
					if input.AgeSeconds > response.PostureAgeSeconds {
						response.PostureAgeSeconds = input.AgeSeconds
					}
				}
				if position.Msg.RevenueMinor == 0 && position.Msg.BurnMinor == 0 {
					response.DefaultAliveGap = "unavailable: revenue and burn observations are incomplete"
				} else {
					gap := float64(position.Msg.BurnMinor)*1.25 - float64(position.Msg.RevenueMinor)
					if gap <= 0 {
						response.DefaultAliveGap = "default-alive threshold met with 1.25 buffer"
					} else {
						response.DefaultAliveGap = fmt.Sprintf("%d minor units below default-alive buffer", int64(gap))
					}
				}
				if s.position != nil {
					goals, goalErr := s.position.ListGoals(deadline, connect.NewRequest(&ledgerpb.ListGoalsRequest{BookId: s.bookID}))
					if goalErr == nil {
						response.Goals = goals.Msg.Goals
					}
				}
			}
		}
	}
	for _, n := range nodes {
		entry := &offerspb.BoardEntry{NodeId: n.Id, Title: n.Name, Status: n.Status}
		actualsSource := "money-ledger.actuals"
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
		if len(entry.Availability) > 0 && entry.Availability[0].Source != "" {
			actualsSource = entry.Availability[0].Source
		}
		entry.RankReason = rankReason(n.Status, entry.ActualsAvailable, entry.ActualMinor, actualsSource)
		out = append(out, entry)
	}
	// Stable sort on (rank, title) so equal-urgency rows keep a predictable,
	// human-scannable order and the board does not reshuffle between reads.
	sort.SliceStable(out, func(i, j int) bool {
		ri := boardRank(out[i].Status, out[i].ActualsAvailable, out[i].ActualMinor)
		rj := boardRank(out[j].Status, out[j].ActualsAvailable, out[j].ActualMinor)
		if ri != rj {
			return ri < rj
		}
		return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title)
	})
	response.Entries = out
	result, scored, reason, at, evaluationErr := s.store.LatestEvaluation(ctx)
	if evaluationErr == nil {
		age := int64(s.clock.Now().Sub(at).Seconds())
		if age < 0 {
			age = 0
		}
		response.Evaluation.LastRunAt = timestamppb.New(at)
		response.Evaluation.NodesScored = int32(scored)
		response.Evaluation.AgeSeconds = age
		response.Evaluation.Reason = reason
		response.Evaluation.LastResult = offerspb.EvaluationResult_EVALUATION_SUCCEEDED
		if result == "failed" {
			response.Evaluation.LastResult = offerspb.EvaluationResult_EVALUATION_FAILED
		}
		response.Evaluation.Degraded = result == "failed" || (s.interval > 0 && time.Duration(age)*time.Second > 3*s.interval)
		if response.Evaluation.Degraded && response.Evaluation.Reason == "" {
			response.Evaluation.Reason = "evaluation reading is older than three scheduler intervals"
		}
	}
	return connect.NewResponse(response), nil
}

func (s *Service) recordEvaluation(ctx context.Context, result string, nodes int, reason string) {
	_ = s.store.RecordEvaluation(ctx, result, nodes, reason, s.clock.Now())
}

func (s *Service) GetProjection(ctx context.Context, r *connect.Request[offerspb.ProjectionRequest]) (*connect.Response[offerspb.SpaceResponse], error) {
	projection := "offers"
	if r != nil && strings.TrimSpace(r.Msg.Projection) != "" {
		projection = strings.TrimSpace(r.Msg.Projection)
	}
	return connect.NewResponse(&offerspb.SpaceResponse{
		SchemaVersion:         "space/v1",
		Projection:            projection,
		Owner:                 "monetization",
		DenominatorConfidence: "sketch",
		ConfidenceRationale:   "The obligation cells are an operator-authored first cut and have not yet been reconciled against an external roster.",
		Source:                "scenarios/offer-desk/docs/spaces/offers.json",
		Cells: []*offerspb.SpaceCell{
			{Id: "catalog-state", Group: "catalog", Question: "What offers and delivery tiers are currently declared?", Owner: "monetization", Status: "active", Notes: "Read from the typed catalog graph."},
			{Id: "promotion-state", Group: "gates", Question: "Which candidate offers have satisfied machine-evaluable triggers?", Owner: "monetization", Status: "active", Notes: "Read from persisted evaluations and lifecycle state."},
			{Id: "financial-posture", Group: "board", Question: "What is the financial posture beside the offer state?", Owner: "monetization", Status: "active", Notes: "Read from Money Ledger when available; unavailable sources remain explicit."},
		},
	}), nil
}
func Schema() string { return (&catalog.Store{}).Schema() }
func ep(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: http.MethodPost, Summary: summary, Category: "offers"}
}

var Endpoints = []module.EndpointDescriptor{
	ep("catalog_create", "/vrooli.offer_desk.v1.offers.CatalogService/CreateNode", "Create a typed offer-graph node"), ep("catalog_list", "/vrooli.offer_desk.v1.offers.CatalogService/ListNodes", "List offer-graph nodes"), ep("catalog_transition", "/vrooli.offer_desk.v1.offers.CatalogService/Transition", "Transition a node through the enforced lifecycle"), ep("catalog_edge", "/vrooli.offer_desk.v1.offers.CatalogService/CreateEdge", "Create a typed graph edge"), ep("catalog_edges", "/vrooli.offer_desk.v1.offers.CatalogService/ListEdges", "List typed graph edges"), ep("catalog_import", "/vrooli.offer_desk.v1.offers.CatalogService/ImportCatalog", "Rehearse or apply a declared catalog source"), ep("catalog_map_account", "/vrooli.offer_desk.v1.offers.CatalogService/MapAccount", "Map a node to the ledger account holding its actuals"), ep("catalog_merge", "/vrooli.offer_desk.v1.offers.CatalogService/MergeNodes", "Dry-run or apply an audited duplicate-node merge"), ep("catalog_verify", "/vrooli.offer_desk.v1.offers.CatalogService/VerifyCatalog", "Verify source counts and graph identity reconciliation"), ep("catalog_set_release_rank", "/vrooli.offer_desk.v1.offers.CatalogService/SetReleaseRank", "Set an operator-owned deliverable release rank"),
	ep("gates_trigger", "/vrooli.offer_desk.v1.offers.GatesService/DeclareTrigger", "Declare a machine-evaluable trigger"), ep("gates_fact", "/vrooli.offer_desk.v1.offers.GatesService/AddFact", "Record an observed fact"), ep("gates_evaluate", "/vrooli.offer_desk.v1.offers.GatesService/Evaluate", "Evaluate candidate triggers"), ep("gates_promote", "/vrooli.offer_desk.v1.offers.GatesService/Promote", "Create an operator promotion proposal"), ep("gates_proposals", "/vrooli.offer_desk.v1.offers.GatesService/ListProposals", "List promotion proposals and decline history"), ep("board_show", "/vrooli.offer_desk.v1.offers.BoardService/GetBoard", "Read the ranked offer board"),
	ep("board_release_ladder", "/vrooli.offer_desk.v1.offers.ReleaseLadderService/GetReleaseLadder", "Read the typed release ladder"), ep("board_prerequisites", "/vrooli.offer_desk.v1.offers.ReleaseLadderService/GetPrerequisites", "Walk stream prerequisites and unshipped deliverables"), ep("space_projection", "/vrooli.offer_desk.v1.offers.SpaceService/GetProjection", "Read monetization obligation cells"),
}
