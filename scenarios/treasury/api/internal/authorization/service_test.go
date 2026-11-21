package authorization_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"treasury/internal/testutil/agentclaims"

	"github.com/stretchr/testify/require"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/cli-core/cliutil"

	"treasury/internal/authorization"
	"treasury/internal/budget"
	"treasury/internal/identity"
	"treasury/internal/mandate"
	mandateflow "treasury/internal/mandate/flow"
)

type verifier struct {
	claims identity.Claims
	err    error
}

func (v verifier) Verify(context.Context, string) (identity.Claims, error) { return v.claims, v.err }

type pairedVerifier struct {
	baseline       identity.Claims
	operation      identity.Claims
	baselineToken  string
	operationToken string
	pairCalls      int
}

func (v *pairedVerifier) Verify(_ context.Context, token string) (identity.Claims, error) {
	if token != v.baselineToken {
		return identity.Claims{}, errors.New("unexpected baseline token")
	}
	return v.baseline, nil
}

func (v *pairedVerifier) VerifyPair(_ context.Context, baseline, operation string) (identity.Claims, error) {
	v.pairCalls++
	if baseline != v.baselineToken || operation != v.operationToken {
		return identity.Claims{}, errors.New("unexpected identity pair")
	}
	return v.operation, nil
}

type mandates struct {
	value mandate.Mandate
	err   error
}

func (m mandates) RequireLive(context.Context, string) (mandate.Mandate, error) {
	return m.value, m.err
}

type budgets struct {
	value budget.Budget
	err   error
}

func (b budgets) Get(context.Context, string) (budget.Budget, error) { return b.value, b.err }

type frozenBudgets struct {
	budgets
	scope budget.FreezeScope
}

func (b frozenBudgets) IsFrozen(context.Context, string, string) (bool, budget.FreezeScope, error) {
	return true, b.scope, nil
}

type evidence struct {
	mu      sync.Mutex
	records []authorization.DecisionEvidence
}

type approvalQueue struct {
	mu         sync.Mutex
	admissions []authorization.ApprovalAdmission
}

func (q *approvalQueue) Admit(_ context.Context, value authorization.ApprovalAdmission) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.admissions = append(q.admissions, value)
	return nil
}

func (e *evidence) RecordDecision(_ context.Context, value authorization.DecisionEvidence) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.records = append(e.records, value)
	return nil
}

type repository struct {
	mu      sync.Mutex
	records map[string]authorization.Record
	keys    map[string]string
}

func newRepository() *repository {
	return &repository{records: map[string]authorization.Record{}, keys: map[string]string{}}
}

func (r *repository) Create(_ context.Context, value authorization.Record) (authorization.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[value.ID] = value
	r.keys[value.IdempotencyKey] = value.ID
	return value, nil
}

func (r *repository) Get(_ context.Context, id string) (authorization.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.records[id]
	if !ok {
		return authorization.Record{}, authorization.ErrNotFound
	}
	return value, nil
}

func (r *repository) GetByIdempotencyKey(_ context.Context, key string) (authorization.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.keys[key]
	if !ok {
		return authorization.Record{}, authorization.ErrNotFound
	}
	return r.records[id], nil
}

func (r *repository) Usage(_ context.Context, budgetID, mandateID string, periodStart, now time.Time) (authorization.Usage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var usage authorization.Usage
	for _, value := range r.records {
		active := value.Verdict == authorization.VerdictSettled || (value.Verdict == authorization.VerdictPending || value.Verdict == authorization.VerdictApproved) && value.ExpiresAt.After(now)
		if !active || value.BudgetID != budgetID {
			continue
		}
		usage.BudgetTotalMinor += value.AmountMinor
		if !value.CreatedAt.Before(periodStart) {
			usage.BudgetPeriodMinor += value.AmountMinor
		}
		if value.MandateID == mandateID {
			usage.MandateTotalMinor += value.AmountMinor
		}
	}
	return usage, nil
}

func (r *repository) Release(_ context.Context, id string) (authorization.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.records[id]
	if !ok {
		return authorization.Record{}, authorization.ErrNotFound
	}
	value.Verdict = authorization.VerdictReleased
	value.HoldMinor = 0
	r.records[id] = value
	return value, nil
}

func (r *repository) Approve(_ context.Context, id string) (authorization.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.records[id]
	if !ok {
		return authorization.Record{}, authorization.ErrNotFound
	}
	value.Verdict = authorization.VerdictApproved
	r.records[id] = value
	return value, nil
}

func (r *repository) Settle(_ context.Context, id string) (authorization.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.records[id]
	if !ok || (value.Verdict != authorization.VerdictApproved && value.Verdict != authorization.VerdictSettled) {
		return authorization.Record{}, authorization.ErrNotFound
	}
	value.Verdict = authorization.VerdictSettled
	value.HoldMinor = 0
	r.records[id] = value
	return value, nil
}

func fixture(now time.Time, repo *repository, verification verifier, policy budget.Budget, grant mandate.Mandate, recorder *evidence) *authorization.Service {
	return authorization.NewService(repo, verification, mandates{value: grant}, budgets{value: policy}, recorder, schedule.NewFake(now), &approvalQueue{})
}

func validPolicy() budget.Budget {
	return budget.Budget{ID: "budget-1", BookID: "book-1", Currency: "USD", TotalCapMinor: 100, PeriodicCapMinor: 100, PerTransactionCapMinor: 100, Period: time.Hour, AllowedCounterparties: []string{"api.example"}, RequiresApproval: true}
}

func validGrant(now time.Time) mandate.Mandate {
	return mandate.Mandate{AgentSubject: "operator:1", AgentRunID: "test-run", ID: "mandate-1", BookID: "book-1", BudgetID: "budget-1", CapMinor: 100, Currency: "USD", AllowedCounterparties: []string{"api.example"}, ExpiresAt: now.Add(time.Hour), Status: mandateflow.MandateLive}
}

func validInput(id string, amount int64) authorization.ProposeInput {
	return authorization.ProposeInput{
		ID: id, IdempotencyKey: "key-" + id, MandateID: "mandate-1", IdentityToken: "opaque",
		AmountMinor: amount, Currency: "USD", Counterparty: "api.example",
		SettlementID: "settlement-" + id, InstrumentID: "instrument-1", SettlementIdempotencyKey: "settlement-key-" + id,
	}
}

// [REQ:TRS-P0-002] [REQ:TRS-P0-005] A verified identity alone is not spending authority.
func TestAgentMustHoldCoveredOperationAndExactMandateDelegate(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name       string
		edit       func(*identity.Claims, *mandate.Mandate)
		constraint string
	}{
		{"missing grant", func(c *identity.Claims, _ *mandate.Mandate) { c.Meta = nil }, "operation_grant"},
		{"missing capability", func(c *identity.Claims, _ *mandate.Mandate) { c.Scopes = nil }, "operation_grant"},
		{"expired identity", func(c *identity.Claims, _ *mandate.Mandate) { c.ExpiresAt = now.Add(-time.Second).Unix() }, "operation_grant"},
		{"different agent", func(_ *identity.Claims, m *mandate.Mandate) { m.AgentSubject = "agent:other" }, "mandate_delegate"},
		{"different run", func(_ *identity.Claims, m *mandate.Mandate) { m.AgentRunID = "other-run" }, "mandate_delegate"},
		{"operator-only mandate", func(_ *identity.Claims, m *mandate.Mandate) { m.AgentSubject = ""; m.AgentRunID = "" }, "mandate_delegate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims, grant := agentclaims.For("operator:1"), validGrant(now)
			tc.edit(&claims, &grant)
			repo := newRepository()
			service := fixture(now, repo, verifier{claims: claims}, validPolicy(), grant, &evidence{})
			result, err := service.Propose(context.Background(), validInput("authority-guard", 10))
			require.NoError(t, err)
			require.Equal(t, authorization.VerdictRefused, result.Verdict)
			require.Equal(t, tc.constraint, result.ViolatedConstraint)
			require.Empty(t, repo.records)
		})
	}
}

// [REQ:TRS-P0-002] A normal Agent Manager run identity and its separate
// approved operation grant must be reverified together at spend admission.
func TestSpendAuthorizationUsesPairedOperationGrant(t *testing.T) {
	now := time.Now().UTC()
	baseline := identity.Claims{RunID: "test-run", Subject: "operator:1", Scopes: []string{"treasury:write"}, ExpiresAt: now.Add(time.Hour).Unix()}
	operation := agentclaims.For("operator:1")
	operation.RunID = baseline.RunID
	operation.ExpiresAt = baseline.ExpiresAt
	verification := &pairedVerifier{baseline: baseline, operation: operation, baselineToken: "run-token", operationToken: "approved-operation-token"}
	repo := newRepository()
	policy := validPolicy()
	policy.RequiresApproval = false
	service := authorization.NewService(repo, verification, mandates{value: validGrant(now)}, budgets{value: policy}, &evidence{}, schedule.NewFake(now), &approvalQueue{})
	in := validInput("paired-operation", 10)
	in.IdentityToken = verification.baselineToken
	in.OperationToken = verification.operationToken
	result, err := service.Propose(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictApproved, result.Verdict)
	require.Equal(t, 1, verification.pairCalls)
	require.Equal(t, 1, len(repo.records))
}

func TestSpendAuthorizationRejectsCrossIntentAndPreservesExactRetry(t *testing.T) {
	now := time.Now().UTC()
	baseline := identity.Claims{RunID: "test-run", Subject: "operator:1", Scopes: []string{"treasury:write"}, ExpiresAt: now.Add(time.Hour).Unix()}
	in := validInput("intent-a", 10)
	in.IdempotencyKey = "intent-a-idem"
	grant := requestidentity.OperationGrant{ID: "grant-a", Approver: "operator", ExpiresAt: now.Add(time.Hour), Rules: []requestidentity.OperationRule{
		{Operations: []string{"treasury.spend"}, Targets: []string{cliutil.TreasurySpendIntentAuthorizationTarget(cliutil.TreasurySpendIntent{
			AuthorizationID: in.ID, MandateID: in.MandateID, AmountMinor: in.AmountMinor, Currency: in.Currency,
			Counterparty: in.Counterparty, AuthorizationIdempotencyKey: in.IdempotencyKey,
			SettlementID: in.SettlementID, InstrumentID: in.InstrumentID, SettlementIdempotencyKey: in.SettlementIdempotencyKey,
		})}},
	}}
	grantJSON, err := json.Marshal(grant)
	require.NoError(t, err)
	operation := agentclaims.For("operator:1")
	operation.RunID = baseline.RunID
	operation.ExpiresAt = baseline.ExpiresAt
	operation.Meta["operation_grant"] = string(grantJSON)
	verification := &pairedVerifier{baseline: baseline, operation: operation, baselineToken: "run-token", operationToken: "approved-operation-token"}
	repo := newRepository()
	policy := validPolicy()
	policy.RequiresApproval = false
	service := authorization.NewService(repo, verification, mandates{value: validGrant(now)}, budgets{value: policy}, &evidence{}, schedule.NewFake(now), &approvalQueue{})
	in.IdentityToken, in.OperationToken = verification.baselineToken, verification.operationToken
	first, err := service.Propose(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictApproved, first.Verdict)

	// The same token and same immutable intent support an idempotent retry.
	retry, err := service.Propose(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, first, retry)
	require.Len(t, repo.records, 1)

	// A grant for intent A cannot authorize a different authorization, amount,
	// or beneficiary. The denial is evidenced and creates no second hold.
	cross := in
	cross.ID, cross.IdempotencyKey = "intent-b", "intent-b-idem"
	cross.AmountMinor = 11
	cross.Counterparty = "other.example"
	denied, err := service.Propose(context.Background(), cross)
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictRefused, denied.Verdict)
	require.Equal(t, "operation_grant", denied.ViolatedConstraint)
	require.Len(t, repo.records, 1)
}

// [REQ:TRS-P0-002] Supplying a grant token without a pair verifier fails
// closed; the baseline is never silently treated as the approved grant.
func TestSpendAuthorizationFailsClosedWhenPairVerifierUnavailable(t *testing.T) {
	now := time.Now().UTC()
	claims := agentclaims.For("operator:1")
	repo := newRepository()
	service := fixture(now, repo, verifier{claims: claims}, validPolicy(), validGrant(now), &evidence{})
	in := validInput("unverifiable-pair", 10)
	in.OperationToken = "operation-token"
	result, err := service.Propose(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictRefused, result.Verdict)
	require.Equal(t, "identity", result.ViolatedConstraint)
	require.Empty(t, repo.records)
}

// [REQ:TRS-P0-006] Above the automatic cap, one exact held charge requires a human decision.
func TestAboveAutomaticCapQueuesOnceWithoutRaisingPolicyOrMandate(t *testing.T) {
	now := time.Now()
	policy := validPolicy()
	policy.PerTransactionCapMinor = 20
	policy.RequiresApproval = false
	repo := newRepository()
	queue := &approvalQueue{}
	grant := validGrant(now)
	service := authorization.NewService(repo, verifier{claims: agentclaims.For("operator:1")}, mandates{value: grant}, budgets{value: policy}, &evidence{}, schedule.NewFake(now), queue)
	in := validInput("above-auto", 25)
	first, err := service.Propose(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictPending, first.Verdict)
	require.Equal(t, "per_transaction_cap", first.ViolatedConstraint)
	second, err := service.Propose(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Len(t, repo.records, 1)
	controller := authorization.NewApprovalController(repo, mandates{value: grant}, budgets{value: policy}, schedule.NewFake(now))
	approved, err := controller.Approve(context.Background(), first.ID)
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictApproved, approved.Verdict)
	require.EqualValues(t, 25, approved.HoldMinor)
	require.EqualValues(t, 20, policy.PerTransactionCapMinor)
	require.EqualValues(t, 100, grant.CapMinor)
}

// [REQ:TRS-P0-005] An unverifiable caller is refused, evidenced, and never persisted as spend authority.
func TestProposeFailsClosedBeforeAuthorizationPersistence(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	repo := newRepository()
	recorder := &evidence{}
	service := fixture(now, repo, verifier{err: errors.New("authority unavailable")}, validPolicy(), validGrant(now), recorder)

	got, err := service.Propose(context.Background(), validInput("auth-1", 10))
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictRefused, got.Verdict)
	require.Equal(t, "identity", got.ViolatedConstraint)
	require.Contains(t, got.Remediation, "active agent-manager identity token")
	require.Empty(t, repo.records, "identity failure must not create spend authority")
	require.Len(t, recorder.records, 1)
}

func TestPendingAuthorizationCreatesLocalApproval(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	repo := newRepository()
	queue := &approvalQueue{}
	service := authorization.NewService(repo, verifier{claims: agentclaims.For("operator:1")}, mandates{value: validGrant(now)}, budgets{value: validPolicy()}, &evidence{}, schedule.NewFake(now), queue)
	result, err := service.Propose(context.Background(), validInput("auth-approval", 10))
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictPending, result.Verdict)
	require.Len(t, queue.admissions, 1)
	require.Equal(t, "auth-approval:approval", queue.admissions[0].ID)
	require.Equal(t, result.HoldMinor, queue.admissions[0].AmountMinor)
}

func TestIdempotentRetryRepairsMissingApprovalAdmission(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	repo := newRepository()
	input := validInput("auth-repair", 10)
	pending := basePendingRecordForTest(input, now)
	_, err := repo.Create(context.Background(), pending)
	require.NoError(t, err)
	queue := &approvalQueue{}
	service := authorization.NewService(repo, verifier{claims: agentclaims.For("operator:1")}, mandates{value: validGrant(now)}, budgets{value: validPolicy()}, &evidence{}, schedule.NewFake(now), queue)

	result, err := service.Propose(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, pending, result)
	require.Len(t, queue.admissions, 1, "retry must reconstruct a potentially missing approval after an authorization-first crash")
}

func TestIdempotencyKeyCannotAliasADifferentCharge(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	repo := newRepository()
	input := validInput("auth-original", 10)
	_, err := repo.Create(context.Background(), basePendingRecordForTest(input, now))
	require.NoError(t, err)
	service := authorization.NewService(repo, verifier{claims: agentclaims.For("operator:1")}, mandates{value: validGrant(now)}, budgets{value: validPolicy()}, &evidence{}, schedule.NewFake(now), &approvalQueue{})

	input.AmountMinor = 11
	_, err = service.Propose(context.Background(), input)
	require.ErrorIs(t, err, authorization.ErrInvalid)
}

func basePendingRecordForTest(input authorization.ProposeInput, now time.Time) authorization.Record {
	return authorization.Record{
		ID: input.ID, IdempotencyKey: input.IdempotencyKey, MandateID: input.MandateID, BudgetID: "budget-1",
		RequestingAgent: "operator:1", AmountMinor: input.AmountMinor, Currency: input.Currency,
		Counterparty: input.Counterparty, SettlementID: input.SettlementID, InstrumentID: input.InstrumentID,
		SettlementIdempotencyKey: input.SettlementIdempotencyKey, Verdict: authorization.VerdictPending, HoldMinor: input.AmountMinor,
		CreatedAt: now, ExpiresAt: now.Add(15 * time.Minute),
	}
}

// [REQ:TRS-P0-002] Policy is recomputed server-side and denials name the constraint and remedy.
func TestProposeEvaluatesStoredPolicyInSecurityOrder(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		amount     int64
		editPolicy func(*budget.Budget)
		editGrant  func(*mandate.Mandate)
		want       string
	}{
		{"expired mandate", 10, func(*budget.Budget) {}, func(m *mandate.Mandate) { m.Status = mandateflow.MandateExpired }, "mandate_live"},
		{"mandate hard cap", 101, func(*budget.Budget) {}, func(*mandate.Mandate) {}, "per_transaction_cap"},
		{"counterparty scope", 10, func(b *budget.Budget) { b.DeniedCounterparties = []string{"api.example"} }, func(*mandate.Mandate) {}, "counterparty_scope"},
		{"frozen budget", 10, func(b *budget.Budget) { b.Frozen = true }, func(*mandate.Mandate) {}, "budget_frozen"},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			policy, grant := validPolicy(), validGrant(now)
			test.editPolicy(&policy)
			test.editGrant(&grant)
			grantSource := mandates{value: grant}
			if grant.Status != mandateflow.MandateLive {
				grantSource.err = mandate.ErrInactive
			}
			repo := newRepository()
			recorder := &evidence{}
			service := authorization.NewService(repo, verifier{claims: agentclaims.For("operator:1")}, grantSource, budgets{value: policy}, recorder, schedule.NewFake(now), &approvalQueue{})
			got, err := service.Propose(context.Background(), validInput(string(rune('a'+index)), test.amount))
			require.NoError(t, err)
			require.Equal(t, authorization.VerdictRefused, got.Verdict)
			require.Equal(t, test.want, got.ViolatedConstraint)
			require.NotEmpty(t, got.Remediation)
		})
	}
}

// [REQ:TRS-P1-006] A broad kill switch is read inside the authorization
// boundary and refuses the next proposal before any hold is persisted.
func TestBookOrScenarioFreezeRefusesNextAuthorization(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	for _, scope := range []budget.FreezeScope{budget.FreezeScopeBook, budget.FreezeScopeScenario} {
		repo := newRepository()
		service := authorization.NewService(repo, verifier{claims: agentclaims.For("operator:1")}, mandates{value: validGrant(now)}, frozenBudgets{budgets: budgets{value: validPolicy()}, scope: scope}, &evidence{}, schedule.NewFake(now), &approvalQueue{})
		result, err := service.Propose(context.Background(), validInput("auth-"+string(scope), 10))
		require.NoError(t, err)
		require.Equal(t, authorization.VerdictRefused, result.Verdict)
		require.Equal(t, string(scope)+"_frozen", result.ViolatedConstraint)
		require.Empty(t, repo.records)
	}
}

// [REQ:TRS-P0-003] A pending authorization reserves derived headroom before settlement.
func TestConcurrentProposalsCannotSpendTheSameHeadroom(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	repo := newRepository()
	recorder := &evidence{}
	service := fixture(now, repo, verifier{claims: agentclaims.For("operator:1")}, validPolicy(), validGrant(now), recorder)

	results := make(chan authorization.Record, 2)
	var group sync.WaitGroup
	for _, id := range []string{"auth-1", "auth-2"} {
		group.Add(1)
		go func() {
			defer group.Done()
			got, err := service.Propose(context.Background(), validInput(id, 75))
			require.NoError(t, err)
			results <- got
		}()
	}
	group.Wait()
	close(results)
	counts := map[authorization.Verdict]int{}
	for result := range results {
		counts[result.Verdict]++
	}
	require.Equal(t, 1, counts[authorization.VerdictPending])
	require.Equal(t, 1, counts[authorization.VerdictRefused])
}

func TestReleaseAndReadTimeExpiryFreeDerivedHeadroom(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	repo := newRepository()
	service := fixture(now, repo, verifier{claims: agentclaims.For("operator:1")}, validPolicy(), validGrant(now), &evidence{})
	first, err := service.Propose(context.Background(), validInput("auth-1", 100))
	require.NoError(t, err)
	_, err = service.Release(context.Background(), first.ID)
	require.NoError(t, err)
	second, err := service.Propose(context.Background(), validInput("auth-2", 100))
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictPending, second.Verdict)
}

// [REQ:TRS-P0-003] Periodic headroom rolls at its boundary while total usage remains consumed.
func TestPeriodicHeadroomRollsWithoutFreeingTotalUsage(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	repo := newRepository()
	repo.records["settled-1"] = authorization.Record{ID: "settled-1", BudgetID: "budget-1", MandateID: "mandate-1", AmountMinor: 100, Verdict: authorization.VerdictSettled, CreatedAt: now}
	policy := validPolicy()
	policy.TotalCapMinor = 200
	clock := schedule.NewFake(now.Add(30 * time.Minute))
	grant := validGrant(now.Add(30 * time.Minute))
	grant.CapMinor = 200
	service := authorization.NewService(repo, verifier{claims: agentclaims.For("operator:1")}, mandates{value: grant}, budgets{value: policy}, &evidence{}, clock, &approvalQueue{})

	withinPeriod, err := service.Propose(context.Background(), validInput("auth-1", 100))
	require.NoError(t, err)
	require.Equal(t, "periodic_headroom", withinPeriod.ViolatedConstraint)

	clock.Advance(30 * time.Minute)
	afterBoundary, err := service.Propose(context.Background(), validInput("auth-2", 100))
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictPending, afterBoundary.Verdict)
}
