package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	internalaccess "token-economy/internal/access"
	"token-economy/internal/catalog"
	"token-economy/internal/earning"
	"token-economy/internal/grants"
	"token-economy/internal/holders"
	"token-economy/internal/journal"
	"token-economy/internal/mints"
	"token-economy/internal/modules"
	"token-economy/internal/redemption"
	"token-economy/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/provenance"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	accessH "token-economy/handlers/access"
	catalogH "token-economy/handlers/catalog"
	earningH "token-economy/handlers/earning"
	grantsH "token-economy/handlers/grants"
	healthH "token-economy/handlers/health"
	holdersH "token-economy/handlers/holders"
	journalH "token-economy/handlers/journal"
	mintsH "token-economy/handlers/mints"
	redemptionH "token-economy/handlers/redemption"
)

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup so devrouting can install lease-owned test roots independently of
// the live tree.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("token-economy")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve token-economy storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "token-economy"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "token-economy",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	if err := journal.EnsureSchema(context.Background(), db.Primary()); err != nil {
		log.Fatalf("journal schema migration failed: %v", err)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)
	clock := schedule.System()
	mintRepository := mints.NewSQLiteRepository(db)
	mintService := mints.NewService(mintRepository, clock)
	catalogRepository := catalog.NewSQLiteRepository(db)
	catalogService := catalog.NewService(catalogRepository, catalog.TokenTypeReaderFunc(func(ctx context.Context, id string) (catalog.TokenTypeState, error) {
		tokenType, getErr := mintRepository.Get(ctx, id)
		switch {
		case errors.Is(getErr, mints.ErrTokenTypeNotFound):
			return catalog.TokenTypeState{}, catalog.ErrTokenTypeNotFound
		case getErr != nil:
			return catalog.TokenTypeState{}, getErr
		default:
			return catalog.TokenTypeState{ID: tokenType.ID, Retired: tokenType.Retired}, nil
		}
	}), clock)
	grantRepository := grants.NewSQLiteRepository(db, func(ctx context.Context, tx *sql.Tx, credit grants.Credit) error {
		_, appendErr := journal.NewTransactionalAppender(tx).Append(ctx, journal.Event{
			ID: credit.ID, TokenTypeID: credit.TokenTypeID, HolderID: credit.HolderID,
			Amount: credit.Amount, Kind: journal.EventKindCredit,
			CauseReference: credit.CauseReference, ActorIdentity: credit.ActorIdentity,
			CreatedAt: credit.CreatedAt,
		})
		return appendErr
	})
	journalRepository := journal.NewSQLiteRepository(db)
	journalService := journal.NewService(journalRepository, clock)
	grantService := grants.NewService(grantRepository, grants.TokenTypeReaderFunc(func(ctx context.Context, id string) (grants.TokenTypeState, error) {
		tokenType, getErr := mintRepository.Get(ctx, id)
		switch {
		case errors.Is(getErr, mints.ErrTokenTypeNotFound):
			return grants.TokenTypeState{}, grants.ErrTokenTypeNotFound
		case getErr != nil:
			return grants.TokenTypeState{}, getErr
		default:
			return grants.TokenTypeState{ID: tokenType.ID, Retired: tokenType.Retired}, nil
		}
	}), grants.NewRuleEvaluator(), clock)
	earningService := earning.NewService(
		earning.NewSQLiteRepository(db),
		earningGrantAdapter{service: grantService},
		clock,
	)
	holderRepository := holders.NewSQLiteRepository(db)
	holderHistory := journal.NewHolderHistoryRepository(journalRepository, holderRepository)
	holderService := holders.NewService(holderRepository, holderHistoryAdapter{history: holderHistory})
	redemptionRepository := redemption.NewSQLiteRepository(
		db,
		func(ctx context.Context, tx *sql.Tx, id string, at time.Time) (redemption.CatalogEntry, error) {
			entry, reserveErr := catalog.ReserveInventory(ctx, tx, id, at)
			if reserveErr != nil {
				return redemption.CatalogEntry{}, reserveErr
			}
			return redemption.CatalogEntry{ID: entry.ID, TokenTypeID: entry.TokenTypeID, CostAmount: entry.CostAmount, ApprovalPosture: redemption.ApprovalPosture(entry.ApprovalPosture)}, nil
		},
		catalog.ReleaseInventory,
		func(ctx context.Context, tx *sql.Tx, holderID, tokenTypeID string) (int64, error) {
			balance, balanceErr := journal.BalanceInTransaction(ctx, tx, holderID, tokenTypeID)
			return balance.Amount, balanceErr
		},
		func(ctx context.Context, tx *sql.Tx, debit redemption.Debit) error {
			_, appendErr := journal.NewTransactionalAppender(tx).Append(ctx, journal.Event{
				ID: debit.ID, TokenTypeID: debit.TokenTypeID, HolderID: debit.HolderID,
				Amount: debit.Amount, Kind: journal.EventKindDebit,
				CauseReference: debit.CauseReference, ActorIdentity: debit.ActorIdentity,
				CreatedAt: debit.CreatedAt,
			})
			return appendErr
		},
	)
	redemptionService := redemption.NewService(
		redemptionRepository,
		redemption.HolderReaderFunc(func(ctx context.Context, subject string) (redemption.Holder, error) {
			holder, holderErr := holderRepository.GetBySubject(ctx, subject)
			return redemption.Holder{ID: holder.ID}, holderErr
		}),
		redemption.CatalogReaderFunc(func(ctx context.Context, id string) (redemption.CatalogEntry, error) {
			entry, entryErr := catalogService.RequireAvailable(ctx, id)
			return redemption.CatalogEntry{ID: entry.ID, TokenTypeID: entry.TokenTypeID, CostAmount: entry.CostAmount, ApprovalPosture: redemption.ApprovalPosture(entry.ApprovalPosture)}, entryErr
		}),
		redemption.GrantEvaluatorFunc(func(ctx context.Context, grantID, scope string, evidence []string, available, requested int64, now time.Time) (redemption.GrantEvaluation, error) {
			decision, decisionErr := grantService.EvaluateRedemption(ctx, grantID, grants.EvaluationRequest{CatalogScope: scope, Evidence: evidence, AvailableBalance: available, RequestedAmount: requested, Now: now})
			return redemption.GrantEvaluation{Allowed: decision.Allowed, Reason: decision.Reason}, decisionErr
		}),
		redemption.NewNotificationRelay(
			discovery.NewResolver(discovery.ResolverConfig{}),
			&http.Client{Timeout: 2 * time.Second},
		),
		clock,
	)

	srv := server.New(
		server.Deps{Clock: clock, Logger: log.Default()},
		healthH.Module(db, "token-economy-api", "1.0.0"),
		accessH.Module(
			mintsH.NewConnectHandler(mintService, log.Default()),
			grantsH.NewConnectHandler(grantService, log.Default()),
			holdersH.NewConnectHandler(holderService, log.Default()),
			earningH.NewConnectHandler(earningService, log.Default()),
			catalogH.NewConnectHandler(catalogService, log.Default()),
			redemptionH.NewConnectHandler(redemptionService, log.Default()),
			journalH.NewConnectHandler(journalService, log.Default()),
			internalaccess.NewJWKSValidator(discovery.NewResolver(discovery.ResolverConfig{}), nil),
		),
		journalH.Module(),
		redemptionH.Module(),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(provenance.Middleware(provenance.CLIUtilVerifier{})(rootMux))

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

type earningGrantAdapter struct{ service grants.Service }

func (a earningGrantAdapter) Issue(ctx context.Context, request earning.GrantRequest) (earning.GrantOutcome, error) {
	grant, err := a.service.Create(ctx, grants.CreateInput{
		TokenTypeID: request.TokenTypeID, GrantSourceID: request.GrantSourceID,
		Authorizer: request.Authorizer, HolderID: request.HolderID, AmountMinor: request.AmountMinor,
		ExpiresAt: request.ExpiresAt, IdempotencyKey: request.IdempotencyKey,
		ActorIdentity: request.ActorIdentity,
	})
	if err != nil {
		return earning.GrantOutcome{}, err
	}
	return earning.GrantOutcome{ID: grant.ID}, nil
}

type holderHistoryAdapter struct {
	history *journal.HolderHistoryRepository
}

func (a holderHistoryAdapter) Read(ctx context.Context, holderID, authenticatedSubject string) (holders.History, error) {
	history, err := a.history.Read(ctx, holderID, authenticatedSubject)
	if err != nil {
		return holders.History{}, err
	}
	out := holders.History{
		Events:   make([]holders.HistoryEvent, 0, len(history.Events)),
		Balances: make([]holders.Balance, 0, len(history.Balances)),
	}
	for _, event := range history.Events {
		out.Events = append(out.Events, holders.HistoryEvent{
			ID: event.ID, TokenTypeID: event.TokenTypeID, Amount: event.Amount,
			Kind: string(event.Kind), Reason: event.Reason, CauseReference: event.CauseReference,
			ActorIdentity: event.ActorIdentity, ActorKind: event.ActorKind,
			ActorVerificationStatus: event.ActorVerificationStatus, ActorRunID: event.ActorRunID,
			CreatedAt: event.CreatedAt,
		})
	}
	for _, balance := range history.Balances {
		out.Balances = append(out.Balances, holders.Balance{TokenTypeID: balance.TokenTypeID, Amount: balance.Amount})
	}
	return out, nil
}
