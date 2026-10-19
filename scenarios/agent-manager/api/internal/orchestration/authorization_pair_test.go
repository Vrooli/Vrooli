//go:build linux

package orchestration

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	runidentity "agent-manager/internal/identity"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

const pairTestSecret = "credential-pair-test-secret"

func newCredentialPairFixture(t *testing.T) (*Orchestrator, context.Context, context.Context, *domain.Run, string) {
	return newCredentialPairFixtureWithOwnerLifetime(t, time.Hour)
}

func newCredentialPairFixtureWithOwnerLifetime(t *testing.T, ownerLifetime time.Duration) (*Orchestrator, context.Context, context.Context, *domain.Run, string) {
	t.Helper()
	ctx := context.Background()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	o := New(nil, nil, repos.Runs, WithEvents(events), WithIdentitySecret([]byte(pairTestSecret)), WithAuthorizations(database.NewAuthorizationRepository(db)))
	policy, ok := authn.InteractivePolicy("swarm-bookkeeping")
	if !ok {
		t.Fatal("fixture policy missing")
	}
	owner := requestidentity.Principal{Kind: requestidentity.ActorHuman, Subject: "pair-owner", Source: requestidentity.SourceCloudflareAccess, Verified: true, Scopes: append([]string{"agent-manager:write"}, policy.Scopes...), ExpiresAt: time.Now().Add(ownerLifetime)}
	ownerCtx := requestidentity.WithPrincipal(ctx, owner)
	attached, err := o.AttachRun(ownerCtx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "pair-fixture", ProcessID: os.Getpid()})
	if err != nil {
		t.Fatalf("attach owner run: %v", err)
	}
	return o, ctx, ownerCtx, attached.Run, attached.Token
}

func approvePairGrant(t *testing.T, o *Orchestrator, ctx, ownerCtx context.Context, runID uuid.UUID, session string) string {
	return approvePairGrantForDuration(t, o, ctx, ownerCtx, runID, session, 3600)
}

func approvePairGrantForDuration(t *testing.T, o *Orchestrator, ctx, ownerCtx context.Context, runID uuid.UUID, session string, durationSeconds int64) string {
	t.Helper()
	grant, err := o.RequestAuthorization(ctx, AuthorizationRequest{RunID: runID, PID: os.Getpid(), HarnessKind: "codex", HarnessSession: session, Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: durationSeconds})
	if err != nil {
		t.Fatalf("request grant: %v", err)
	}
	approved, err := o.ApproveAuthorization(ownerCtx, grant.ID)
	if err != nil {
		t.Fatalf("approve grant: %v", err)
	}
	var claims runidentity.Claims
	if err := json.Unmarshal([]byte(approved.ClaimsJSON), &claims); err != nil {
		t.Fatalf("decode grant claims: %v", err)
	}
	token, err := runidentity.GenerateToken(&claims, []byte(pairTestSecret))
	if err != nil {
		t.Fatalf("mint fixture grant token: %v", err)
	}
	return token
}

func expireStoredBaselineToken(t *testing.T, o *Orchestrator, ctx context.Context, runID uuid.UUID) string {
	t.Helper()
	run, err := o.GetRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	claims := runidentity.Claims{RunID: runID, Subject: run.OwnerSubject, Scopes: []string{"prompt-manager:read"}, ExpiresAt: time.Now().Add(-time.Hour).Unix()}
	token, err := runidentity.GenerateToken(&claims, []byte(pairTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	run.IdentityTokenHash = runidentity.HashToken(token)
	if err := o.runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	return token
}

func TestCredentialPairKeepsBaselineAndOperationSeparateThroughBroker(t *testing.T) {
	o, ctx, ownerCtx, run, baselineToken := newCredentialPairFixture(t)
	operationToken := approvePairGrant(t, o, ctx, ownerCtx, run.ID, run.HarnessSessionID)
	pair, err := o.VerifyIdentityPair(ctx, baselineToken, operationToken)
	if err != nil || pair == nil || !pair.Valid || pair.Operation == nil || !pair.Operation.Valid {
		t.Fatalf("valid pair refused: result=%+v err=%v", pair, err)
	}
	credentials, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), baselineToken)
	if err != nil || credentials.BaselineToken != baselineToken || credentials.OperationToken == "" || credentials.LegacyToken != "" {
		t.Fatalf("credential resolver merged or replaced the pair: %+v err=%v", credentials, err)
	}

	dir, err := os.MkdirTemp("/tmp", "amcred-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "identity.sock")
	stop, err := o.startAuthorizationBrokerAt(socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stop() })
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	t.Cleanup(transport.CloseIdleConnections)
	req, err := http.NewRequest(http.MethodGet, "http://local-agent-manager/credential", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(cliutil.HeaderAgentIdentityToken, baselineToken)
	response, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var received AuthorizationCredentials
	if err := json.NewDecoder(response.Body).Decode(&received); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || received.BaselineToken != baselineToken || received.OperationToken == "" || received.LegacyToken != "" {
		t.Fatalf("broker did not preserve distinct credential roles: status=%d credentials=%+v", response.StatusCode, received)
	}
	_ = response.Body.Close()

	identityReq, err := http.NewRequest(http.MethodGet, "http://local-agent-manager/credential?purpose=ordinary_identity", nil)
	if err != nil {
		t.Fatal(err)
	}
	identityReq.Header.Set(cliutil.HeaderAgentIdentityToken, baselineToken)
	identityResponse, err := (&http.Client{Transport: transport}).Do(identityReq)
	if err != nil {
		t.Fatal(err)
	}
	var ordinary AuthorizationCredentials
	if err := json.NewDecoder(identityResponse.Body).Decode(&ordinary); err != nil {
		t.Fatal(err)
	}
	_ = identityResponse.Body.Close()
	if identityResponse.StatusCode != http.StatusOK || ordinary.BaselineToken != baselineToken || ordinary.LegacyToken != "" || ordinary.OperationToken != "" {
		t.Fatalf("identity-purpose selector returned the wrong role set: status=%d baseline=%t legacy=%t operation=%t", identityResponse.StatusCode, ordinary.BaselineToken != "", ordinary.LegacyToken != "", ordinary.OperationToken != "")
	}

	unknownReq, err := http.NewRequest(http.MethodGet, "http://local-agent-manager/credential?purpose=unexpected", nil)
	if err != nil {
		t.Fatal(err)
	}
	unknownReq.Header.Set(cliutil.HeaderAgentIdentityToken, baselineToken)
	unknownResponse, err := (&http.Client{Transport: transport}).Do(unknownReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = unknownResponse.Body.Close()
	if unknownResponse.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown credential purpose status=%d, want 400", unknownResponse.StatusCode)
	}
	duplicateReq, err := http.NewRequest(http.MethodGet, "http://local-agent-manager/credential?purpose=ordinary_identity&purpose=ordinary_identity", nil)
	if err != nil {
		t.Fatal(err)
	}
	duplicateReq.Header.Set(cliutil.HeaderAgentIdentityToken, baselineToken)
	duplicateResponse, err := (&http.Client{Transport: transport}).Do(duplicateReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = duplicateResponse.Body.Close()
	if duplicateResponse.StatusCode != http.StatusBadRequest {
		t.Fatalf("duplicate credential purpose status=%d, want 400", duplicateResponse.StatusCode)
	}
}

func TestCredentialPairAcceptsDistinctAttachedRunIDsForSameOwnerAndProcess(t *testing.T) {
	o, ctx, ownerCtx, baselineRun, baselineToken := newCredentialPairFixture(t)
	grantAttached, err := o.AttachRun(ownerCtx, AttachRunRequest{
		HarnessKind: "codex", HarnessSession: "pair-distinct-grant-fixture", ProcessID: os.Getpid(),
	})
	if err != nil {
		t.Fatalf("attach grant alias run: %v", err)
	}
	operationToken := approvePairGrant(t, o, ctx, ownerCtx, grantAttached.Run.ID, grantAttached.Run.HarnessSessionID)
	baselineClaims, err := runidentity.VerifyToken(baselineToken, []byte(pairTestSecret))
	if err != nil {
		t.Fatalf("verify baseline token: %v", err)
	}
	operationClaims, err := runidentity.VerifyToken(operationToken, []byte(pairTestSecret))
	if err != nil {
		t.Fatalf("verify operation token: %v", err)
	}
	if baselineClaims.RunID != baselineRun.ID || operationClaims.RunID != grantAttached.Run.ID || baselineClaims.RunID == operationClaims.RunID {
		t.Fatalf("fixture must use distinct attached run IDs: baseline=%s operation=%s", baselineClaims.RunID, operationClaims.RunID)
	}
	if baselineClaims.Subject == "" || baselineClaims.Subject != operationClaims.Subject || baselineRun.RunnerPID != grantAttached.Run.RunnerPID || baselineRun.RunnerPID != os.Getpid() {
		t.Fatalf("fixture must bind the same verified owner and current process: baseline=%+v operation=%+v", baselineRun, grantAttached.Run)
	}

	pair, err := o.VerifyIdentityPair(ctx, baselineToken, operationToken)
	if err != nil || pair == nil || !pair.Valid || pair.Operation == nil || !pair.Operation.Valid || pair.Operation.Claims.RunID != grantAttached.Run.ID {
		t.Fatalf("valid distinct-run pair refused or rewritten: result=%+v err=%v", pair, err)
	}
	if len(pair.Operation.Claims.Scopes) != len(operationClaims.Scopes) {
		t.Fatalf("pair changed operation scopes: verified=%v token=%v", pair.Operation.Claims.Scopes, operationClaims.Scopes)
	}
	for i := range operationClaims.Scopes {
		if pair.Operation.Claims.Scopes[i] != operationClaims.Scopes[i] {
			t.Fatalf("pair changed operation scopes: verified=%v token=%v", pair.Operation.Claims.Scopes, operationClaims.Scopes)
		}
	}
	credentials, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), baselineToken)
	if err != nil || credentials.BaselineToken != baselineToken || credentials.OperationToken != operationToken || credentials.LegacyToken != "" {
		t.Fatalf("broker resolver did not preserve the separate distinct-run pair: credentials=%+v err=%v", credentials, err)
	}
}

func TestCredentialPairRejectsParentGrantForManagedChild(t *testing.T) {
	o, ctx, ownerCtx, parent, parentToken := newCredentialPairFixture(t)
	parentOperation := approvePairGrant(t, o, ctx, ownerCtx, parent.ID, parent.HarnessSessionID)
	child, err := o.AttachRun(ownerCtx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "child-pair-fixture", ProcessID: os.Getpid()})
	if err != nil {
		t.Fatal(err)
	}
	storedChild, err := o.GetRun(ctx, child.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	storedChild.ParentRunID = &parent.ID
	storedChild.ExecutionMode = domain.ExecutionModeCodecPipe
	if err := o.runs.Update(ctx, storedChild); err != nil {
		t.Fatal(err)
	}
	parentResult, err := o.VerifyIdentityPair(ctx, child.Token, parentOperation)
	if err != nil || parentResult == nil || parentResult.Valid {
		t.Fatalf("child baseline borrowed its parent's grant: result=%+v err=%v", parentResult, err)
	}
	resolved, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), child.Token)
	if err != nil || resolved.BaselineToken != child.Token || resolved.OperationToken != "" || resolved.LegacyToken != "" {
		t.Fatalf("parent grant fell back to a grant-only child credential: %+v err=%v", resolved, err)
	}
	withoutBaseline, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), "")
	if err != nil || withoutBaseline.LegacyToken != "" {
		t.Fatalf("managed child without a baseline inherited its parent's grant: %+v err=%v", withoutBaseline, err)
	}
	parentGrantAsPrimary, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), parentOperation)
	if err != nil || parentGrantAsPrimary.LegacyToken != "" || parentGrantAsPrimary.BaselineToken != "" {
		t.Fatalf("managed child reused a parent grant as its primary credential: %+v err=%v", parentGrantAsPrimary, err)
	}
	_ = parentToken
}

func TestOwnerlessAttachedRootKeepsLegacyGrantWhenBaselineIsAbsent(t *testing.T) {
	o, ctx, ownerCtx, run, _ := newCredentialPairFixture(t)
	_ = approvePairGrant(t, o, ctx, ownerCtx, run.ID, run.HarnessSessionID)
	credentials, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), "")
	if err != nil || credentials.LegacyToken == "" || credentials.BaselineToken != "" {
		t.Fatalf("ownerless attached root lost its independent legacy grant path: %+v err=%v", credentials, err)
	}
	claims, err := runidentity.VerifyToken(credentials.LegacyToken, []byte(pairTestSecret))
	if err != nil || claims.RunID != run.ID {
		t.Fatalf("attached root resolved another run's grant: claims=%+v err=%v", claims, err)
	}
}

func TestValidOwnerlessAttachedBaselineKeepsIndependentLongGrant(t *testing.T) {
	o, ctx, ownerCtx, _, _ := newCredentialPairFixtureWithOwnerLifetime(t, 24*time.Hour)
	baseline, err := o.AttachRun(ctx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "ownerless-valid-baseline", ProcessID: os.Getpid()})
	if err != nil {
		t.Fatalf("attach ownerless baseline: %v", err)
	}
	withoutGrant, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), baseline.Token)
	if err != nil || withoutGrant.BaselineToken != baseline.Token || withoutGrant.LegacyToken != "" || withoutGrant.OperationToken != "" {
		t.Fatalf("ownerless baseline without a grant changed its ordinary identity: baseline=%t legacy=%t operation=%t err=%v", withoutGrant.BaselineToken != "", withoutGrant.LegacyToken != "", withoutGrant.OperationToken != "", err)
	}
	alias, err := o.AttachRun(ownerCtx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "ownerless-valid-alias", ProcessID: os.Getpid()})
	if err != nil {
		t.Fatalf("attach independently owned grant alias: %v", err)
	}
	grantToken := approvePairGrantForDuration(t, o, ctx, ownerCtx, alias.Run.ID, alias.Run.HarnessSessionID, 16*60*60)
	grantClaims, err := runidentity.VerifyToken(grantToken, []byte(pairTestSecret))
	if err != nil || grantClaims.ExpiresAt < time.Now().Add(15*time.Hour).Unix() {
		t.Fatalf("fixture did not mint a retained long grant: claims=%+v err=%v", grantClaims, err)
	}
	baselineClaims, err := runidentity.VerifyToken(baseline.Token, []byte(pairTestSecret))
	if err != nil || baselineClaims.Subject != "" || len(baselineClaims.Scopes) != 0 || baselineClaims.RunID != baseline.Run.ID {
		t.Fatalf("fixture baseline is not the valid ownerless attached identity: claims=%+v err=%v", baselineClaims, err)
	}
	if baseline.Run.RunnerPID != alias.Run.RunnerPID || baseline.Run.RunnerPID != os.Getpid() {
		t.Fatalf("fixture grants are not bound to the same current process: baseline=%+v alias=%+v", baseline.Run, alias.Run)
	}
	ordinary, err := o.resolveOrdinaryRunIdentityCredential(ctx, os.Getpid(), baseline.Token)
	if err != nil || ordinary.BaselineToken != baseline.Token || ordinary.LegacyToken != "" || ordinary.OperationToken != "" {
		t.Fatalf("identity-purpose resolution did not preserve the ordinary baseline: baseline=%t legacy=%t operation=%t err=%v", ordinary.BaselineToken != "", ordinary.LegacyToken != "", ordinary.OperationToken != "", err)
	}
	credentials, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), baseline.Token)
	if err != nil || credentials.LegacyToken == "" || credentials.BaselineToken != "" || credentials.OperationToken != "" {
		t.Fatalf("valid ownerless baseline hid or paired its separate grant: baseline=%t legacy=%t operation=%t refusal=%s err=%v", credentials.BaselineToken != "", credentials.LegacyToken != "", credentials.OperationToken != "", credentials.OperationRefusal, err)
	}
	resolvedClaims, err := runidentity.VerifyToken(credentials.LegacyToken, []byte(pairTestSecret))
	if err != nil || resolvedClaims.RunID != alias.Run.ID || resolvedClaims.ExpiresAt != grantClaims.ExpiresAt || resolvedClaims.Meta["authorization_id"] != grantClaims.Meta["authorization_id"] {
		t.Fatalf("ownerless baseline did not preserve the exact independently approved grant: claims=%+v expected=%+v err=%v", resolvedClaims, grantClaims, err)
	}
}

func TestOwnerlessAttachedBaselineCannotGiveManagedChildParentGrant(t *testing.T) {
	o, ctx, ownerCtx, _, _ := newCredentialPairFixture(t)
	parent, err := o.AttachRun(ctx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "ownerless-parent-baseline", ProcessID: os.Getpid()})
	if err != nil {
		t.Fatalf("attach ownerless parent baseline: %v", err)
	}
	parentAlias, err := o.AttachRun(ownerCtx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "ownerless-parent-grant-alias", ProcessID: os.Getpid()})
	if err != nil {
		t.Fatalf("attach parent grant alias: %v", err)
	}
	parentGrant := approvePairGrant(t, o, ctx, ownerCtx, parentAlias.Run.ID, parentAlias.Run.HarnessSessionID)
	child, err := o.AttachRun(ctx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "ownerless-managed-child", ProcessID: os.Getpid()})
	if err != nil {
		t.Fatalf("attach ownerless child: %v", err)
	}
	storedChild, err := o.GetRun(ctx, child.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	storedChild.ParentRunID = &parent.Run.ID
	storedChild.ExecutionMode = domain.ExecutionModeCodecPipe
	if err := o.runs.Update(ctx, storedChild); err != nil {
		t.Fatal(err)
	}

	borrowed, borrowErr := o.resolveOrdinaryRunIdentityCredential(ctx, os.Getpid(), parent.Token)
	if borrowErr == nil || borrowed.BaselineToken != "" || borrowed.LegacyToken != "" || borrowed.OperationToken != "" {
		t.Fatalf("managed child borrowed an ancestor ordinary identity (baseline=%t legacy=%t operation=%t) err=%v", borrowed.BaselineToken != "", borrowed.LegacyToken != "", borrowed.OperationToken != "", borrowErr)
	}
	credentials, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), child.Token)
	if err != nil || credentials.BaselineToken != child.Token || credentials.LegacyToken != "" || credentials.OperationToken != "" {
		t.Fatalf("ownerless managed child inherited the parent's independent grant: credentials=%+v err=%v", credentials, err)
	}
	parentClaims, err := runidentity.VerifyToken(parentGrant, []byte(pairTestSecret))
	if err != nil || parentClaims.RunID != parentAlias.Run.ID {
		t.Fatalf("fixture parent grant is invalid: claims=%+v err=%v", parentClaims, err)
	}
}

func TestExpiredAttachedBaselineKeepsOnlyItsCurrentlyBoundGrant(t *testing.T) {
	o, ctx, ownerCtx, run, _ := newCredentialPairFixture(t)
	expiredBaseline := expireStoredBaselineToken(t, o, ctx, run.ID)
	_ = approvePairGrant(t, o, ctx, ownerCtx, run.ID, run.HarnessSessionID)
	credentials, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), expiredBaseline)
	if err != nil || credentials.LegacyToken == "" || credentials.BaselineToken != "" || credentials.OperationToken != "" {
		t.Fatalf("expired attached baseline suppressed or replaced its independent current grant: %+v err=%v", credentials, err)
	}
	claims, err := runidentity.VerifyToken(credentials.LegacyToken, []byte(pairTestSecret))
	if err != nil || claims.RunID != run.ID {
		t.Fatalf("expired baseline resolved a different run's grant: claims=%+v err=%v", claims, err)
	}
}

func TestExpiredOwnerlessAttachedRootKeepsIndependentLongGrant(t *testing.T) {
	o, ctx, ownerCtx, baselineRun, _ := newCredentialPairFixtureWithOwnerLifetime(t, 24*time.Hour)
	stored, err := o.GetRun(ctx, baselineRun.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.OwnerSubject = ""
	if err := o.runs.Update(ctx, stored); err != nil {
		t.Fatal(err)
	}
	aliasAttached, err := o.AttachRun(ownerCtx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "ownerless-alias-fixture", ProcessID: os.Getpid()})
	if err != nil {
		t.Fatal(err)
	}
	grantToken := approvePairGrantForDuration(t, o, ctx, ownerCtx, aliasAttached.Run.ID, aliasAttached.Run.HarnessSessionID, 16*60*60)
	expiredBaseline := expireStoredBaselineToken(t, o, ctx, baselineRun.ID)
	grantClaims, err := runidentity.VerifyToken(grantToken, []byte(pairTestSecret))
	if err != nil || grantClaims.ExpiresAt < time.Now().Add(15*time.Hour).Unix() {
		t.Fatalf("fixture did not mint the retained long grant: claims=%+v err=%v", grantClaims, err)
	}
	credentials, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), expiredBaseline)
	if err != nil || credentials.LegacyToken == "" || credentials.BaselineToken != "" || credentials.OperationToken != "" {
		t.Fatalf("expired ownerless attached root lost its independent long grant: %+v err=%v", credentials, err)
	}
	claims, err := runidentity.VerifyToken(credentials.LegacyToken, []byte(pairTestSecret))
	if err != nil || claims.RunID != aliasAttached.Run.ID || claims.ExpiresAt != grantClaims.ExpiresAt {
		t.Fatalf("expired baseline resolved a different or shortened grant: claims=%+v err=%v", claims, err)
	}
}

func TestExpiredChildBaselineCannotFallBackToParentGrant(t *testing.T) {
	o, ctx, ownerCtx, parent, _ := newCredentialPairFixture(t)
	expiredParentToken := expireStoredBaselineToken(t, o, ctx, parent.ID)
	_ = approvePairGrant(t, o, ctx, ownerCtx, parent.ID, parent.HarnessSessionID)
	child, err := o.AttachRun(ownerCtx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "expired-child-fixture", ProcessID: os.Getpid()})
	if err != nil {
		t.Fatal(err)
	}
	storedChild, err := o.GetRun(ctx, child.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	storedChild.ParentRunID = &parent.ID
	storedChild.ExecutionMode = domain.ExecutionModeCodecPipe
	if err := o.runs.Update(ctx, storedChild); err != nil {
		t.Fatal(err)
	}
	parentHeader, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), expiredParentToken)
	if err != nil || parentHeader.LegacyToken != "" || parentHeader.OperationToken != "" {
		t.Fatalf("managed child recovered its parent's grant from the expired parent baseline: %+v err=%v", parentHeader, err)
	}
	expiredChildToken := expireStoredBaselineToken(t, o, ctx, child.Run.ID)
	credentials, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), expiredChildToken)
	if err != nil || credentials.LegacyToken != "" {
		t.Fatalf("expired child baseline fell back to the parent's grant: %+v err=%v", credentials, err)
	}
}

type authorizationProjectionCountingStore struct {
	event.Store
	reads int
}

func (s *authorizationProjectionCountingStore) Get(ctx context.Context, runID uuid.UUID, opts event.GetOptions) ([]*domain.RunEvent, error) {
	s.reads++
	return s.Store.Get(ctx, runID, opts)
}

func TestAuthorizationCredentialReadsAvoidDerivedRunProjection(t *testing.T) {
	o, ctx, ownerCtx, run, baselineToken := newCredentialPairFixture(t)
	operationToken := approvePairGrant(t, o, ctx, ownerCtx, run.ID, run.HarnessSessionID)
	stored, err := o.GetStoredRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.ExecutionMode = domain.ExecutionModeInteractive
	stored.ResolvedConfig = domain.DefaultRunConfig()
	stored.ResolvedConfig.Until = "retain this interactive goal"
	stored.WebConsoleSessionID = "projection-must-not-resolve"
	if err := o.runs.Update(ctx, stored); err != nil {
		t.Fatalf("prepare interactive authorization fixture: %v", err)
	}

	events := &authorizationProjectionCountingStore{Store: o.events}
	o.events = events
	resolverCalls := 0
	previousResolver := resolveExternalURL
	resolveExternalURL = func(context.Context, string, string) (string, error) {
		resolverCalls++
		return "http://unexpected-projection", nil
	}
	t.Cleanup(func() { resolveExternalURL = previousResolver })

	pair, err := o.VerifyIdentityPair(ctx, baselineToken, operationToken)
	if err != nil || pair == nil || !pair.Valid {
		t.Fatalf("valid persisted identity pair changed: pair=%+v err=%v", pair, err)
	}
	credentials, err := o.ResolveAuthorizationCredentials(ctx, os.Getpid(), baselineToken)
	if err != nil || credentials.BaselineToken != baselineToken || credentials.OperationToken == "" || credentials.LegacyToken != "" {
		t.Fatalf("valid credentials changed: credentials=%+v err=%v", credentials, err)
	}
	if events.reads != 0 || resolverCalls != 0 {
		t.Fatalf("authorization metadata reads ran derived projection: eventReads=%d externalURLResolutions=%d", events.reads, resolverCalls)
	}
}
