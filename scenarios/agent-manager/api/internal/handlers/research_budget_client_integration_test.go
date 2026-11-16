package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	adb "agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/researchbudget"
	"agent-manager/internal/supervision"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/eventbus"
	coreidentity "github.com/vrooli/api-core/identity"
	accountsV1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsConnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	cf7RequestOnlyCredential = "cf7-agent-manager-test-credential"
	cf7RequestOnlyToken      = "cf7-agent-manager-test-service-token"
)

// TestResearchBudgetAuthenticatedClientRaisesExactRequestOnlyDecision crosses
// the module internal-package boundary through Treasury's real authenticated
// client. It first proves request-only refusal, then applies an explicit Hub
// resolution and proves the same client can write only the approved run's data.
func TestResearchBudgetAuthenticatedClientRaisesExactRequestOnlyDecision(t *testing.T) {
	db, runID := researchBudgetIngressDB(t)
	for _, statement := range []string{
		`CREATE TABLE standing_grants (id TEXT PRIMARY KEY, subject TEXT NOT NULL, session_id TEXT NOT NULL DEFAULT '', state TEXT NOT NULL, body TEXT NOT NULL, approved_by TEXT NOT NULL DEFAULT '', approved_at TEXT NOT NULL, expires_at TEXT NOT NULL, revoked_at TEXT)`,
		`CREATE TABLE pending_authorization_requests (id TEXT PRIMARY KEY, subject TEXT NOT NULL, state TEXT NOT NULL, body TEXT NOT NULL, raised_at TEXT NOT NULL, expires_at TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	var publisher cf7CapturedApprovalPublisher
	sqlxDB := sqlx.NewDb(db, "sqlite")
	store := adb.NewDB(sqlxDB, logrus.New())
	grants := adb.NewStandingGrantRepository(store)
	owner := orchestration.New(nil, nil, nil,
		orchestration.WithPendingAuthorizations(adb.NewPendingAuthorizationRepository(store)),
		orchestration.WithStandingGrants(grants),
		orchestration.WithApprovalEventPublisher(&publisher),
	)
	raiser := cf7PersistedTestRunRaiser{Orchestrator: owner, db: db}
	grantAuthority := NewStandingGrantAuthority(grants, nil, nil)
	routes := http.NewServeMux()
	connectPath, exchangeHandler := accountsConnect.NewAccountsServiceHandler(&cf7AgentManagerCredentialExchange{})
	routes.Handle(connectPath, exchangeHandler)

	// Agent Manager's real report routes and admission middleware run on a real
	// local HTTP socket. This test-only token verifier accepts only the
	// authenticator fixture's exact service token, then installs its verified
	// Treasury principal for the owner handler.
	muxRouter := newTestRouter()
	RegisterResearchBudgetIngestion(muxRouter, researchbudget.Ledger{DB: db})
	reportHandler := StandingGrantMiddleware(grantAuthority, &raiser)(AgentOperationMiddleware(muxRouter))
	routes.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+cf7RequestOnlyToken {
			http.Error(w, "verified Treasury service identity required", http.StatusUnauthorized)
			return
		}
		principal := coreidentity.Principal{Kind: coreidentity.ActorService, Subject: "treasury-agent-manager", Verified: true, Scopes: []string{"agent-manager:write"}}
		reportHandler.ServeHTTP(w, r.WithContext(coreidentity.WithPrincipal(r.Context(), principal)))
	}))
	server := httptest.NewServer(routes)
	t.Cleanup(server.Close)

	runTreasuryClient := func(approved bool) {
		t.Helper()
		cmdCtx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
		defer cancel()
		_, sourceFile, _, ok := runtime.Caller(0)
		if !ok {
			t.Fatal("resolve test source path")
		}
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../../../.."))
		cmd := exec.CommandContext(cmdCtx, "go", "test", "./internal/researchbudget", "-run", "^TestCF7RequestOnlyBridgeClient$", "-count=1")
		cmd.Dir = filepath.Join(repoRoot, "scenarios/treasury/api")
		cmd.Env = append(os.Environ(),
			"CF7_AM_REQUEST_ONLY_URL="+server.URL,
			"CF7_AM_REQUEST_ONLY_RUN_ID="+runID.String(),
			"CF7_AM_REQUEST_ONLY_APPROVED="+strconv.FormatBool(approved),
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Treasury authenticated-client request failed: %v\n%s", err, output)
		}
	}
	runTreasuryClient(false)

	var pendingRows []string
	if err := sqlxDB.Select(&pendingRows, `SELECT body FROM pending_authorization_requests WHERE subject='treasury-agent-manager' AND state='pending'`); err != nil {
		t.Fatal(err)
	}
	if len(pendingRows) != 1 {
		t.Fatalf("expected one durable pending authorization, got %d", len(pendingRows))
	}
	var pending domain.PendingAuthorizationRequest
	if err := json.Unmarshal([]byte(pendingRows[0]), &pending); err != nil {
		t.Fatal(err)
	}
	wantTarget := "agent-manager:run/" + runID.String()
	if pending.Subject != "treasury-agent-manager" || pending.Operation != researchBudgetOperation || pending.Target != wantTarget || pending.Kind != domain.PendingAuthorizationKindOneTime || pending.State != domain.PendingAuthorizationPending {
		t.Fatalf("pending authorization is not bound to the exact persisted research run: %+v", pending)
	}
	if len(publisher.events) != 1 || publisher.events[0].EventType != orchestration.AuthorizationDecisionEventType {
		t.Fatalf("expected one captured authorization decision and no Hub write, got %+v", publisher.events)
	}
	question, _ := publisher.events[0].Payload["question"].(string)
	big, _ := publisher.events[0].Payload["big_picture"].(string)
	technical, _ := publisher.events[0].Payload["technical_details"].(string)
	if !strings.Contains(question, "one agent run") || !strings.Contains(big, "one-time permission") || !strings.Contains(technical, wantTarget) {
		t.Fatalf("human decision does not explain a one-time permission and exact run target: %s / %s / %s", question, big, technical)
	}
	for table, query := range map[string]string{
		"standing grants":        `SELECT COUNT(*) FROM standing_grants`,
		"budget snapshots":       `SELECT COUNT(*) FROM research_run_budgets WHERE run_id=?`,
		"settlement projections": `SELECT COUNT(*) FROM research_run_spend WHERE run_id=?`,
	} {
		var count int
		var queryErr error
		if strings.Contains(query, "?") {
			queryErr = sqlxDB.Get(&count, query, runID.String())
		} else {
			queryErr = sqlxDB.Get(&count, query)
		}
		if queryErr != nil || count != 0 {
			t.Fatalf("request-only path created %s before approval: count=%d err=%v", table, count, queryErr)
		}
	}

	// The operator's explicit answer is applied through the same canonical Hub
	// resolution effect that issues the exact one-time grant in production.
	handled, err := owner.ResolveAuthorizationFromHub(context.Background(), supervision.DecisionResolution{
		AskID: pending.ID, State: "answered", Answer: "approve", AnsweredBy: "cf7-local-test-operator",
		SourceEventType: orchestration.AuthorizationDecisionEventType, RequesterRef: pending.EventID,
		Correlation: map[string]string{"authorization_request_id": pending.ID},
	})
	if err != nil || !handled {
		t.Fatalf("apply explicit exact-run operator resolution: handled=%t err=%v", handled, err)
	}
	resolved, err := adb.NewPendingAuthorizationRepository(store).Get(context.Background(), pending.ID)
	if err != nil || resolved == nil || resolved.State != domain.PendingAuthorizationApproved || resolved.GrantID == "" || resolved.ResolvedBy != "cf7-local-test-operator" {
		t.Fatalf("operator resolution did not persist the approved exact-run grant: request=%+v err=%v", resolved, err)
	}
	runTreasuryClient(true)
	var grantRows int
	if err := sqlxDB.Get(&grantRows, `SELECT COUNT(*) FROM standing_grants WHERE id=? AND subject=? AND state='approved'`, resolved.GrantID, orchestration.TreasuryServiceSubject); err != nil || grantRows != 1 {
		t.Fatalf("exact Treasury run-report grant was not durable: count=%d err=%v", grantRows, err)
	}
	var snapshotRows, spendRows int
	if err := sqlxDB.Get(&snapshotRows, `SELECT COUNT(*) FROM research_run_budgets WHERE run_id=?`, runID.String()); err != nil || snapshotRows != 1 {
		t.Fatalf("approved Treasury client did not write exactly one run snapshot: count=%d err=%v", snapshotRows, err)
	}
	if err := sqlxDB.Get(&spendRows, `SELECT COUNT(*) FROM research_run_spend WHERE run_id=?`, runID.String()); err != nil || spendRows != 1 {
		t.Fatalf("approved Treasury client did not write exactly one run settlement: count=%d err=%v", spendRows, err)
	}
}

// newTestRouter isolates the Gorilla router used by the registered report
// routes without introducing another service or lifecycle owner.
func newTestRouter() *mux.Router { return mux.NewRouter() }

type cf7CapturedApprovalPublisher struct{ events []eventbus.DomainEvent }

func (p *cf7CapturedApprovalPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.events = append(p.events, event)
	return nil
}

type cf7PersistedTestRunRaiser struct {
	*orchestration.Orchestrator
	db *sql.DB
}

func (r *cf7PersistedTestRunRaiser) IsResearchBudgetRun(ctx context.Context, runID uuid.UUID) bool {
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM runs WHERE id=?`, runID.String()).Scan(&exists)
	return err == nil && exists == 1
}

type cf7AgentManagerCredentialExchange struct {
	accountsConnect.UnimplementedAccountsServiceHandler
}

func (*cf7AgentManagerCredentialExchange) ExchangeServiceCredential(_ context.Context, req *connect.Request[accountsV1.ExchangeServiceCredentialRequest]) (*connect.Response[accountsV1.ExchangeServiceCredentialResponse], error) {
	if req.Msg.GetCredential() != cf7RequestOnlyCredential {
		return nil, connect.NewError(connect.CodeUnauthenticated, sql.ErrNoRows)
	}
	return connect.NewResponse(&accountsV1.ExchangeServiceCredentialResponse{
		AccessToken: cf7RequestOnlyToken, AccessTokenExpiresAt: timestamppb.New(time.Now().Add(time.Hour)),
	}), nil
}
