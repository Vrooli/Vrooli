package orchestration_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/maintenance"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/repository"
	"agent-manager/internal/storage"

	"github.com/google/uuid"
)

func TestFreshRecoveryRefusesDispatcherBoundSourceIncludingRevokedAndWithdrawn(t *testing.T) {
	for _, state := range []string{"active", "revoked", "withdrawn"} {
		t.Run(state, func(t *testing.T) {
			svc, repos := newResumeTestOrchestrator(t)
			_, _, source := seedFailedRun(t, svc, repos, "original", nil)
			source.DispatchBinding = &domain.DispatchBinding{EffortRef: "test-effort", AuthorizationID: "original-authorization"}
			source.OwnerSubject = "original-owner"
			expires := time.Now().Add(time.Hour)
			source.OwnerExpiresAt = &expires
			source.OwnerScopes = []string{"agent-manager:supervise"}
			source.RequestedScopes = []string{"agent-manager:supervise"}
			if err := repos.Runs.Update(t.Context(), source); err != nil {
				t.Fatal(err)
			}
			authority := &recoveryDispatchAuthority{}
			if state != "active" {
				authority.refusal = errors.New("dispatch authorization " + state)
			}
			svc.SetSupervisorDispatch(authority)
			req := orchestration.ResumeFromFailedRunRequest{RunID: source.ID}
			got, err := svc.ResumeFromFailedRun(t.Context(), req)
			if got != nil || !domain.IsPreEffectRefusal(err) || !strings.Contains(err.Error(), "dispatcher-bound") {
				t.Fatalf("%s dispatch binding escaped into a replacement identity: run=%v err=%v", state, got, err)
			}
			// The automatic owner route must apply the same conservative gate,
			// before requiring physical scope or reading a native session store.
			source.SessionID = "ses_retained"
			source.ResolvedConfig.RunnerType = domain.RunnerTypeOpenCode
			if err := repos.Runs.Update(t.Context(), source); err != nil {
				t.Fatal(err)
			}
			got, err = svc.RecoverMissingSessionRun(t.Context(), req)
			if got != nil || !domain.IsPreEffectRefusal(err) || !strings.Contains(err.Error(), "dispatcher-bound") {
				t.Fatalf("automatic recovery dropped %s dispatch binding: run=%v err=%v", state, got, err)
			}
			hash, err := repos.Runs.(repository.RunFreshRecoveryClaimer).GetFreshRecoveryClaim(t.Context(), source.ID)
			if err != nil || hash != "" {
				t.Fatalf("unsupported binding claimed source: %q %v", hash, err)
			}
			if accepted, err := svc.FreshRecoveryAccepted(t.Context(), source.ID, ""); err != nil || accepted != nil {
				t.Fatalf("unsupported binding created replacement: %v %v", accepted, err)
			}
		})
	}
}

type recoveryDispatchAuthority struct {
	orchestration.SupervisorDispatchAuthority
	refusal error
}

func (a *recoveryDispatchAuthority) CheckDispatchIdentity(context.Context, *identity.Claims) error {
	return a.refusal
}

func TestFreshRecoveryReceiptRejectsLostDispatchBinding(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	_, _, source := seedFailedRun(t, svc, repos, "original", nil)
	source.DispatchBinding = &domain.DispatchBinding{EffortRef: "test-effort", AuthorizationID: "original-authorization"}
	if err := repos.Runs.Update(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	// Reconstruct the durable bad receipt left by the old owner path: the
	// source was claimed, but the replacement omitted its dispatch binding.
	req := orchestration.ResumeFromFailedRunRequest{RunID: source.ID}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	claims := repos.Runs.(repository.RunFreshRecoveryClaimer)
	if won, err := claims.ClaimFreshRecovery(t.Context(), source.ID, source.LifecycleVersion, hex.EncodeToString(hash[:]), true); err != nil || !won {
		t.Fatalf("source fixture claim: %v %v", won, err)
	}
	replacement := &domain.Run{ID: uuid.New(), TaskID: source.TaskID, SourceRunIDs: []uuid.UUID{source.ID}, IdempotencyKey: "resume-from-failed:" + source.ID.String(), Status: domain.RunStatusPending}
	if err := repos.Runs.Create(t.Context(), replacement); err != nil {
		t.Fatal(err)
	}
	if got, err := svc.FreshRecoveryAccepted(t.Context(), source.ID, ""); got != nil || err == nil || domain.IsPreEffectRefusal(err) {
		t.Fatalf("dropped binding was accepted or prior effects called refused: %v %v", got, err)
	}
	if got, err := svc.ResumeFromFailedRun(t.Context(), req); got != nil || err == nil || domain.IsPreEffectRefusal(err) {
		t.Fatalf("lost-binding replay was exposed as qualified identity: %v %v", got, err)
	}
}

func TestResumeFromFailedRunPreservesTaskPinsEnvironmentAndReplay(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	profile, task, prior := seedFailedRun(t, svc, repos, "keep the original assignment", nil)
	prior.ResolvedConfig.Model = "mock-model"
	prior.ResolvedConfig.RoleRef = "code.default"
	prior.ResolvedConfig.MaxTurns = 7
	prior.ResolvedConfig.Timeout = 2 * time.Minute
	prior.ResolvedConfig.SkillPack = []string{"scientific-debugging"}
	prior.ResolvedConfig.SkillExperimentID = "pinned-experiment"
	prior.ResolvedConfig.ToolRestrictionPolicy = domain.ToolRestrictionPolicyAdvisory
	prior.ResolvedConfig.DeniedPaths = []string{"private"}
	prior.ResolvedConfig.NetworkAccess = domain.NetworkAccessNone
	// The mutable profile has drifted. Recovery must not import its new
	// capabilities or silently replace the original experiment/permission policy.
	profile.SkillPack = []string{"different-skill"}
	profile.SkillExperimentID = "different-experiment"
	profile.ToolRestrictionPolicy = domain.ToolRestrictionPolicyEnforced
	if err := repos.Profiles.Update(t.Context(), profile); err != nil {
		t.Fatal(err)
	}
	prior.CustomEnv = map[string]string{"VROOLI_WORK_ID": "original-work"}
	prior.ConversationID = uuid.NewString()
	prior.OwnerSubject = "original-owner"
	expires := time.Now().Add(time.Hour)
	prior.OwnerExpiresAt = &expires
	prior.OwnerScopes = []string{"agent-manager:run:read"}
	if err := repos.Runs.Update(t.Context(), prior); err != nil {
		t.Fatal(err)
	}
	request := orchestration.ResumeFromFailedRunRequest{RunID: prior.ID}
	got, err := svc.ResumeFromFailedRun(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if got.TaskID != task.ID || got.ConversationID != prior.ConversationID || got.OwnerSubject != prior.OwnerSubject || !reflect.DeepEqual(got.OwnerScopes, prior.OwnerScopes) || !reflect.DeepEqual(got.CustomEnv, prior.CustomEnv) {
		t.Errorf("fresh recovery lost task/lineage/owner/environment: task=%s conversation=%s owner=%s env=%v", got.TaskID, got.ConversationID, got.OwnerSubject, got.CustomEnv)
	}
	if got.ResolvedConfig.Model != "mock-model" || got.ResolvedConfig.MaxTurns != 7 || got.ResolvedConfig.Timeout != 2*time.Minute {
		t.Errorf("fresh recovery lost pins: %+v", got.ResolvedConfig)
	}
	if !reflect.DeepEqual(got.ResolvedConfig.SkillPack, prior.ResolvedConfig.SkillPack) || got.ResolvedConfig.SkillExperimentID != prior.ResolvedConfig.SkillExperimentID || got.ResolvedConfig.ToolRestrictionPolicy != prior.ResolvedConfig.ToolRestrictionPolicy || !reflect.DeepEqual(got.ResolvedConfig.DeniedPaths, prior.ResolvedConfig.DeniedPaths) || got.ResolvedConfig.NetworkAccess != prior.ResolvedConfig.NetworkAccess {
		t.Fatalf("fresh recovery changed immutable policy: %+v", got.ResolvedConfig)
	}
	replay, err := svc.ResumeFromFailedRun(t.Context(), request)
	if err != nil || replay.ID != got.ID {
		t.Errorf("recovery replay launched another run: got=%v err=%v", replay, err)
	}
}

func TestResumeFromFailedRunRejectsRetainedExcludedModelBeforeReplacementClaim(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	_, _, source := seedFailedRun(t, svc, repos, "original", nil)
	source.ResolvedConfig.Model = "blocked-model"
	source.ResolvedConfig.PolicySnapshot = &domain.ExecutionPolicySnapshot{
		SelectedCandidate: domain.ExecutionCandidate{
			RunnerType:     domain.RunnerTypeClaudeCode,
			SelectionType:  domain.ModelSelectionTypeModel,
			Model:          "blocked-model",
			ExcludedModels: []string{"blocked-model"},
		},
	}
	if err := repos.Runs.Update(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	request := orchestration.ResumeFromFailedRunRequest{RunID: source.ID}
	got, err := svc.ResumeFromFailedRun(t.Context(), request)
	if got != nil || err == nil || !domain.IsPreEffectRefusal(err) || !strings.Contains(err.Error(), "excluded") {
		t.Fatalf("retained excluded recovery escaped admission: run=%v err=%v", got, err)
	}
	claims := repos.Runs.(repository.RunFreshRecoveryClaimer)
	if hash, readErr := claims.GetFreshRecoveryClaim(t.Context(), source.ID); readErr != nil || hash != "" {
		t.Fatalf("rejected recovery left a source claim: hash=%q err=%v", hash, readErr)
	}
	if accepted, readErr := svc.FreshRecoveryAccepted(t.Context(), source.ID, ""); readErr != nil || accepted != nil {
		t.Fatalf("rejected recovery created a replacement receipt: run=%v err=%v", accepted, readErr)
	}
}

func TestResumeFromFailedRunRejectsChangedRecoveryRequest(t *testing.T) {
	for _, changed := range []string{"context", "attachments"} {
		t.Run(changed, func(t *testing.T) {
			svc, repos := newResumeTestOrchestrator(t)
			_, _, prior := seedFailedRun(t, svc, repos, "original", nil)
			req := orchestration.ResumeFromFailedRunRequest{RunID: prior.ID, CustomContext: "finish the original work"}
			if _, err := svc.ResumeFromFailedRun(t.Context(), req); err != nil {
				t.Fatal(err)
			}
			if changed == "context" {
				req.CustomContext = "different assignment"
			} else {
				req.AttachmentIDs = []string{"different-image"}
			}
			if _, err := svc.ResumeFromFailedRun(t.Context(), req); err == nil {
				t.Fatal("recovery accepted a different payload under the consumed source key")
			}
		})
	}
}

func TestFreshRecoveryReceiptSurvivesCacheExpiryAndReplacementDeletion(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	_, _, prior := seedFailedRun(t, svc, repos, "original", nil)
	req := orchestration.ResumeFromFailedRunRequest{RunID: prior.ID, CustomContext: "preserve this exact context"}
	recovered, err := svc.ResumeFromFailedRun(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	key := "resume-from-failed:" + prior.ID.String()
	// Exercise real expiration/cleanup through the cache's owning API, without
	// changing the source or creating a fake native session database.
	if err := repos.Idempotency.Fail(t.Context(), key); err != nil {
		t.Fatal(err)
	}
	if _, err := repos.Idempotency.Reserve(t.Context(), key, -time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := repos.Idempotency.CleanupExpired(t.Context()); err != nil {
		t.Fatal(err)
	}
	restarted := resumeTestOrchestrator(t, repos, nil)
	accepted, err := restarted.FreshRecoveryAccepted(t.Context(), prior.ID, req.CustomContext)
	if err != nil || accepted == nil || accepted.ID != recovered.ID {
		t.Fatalf("durable receipt lost across owner/cache expiry: run=%v err=%v", accepted, err)
	}
	if replayed, err := restarted.ResumeFromFailedRun(t.Context(), req); err != nil || replayed.ID != recovered.ID {
		t.Fatalf("exact replay did not return original replacement: run=%v err=%v", replayed, err)
	}
	if _, err := restarted.FreshRecoveryAccepted(t.Context(), prior.ID, "different context"); err == nil {
		t.Fatal("receipt accepted a different request")
	}
	if err := repos.Runs.Delete(t.Context(), recovered.ID); err != nil {
		t.Fatal(err)
	}
	if accepted, err := restarted.FreshRecoveryAccepted(t.Context(), prior.ID, req.CustomContext); accepted != nil || domain.IsPreEffectRefusal(err) {
		t.Fatalf("deleted replacement was treated as readable acceptance: %v %v", accepted, err)
	}
	if _, err := restarted.ResumeFromFailedRun(t.Context(), req); err == nil || domain.IsPreEffectRefusal(err) {
		t.Fatalf("consumed source must stay unresolved, not retryable: %v", err)
	}
	if replacement, err := repos.Runs.GetByIdempotencyKey(t.Context(), key); replacement != nil || domain.IsPreEffectRefusal(err) {
		t.Fatalf("deleted recovery was replaced: %v %v", replacement, err)
	}
}

func TestFreshRecoveryReceiptMissingIsReadOnlyAndLineageMustMatch(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	_, _, source := seedFailedRun(t, svc, repos, "original", nil)
	if got, err := svc.FreshRecoveryAccepted(t.Context(), source.ID, "exact"); err != nil || got != nil {
		t.Fatalf("missing receipt: %v %v", got, err)
	}
	claims := repos.Runs.(repository.RunFreshRecoveryClaimer)
	if hash, err := claims.GetFreshRecoveryClaim(t.Context(), source.ID); err != nil || hash != "" {
		t.Fatal("read-only receipt created a source claim")
	}
	req := orchestration.ResumeFromFailedRunRequest{RunID: source.ID, CustomContext: "exact"}
	_, err := svc.ResumeFromFailedRun(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	// A matching key alone cannot certify the wrong source lineage.
	reader := orchestration.New(repos.Profiles, repos.Tasks, &wrongRecoveryLineageRuns{repos.Runs, claims})
	if got, err := reader.FreshRecoveryAccepted(t.Context(), source.ID, "exact"); err == nil || got != nil {
		t.Fatal("receipt accepted mismatched source lineage")
	}
}

func TestFreshRecoveryClaimSurvivesCrashBeforeAcceptance(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	_, _, source := seedFailedRun(t, svc, repos, "original", nil)
	req := orchestration.ResumeFromFailedRunRequest{RunID: source.ID, CustomContext: "exact"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	claims := repos.Runs.(repository.RunFreshRecoveryClaimer)
	if won, err := claims.ClaimFreshRecovery(t.Context(), source.ID, source.LifecycleVersion, hex.EncodeToString(sum[:]), false); err != nil || !won {
		t.Fatalf("claim: %v %v", won, err)
	}
	restarted := resumeTestOrchestrator(t, repos, nil)
	if got, err := restarted.FreshRecoveryAccepted(t.Context(), source.ID, req.CustomContext); got != nil || err != nil {
		t.Fatalf("unresolved claim invented acceptance: %v %v", got, err)
	}
	if _, err := restarted.ResumeFromFailedRun(t.Context(), req); err == nil || domain.IsPreEffectRefusal(err) {
		t.Fatalf("crashed claimant was treated as safe to retry: %v", err)
	}
	if replacement, err := repos.Runs.GetByIdempotencyKey(t.Context(), "resume-from-failed:"+source.ID.String()); err != nil || replacement != nil {
		t.Fatalf("crash caused replacement dispatch: %v %v", replacement, err)
	}
}

func TestFreshRecoveryIndependentOwnersReuseOneReplacement(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	_, _, source := seedFailedRun(t, svc, repos, "original", nil)
	req := orchestration.ResumeFromFailedRunRequest{RunID: source.ID, CustomContext: "same work"}
	owners := []*orchestration.Orchestrator{svc, resumeTestOrchestrator(t, repos, nil)}
	start := make(chan struct{})
	ids := make(chan uuid.UUID, len(owners))
	var wg sync.WaitGroup
	for _, owner := range owners {
		wg.Add(1)
		go func(owner *orchestration.Orchestrator) {
			defer wg.Done()
			<-start
			run, err := owner.ResumeFromFailedRun(t.Context(), req)
			if err == nil {
				ids <- run.ID
			}
		}(owner)
	}
	close(start)
	wg.Wait()
	close(ids)
	accepted, err := svc.FreshRecoveryAccepted(t.Context(), source.ID, req.CustomContext)
	if err != nil || accepted == nil {
		t.Fatalf("neither owner accepted recovery: %v %v", accepted, err)
	}
	for id := range ids {
		if id != accepted.ID {
			t.Fatal("independent owners returned different replacement identities")
		}
	}
	runs, err := repos.Runs.ListByTask(t.Context(), source.TaskID, repository.ListFilter{})
	if err != nil || len(runs) != 2 {
		t.Fatalf("want original and one replacement, got %d: %v", len(runs), err)
	}
}

func TestFreshRecoveryAutomaticReceiptReplayAndNoContinuationAfterClaim(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	_, _, source := seedFailedRun(t, svc, repos, "original", nil)
	registry := runner.NewRegistry()
	opencode := runner.NewMockRunner(domain.RunnerTypeOpenCode)
	if err := registry.Register(opencode); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(runner.NewMockRunner(domain.RunnerTypeClaudeCode)); err != nil {
		t.Fatal(err)
	}
	orchestration.WithRunners(registry)(svc)
	source.SessionID = "session-whose-native-store-is-missing"
	source.ResolvedConfig.RunnerType = domain.RunnerTypeOpenCode
	source.ResolvedConfig.Model = "mock-model"
	if err := repos.Runs.Update(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	req := orchestration.ResumeFromFailedRunRequest{RunID: source.ID, CustomContext: "keep working"}
	// Seed a real owner acceptance through explicit fresh recovery. Automatic
	// replay must consult that exact receipt before native/physical preflight;
	// it must not require a new proof or dispatch another executor.
	recovered, err := svc.ResumeFromFailedRun(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	restarted := resumeTestOrchestrator(t, repos, nil, orchestration.WithIdempotency(nil))
	if replayed, err := restarted.RecoverMissingSessionRun(t.Context(), req); err != nil || replayed.ID != recovered.ID {
		t.Fatalf("accepted auto recovery was redispatched or lost: %v %v", replayed, err)
	}
	// A fresh reader must remain fenced even if the provider later reports
	// that continuation is supported/ready. The repository, not wakeMu, owns it.
	var continued atomic.Int32
	ready := runner.NewMockRunner(domain.RunnerTypeOpenCode)
	caps := ready.Capabilities()
	caps.SupportsContinuation = true
	ready.SetCapabilities(caps)
	ready.ContinueFunc = func(context.Context, runner.ContinueRequest) (*runner.ExecuteResult, error) {
		continued.Add(1)
		return &runner.ExecuteResult{Success: true}, nil
	}
	readyRegistry := runner.NewRegistry()
	if err := readyRegistry.Register(ready); err != nil {
		t.Fatal(err)
	}
	orchestration.WithRunners(readyRegistry)(restarted)
	if _, err := restarted.ContinueRun(t.Context(), orchestration.ContinueRunRequest{RunID: source.ID, Message: "must not reactivate"}); err == nil {
		t.Fatal("continuation reactivated an already replaced source")
	}
	if continued.Load() != 0 {
		t.Fatal("claimed source started a continuation executor")
	}
}

func TestFreshRecoveryKeepsAdmissionThroughSourceClaimAndCreation(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	gate := maintenance.NewGate(maintenance.NewRepository(db))
	claims := &maintenanceClosingRecoveryClaims{
		RunRepository: repos.Runs, RunFreshRecoveryClaimer: repos.Runs.(repository.RunFreshRecoveryClaimer), gate: gate,
	}
	repos.Runs = claims
	svc := resumeTestOrchestrator(t, repos, events, orchestration.WithMaintenanceGate(gate))
	_, task, source := seedFailedRun(t, svc, repos, "original", nil)
	req := orchestration.ResumeFromFailedRunRequest{RunID: source.ID, CustomContext: "finish admitted work"}
	recovered, err := svc.ResumeFromFailedRun(t.Context(), req)
	if err != nil {
		t.Fatalf("maintenance closed after outer admission and stranded the source claim: %v", err)
	}
	standing, err := gate.Status(t.Context())
	if err != nil || !standing.Closed || standing.Admitting != 0 {
		t.Fatalf("admission did not cover and release durable creation: %+v %v", standing, err)
	}
	accepted, err := svc.FreshRecoveryAccepted(t.Context(), source.ID, req.CustomContext)
	if err != nil || accepted == nil || accepted.ID != recovered.ID {
		t.Fatalf("admitted recovery lost its receipt: %v %v", accepted, err)
	}
	if _, err := svc.CreateRun(t.Context(), orchestration.CreateRunRequest{TaskID: task.ID, Force: true, Prompt: "not admitted", IdempotencyKey: "public-must-stay-fenced"}); err == nil {
		t.Fatal("private nested admission bypass escaped to public Force")
	}
	if claims.admittedCtx == nil {
		t.Fatal("source claim did not receive its admission context")
	}
	if _, err := svc.CreateRun(claims.admittedCtx, orchestration.CreateRunRequest{TaskID: task.ID, Force: true, Prompt: "expired admission", IdempotencyKey: "expired-context-must-stay-fenced"}); !errors.Is(err, maintenance.ErrClosed) {
		t.Fatalf("released recovery context bypassed the public creation gate: %v", err)
	}
}

type maintenanceClosingRecoveryClaims struct {
	repository.RunRepository
	repository.RunFreshRecoveryClaimer
	gate        *maintenance.Gate
	admittedCtx context.Context
}

func TestFreshRecoveryReleasesOnlyProvenPreEffectRefusal(t *testing.T) {
	for _, mode := range []string{"refused", "ambiguous", "release-error"} {
		t.Run(mode, func(t *testing.T) {
			_, repos := newResumeTestOrchestrator(t)
			wrapped := &refusedRecoveryCreation{RunRepository: repos.Runs, RunFreshRecoveryClaimer: repos.Runs.(repository.RunFreshRecoveryClaimer), mode: mode}
			repos.Runs = wrapped
			svc := resumeTestOrchestrator(t, repos, nil)
			_, _, source := seedFailedRun(t, svc, repos, "original", nil)
			_, err := svc.ResumeFromFailedRun(t.Context(), orchestration.ResumeFromFailedRunRequest{RunID: source.ID})
			if err == nil {
				t.Fatal("creation failure was not returned")
			}
			hash, readErr := wrapped.GetFreshRecoveryClaim(t.Context(), source.ID)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if mode == "refused" {
				if hash != "" || !domain.IsPreEffectRefusal(err) {
					t.Fatalf("known refusal stranded claim or lost class: hash=%q err=%v", hash, err)
				}
			} else if hash == "" || domain.IsPreEffectRefusal(err) {
				t.Fatalf("ambiguity released claim or became refusal: hash=%q err=%v", hash, err)
			}
		})
	}
}

type refusedRecoveryCreation struct {
	repository.RunRepository
	repository.RunFreshRecoveryClaimer
	claimed bool
	mode    string
}

func (r *refusedRecoveryCreation) ClaimFreshRecovery(ctx context.Context, id uuid.UUID, version int64, hash string, allowCancelled bool) (bool, error) {
	won, err := r.RunFreshRecoveryClaimer.ClaimFreshRecovery(ctx, id, version, hash, allowCancelled)
	r.claimed = won
	return won, err
}

func (r *refusedRecoveryCreation) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Run, error) {
	if r.claimed {
		// Inject the owner's delivery classification at createRun's initial
		// durable lookup, before it can reserve or create anything.
		err := errors.New("creation outcome unavailable")
		if r.mode != "ambiguous" {
			err = domain.RefuseBeforeEffects(err)
		}
		return nil, err
	}
	return r.RunRepository.GetByIdempotencyKey(ctx, key)
}

func (r *refusedRecoveryCreation) ReleaseFreshRecoveryClaim(ctx context.Context, id uuid.UUID, version int64, hash string) (bool, error) {
	if r.mode == "release-error" {
		return false, errors.New("release outcome unavailable")
	}
	return r.RunFreshRecoveryClaimer.ReleaseFreshRecoveryClaim(ctx, id, version, hash)
}

func (r *maintenanceClosingRecoveryClaims) ClaimFreshRecovery(ctx context.Context, id uuid.UUID, version int64, hash string, allowCancelled bool) (bool, error) {
	r.admittedCtx = ctx
	won, err := r.RunFreshRecoveryClaimer.ClaimFreshRecovery(ctx, id, version, hash, allowCancelled)
	if err != nil || !won {
		return won, err
	}
	_, err = r.gate.Enter(ctx, "fixture-owner", "close between source claim and nested creation")
	return won, err
}

type wrongRecoveryLineageRuns struct {
	repository.RunRepository
	repository.RunFreshRecoveryClaimer
}

func (r *wrongRecoveryLineageRuns) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Run, error) {
	run, err := r.RunRepository.GetByIdempotencyKey(ctx, key)
	if run != nil {
		run.SourceRunIDs = []uuid.UUID{uuid.New()}
	}
	return run, err
}

// TestCanResumeFromFailureRun_AllowsTerminalFailures asserts the predicate
// allows resume only for failed/cancelled runs and rejects everything else.
func TestCanResumeFromFailureRun_AllowsTerminalFailures(t *testing.T) {
	cases := []struct {
		name   string
		status domain.RunStatus
		want   bool
	}{
		{"failed", domain.RunStatusFailed, true},
		{"cancelled", domain.RunStatusCancelled, true},
		{"complete", domain.RunStatusComplete, false},
		{"running", domain.RunStatusRunning, false},
		{"starting", domain.RunStatusStarting, false},
		{"pending", domain.RunStatusPending, false},
		{"needs_review", domain.RunStatusNeedsReview, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := &domain.Run{Status: tc.status}
			got, reason := domain.CanResumeFromFailureRun(run)
			if got != tc.want {
				t.Fatalf("status=%s: want allowed=%v, got %v (reason=%q)", tc.status, tc.want, got, reason)
			}
			if !got && reason == "" {
				t.Fatalf("status=%s: rejection must include a non-empty reason", tc.status)
			}
		})
	}
}

func TestAutomaticMissingSessionRecoveryRejectsCancelledImportedAndLive(t *testing.T) {
	for _, status := range []domain.RunStatus{domain.RunStatusCancelled, domain.RunStatusRunning, domain.RunStatusFailed} {
		t.Run(string(status), func(t *testing.T) {
			svc, repos := newResumeTestOrchestrator(t)
			_, _, prior := seedFailedRun(t, svc, repos, "original", nil)
			prior.Status = status
			prior.SessionID = "ses_retained"
			prior.ResolvedConfig.RunnerType = domain.RunnerTypeOpenCode
			if status == domain.RunStatusFailed {
				prior.ExecutionMode = domain.ExecutionModeImported
			}
			if err := repos.Runs.Update(t.Context(), prior); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.RecoverMissingSessionRun(t.Context(), orchestration.ResumeFromFailedRunRequest{RunID: prior.ID}); err == nil {
				t.Fatal("automatic recovery accepted forbidden source")
			}
			runs, err := svc.ListRuns(t.Context(), orchestration.RunListOptions{TagPrefix: prior.Tag})
			if err != nil || len(runs) != 1 {
				t.Fatalf("refusal created a run: %d %v", len(runs), err)
			}
		})
	}
}

func TestFreshRecoveryRequiresCanonicalExecutorExclusionBeforeClaim(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	_, _, source := seedFailedRun(t, svc, repos, "original", nil)
	source.SessionID = "ses_retained"
	source.ResolvedConfig.RunnerType = domain.RunnerTypeOpenCode
	if err := repos.Runs.Update(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	// A missing control-plane client is unknown physical scope, never absence.
	// No native session store is fabricated and no executor can be launched.
	t.Setenv("PATH", t.TempDir())
	_, err := svc.RecoverMissingSessionRun(t.Context(), orchestration.ResumeFromFailedRunRequest{RunID: source.ID})
	if err == nil || !strings.Contains(err.Error(), "control-plane") || !domain.IsPreEffectRefusal(err) {
		t.Fatalf("missing canonical exclusion did not refuse before effects: %v", err)
	}
	hash, err := repos.Runs.(repository.RunFreshRecoveryClaimer).GetFreshRecoveryClaim(t.Context(), source.ID)
	if err != nil || hash != "" {
		t.Fatalf("unknown physical scope claimed the source: %q %v", hash, err)
	}
	accepted, err := svc.FreshRecoveryAccepted(t.Context(), source.ID, "")
	if err != nil || accepted != nil {
		t.Fatalf("unknown physical scope created a replacement: %v %v", accepted, err)
	}
}

func TestCanResumeFromFailureRun_NilRun(t *testing.T) {
	allowed, reason := domain.CanResumeFromFailureRun(nil)
	if allowed {
		t.Fatal("nil run must not be resumable")
	}
	if reason == "" {
		t.Fatal("nil rejection must include a reason")
	}
}

// newResumeTestOrchestrator wires the orchestrator with the runners +
// storage that ResumeFromFailedRun exercises. Mirrors the investigation
// attachments test setup.
func newResumeTestOrchestrator(t *testing.T) (*orchestration.Orchestrator, *database.Repositories) {
	t.Helper()

	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	return resumeTestOrchestrator(t, repos, eventStore), repos
}

func resumeTestOrchestrator(t *testing.T, repos *database.Repositories, eventStore event.Store, options ...orchestration.Option) *orchestration.Orchestrator {
	t.Helper()
	// These are admission/receipt tests, not execution tests. Hold the real
	// dispatcher's starting slot so accepted jobs remain pending until cleanup;
	// no async mock executor can race a deleted row or a temporary run root.
	dispatcher := spawn.New(spawn.Config{MaxStartingConcurrency: 1, QueueCapacity: 20})
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	if err := dispatcher.Enqueue(&spawn.Job{RunID: uuid.New(), Fn: func(spawn.StartedFn) {
		close(entered)
		<-release
		close(done)
	}}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("test dispatcher did not hold its starting slot")
	}
	t.Cleanup(func() {
		dispatcher.Close()
		close(release)
		<-done
	})

	registry := runner.NewRegistry()
	claudeRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	claudeRunner.SetAvailable(true, "mock claude available")
	if err := registry.Register(claudeRunner); err != nil {
		t.Fatalf("register claude runner: %v", err)
	}

	mockStorage := storage.NewMockService()

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          5 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: false,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
		newTestRolePolicyOption(t),
		orchestration.WithInvestigationSettings(repos.InvestigationSettings),
		orchestration.WithAttachmentStorage(mockStorage),
		orchestration.WithRunStateRoot(t.TempDir()),
		orchestration.WithSpawnDispatcher(dispatcher),
	)
	for _, option := range options {
		option(svc)
	}
	return svc
}

// seedFailedRun inserts a profile, task, and a failed run that points to
// them. Returns the created records for assertions.
func seedFailedRun(
	t *testing.T,
	svc *orchestration.Orchestrator,
	repos *database.Repositories,
	originalDescription string,
	originalAttachments []domain.ContextAttachment,
) (*domain.AgentProfile, *domain.Task, *domain.Run) {
	t.Helper()
	ctx := context.Background()

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:       "resume-source-profile",
		ProfileKey: "resume-source-" + uuid.New().String()[:8],

		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, RoleRef: "code.default",
	})

	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:              "resume-source-task",
		Description:        originalDescription,
		ScopePath:          "src/",
		ProjectRoot:        "/tmp/resume-test-root",
		ContextAttachments: originalAttachments,
	})

	now := time.Now()
	failedRunID := uuid.New()
	failedRun := &domain.Run{
		ID:             failedRunID,
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Tag:            "resume-test-" + failedRunID.String()[:6],
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusFailed,
		Phase:          domain.RunPhaseCompleted,
		StartedAt:      &now,
		EndedAt:        &now,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		ApprovalState:  domain.ApprovalStateNone,
		ErrorMsg:       "simulated failure for test",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Runs.Create(ctx, failedRun); err != nil {
		t.Fatalf("insert failed run: %v", err)
	}
	return profile, task, failedRun
}

// TestResumeFromFailedRun_BuildsAttachments asserts that the resumed task
// carries forward the original task's attachments AND adds previous-attempt
// context (overview + timeline) plus the user's custom guidance.
func TestResumeFromFailedRun_BuildsAttachments(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	ctx := context.Background()

	originalAttachments := []domain.ContextAttachment{
		{
			Type:    "note",
			Key:     "original-spec",
			Label:   "Original Spec",
			Content: "build the widget",
			Format:  "markdown",
		},
	}

	_, _, failedRun := seedFailedRun(t, svc, repos, "Build the widget end to end.", originalAttachments)

	resumedRun, err := svc.ResumeFromFailedRun(ctx, orchestration.ResumeFromFailedRunRequest{
		RunID:         failedRun.ID,
		CustomContext: "skip the part that already wrote the README",
	})
	if err != nil {
		t.Fatalf("ResumeFromFailedRun: %v", err)
	}
	if resumedRun == nil {
		t.Fatal("expected a resumed run, got nil")
	}

	resumedTask, err := svc.GetTask(ctx, resumedRun.TaskID)
	if err != nil {
		t.Fatalf("GetTask for resumed run: %v", err)
	}

	if resumedTask.ID != failedRun.TaskID {
		t.Fatal("fresh recovery must retain the original task identity")
	}
	if resumedTask.ProjectRoot != "/tmp/resume-test-root" {
		t.Errorf("resumed task project root = %q, want inherited /tmp/resume-test-root", resumedTask.ProjectRoot)
	}

	keys := map[string]bool{}
	for _, att := range resumedTask.ContextAttachments {
		keys[att.Key] = true
	}

	if !keys["original-spec"] {
		t.Error("resumed task missing original task's ContextAttachment (key=original-spec)")
	}

	hasPreviousReport := false
	for k := range keys {
		if strings.HasPrefix(k, "previous-run-report-prev-") {
			hasPreviousReport = true
			break
		}
	}
	if hasPreviousReport {
		t.Errorf("recovery context must not mutate original task attachments; keys=%v", keys)
	}

	if keys["user-resume-context"] {
		t.Error("recovery guidance mutated original task attachments")
	}

	if resumedTask.Description != "Build the widget end to end." {
		t.Error("recovery changed the original task description")
	}
}

// TestResumeFromFailedRun_LinksLineage asserts the resumed run records the
// original failed run in SourceRunIDs so downstream consumers (UI, list
// filters) can trace ancestry.
func TestResumeFromFailedRun_LinksLineage(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	ctx := context.Background()

	_, _, failedRun := seedFailedRun(t, svc, repos, "do the thing", nil)

	resumedRun, err := svc.ResumeFromFailedRun(ctx, orchestration.ResumeFromFailedRunRequest{RunID: failedRun.ID})
	if err != nil {
		t.Fatalf("ResumeFromFailedRun: %v", err)
	}

	got, err := svc.ListRuns(ctx, orchestration.RunListOptions{
		TagPrefix: failedRun.Tag,
	})
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}

	var foundResumed bool
	for _, run := range got {
		if run.ID == resumedRun.ID {
			foundResumed = true
			if !strings.HasSuffix(run.Tag, "-resume") {
				t.Errorf("resumed run tag = %q, want a -resume suffix for traceability", run.Tag)
			}
		}
	}
	if !foundResumed {
		t.Errorf("ListRuns by tag prefix %q did not return the resumed run; got %d runs", failedRun.Tag, len(got))
	}
}

// TestResumeFromFailedRun_InheritsProfile asserts the resumed run uses the
// same agent profile as the failed run (so it has the same tools and
// permissions the original attempt had).
func TestResumeFromFailedRun_InheritsProfile(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	ctx := context.Background()

	profile, _, failedRun := seedFailedRun(t, svc, repos, "do the thing", nil)

	resumedRun, err := svc.ResumeFromFailedRun(ctx, orchestration.ResumeFromFailedRunRequest{RunID: failedRun.ID})
	if err != nil {
		t.Fatalf("ResumeFromFailedRun: %v", err)
	}

	if resumedRun.AgentProfileID == nil {
		t.Fatal("resumed run must inherit the failed run's AgentProfileID, got nil")
	}
	if *resumedRun.AgentProfileID != profile.ID {
		t.Errorf("resumed run profile = %s, want inherited %s", resumedRun.AgentProfileID, profile.ID)
	}
}

// TestResumeFromFailedRun_RejectsRunning asserts the orchestrator refuses to
// resume a run that's still in flight, mirroring the predicate's contract.
func TestResumeFromFailedRun_RejectsRunning(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	ctx := context.Background()

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:       "running-profile",
		ProfileKey: "running-" + uuid.New().String()[:8],

		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, RoleRef: "code.default",
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:       "running-task",
		Description: "in flight",
		ScopePath:   "src/",
	})

	now := time.Now()
	runID := uuid.New()
	if err := repos.Runs.Create(ctx, &domain.Run{
		ID:             runID,
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Tag:            runID.String(),
		Status:         domain.RunStatusRunning,
		Phase:          domain.RunPhaseExecuting,
		StartedAt:      &now,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("insert running run: %v", err)
	}

	_, err := svc.ResumeFromFailedRun(ctx, orchestration.ResumeFromFailedRunRequest{RunID: runID})
	if err == nil {
		t.Fatal("expected ResumeFromFailedRun to reject a still-running run, got nil error")
	}
}

// TestResumeTag_AppendsSuffix is a small regression for the tag derivation
// helper to make sure we never double-append -resume.
func TestResumeTag_AppendsSuffixOnce(t *testing.T) {
	svc, repos := newResumeTestOrchestrator(t)
	ctx := context.Background()

	_, _, failedRun := seedFailedRun(t, svc, repos, "x", nil)
	failedRun.Tag += "-resume"
	if err := repos.Runs.Update(ctx, failedRun); err != nil {
		t.Fatalf("mark failed run tag as already resumed: %v", err)
	}

	resumed, err := svc.ResumeFromFailedRun(ctx, orchestration.ResumeFromFailedRunRequest{RunID: failedRun.ID})
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if strings.Count(resumed.Tag, "-resume") != 1 {
		t.Errorf("resumed tag = %q, want exactly one -resume suffix (no double-append)", resumed.Tag)
	}
}
