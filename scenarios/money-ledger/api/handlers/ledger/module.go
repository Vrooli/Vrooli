package ledger

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"money-ledger/internal/ledger"
	"money-ledger/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
)

type Service struct {
	store  *ledger.Store
	logger *log.Logger
}

func NewService(db *database.RoutedDB, logger *log.Logger) *Service {
	return &Service{store: ledger.NewStore(db, nil), logger: logger}
}

func EnsureMigrations(db *database.RoutedDB) error {
	return ledger.NewStore(db, nil).EnsureMigrations(context.Background())
}

func Module(db *database.RoutedDB, _ interface{}, logger *log.Logger) module.Module {
	s := NewService(db, logger)
	return module.Module{Name: "ledger", Mount: func(r *mux.Router) {
		p, h := ledgerconnect.NewBooksServiceHandler(s)
		r.PathPrefix(p).Handler(h)
		p, h = ledgerconnect.NewJournalServiceHandler(s)
		r.PathPrefix(p).Handler(h)
		p, h = ledgerconnect.NewPositionServiceHandler(s)
		r.PathPrefix(p).Handler(h)
	}, Endpoints: Endpoints}
}

func invalid(err error) error  { return connect.NewError(connect.CodeInvalidArgument, err) }
func internal(err error) error { return connect.NewError(connect.CodeInternal, err) }
func notFound(err error) error { return connect.NewError(connect.CodeNotFound, err) }
func actor(ctx context.Context) string {
	if value := ctx.Value(actorKey{}); value != nil {
		if v, ok := value.(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return "operator"
}

type actorKey struct{}

func (s *Service) CreateBook(ctx context.Context, req *connect.Request[ledgerpb.CreateBookRequest]) (*connect.Response[ledgerpb.CreateBookResponse], error) {
	b, err := s.store.CreateBook(ctx, req.Msg.Name, req.Msg.Currency)
	if err != nil {
		return nil, invalid(err)
	}
	return connect.NewResponse(&ledgerpb.CreateBookResponse{Book: b}), nil
}

func (s *Service) ListBooks(ctx context.Context, req *connect.Request[ledgerpb.ListBooksRequest]) (*connect.Response[ledgerpb.ListBooksResponse], error) {
	b, err := s.store.ListBooks(ctx, req.Msg.IncludeArchived)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&ledgerpb.ListBooksResponse{Books: b}), nil
}

func (s *Service) CreateAccount(ctx context.Context, req *connect.Request[ledgerpb.CreateAccountRequest]) (*connect.Response[ledgerpb.CreateAccountResponse], error) {
	a, err := s.store.CreateAccount(ctx, req.Msg.BookId, req.Msg.Name, req.Msg.AccountKind)
	if err != nil {
		return nil, invalid(err)
	}
	return connect.NewResponse(&ledgerpb.CreateAccountResponse{Account: a}), nil
}

func (s *Service) ArchiveBook(ctx context.Context, req *connect.Request[ledgerpb.ArchiveBookRequest]) (*connect.Response[ledgerpb.ArchiveBookResponse], error) {
	b, err := s.store.ArchiveBook(ctx, req.Msg.BookId, actor(ctx))
	if err != nil {
		return nil, notFound(err)
	}
	return connect.NewResponse(&ledgerpb.ArchiveBookResponse{Book: b}), nil
}

func (s *Service) ListAccounts(ctx context.Context, req *connect.Request[ledgerpb.ListAccountsRequest]) (*connect.Response[ledgerpb.ListAccountsResponse], error) {
	a, err := s.store.ListAccounts(ctx, req.Msg.BookId)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&ledgerpb.ListAccountsResponse{Accounts: a}), nil
}

func (s *Service) ListPostings(ctx context.Context, req *connect.Request[ledgerpb.ListPostingsRequest]) (*connect.Response[ledgerpb.ListPostingsResponse], error) {
	p, err := s.store.ListPostings(ctx, req.Msg.AccountId, req.Msg.BookId, req.Msg.From, req.Msg.To, int(req.Msg.Limit))
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&ledgerpb.ListPostingsResponse{Postings: p}), nil
}

func (s *Service) GetPosting(ctx context.Context, req *connect.Request[ledgerpb.GetPostingRequest]) (*connect.Response[ledgerpb.GetPostingResponse], error) {
	p, err := s.store.GetPosting(ctx, req.Msg.PostingId)
	if err != nil {
		return nil, notFound(err)
	}
	return connect.NewResponse(&ledgerpb.GetPostingResponse{Posting: p}), nil
}

func (s *Service) ReversePosting(ctx context.Context, req *connect.Request[ledgerpb.ReversePostingRequest]) (*connect.Response[ledgerpb.ReversePostingResponse], error) {
	p, err := s.store.Reverse(ctx, req.Msg.PostingId, req.Msg.Reason, actor(ctx))
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, notFound(err)
	}
	return connect.NewResponse(&ledgerpb.ReversePostingResponse{Posting: p}), nil
}

func (s *Service) Transfer(ctx context.Context, req *connect.Request[ledgerpb.TransferRequest]) (*connect.Response[ledgerpb.TransferResponse], error) {
	posts, err := s.store.Transfer(ctx, req.Msg.FromAccountId, req.Msg.ToAccountId, req.Msg.AmountMinor, req.Msg.Currency, req.Msg.ExternalId, req.Msg.Description, req.Msg.OccurredAt, actor(ctx))
	if err != nil {
		return nil, invalid(err)
	}
	return connect.NewResponse(&ledgerpb.TransferResponse{Postings: posts}), nil
}

func (s *Service) GetPosition(ctx context.Context, req *connect.Request[ledgerpb.PositionRequest]) (*connect.Response[ledgerpb.PositionResponse], error) {
	p, err := s.store.Position(ctx, req.Msg.BookId)
	if err != nil {
		return nil, notFound(err)
	}
	return connect.NewResponse(p), nil
}

func (s *Service) GetStatement(ctx context.Context, req *connect.Request[ledgerpb.StatementRequest]) (*connect.Response[ledgerpb.StatementResponse], error) {
	statement, err := s.store.Statement(ctx, req.Msg.BookId, req.Msg.From, req.Msg.To)
	if err != nil {
		return nil, notFound(err)
	}
	return connect.NewResponse(statement), nil
}

func (s *Service) DeclareGoal(ctx context.Context, req *connect.Request[ledgerpb.DeclareGoalRequest]) (*connect.Response[ledgerpb.DeclareGoalResponse], error) {
	g, err := s.store.DeclareGoal(ctx, req.Msg.Goal.BookId, req.Msg.Goal)
	if err != nil {
		return nil, invalid(err)
	}
	return connect.NewResponse(&ledgerpb.DeclareGoalResponse{Goal: g}), nil
}

func (s *Service) ListGoals(ctx context.Context, req *connect.Request[ledgerpb.ListGoalsRequest]) (*connect.Response[ledgerpb.ListGoalsResponse], error) {
	g, err := s.store.ListGoals(ctx, req.Msg.BookId, req.Msg.IncludeArchived)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&ledgerpb.ListGoalsResponse{Goals: g}), nil
}

func (s *Service) ArchiveGoal(ctx context.Context, req *connect.Request[ledgerpb.ArchiveGoalRequest]) (*connect.Response[ledgerpb.ArchiveGoalResponse], error) {
	g, err := s.store.ArchiveGoal(ctx, req.Msg.GoalId, actor(ctx))
	if err != nil {
		return nil, notFound(err)
	}
	return connect.NewResponse(&ledgerpb.ArchiveGoalResponse{Goal: g}), nil
}

func (s *Service) ReparentGoal(ctx context.Context, req *connect.Request[ledgerpb.ReparentGoalRequest]) (*connect.Response[ledgerpb.ReparentGoalResponse], error) {
	g, err := s.store.ReparentGoal(ctx, req.Msg.GoalId, req.Msg.BookId, actor(ctx))
	if err != nil {
		return nil, invalid(err)
	}
	return connect.NewResponse(&ledgerpb.ReparentGoalResponse{Goal: g}), nil
}

func Schema() string { return (&ledger.Store{}).Schema() }

func endpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: http.MethodPost, Summary: summary, Category: "ledger"}
}

var Endpoints = []module.EndpointDescriptor{
	endpoint("books_create", "/vrooli.money_ledger.v1.ledger.BooksService/CreateBook", "Create an accounting book"), endpoint("books_list", "/vrooli.money_ledger.v1.ledger.BooksService/ListBooks", "List accounting books"), endpoint("books_archive", "/vrooli.money_ledger.v1.ledger.BooksService/ArchiveBook", "Archive a book without deleting its journal"), endpoint("accounts_create", "/vrooli.money_ledger.v1.ledger.BooksService/CreateAccount", "Create an account in a book"), endpoint("accounts_list", "/vrooli.money_ledger.v1.ledger.BooksService/ListAccounts", "List accounts"),
	endpoint("journal_get", "/vrooli.money_ledger.v1.ledger.JournalService/GetPosting", "Read one immutable posting"), endpoint("journal_list", "/vrooli.money_ledger.v1.ledger.JournalService/ListPostings", "List immutable postings"), endpoint("journal_reverse", "/vrooli.money_ledger.v1.ledger.JournalService/ReversePosting", "Create a reversing entry"), endpoint("journal_transfer", "/vrooli.money_ledger.v1.ledger.JournalService/Transfer", "Create paired inter-book postings"),
	endpoint("position_get", "/vrooli.money_ledger.v1.ledger.PositionService/GetPosition", "Compute financial position at read time"), endpoint("statement_get", "/vrooli.money_ledger.v1.ledger.PositionService/GetStatement", "Read a period statement"), endpoint("goals_declare", "/vrooli.money_ledger.v1.ledger.PositionService/DeclareGoal", "Declare a sustained financial goal"), endpoint("goals_list", "/vrooli.money_ledger.v1.ledger.PositionService/ListGoals", "List goal verdicts"), endpoint("goals_archive", "/vrooli.money_ledger.v1.ledger.PositionService/ArchiveGoal", "Archive a goal without deleting its declaration"), endpoint("goals_reparent", "/vrooli.money_ledger.v1.ledger.PositionService/ReparentGoal", "Move a goal while preserving its declaration"),
}
