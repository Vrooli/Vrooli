package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"treasury/internal/approval"
	"treasury/internal/authorization"
	"treasury/internal/baseusdc"
	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/capabilities"
	"treasury/internal/evidence"
	"treasury/internal/identity"
	"treasury/internal/instrument"
	"treasury/internal/ledger"
	"treasury/internal/mandate"
	"treasury/internal/module"
	"treasury/internal/modules"
	"treasury/internal/nativeusdc"
	"treasury/internal/operatorauth"
	"treasury/internal/rail"
	"treasury/internal/rail/card"
	lithicrail "treasury/internal/rail/card/lithic"
	"treasury/internal/rail/manual"
	x402rail "treasury/internal/rail/x402"
	"treasury/internal/receiptfunding"
	treasuryresearchbudget "treasury/internal/researchbudget"
	"treasury/internal/server"
	"treasury/internal/settlement"
	"treasury/internal/wallet"
	"treasury/internal/walletops"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/schedule"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"github.com/vrooli/vrooli/packages/credential-authority-go/eventpublishing"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	_ "modernc.org/sqlite"

	agentspendH "treasury/handlers/agentspend"
	bookH "treasury/handlers/book"
	budgetH "treasury/handlers/budget"
	capsH "treasury/handlers/capabilities"
	fundingH "treasury/handlers/funding"
	healthH "treasury/handlers/health"
	instrumentH "treasury/handlers/instrument"
	ledgeradminH "treasury/handlers/ledgeradmin"
	mandateH "treasury/handlers/mandate"
	treasuryadminH "treasury/handlers/treasuryadmin"
	walletH "treasury/handlers/wallet"
	x402gateH "treasury/handlers/x402gate"
	x402receiptH "treasury/handlers/x402receipt"
	x402resourceH "treasury/handlers/x402resource"
)

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup. File writers must select their class through fileRootPath so a
// test-mode request uses the lease-owned root instead of the live tree.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("treasury")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve treasury storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

type credentialReader interface {
	Resolve(context.Context, string, string) (string, error)
}

func runtimeCredential(ctx context.Context, reader credentialReader, envValue, identityName, field string) (string, error) {
	if value := strings.TrimSpace(envValue); value != "" {
		return value, nil
	}
	if reader == nil {
		return "", fmt.Errorf("credential authority unavailable")
	}
	value, err := reader.Resolve(ctx, identityName, field)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("credential value is empty")
	}
	return value, nil
}

func scenarioURL(ctx context.Context, override, scenario string) string {
	if value := strings.TrimSpace(override); value != "" {
		return value
	}
	value, err := discovery.ResolveScenarioURLDefault(ctx, scenario)
	if err != nil {
		return ""
	}
	return value
}

// fileRootPath is the template's mandatory file-store seam. Domain stores
// compose their relative paths from it rather than retaining startup root
// strings, so X-Vrooli-Test-Mode is honored independently per request.
func fileRootPath(ctx context.Context, roots *filerouting.RoutedRoots, class storage.Class, rel string) (string, error) {
	root, err := roots.Pick(ctx, class)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, rel), nil
}

// openCredentialAuthority is selected at compile time. Only the explicitly tagged
// local-fork rehearsal build substitutes an isolated encrypted test authority.
var openCredentialAuthority = credentialauthority.Default

func main() {
	eventpublishing.Register()
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "treasury"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "treasury",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)

	clock := schedule.System()
	startupCtx := context.Background()
	var credentialClient credentialclient.Client
	authority, authorityErr := openCredentialAuthority()
	if authorityErr == nil {
		credentialClient, authorityErr = credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: authority})
	}
	var identityVerifier identity.Verifier
	identityVerifier, err = identity.NewHTTPVerifier(scenarioURL(startupCtx, os.Getenv("AGENT_MANAGER_API_URL"), "agent-manager"), &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		log.Printf("agent-manager identity verification unavailable; automated spend will fail closed: %v", err)
		identityVerifier = identity.UnavailableVerifier{Cause: err}
	}

	operatorToken, operatorTokenErr := runtimeCredential(startupCtx, credentialClient, os.Getenv("TREASURY_OPERATOR_TOKEN"), "vrooli/treasury", "operator-token")
	if operatorTokenErr != nil {
		operatorToken = strings.TrimSpace(os.Getenv("API_TOKEN"))
	}
	var operatorAuthorizer operatorauth.Authorizer
	operatorAuthorizer, err = operatorauth.NewStaticToken(operatorToken)
	if err != nil {
		log.Printf("operator realm unavailable; TreasuryAdmin will fail closed: %v", err)
		operatorAuthorizer = operatorauth.Unavailable{Cause: err}
	}
	var approvalRelay approval.Relay
	if os.Getenv("TREASURY_CF5_NOTIFICATION_TEST_MODE") == "1" {
		approvalRelay, err = approval.NewLeasedTestNotificationRelay(scenarioURL(startupCtx, os.Getenv("NOTIFICATION_HUB_API_URL"), "notification-hub"), &http.Client{Timeout: 5 * time.Second})
	} else {
		approvalRelay, err = approval.NewNotificationRelay(scenarioURL(startupCtx, os.Getenv("NOTIFICATION_HUB_API_URL"), "notification-hub"), &http.Client{Timeout: 5 * time.Second})
	}
	if err != nil {
		log.Printf("notification relay unavailable; approvals remain local and relay attempts will record failure: %v", err)
		approvalRelay = approval.UnavailableRelay{Cause: err}
	}
	var mandateSigner mandate.Signer
	mandateSigningKey, signingKeyErr := runtimeCredential(startupCtx, credentialClient, os.Getenv("TREASURY_MANDATE_SIGNING_KEY"), "vrooli/treasury", "mandate-signing-key")
	if signingKeyErr == nil {
		mandateSigner, err = mandate.NewHMACSigner([]byte(mandateSigningKey))
	} else {
		err = signingKeyErr
	}
	if err != nil {
		log.Printf("mandate issuance unavailable; TREASURY_MANDATE_SIGNING_KEY is required: %v", err)
		mandateSigner = nil
	}
	x402HTTPClient := &http.Client{Timeout: 30 * time.Second}
	x402Signer, err := x402rail.NewRPCSigner(x402HTTPClient)
	if err != nil {
		log.Fatalf("x402 signer configuration failed: %v", err)
	}
	x402Adapter, err := x402rail.New(x402HTTPClient, x402Signer)
	if err != nil {
		log.Fatalf("x402 rail configuration failed: %v", err)
	}
	cardHTTPClient := &http.Client{Timeout: 30 * time.Second}
	lithicURL := strings.TrimSpace(os.Getenv("LITHIC_SANDBOX_URL"))
	if lithicURL == "" {
		lithicURL = "https://sandbox.lithic.com"
	}
	lithicAdapter, err := lithicrail.New(lithicURL, cardHTTPClient)
	if err != nil {
		log.Fatalf("scoped card rail configuration failed: %v", err)
	}
	railRegistry, err := rail.NewRegistry(manual.New(), x402Adapter, lithicAdapter)
	if err != nil {
		log.Fatalf("rail registry configuration failed: %v", err)
	}
	cardRegistry, err := card.NewRegistry(lithicAdapter)
	if err != nil {
		log.Fatalf("card issuer registry configuration failed: %v", err)
	}
	facilitatorURL := strings.TrimSpace(os.Getenv("X402_FACILITATOR_URL"))
	if facilitatorURL == "" {
		facilitatorURL = "http://127.0.0.1:14020"
	}
	x402Facilitator, err := x402rail.NewHTTPFacilitator(facilitatorURL, x402HTTPClient)
	if err != nil {
		log.Fatalf("x402 facilitator configuration failed: %v", err)
	}
	x402Gate, err := x402rail.NewGate(x402rail.NewSQLiteInboundRepository(db), x402Facilitator)
	if err != nil {
		log.Fatalf("x402 inbound gate configuration failed: %v", err)
	}
	if runtimeMode := strings.TrimSpace(os.Getenv("X402_RUNTIME_MODE")); runtimeMode != "" {
		if err := x402Gate.SetRuntimeMode(runtimeMode); err != nil {
			log.Fatalf("x402 runtime mode configuration failed: %v", err)
		}
	}
	var offerDeskPrices x402rail.OfferDeskPriceReader
	if offerDeskURL := scenarioURL(startupCtx, os.Getenv("OFFER_DESK_BASE_URL"), "offer-desk"); offerDeskURL != "" {
		offerDeskPrices, err = x402rail.NewOfferDeskPriceSource(offerDeskURL, x402HTTPClient)
		if err != nil {
			log.Printf("Offer Desk x402 pricing unavailable; Offer Desk-priced admissions fail closed: %v", err)
			offerDeskPrices = nil
		}
	} else {
		log.Printf("Offer Desk x402 pricing unavailable; Offer Desk-priced admissions fail closed")
	}
	var credentialResolver instrument.CredentialResolver
	if authorityErr == nil {
		credentialResolver, authorityErr = instrument.NewCredentialClientResolver(credentialClient)
	}
	if authorityErr != nil {
		log.Printf("instrument credential resolution unavailable; instrument use will fail closed: %v", authorityErr)
		credentialResolver = instrument.UnavailableResolver{Cause: authorityErr}
	}
	moneyLedgerURL := scenarioURL(startupCtx, os.Getenv("MONEY_LEDGER_API_URL"), "money-ledger")
	authenticatorURL := scenarioURL(startupCtx, os.Getenv("SCENARIO_AUTHENTICATOR_API_URL"), "scenario-authenticator")
	// Treasury writes to Money Ledger as its own money-ledger principal, which
	// Money Ledger requires for every journal write.
	serviceTokens, tokenErr := ledger.NewServiceTokenSource(credentialClient, authenticatorURL, nil)
	// The settlement destination is the operator's choice, recorded in Treasury
	// (treasury ledger destination-set). The environment pair is only a seed for
	// rehearsals and never overrides a stored choice.
	destinationStore := ledger.NewDestinationStore(db)
	destination := ledger.SeededDestination{Store: destinationStore, SeedBook: os.Getenv("TREASURY_LEDGER_BOOK_ID"), SeedAccount: os.Getenv("TREASURY_LEDGER_ACCOUNT_ID")}
	var ledgerEmitter ledger.Emitter
	if tokenErr != nil {
		err = tokenErr
	} else {
		ledgerEmitter, err = ledger.NewMoneyLedgerEmitter(moneyLedgerURL, destination, serviceTokens, &http.Client{Timeout: 5 * time.Second})
	}
	if err != nil {
		log.Printf("money-ledger emission unavailable; settlements will remain durably queued: %v", err)
		ledgerEmitter = ledger.UnavailableEmitter{Cause: err}
	}
	var destinationValidator ledger.DestinationValidator
	if validator, validatorErr := ledger.NewMoneyLedgerValidator(moneyLedgerURL, nil); validatorErr != nil {
		log.Printf("settlement destination cannot be checked against Money Ledger: %v", validatorErr)
	} else {
		destinationValidator = validator
	}
	if _, destinationErr := destination.Destination(startupCtx); destinationErr != nil {
		log.Printf("%v; settlements stay queued until it is chosen", destinationErr)
	}
	ledgerService := ledger.NewService(ledger.NewSQLiteRepository(db), ledgerEmitter, time.Now)
	ledgerCtx, stopLedger := context.WithCancel(context.Background())
	go ledgerService.Run(ledgerCtx, 5*time.Second, log.Default())

	var walletService *wallet.Service
	if authorityErr == nil && authority != nil {
		walletService = wallet.NewService(wallet.NewSQLiteRepository(db), book.NewService(book.NewSQLiteRepository(db), clock), authority, clock, func(ctx context.Context, rel string) (string, error) {
			return fileRootPath(ctx, fileRoots, storage.ClassData, rel)
		})
	}
	var nativeUSDCAdapter *nativeusdc.Adapter
	if walletService != nil && strings.TrimSpace(os.Getenv("TREASURY_BASE_USDC_MODE")) == "local-fork" {
		nativeUSDCAdapter, err = nativeusdc.NewWithClock(db, walletService, strings.TrimSpace(os.Getenv("TREASURY_BASE_RPC_URL")), "local-fork", clock)
		if err != nil {
			log.Printf("local-fork native USDC rail unavailable: %v", err)
			nativeUSDCAdapter = nil
		} else {
			adapters := append(railRegistry.Adapters(), nativeUSDCAdapter)
			railRegistry, err = rail.NewRegistry(adapters...)
			if err != nil {
				log.Printf("local-fork native USDC rail registration failed: %v", err)
				nativeUSDCAdapter = nil
			}
		}
	}
	baseRPCURL := strings.TrimSpace(os.Getenv("TREASURY_BASE_RPC_URL"))
	baseProviderID := "base-mainnet-public-rpc"
	if baseRPCURL == "" {
		baseRPCURL = baseusdc.DefaultRPCURL
	} else {
		baseProviderID = "operator-configured-base-rpc"
	}
	var baseReceiptVerifier *baseusdc.Verifier
	baseRPC, baseRPCErr := baseusdc.NewHTTPRPC(baseRPCURL, nil)
	if baseRPCErr == nil {
		baseReceiptVerifier, baseRPCErr = baseusdc.NewVerifier(baseRPC, baseProviderID, clock.Now)
	}
	if baseRPCErr != nil {
		log.Printf("Base native-USDC receipt checks unavailable: operator RPC configuration is invalid")
	}
	var baseReceiptService *baseusdc.Service
	adminServices := make([]any, 0, 2)
	if baseReceiptVerifier != nil {
		baseReceiptService = baseusdc.NewService(baseusdc.NewSQLiteStore(db), baseReceiptVerifier, baseProviderID, clock.Now)
	}
	var receiptAcceptor receiptfunding.Acceptor = receiptfunding.UnavailableAcceptor{}
	var creditLotGranter receiptfunding.CreditLotGranter
	noticeTokens, noticeTokenErr := ledger.NewServiceTokenSourceFor(credentialClient, authenticatorURL, nil, ledger.NoticePrincipal, ledger.NoticeResource)
	if noticeTokenErr != nil {
		log.Printf("funding notification service identity unavailable: %v", noticeTokenErr)
	}
	var researchBudgetClient *treasuryresearchbudget.Client
	if credentialClient != nil {
		researchBudgetClient, err = treasuryresearchbudget.NewAuthenticatedClient(
			scenarioURL(startupCtx, os.Getenv("AGENT_MANAGER_API_URL"), "agent-manager"),
			&http.Client{Timeout: 5 * time.Second}, credentialClient, authenticatorURL,
		)
		if err != nil {
			log.Printf("run-bound Agent Manager budget reporting unavailable: %v", err)
			researchBudgetClient = nil
		}
	}
	if tokenErr == nil {
		if client, err := receiptfunding.NewMoneyLedgerClient(moneyLedgerURL, serviceTokens, nil); err == nil {
			receiptAcceptor = client
		}
	}
	if credentialClient != nil {
		if client, err := receiptfunding.NewLPBSCreditLotClient(scenarioURL(startupCtx, os.Getenv("LANDING_PAGE_BUSINESS_SUITE_API_URL"), "landing-page-business-suite"), credentialClient, nil); err == nil {
			creditLotGranter = client
		}
	}
	fundingService := receiptfunding.NewService(receiptfunding.NewStore(db), baseusdc.NewSQLiteStore(db), walletService, budget.NewSQLiteRepository(db), receiptAcceptor, clock.Now)
	fundingService.SetCreditLotGranter(creditLotGranter)
	bookService := book.NewService(book.NewSQLiteRepository(db), clock)
	authorizationRepository := authorization.NewSQLiteRepository(db)
	budgetService := budget.NewService(budget.NewSQLiteRepository(db), clock, authorizationRepository)
	mandateService := mandate.NewService(mandate.NewSQLiteRepository(db), clock, mandateSigner)
	evidenceRecorder := evidence.NewRecorder(evidence.NewSQLiteRecorder(db))
	approvalService := approval.NewService(approval.NewSQLiteRepository(db), authorization.NewApprovalController(authorizationRepository, mandateService, budgetService, clock), approvalRelay, clock, evidenceRecorder)
	var approvalDecisionFlow *approval.DecisionFlow
	var approvalDecisionModule module.Module
	if os.Getenv("TREASURY_CF5_NOTIFICATION_TEST_MODE") != "1" {
		approvalDecisionFlow = approval.NewDecisionFlow(approval.NewSQLiteDecisionAskStore(db), eventbus.NewDiscoveredClient(startupCtx), clock).WithApprovalService(approvalService).
			WithLimits(func(ctx context.Context, request approval.Request) (approval.Limits, error) {
				record, err := authorizationRepository.Get(ctx, request.AuthorizationID)
				if err != nil {
					return approval.Limits{}, err
				}
				headroom, err := budgetService.Headroom(ctx, record.BudgetID)
				return approval.Limits{HeldBecause: record.Remediation, Headroom: headroom}, err
			})
		approvalService.WithDecisionAsker(approvalDecisionFlow)
		webhookSecret, secretErr := approval.ResolveEventsWebhookSecret()
		if secretErr != nil {
			log.Printf("Treasury approval decision callback unavailable: %v", secretErr)
		}
		approvalDecisionModule = approval.DecisionWebhookModule(webhookSecret, approval.NewDecisionAskReader(func(ctx context.Context) (string, error) {
			if credentialClient == nil {
				return "", fmt.Errorf("Treasury ask-reader credential authority is unavailable")
			}
			identity, err := credentialauthority.ParseIdentity(askreader.CredentialIdentity(approval.DecisionSource))
			if err != nil {
				return "", fmt.Errorf("parse Treasury ask-reader credential identity: %w", err)
			}
			return credentialClient.Resolve(ctx, string(identity), askreader.CredentialField)
		}), approvalDecisionFlow)
	}
	walletInstrumentService := instrument.NewServiceWithCardIssuers(instrument.NewSQLiteRepository(db), mandateService, railRegistry, credentialResolver, clock, cardRegistry)
	x402Gate.SetRecoveryVault(walletInstrumentService)
	walletAuthorizationService := authorization.NewService(authorizationRepository, identityVerifier, mandateService, budgetService, evidenceRecorder, clock, approvalService)
	walletSettlementService := settlement.NewService(settlement.NewSQLiteRepository(db), authorizationRepository, walletInstrumentService, railRegistry, identityVerifier, clock, budgetService)
	var credentialWriter instrument.CredentialWriter
	if writer, ok := credentialResolver.(instrument.CredentialWriter); ok {
		credentialWriter = writer
	}
	walletOperations := walletops.New(walletops.Config{
		DB: db, Wallets: walletService, Books: bookService, Budgets: budgetService,
		Receipts: baseReceiptService, Funding: fundingService, Native: nativeUSDCAdapter,
		Mandates: mandateService, Authorizations: walletAuthorizationService, Approvals: approvalService,
		Instruments: walletInstrumentService, Credentials: credentialWriter, Settlements: walletSettlementService,
		Clock: clock,
	})
	if walletService != nil {
		walletService.SetSessionSignAuthorizer(walletOperations)
	}
	// Both RPC surfaces share these policy services and their in-process
	// serialization, not merely the same SQLite rows.
	adminServices = append(adminServices, budgetService, approvalService, authorizationRepository, walletAuthorizationService, walletSettlementService)
	notificationHubURL := scenarioURL(startupCtx, os.Getenv("NOTIFICATION_HUB_API_URL"), "notification-hub")
	if notificationHubURL != "" {
		var notifier *walletops.NotificationHub
		var notifyErr error
		if os.Getenv("TREASURY_CF5_NOTIFICATION_TEST_MODE") == "1" {
			notifier, notifyErr = walletops.NewLeasedTestNotificationHub(notificationHubURL, os.Getenv("TREASURY_CF5_NOTIFICATION_TEST_AGENT_IDENTITY_TOKEN"))
		} else {
			notifier, notifyErr = walletops.NewAuthenticatedNotificationHub(notificationHubURL, &http.Client{Timeout: 5 * time.Second}, noticeTokens)
		}
		if notifyErr == nil {
			walletOperations.Notifications = notifier
		} else {
			log.Printf("funding notifications unavailable; durable receipt credit continues: %v", notifyErr)
		}
	}
	go fundingService.Run(ledgerCtx, 2*time.Second, log.Default())
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		lastDecisionSubscriptionAttempt := time.Time{}
		for {
			bounded, cancel := context.WithTimeout(ledgerCtx, 10*time.Second)
			if approvalDecisionFlow != nil {
				if err := approvalDecisionFlow.RetryQueued(bounded, approvalService); err != nil {
					log.Printf("Treasury approval decision retry deferred: %v", err)
				}
				if port := strings.TrimSpace(os.Getenv("API_PORT")); port != "" &&
					(lastDecisionSubscriptionAttempt.IsZero() || time.Since(lastDecisionSubscriptionAttempt) >= 30*time.Second) {
					lastDecisionSubscriptionAttempt = time.Now()
					target := "http://127.0.0.1:" + port + approval.DecisionWebhookPath
					if err := approval.ReconcileDecisionEventSubscription(bounded, os.Getenv("VROOLI_EVENTS_API_BASE"), target); err != nil {
						log.Printf("Treasury approval answer subscription reconciliation deferred: %v", err)
					}
				}
			}
			if err := walletOperations.WatchFundingIntents(bounded); err != nil {
				log.Printf("funding receipt watcher deferred: %v", err)
			}
			if err := walletOperations.RetryNotifications(bounded); err != nil {
				log.Printf("funding notification retry incomplete: %v", err)
			}
			cancel()
			select {
			case <-ledgerCtx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	var walletAPI walletH.Wallets
	var portableRecovery walletH.PortableRecovery
	var walletReceiptReader walletH.ReceiptChecker
	if walletService != nil {
		walletAPI = walletService
		portableRecovery = walletService
	}
	if baseReceiptService != nil {
		walletReceiptReader = baseReceiptService
	}
	srv := server.New(
		server.Deps{Clock: clock, Logger: log.Default()},
		healthH.Module(db, "treasury-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		agentspendH.Module(db, identityVerifier, clock, approvalRelay, railRegistry, cardRegistry, credentialResolver, agentspendH.HandlerOverrides{
			WalletOps: walletOperations, Authorization: walletAuthorizationService,
			Settlements: walletSettlementService, ResearchBudget: researchBudgetClient,
		}),
		budgetH.Module(operatorAuthorizer, budgetService),
		bookH.Module(operatorAuthorizer, bookService),
		fundingH.Module(operatorAuthorizer, walletOperations, walletOperations, fundingService, identityVerifier),
		instrumentH.Module(operatorAuthorizer, walletInstrumentService),
		mandateH.Module(operatorAuthorizer, mandateService, walletOperations),
		walletH.Module(operatorAuthorizer, walletAPI, portableRecovery, walletReceiptReader),
		treasuryadminH.Module(db, operatorAuthorizer, clock, approvalRelay, railRegistry, cardRegistry, credentialResolver, mandateSigner, adminServices...),
		ledgeradminH.Module(db, operatorAuthorizer, ledgeradminH.Destinations{Current: destination, Store: destinationStore, Validator: destinationValidator}),
		x402gateH.ModuleWithOfferDesk(x402Gate, operatorAuthorizer, offerDeskPrices),
		x402resourceH.Module(x402Gate),
		x402receiptH.Module(ledger.NewSQLiteRepository(db)),
		approvalDecisionModule,
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
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error {
			stopLedger()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
