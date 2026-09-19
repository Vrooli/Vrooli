package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/supervision"
	"connectrpc.com/connect"
	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Only the human credential verifier is stubbed. RPCs, signer, enrollment,
// profile selection, run creation, identity minting and revocation use owners.
type dispatchOwnerAuth struct{ allowWatchAction }

func (dispatchOwnerAuth) AuthorizeEffortAction(_ context.Context, token string, authority pb.WatchAuthority) (supervision.EffortActor, error) {
	if token != "human-owner-fixture" || authority != pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR {
		return supervision.EffortActor{}, errors.New("owner refused")
	}
	return supervision.EffortActor{ID: "owner", Operator: true, Scopes: []string{"agent-manager:supervise"}}, nil
}

func TestSupervisorDispatchRPCToMintedChildAndLiveRevocation(t *testing.T) {
	ctx := context.Background()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	tokens := make(chan string, 1)
	release := make(chan struct{})
	var releaseOnce sync.Once
	finish := func() { releaseOnce.Do(func() { close(release) }) }
	var executions atomic.Int32
	mock.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		executions.Add(1)
		tokens <- req.Environment["VROOLI_AGENT_IDENTITY_TOKEN"]
		select {
		case <-release:
		case <-ctx.Done():
		}
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	h, _, repos, _ := setupTestHandlerWithRunnerAndRepos(t, mock)
	orch := h.svc.RunService.(*orchestration.Orchestrator)
	orchestration.WithIdempotency(repos.Idempotency)(orch)
	secret := []byte("isolated-dispatch-owner-signer-fixture")
	orchestration.WithIdentitySecret(secret)(orch)
	db, err := sqlx.Connect("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	t.Cleanup(finish)
	if err = coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(supervision.Schema)); err != nil {
		t.Fatal(err)
	}
	repo := supervision.NewRepository(db)
	s := supervision.NewEffortService(repo, nil, nil, supervision.EffortDiscoveryConfig{Root: t.TempDir()})
	var dispatcherToken string
	s.ConfigureDispatch(secret, func(token string) error { dispatcherToken = token; return nil }, func(ctx context.Context, key string) error { _, err := repos.Profiles.GetByKey(ctx, key); return err })
	orch.SetSupervisorDispatch(s)
	profile, err := orch.CreateProfile(ctx, &domain.AgentProfile{Name: "Bound supervisor", ProfileKey: "qualified-supervisor", DeclaredScopes: []string{"agent-manager:supervise"}, RoleRef: "code.default", SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}})
	if err != nil {
		t.Fatal(err)
	}
	task, err := orch.CreateTask(ctx, &domain.Task{Title: "Fixture wake", ScopePath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	rpc := NewAgentManagerConnectHandler(h, &supervision.Service{Efforts: s})
	rpc.SetWatchActionAuthorizer(dispatchOwnerAuth{})
	path, handler := apiconnect.NewAgentManagerServiceHandler(rpc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL, connect.WithProtoJSON())
	enroll := connect.NewRequest(&pb.EnrollEffortRequest{Enrollment: &pb.EffortEnrollment{EffortRef: "service:standing", SupervisorOwnerSubject: "owner", SupervisorScope: "agent-manager:supervise"}, IdempotencyKey: "enroll"})
	enroll.Header().Set("Authorization", "Bearer human-owner-fixture")
	enrolled, err := client.EnrollEffort(ctx, enroll)
	if err != nil {
		t.Fatal(err)
	}
	issue := connect.NewRequest(&api.IssueSupervisorDispatchRequest{EffortRef: "service:standing", ExpectedRevision: enrolled.Msg.Revision, TeamId: "supervisors", MemberId: "leader", ProfileKey: profile.ProfileKey, MaximumRuns: 1, MinimumIntervalSeconds: 60, ExpiresAt: timestamppb.New(time.Now().Add(3 * time.Hour)), IdempotencyKey: "issue"})
	issue.Header().Set("Authorization", "Bearer human-owner-fixture")
	issued, err := client.IssueSupervisorDispatch(ctx, issue)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := protojson.Marshal(issued.Msg)
	if dispatcherToken == "" || strings.Contains(string(encoded), dispatcherToken) {
		t.Fatal("issuance failed or returned bearer")
	}
	if result, err := orch.VerifyIdentityToken(ctx, dispatcherToken); err != nil || result.Valid {
		t.Fatal("dispatcher is not an ordinary run identity")
	}
	wake := connect.NewRequest(&api.CreateSupervisorRunRequest{EffortRef: "service:standing", AuthorizationId: issued.Msg.DispatchAuthorization.AuthorizationId, TeamId: "supervisors", MemberId: "leader", TaskId: task.ID.String(), IdempotencyKey: "future-wake"})
	wake.Header().Set("Authorization", "Bearer "+dispatcherToken)
	wake.Header().Set("X-Agent-Identity-Token", "identified-run")
	if _, err := client.CreateSupervisorRun(ctx, wake); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatal("identified run bypassed direct lifecycle guard")
	}
	wake.Header().Del("X-Agent-Identity-Token")
	// Admission can fail before any effect. Restoring the granted profile must
	// permit the identical admission to create its one original run.
	profile.DeclaredScopes = []string{}
	if err := repos.Profiles.Update(ctx, profile); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CreateSupervisorRun(ctx, wake); err == nil || executions.Load() != 0 {
		t.Fatal("pre-effect refusal executed work")
	}
	profile.DeclaredScopes = []string{"agent-manager:supervise"}
	if err := repos.Profiles.Update(ctx, profile); err != nil {
		t.Fatal(err)
	}
	created, err := client.CreateSupervisorRun(ctx, wake)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := client.CreateSupervisorRun(ctx, wake)
	if err != nil || created.Msg.Run.Id != replayed.Msg.Run.Id {
		t.Fatal("replayed admission duplicated run", err)
	}
	var childToken string
	select {
	case childToken = <-tokens:
	case <-time.After(5 * time.Second):
		t.Fatal("runner did not receive child identity")
	}
	verified, err := orch.VerifyIdentityToken(ctx, childToken)
	if err != nil || !verified.Valid {
		t.Fatalf("child verification failed: %v %+v", err, verified)
	}
	claims := verified.Claims
	if claims.Subject != "owner" || claims.ProfileKey != profile.ProfileKey || !slices.Equal(claims.Scopes, []string{"agent-manager:supervise"}) || claims.DispatchAuthorizationID != issued.Msg.DispatchAuthorization.AuthorizationId || claims.ExpiresAt > issued.Msg.DispatchAuthorization.ExpiresAt.AsTime().Unix() {
		t.Fatalf("child lost exact authority attenuation: subject=%q profile=%q scopes=%v binding=%q expiry=%d", claims.Subject, claims.ProfileKey, claims.Scopes, claims.DispatchAuthorizationID, claims.ExpiresAt)
	}
	persisted, err := repos.Runs.Get(ctx, claims.RunID)
	if err != nil || persisted.DispatchBinding == nil || persisted.DispatchBinding.AuthorizationID != claims.DispatchAuthorizationID {
		t.Fatal("binding missing from durable run")
	}
	finish()
	deadline := time.Now().Add(5 * time.Second)
	for {
		persisted, err = repos.Runs.Get(ctx, claims.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if persisted.Status == domain.RunStatusComplete {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("original executor did not complete: %s", persisted.Status)
		}
		time.Sleep(time.Millisecond)
	}
	if err := orch.DeleteRun(ctx, claims.RunID); err != nil {
		t.Fatal("completed run deletion should preserve a separate acceptance receipt", err)
	}
	// Model expiry/cleanup of the ordinary one-hour cache while the three-hour
	// grant remains valid: durable run evidence, not that cache, must fence replay.
	orchestration.WithIdempotency(nil)(orch)
	afterExpiry, err := client.CreateSupervisorRun(ctx, wake)
	if err == nil || afterExpiry != nil || executions.Load() != 1 {
		t.Fatal("cache expiry and deletion reopened exhausted admission", err)
	}
	receipt, err := repos.Runs.GetCreationReceipt(ctx, "supervisor-dispatch:"+claims.DispatchAuthorizationID+":"+wake.Msg.IdempotencyKey)
	if err != nil || receipt == nil || receipt.RunID != claims.RunID {
		t.Fatal("deleted original run lost owner acceptance evidence", err)
	}
	currentBudget, _, err := repo.GetEffort(ctx, "service:standing")
	if err != nil || currentBudget.DispatchAuthorization.DispatchedRuns != 1 || currentBudget.DispatchAuthorization.MaximumRuns != 1 {
		t.Fatal("original run did not exhaust finite allowance")
	}
	current, _, err := repo.GetEffort(ctx, "service:standing")
	if err != nil {
		t.Fatal(err)
	}
	revoke := connect.NewRequest(&api.RevokeSupervisorDispatchRequest{EffortRef: "service:standing", AuthorizationId: claims.DispatchAuthorizationID, ExpectedRevision: current.Revision, Reason: "fixture complete", IdempotencyKey: "revoke"})
	revoke.Header().Set("Authorization", "Bearer human-owner-fixture")
	if _, err := client.RevokeSupervisorDispatch(ctx, revoke); err != nil {
		t.Fatal(err)
	}
	if result, err := orch.VerifyIdentityToken(ctx, childToken); err != nil || result.Valid {
		t.Fatal("revoked grant left active child authorized")
	}
	if _, err := client.CreateSupervisorRun(ctx, wake); err == nil || strings.Contains(err.Error(), dispatcherToken) {
		t.Fatal("revoked dispatch replay succeeded or leaked bearer")
	}
}
