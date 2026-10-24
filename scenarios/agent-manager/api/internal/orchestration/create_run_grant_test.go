package orchestration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	grantFixtureBaselineToken  = "baseline-fixture-token"
	grantFixtureOperationToken = "operation-fixture-token"
	grantFixtureApprover       = "operator@example.test"
	runCreateGrantTarget       = "agent-manager:*"
)

type runCreatePairVerifier struct {
	baseline  *cliutil.VerifiedClaims
	operation *cliutil.VerifiedClaims
	valid     bool
}

func (v *runCreatePairVerifier) Verify(token string) (*cliutil.VerifyResult, error) {
	if token != grantFixtureBaselineToken {
		return &cliutil.VerifyResult{Error: "invalid"}, nil
	}
	return &cliutil.VerifyResult{Valid: true, Claims: v.baseline}, nil
}

func (v *runCreatePairVerifier) VerifyPair(_ context.Context, baseline, operation string) (*cliutil.VerifyPairResult, error) {
	if baseline != grantFixtureBaselineToken || operation != grantFixtureOperationToken || !v.valid {
		return &cliutil.VerifyPairResult{Error: "invalid_pair"}, nil
	}
	return &cliutil.VerifyPairResult{Valid: true, Operation: &cliutil.VerifyResult{Valid: true, Claims: v.operation}}, nil
}

type runCreateGrantRunRepository struct {
	repository.RunRepository
	run *domain.Run
}

type runCreateAuthorizationRepository struct {
	domain.AuthorizationRepository
	grant *domain.AuthorizationGrant
}

func (r runCreateAuthorizationRepository) Get(_ context.Context, id uuid.UUID) (*domain.AuthorizationGrant, error) {
	if r.grant == nil || r.grant.ID != id {
		return nil, nil
	}
	copy := *r.grant
	return &copy, nil
}

type runCreateAgentIdentityVerifier struct{ o *Orchestrator }

func (v runCreateAgentIdentityVerifier) Verify(token string) (*cliutil.VerifyResult, error) {
	verified, err := v.o.VerifyIdentityToken(context.Background(), token)
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
		return &cliutil.VerifyResult{Error: "invalid"}, err
	}
	encoded, err := json.Marshal(verified.Claims)
	if err != nil {
		return nil, err
	}
	var claims cliutil.VerifiedClaims
	if err := json.Unmarshal(encoded, &claims); err != nil {
		return nil, err
	}
	return &cliutil.VerifyResult{Valid: true, Claims: &claims, RunStatus: string(verified.RunStatus)}, nil
}

func (r runCreateGrantRunRepository) Get(_ context.Context, id uuid.UUID) (*domain.Run, error) {
	if r.run != nil && r.run.ID == id {
		copy := *r.run
		return &copy, nil
	}
	return nil, nil
}

func runCreateRule(policy string, operations, targets []string) coreidentity.OperationRule {
	return coreidentity.OperationRule{Policy: policy, Operations: operations, Targets: targets}
}

func runCreateRequestContext(t *testing.T, verifier *runCreatePairVerifier, call func(context.Context) error) error {
	t.Helper()
	var callErr error
	handler := provenance.Middleware(verifier)(authn.Middleware(authn.Config{})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		callErr = call(r.Context())
	})))
	req := httptest.NewRequest(http.MethodPost, "http://agent-manager/api/v1/runs", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, grantFixtureBaselineToken)
	req.Header.Set(cliutil.HeaderAgentOperationToken, grantFixtureOperationToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("test middleware status = %d", response.Code)
	}
	return callErr
}

func TestRunCreateAcceptsExactRuleFromMultiPolicyGrantWithoutScopeUnion(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	baselineID, grantRunID := uuid.New(), uuid.New()
	grantID := uuid.NewString()
	grantExpires := now.Add(time.Hour)
	aliasOwnerExpires := now.Add(30 * time.Minute)
	grant := &coreidentity.OperationGrant{
		ID: grantID, Policy: "agent-management", Approver: grantFixtureApprover,
		Operations: []string{"agent.investigate", "agent.manage", "run.create", "deployment.manage"},
		Targets:    []string{"agent-manager:*", "deployment-manager:*"},
		Rules: []coreidentity.OperationRule{
			runCreateRule("agent-management", []string{"agent.investigate", "agent.manage"}, []string{"agent-manager:*"}),
			runCreateRule("run-creation", []string{"run.create"}, []string{runCreateGrantTarget}),
			runCreateRule("deployment-operations", []string{"deployment.manage"}, []string{"deployment-manager:*"}),
		},
		ExpiresAt: grantExpires,
	}
	grantJSON, err := json.Marshal(grant)
	if err != nil {
		t.Fatal(err)
	}
	verifier := &runCreatePairVerifier{
		valid: true,
		baseline: &cliutil.VerifiedClaims{
			RunID: baselineID.String(), Subject: grantFixtureApprover,
			Scopes: []string{"prompt-manager:read"}, ExpiresAt: grantExpires.Unix(),
		},
		operation: &cliutil.VerifiedClaims{
			RunID: grantRunID.String(), Subject: grantFixtureApprover,
			Scopes:    []string{"agent-manager:read", "agent-manager:write", "deployment-manager:write"},
			ExpiresAt: grantExpires.Unix(), Meta: map[string]string{"authorization_id": grantID, "operation_grant": string(grantJSON)},
		},
	}
	grantRun := &domain.Run{ID: grantRunID, Status: domain.RunStatusRunning, OwnerExpiresAt: &aliasOwnerExpires}
	o := &Orchestrator{clock: func() time.Time { return now }, runs: runCreateGrantRunRepository{run: grantRun}}
	created := CreateRunRequest{RunIdentityToken: grantFixtureBaselineToken}

	err = runCreateRequestContext(t, verifier, func(ctx context.Context) error {
		if err := o.authenticateCreateRunCaller(ctx, &created); err != nil {
			return err
		}
		return o.revalidateCreateRunCaller(ctx, created)
	})
	if err != nil {
		t.Fatalf("approved run.create rule did not admit a top-level run: %v", err)
	}
	if created.caller == nil || created.caller.Kind != "operation-grant" || created.caller.Subject != grantFixtureApprover || created.caller.RunID != grantRunID || created.caller.GrantID != grantID {
		t.Fatalf("grant attribution was not retained: %+v", created.caller)
	}
	if !reflect.DeepEqual(created.OwnerScopes, []string{"agent-manager:read", "agent-manager:write"}) {
		t.Fatalf("new run ceiling included baseline or unrelated grant scopes: %v", created.OwnerScopes)
	}
	if created.OwnerSubject != grantFixtureApprover || created.OwnerExpiresAt == nil || !created.OwnerExpiresAt.Equal(aliasOwnerExpires) {
		t.Fatalf("new run owner or expiry was not bounded by verified grant record: owner=%q expiry=%v", created.OwnerSubject, created.OwnerExpiresAt)
	}
}

func TestRunCreateRejectsWrongRuleAndUnrelatedRequestedCapability(t *testing.T) {
	for _, tc := range []struct {
		name      string
		wrongRule bool
		requested []string
	}{
		{name: "no-run-create-rule", wrongRule: true},
		{name: "cannot-request-unrelated-capability", requested: []string{"deployment-manager:write"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now().UTC().Truncate(time.Second)
			grantRunID := uuid.New()
			grantID := uuid.NewString()
			expires := now.Add(time.Hour)
			grant := &coreidentity.OperationGrant{ID: grantID, Policy: "agent-management", Approver: grantFixtureApprover,
				Operations: []string{"agent.manage", "run.create"}, Targets: []string{runCreateGrantTarget},
				Rules: []coreidentity.OperationRule{runCreateRule("agent-management", []string{"agent.manage"}, []string{runCreateGrantTarget})}, ExpiresAt: expires}
			if !tc.wrongRule {
				grant.Rules = append(grant.Rules, runCreateRule("run-creation", []string{"run.create"}, []string{runCreateGrantTarget}))
			}
			grantJSON, err := json.Marshal(grant)
			if err != nil {
				t.Fatal(err)
			}
			verifier := &runCreatePairVerifier{valid: true,
				baseline:  &cliutil.VerifiedClaims{RunID: uuid.NewString(), Subject: grantFixtureApprover, Scopes: []string{"prompt-manager:read"}, ExpiresAt: expires.Unix()},
				operation: &cliutil.VerifiedClaims{RunID: grantRunID.String(), Subject: grantFixtureApprover, Scopes: []string{"agent-manager:read", "agent-manager:write", "deployment-manager:write"}, ExpiresAt: expires.Unix(), Meta: map[string]string{"authorization_id": grantID, "operation_grant": string(grantJSON)}},
			}
			o := &Orchestrator{clock: func() time.Time { return now }, runs: runCreateGrantRunRepository{run: &domain.Run{ID: grantRunID, Status: domain.RunStatusRunning}}}
			created := CreateRunRequest{RunIdentityToken: grantFixtureBaselineToken, RequestedScopes: tc.requested}
			err = runCreateRequestContext(t, verifier, func(ctx context.Context) error { return o.authenticateCreateRunCaller(ctx, &created) })
			if err == nil {
				t.Fatal("unapproved run-create authority was admitted")
			}
		})
	}
}

func TestRunCreateRefusesMissingGrantRunRecord(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	grantID, grantRunID := uuid.NewString(), uuid.New()
	expires := now.Add(time.Hour)
	grant := &coreidentity.OperationGrant{ID: grantID, Policy: "run-creation", Approver: grantFixtureApprover, Operations: []string{"run.create"}, Targets: []string{runCreateGrantTarget}, ExpiresAt: expires}
	grantJSON, err := json.Marshal(grant)
	if err != nil {
		t.Fatal(err)
	}
	verifier := &runCreatePairVerifier{valid: true,
		baseline:  &cliutil.VerifiedClaims{RunID: uuid.NewString(), Subject: grantFixtureApprover, Scopes: []string{"prompt-manager:read"}, ExpiresAt: expires.Unix()},
		operation: &cliutil.VerifiedClaims{RunID: grantRunID.String(), Subject: grantFixtureApprover, Scopes: []string{"agent-manager:write"}, ExpiresAt: expires.Unix(), Meta: map[string]string{"authorization_id": grantID, "operation_grant": string(grantJSON)}},
	}
	o := &Orchestrator{clock: func() time.Time { return now }, runs: runCreateGrantRunRepository{}}
	created := CreateRunRequest{RunIdentityToken: grantFixtureBaselineToken}
	err = runCreateRequestContext(t, verifier, func(ctx context.Context) error { return o.authenticateCreateRunCaller(ctx, &created) })
	if err == nil {
		t.Fatal("grant without a live retained run was admitted")
	}
}

func TestOwnerlessAttachedGrantRunCreatesWithinRunCreationCeiling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	secret := []byte("disposable-ownerless-grant-key")
	grantID, grantRunID := uuid.New(), uuid.New()
	expires := now.Add(16 * time.Hour)
	grant := &coreidentity.OperationGrant{
		ID: grantID.String(), Policy: "agent-management", Approver: grantFixtureApprover,
		Operations: []string{"agent.investigate", "agent.manage", "team.prepare", "effort.reconcile", "run.create"},
		Targets:    []string{"agent-manager:*", "prompt-manager:team/demo", "agent-manager:effort/goal"},
		Rules: []coreidentity.OperationRule{
			runCreateRule("agent-investigation", []string{"agent.investigate"}, []string{"agent-manager:*"}),
			runCreateRule("agent-management", []string{"agent.manage"}, []string{"agent-manager:*"}),
			runCreateRule("team-preparation", []string{"team.prepare"}, []string{"prompt-manager:team/demo"}),
			runCreateRule("effort-metadata", []string{"effort.reconcile"}, []string{"agent-manager:effort/goal"}),
			runCreateRule("run-creation", []string{"run.create"}, []string{runCreateGrantTarget}),
		},
		ExpiresAt: expires,
	}
	grantJSON, err := json.Marshal(grant)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatalf("current process binding unavailable for ownerless grant fixture: %v", err)
	}
	claims := &identity.Claims{
		RunID: grantRunID, TaskID: uuid.New(), Subject: grantFixtureApprover,
		Scopes:   []string{"agent-manager:read", "agent-manager:write", "prompt-manager:read", "prompt-manager:write", "deployment-manager:write"},
		IssuedAt: now.Unix(), ExpiresAt: expires.Unix(),
		Meta: map[string]string{"authorization_id": grantID.String(), "operation_grant": string(grantJSON)},
	}
	token, err := identity.GenerateToken(claims, secret)
	if err != nil {
		t.Fatal(err)
	}
	grantRecord := &domain.AuthorizationGrant{
		ID: grantID, RunID: grantRunID, State: domain.AuthorizationApproved, Policy: "agent-management",
		Binding: binding, Grant: grant, TokenHash: identity.HashToken(token),
	}
	grantRun := &domain.Run{
		ID: grantRunID, Status: domain.RunStatusRunning, ExecutionMode: domain.ExecutionModeAttached,
		RunnerPID: binding.PID, OwnerSubject: "", OwnerScopes: []string{},
	}
	o := &Orchestrator{
		clock: func() time.Time { return now }, identitySecret: secret,
		authorizations: runCreateAuthorizationRepository{grant: grantRecord},
		runs:           runCreateGrantRunRepository{run: grantRun},
	}
	request := CreateRunRequest{RunIdentityToken: token}
	var callErr error
	handler := provenance.Middleware(runCreateAgentIdentityVerifier{o: o})(authn.Middleware(authn.Config{})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		callErr = o.authenticateCreateRunCaller(r.Context(), &request)
		if callErr == nil {
			callErr = o.revalidateCreateRunCaller(r.Context(), request)
		}
	})))
	httpRequest := httptest.NewRequest(http.MethodPost, "http://agent-manager/api/v1/runs", nil)
	httpRequest.Header.Set(cliutil.HeaderAgentIdentityToken, token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httpRequest)
	if response.Code != http.StatusOK {
		t.Fatalf("middleware status = %d", response.Code)
	}
	if callErr != nil {
		t.Fatalf("ownerless approved grant did not admit bounded run creation: %v", callErr)
	}
	if request.caller == nil || request.caller.Kind != "operation-grant" || request.caller.RunID != grantRunID || request.caller.GrantID != grantID.String() || request.OwnerSubject != grantFixtureApprover {
		t.Fatalf("new run was not attributed to independently approved grant: caller=%+v owner=%q", request.caller, request.OwnerSubject)
	}
	if !reflect.DeepEqual(request.OwnerScopes, []string{"agent-manager:read", "agent-manager:write"}) {
		t.Fatalf("ownerless baseline or unrelated approved rules widened the new run: %v", request.OwnerScopes)
	}
}
