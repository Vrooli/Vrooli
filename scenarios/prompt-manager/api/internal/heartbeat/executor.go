package heartbeat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"

	"github.com/google/uuid"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ExecutionResult represents the result of a heartbeat execution
type ExecutionResult struct {
	TeamID    string
	AgentID   string
	RunID     string
	Status    string
	StartedAt time.Time
	EndedAt   *time.Time
	LogPath   string
	Error     error
}

// TeamExecStoreRegistrar is the subset of TeamExecutionStore that Executor
// needs to keep a member's durable execution obligation in step with owner
// dispatch. Kept as an interface so tests can inject a fake without depending
// on the full store.
type TeamExecStoreRegistrar interface {
	// BeginDispatch persists the dispatch intent (including its idempotency
	// key and owner request essentials) before the owner start request, so a
	// crash mid-dispatch recovers as a visible, replayable obligation rather
	// than a silently free slot.
	BeginDispatch(teamID, agentID string, intent DispatchIntent) error
	// ConfirmTaskDispatch persists exact task confirmation before CreateRun.
	ConfirmTaskDispatch(teamID, agentID string) error
	// CancelUnsentTaskDispatch durably removes only the matching unconfirmed
	// task intent after the owner proves the request was never sent.
	CancelUnsentTaskDispatch(teamID, agentID, taskID, idempotencyKey, taskSnapshot string) error
	// SetRunningRunID binds a confirmed run identity, clearing uncertainty.
	SetRunningRunID(teamID, agentID, runID string, ownerStartedAt ...string)
	// MarkApprovalPending retains an existing run intent while the owner waits
	// for explicit operator approval.
	MarkApprovalPending(teamID, agentID string)
	// MarkAdmissionDeferred retains an exact dispatch intent after a typed
	// definitive capacity refusal.
	MarkAdmissionDeferred(teamID, agentID string) error
}

// Executor handles the actual execution of heartbeats
type Executor struct {
	teamStore        *store.FileTeamStore
	agentStore       *store.FileAgentStore
	agentClient      AgentClient
	vrooliRoot       string
	promptBuilder    *PromptBuilder
	runRegistry      *RunRegistry
	handoffExtractor HandoffExtractor
	teamExecStore    TeamExecStoreRegistrar
	OnRunComplete    func(teamID, agentID, runID string)
	TeamReviewURL    func(context.Context, string) string

	// Completion waiters outlive the request that started them, so they are
	// tracked here to give the executor a point where that work is known to
	// be finished. See Shutdown.
	mu           sync.Mutex
	waiters      map[int64]context.CancelFunc
	waiterKeys   map[string]int64
	nextWaiterID int64
	stopped      bool
	inflight     sync.WaitGroup
}

const heartbeatAttentionFailureThreshold = 3

func heartbeatFailureEpisodeID(teamID, agentID string, config *store.HeartbeatConfig) string {
	resetAt := "before-first-success"
	if config != nil {
		lastSuccess := config.LastSuccessfulExecution
		if lastSuccess == nil && config.LastExecution != nil && config.LastExecution.Status == store.HeartbeatStatusCompleted {
			lastSuccess = config.LastExecution
		}
		if lastSuccess != nil {
			resetAt = strings.TrimSpace(lastSuccess.RunID) + "\x00" + strings.TrimSpace(lastSuccess.EndedAt)
		}
	}
	identity := teamID + "\x00" + agentID + "\x00" + resetAt
	digest := sha256.Sum256([]byte(identity))
	return "pm-heartbeat-failure-" + hex.EncodeToString(digest[:])
}

// SetTeamReviewURL wires the existing browser-facing Prompt Manager origin
// resolver. The returned URL is captured on the immutable threshold attempt;
// an unresolved origin yields an omitted link.
func (e *Executor) SetTeamReviewURL(resolver func(context.Context, string) string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.TeamReviewURL = resolver
}

// trackWaiter registers a completion waiter's cancel function so Shutdown can
// reach it. It reports false once the executor has shut down, in which case
// the caller must not start the waiter.
func (e *Executor) trackWaiter(key string, cancel context.CancelFunc) (int64, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopped {
		return 0, false
	}
	if key != "" && e.waiterKeys != nil {
		if _, exists := e.waiterKeys[key]; exists {
			return 0, false
		}
	}
	if e.waiters == nil {
		e.waiters = make(map[int64]context.CancelFunc)
	}
	if e.waiterKeys == nil {
		e.waiterKeys = make(map[string]int64)
	}
	e.nextWaiterID++
	id := e.nextWaiterID
	e.waiters[id] = cancel
	if key != "" {
		e.waiterKeys[key] = id
	}
	// Register work while holding the same lock Shutdown uses to close
	// admission, so Wait cannot race an Add after shutdown begins.
	e.inflight.Add(1)
	return id, true
}

func (e *Executor) scheduleHandoffExtraction(teamID, agentID, runID string, endedAt time.Time) {
	ctx, cancel := context.WithCancel(context.Background())
	key := "handoff:" + heartbeatWaiterKey(teamID, agentID, runID)
	id, started := e.trackWaiter(key, cancel)
	if !started {
		cancel()
		return
	}
	go func() {
		defer e.inflight.Done()
		defer e.releaseWaiter(id, key)
		defer cancel()
		e.extractAndStoreHandoff(ctx, teamID, agentID, runID, endedAt)
	}()
}

func (e *Executor) releaseWaiter(id int64, key string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.waiters, id)
	if key != "" && e.waiterKeys[key] == id {
		delete(e.waiterKeys, key)
	}
}

// Shutdown cancels every in-flight completion waiter and blocks until they
// return. Those waiters run on a background context and keep writing run
// state after the triggering request has been answered, so both a server
// restart and a finishing test need a point where that writing has stopped.
func (e *Executor) Shutdown() {
	e.mu.Lock()
	e.stopped = true
	cancels := make([]context.CancelFunc, 0, len(e.waiters))
	for _, cancel := range e.waiters {
		cancels = append(cancels, cancel)
	}
	e.waiters = nil
	e.waiterKeys = nil
	e.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	e.inflight.Wait()
}

// SetTeamExecStore wires the team execution store so Execute can register
// the agent-manager RunID against the team's running entry. Called from
// main.go after both the executor and the store have been constructed
// (they have a circular dependency at construction time).
func (e *Executor) SetTeamExecStore(s TeamExecStoreRegistrar) {
	e.teamExecStore = s
}

// HydrateRecoveredRuns updates the presentation registry from TeamExecution's
// authoritative startup reconciliation without publishing synthetic starts.
func (e *Executor) HydrateRecoveredRuns(runs []ActiveRun) {
	if e == nil || e.runRegistry == nil {
		return
	}
	e.runRegistry.HydrateQueueOwners(runs)
}

func (e *Executor) isCurrentRunOwner(teamID, agentID, runID string) bool {
	if owner, ok := e.teamExecStore.(interface {
		IsCurrentRun(teamID, agentID, runID string) bool
	}); ok && !owner.IsCurrentRun(teamID, agentID, runID) {
		return false
	}
	if e.runRegistry != nil {
		if active, ok := e.runRegistry.GetActiveRun(teamID, agentID); ok && active.RunID != runID {
			return false
		}
	}
	// Production recovery has the durable TeamExecutionStore owner. Tests and
	// compatibility callers without that optional owner retain the prior path.
	return true
}

// SetContractFindingsProvider forwards the findings source to the executor's
// prompt builder. This is the builder that produces the prompt an agent is
// actually spawned with, so a provider wired only to the preview builder
// would show findings to the operator and never to the member.
func (e *Executor) SetContractFindingsProvider(provider ContractFindingsProvider) {
	if e == nil || e.promptBuilder == nil {
		return
	}
	e.promptBuilder.SetContractFindingsProvider(provider)
}

// SetTeamObjectiveProvider forwards the canonical objective-authority read to
// the executor's prompt builder. This is the builder that produces the prompt an
// agent is spawned with and the one `team prompt-preview` renders, so wiring it
// here keeps preview and runtime identity in agreement.
func (e *Executor) SetTeamObjectiveProvider(provider TeamObjectiveProvider) {
	if e == nil || e.promptBuilder == nil {
		return
	}
	e.promptBuilder.SetTeamObjectiveProvider(provider)
}

// NewExecutor creates a new heartbeat executor
func NewExecutor(
	teamStore *store.FileTeamStore,
	agentStore *store.FileAgentStore,
	agentClient AgentClient,
	vrooliRoot string,
	runRegistry *RunRegistry,
	handoffExtractor HandoffExtractor,
) *Executor {
	if handoffExtractor == nil {
		handoffExtractor = NewSentinelExtractor()
	}
	promptBuilder := NewPromptBuilder(teamStore, agentStore)
	return &Executor{
		teamStore:        teamStore,
		agentStore:       agentStore,
		agentClient:      agentClient,
		vrooliRoot:       vrooliRoot,
		promptBuilder:    promptBuilder,
		runRegistry:      runRegistry,
		handoffExtractor: handoffExtractor,
	}
}

// Execute runs a heartbeat for a team member
func (e *Executor) Execute(ctx context.Context, teamID, agentID, profileKey string) (*ExecutionResult, error) {
	// The real owner HTTP adapter requires verified request-local identity
	// before configuration, source-ledger or task effects.
	if client, ok := e.agentClient.(*AgentManagerClient); ok {
		qualified, err := client.prepareCreateRunCaller(ctx)
		if err != nil {
			return nil, NewDispatchRejectedError(err)
		}
		ctx = qualified
	}
	startedAt := time.Now().UTC()
	result := &ExecutionResult{
		TeamID:    teamID,
		AgentID:   agentID,
		StartedAt: startedAt,
	}

	// Enforce team-level heartbeat gating before any prompt or config work.
	team, err := e.teamStore.Get(ctx, teamID)
	if err != nil {
		result.Error = fmt.Errorf("getting team: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}
	if err := validateTeamEnabled(team); err != nil {
		result.Error = fmt.Errorf("heartbeat blocked: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}
	if err := e.teamStore.EnsureTeamScope(ctx, teamID); err != nil {
		result.Error = fmt.Errorf("ensuring team source-ledger scope: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}

	// Auto-prune stale shared state before building the prompt.
	if _, pruneErr := e.teamStore.PruneSharedState(ctx, teamID); pruneErr != nil {
		log.Printf("Warning: prune shared state for %s: %v", teamID, pruneErr)
	}

	// Resolve default profile key based on runtime mode when not explicitly set.
	if profileKey == "" {
		profileKey, err = DefaultProfileKeyForRuntimeMode(team.Runtime.Mode)
		if err != nil {
			result.Error = fmt.Errorf("resolving heartbeat profile: %w", err)
			result.Status = store.HeartbeatStatusFailed
			return result, result.Error
		}
	}

	contract := team.Contract()

	// Build the prompt (single-process leader-led teams use Claude Code interop)
	var prompt string
	if teamconfig.UsesSingleProcessInterop(contract) {
		prompt, err = e.promptBuilder.BuildTeamLeadPrompt(ctx, teamID, agentID, e.vrooliRoot)
	} else {
		prompt, err = e.BuildPrompt(ctx, teamID, agentID)
	}
	if err != nil {
		result.Error = fmt.Errorf("building prompt: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}

	// Update config with running status
	config, err := e.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if err != nil {
		result.Error = fmt.Errorf("getting heartbeat config: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}
	if config == nil {
		result.Error = fmt.Errorf("heartbeat config not found")
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}
	attemptID := uuid.NewString()
	workReferences, err := e.orchestratorWorkReferences(ctx, team, agentID, profileKey, config)
	if err != nil {
		result.Error = fmt.Errorf("confirming finite effort leader identity: %w", err)
		if config.LastExecution != nil && config.LastExecution.RunID != "" {
			result.Status = store.HeartbeatStatusObserverUnavailable
			return result, e.retainPriorFiniteOwner(teamID, agentID, config.LastExecution.RunID, result.Error)
		}
		result.Status = store.HeartbeatStatusFailed
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, result.Error, result.Error.Error(), attemptID, profileKey, "", "", "", "resolving_effort_identity", false, false)
		return result, result.Error
	}
	if len(workReferences) > 0 {
		if recovered, handled, recoveryErr := e.recoverPriorFiniteOwner(ctx, team, agentID, config, workReferences); handled {
			if recoveryErr != nil {
				result.Error = recoveryErr
				result.Status = store.HeartbeatStatusObserverUnavailable
				return result, recoveryErr
			}
			return &recovered, nil
		}
	}

	// Generate log path
	timestamp := startedAt.Format("2006-01-02T15-04-05Z")
	logPath := e.teamStore.GetMemberLogPathForContext(ctx, teamID, agentID, timestamp)
	result.LogPath = logPath

	// Update config to running
	admissionOrigin := heartbeatAdmissionOrigin(ctx)
	config.LastExecution = &store.HeartbeatExecResult{
		StartedAt:       startedAt.Format(time.RFC3339),
		Status:          store.HeartbeatStatusRunning,
		LogPath:         logPath,
		AdmissionOrigin: admissionOrigin,
	}
	if err := e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config); err != nil {
		result.Error = fmt.Errorf("updating config to running: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}

	// Guard: agent client must be configured
	if e.agentClient == nil {
		result.Error = fmt.Errorf("agent client is not configured")
		result.Status = store.HeartbeatStatusFailed
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, nil, result.Error.Error(), attemptID, profileKey, "", "", "", "agent_client_missing", true, true)
		return result, result.Error
	}

	task := &Task{
		ID:          uuid.NewString(),
		Title:       fmt.Sprintf("Heartbeat: %s/%s", teamID, agentID),
		Description: prompt,
		ScopePath:   e.vrooliRoot,
		ProjectRoot: e.vrooliRoot,
	}

	// Create a run using the reconciled, scenario-owned profile. Runtime
	// selection belongs to Agent Manager's role resolver, not heartbeat code.
	//
	// Environment carries VROOLI_PROMPT_MANAGER_ATTRIBUTION so the spawned
	// agent's CLI inherits structured attribution and forwards it as the
	// X-Vrooli-Attribution header on every API write (canon:
	// docs/agent-system/RUNTIME_ATTRIBUTION.md § Env-var bridge).
	runTag := fmt.Sprintf("heartbeat-%s-%s-%s", teamID, agentID, timestamp)
	attribKey, attribValue := buildHeartbeatAttributionEnv(teamID, agentID)
	// The idempotency key is derived from the durable dispatch attempt, so a
	// replayed create after an uncertain response resolves to the same run
	// instead of starting a second one.
	dispatchKey := fmt.Sprintf("prompt-manager-heartbeat:%s:%s:%s", teamID, agentID, attemptID)
	workloadKind, workloadKey, workloadInstance := teamWorkloadIdentity(team, dispatchKey)
	environment := map[string]string{attribKey: attribValue}
	addWorkloadEnvironment(environment, workloadKind, workloadKey, workloadInstance)
	runReq := &CreateRunRequest{
		TaskID: task.ID,
		ProfileRef: &ProfileRef{
			ProfileKey: profileKey,
		},
		Tag:            &runTag,
		IdempotencyKey: dispatchKey,
		Environment:    environment,
		WorkReferences: workReferences,
	}

	if e.teamExecStore == nil {
		result.Error = errors.New("durable team execution store is required before heartbeat task dispatch")
		result.Status = store.HeartbeatStatusFailed
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, result.Error, result.Error.Error(), attemptID, profileKey, task.ID, "", runTag, "recording_dispatch_intent", true, true)
		return result, result.Error
	}
	taskSnapshot, snapshotErr := freezeTaskDispatchRequest(task)
	if snapshotErr != nil {
		result.Error = fmt.Errorf("snapshotting task dispatch request: %w", snapshotErr)
		result.Status = store.HeartbeatStatusFailed
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, snapshotErr, result.Error.Error(), attemptID, profileKey, task.ID, "", runTag, "recording_dispatch_intent", true, true)
		return result, result.Error
	}
	requestSnapshot, snapshotErr := json.Marshal(runReq)
	if snapshotErr != nil {
		result.Error = fmt.Errorf("snapshotting run dispatch request: %w", snapshotErr)
		result.Status = store.HeartbeatStatusFailed
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, snapshotErr, result.Error.Error(), attemptID, profileKey, task.ID, "", runTag, "recording_dispatch_intent", true, true)
		return result, result.Error
	}
	callerProof := createRunCaller(ctx)
	callerMode := callerProof.origin
	if !validCreateRunCallerMode(callerMode) {
		callerMode = createRunCallerOriginUnqualified
		if callerProof.authorization != "" || callerProof.runIdentity != "" {
			callerMode = createRunCallerOriginCaller
		}
	}
	intent := DispatchIntent{
		AdmissionOrigin:          heartbeatAdmissionOrigin(ctx),
		IdempotencyKey:           dispatchKey,
		TaskID:                   task.ID,
		TaskRequestSnapshot:      taskSnapshot,
		RunTag:                   runTag,
		WorkloadKind:             workloadKind,
		WorkloadKey:              workloadKey,
		WorkloadInstance:         workloadInstance,
		CreateRunRequestSnapshot: string(requestSnapshot),
		CreateRunCallerMode:      callerMode,
		CreateRunCallerBinding:   callerProof.binding,
	}
	if err := e.teamExecStore.BeginDispatch(teamID, agentID, intent); err != nil {
		result.Error = fmt.Errorf("recording task and run dispatch intent: %w", err)
		result.Status = store.HeartbeatStatusFailed
		reportCtx, cancelReport := boundedHeartbeatReportContext(ctx)
		e.updateConfigFailed(reportCtx, teamID, agentID, config, startedAt, err, result.Error.Error(), attemptID, profileKey, task.ID, "", runTag, "recording_dispatch_intent", true, true)
		cancelReport()
		return result, result.Error
	}

	createdTask, err := resolveTaskDispatch(ctx, e.agentClient, taskSnapshot)
	if err != nil {
		result.Error = fmt.Errorf("creating task: %w", err)
		if IsApprovalPending(err) {
			result.Status = store.HeartbeatStatusApprovalPending
			e.updateConfigApprovalPending(ctx, teamID, agentID, config, startedAt, result.Error.Error(), attemptID, profileKey, task.ID, "", runTag, "creating_task")
			return result, result.Error
		}
		if isPreSendCancellation(err) {
			if cancelErr := e.teamExecStore.CancelUnsentTaskDispatch(teamID, agentID, task.ID, dispatchKey, taskSnapshot); cancelErr != nil {
				retained := NewDispatchUncertainError(&dispatchStageFailure{
					phase: dispatchFailurePhaseTaskCancellation,
					code:  "task_cancellation_failed",
					cause: fmt.Errorf("canceling unsent task dispatch: %w", cancelErr),
				})
				result.Error = retained
				result.Status = store.HeartbeatStatusFailed
				reportCtx, cancelReport := boundedHeartbeatReportContext(ctx)
				e.updateConfigFailed(reportCtx, teamID, agentID, config, startedAt, retained, retained.Error(), attemptID, profileKey, task.ID, "", runTag, "canceling_task_dispatch", false, false)
				cancelReport()
				return result, result.Error
			}
			result.Status = store.HeartbeatStatusCancelled
			reportCtx, cancelReport := boundedHeartbeatReportContext(ctx)
			e.updateConfigCancelled(reportCtx, teamID, agentID, config, startedAt, result.Error.Error(), attemptID, profileKey, task.ID, "", runTag, "creating_task")
			cancelReport()
			return result, result.Error
		}
		if admissionReasonFromError(err) == "grant_limit_reached" {
			if deferErr := e.teamExecStore.MarkAdmissionDeferred(teamID, agentID); deferErr != nil {
				retained := NewDispatchUncertainError(&dispatchStageFailure{
					phase: dispatchFailurePhaseCapacityDeferral,
					code:  "capacity_deferral_failed",
					cause: errors.Join(err, fmt.Errorf("retaining capacity-deferred task intent: %w", deferErr)),
				})
				result.Error = retained
			}
			result.Status = store.HeartbeatStatusFailed
			e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, err, result.Error.Error(), attemptID, profileKey, task.ID, "", runTag, "creating_task", true, true)
			return result, result.Error
		}
		result.Status = store.HeartbeatStatusFailed
		attentionEligible := !IsDispatchUncertain(err)
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, err, result.Error.Error(), attemptID, profileKey, task.ID, "", runTag, "creating_task", attentionEligible, attentionEligible)
		return result, result.Error
	}
	if err := e.teamExecStore.ConfirmTaskDispatch(teamID, agentID); err != nil {
		uncertain := NewDispatchUncertainError(&dispatchStageFailure{
			phase: dispatchFailurePhaseTaskConfirmation,
			code:  "task_confirmation_failed",
			cause: fmt.Errorf("persisting confirmed task stage: %w", err),
		})
		result.Error = uncertain
		result.Status = store.HeartbeatStatusFailed
		reportCtx, cancelReport := boundedHeartbeatReportContext(ctx)
		e.updateConfigFailed(reportCtx, teamID, agentID, config, startedAt, uncertain, uncertain.Error(), attemptID, profileKey, createdTask.ID, "", runTag, "confirming_task", false, false)
		cancelReport()
		return result, result.Error
	}

	run, err := e.agentClient.CreateRun(ctx, runReq)
	if err != nil {
		if IsApprovalPending(err) {
			if e.teamExecStore != nil {
				e.teamExecStore.MarkApprovalPending(teamID, agentID)
			}
			result.Error = fmt.Errorf("creating run: %w", err)
			result.Status = store.HeartbeatStatusApprovalPending
			e.updateConfigApprovalPending(ctx, teamID, agentID, config, startedAt, result.Error.Error(), attemptID, profileKey, createdTask.ID, "", runTag, "creating_run")
			return result, result.Error
		}
		if isPreSendCancellation(err) {
			result.Error = fmt.Errorf("creating run: %w", err)
			result.Status = store.HeartbeatStatusCancelled
			if cancelErr := e.agentClient.CancelTask(ctx, createdTask.ID); cancelErr == nil {
				if deleteErr := e.agentClient.DeleteTask(ctx, createdTask.ID); deleteErr != nil {
					log.Printf("Warning: failed to delete cancelled heartbeat task %s: %v", createdTask.ID, deleteErr)
				}
			}
			e.updateConfigCancelled(ctx, teamID, agentID, config, startedAt, result.Error.Error(), attemptID, profileKey, createdTask.ID, "", runTag, "creating_run")
			return result, result.Error
		}
		if admissionReasonFromError(err) == "grant_limit_reached" {
			// The exact owner refusal proves no run was accepted, but the confirmed
			// task and frozen idempotency key are retained for the next scheduled
			// admission check. Do not cancel/delete the task or rebuild the request.
			result.Error = fmt.Errorf("creating run: %w", err)
			if deferErr := e.teamExecStore.MarkAdmissionDeferred(teamID, agentID); deferErr != nil {
				retained := NewDispatchUncertainError(&dispatchStageFailure{
					phase: dispatchFailurePhaseCapacityDeferral,
					code:  "capacity_deferral_failed",
					cause: errors.Join(err, fmt.Errorf("retaining capacity-deferred run intent: %w", deferErr)),
				})
				result.Error = retained
			}
			result.Status = store.HeartbeatStatusFailed
			e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, err, result.Error.Error(), attemptID, profileKey, createdTask.ID, "", runTag, "creating_run", true, true)
			return result, result.Error
		}
		attentionEligible := !IsDispatchUncertain(err)
		if !attentionEligible {
			// The owner may have accepted the run. Do not cancel the task and
			// do not release the slot. The TeamExecutionContext persists this
			// uncertain result once, with its run-creation phase.
			log.Printf("heartbeat dispatch outcome uncertain for %s/%s (task %s, key %s): %v", teamID, agentID, createdTask.ID, dispatchKey, err)
			result.Error = NewDispatchUncertainError(&dispatchStageFailure{
				phase: dispatchFailurePhaseRunCreation,
				code:  "run_create_uncertain",
				cause: err,
			})
		} else {
			// Definitive refusal: no run was accepted, so the orphaned task is
			// safe to cancel and the slot is safe to release.
			if cancelErr := e.agentClient.CancelTask(ctx, createdTask.ID); cancelErr != nil {
				log.Printf("Warning: failed to cancel orphaned heartbeat task %s: %v", createdTask.ID, cancelErr)
			} else if deleteErr := e.agentClient.DeleteTask(ctx, createdTask.ID); deleteErr != nil {
				log.Printf("Warning: failed to delete cancelled heartbeat task %s: %v", createdTask.ID, deleteErr)
			}
		}
		if result.Error == nil {
			result.Error = fmt.Errorf("creating run: %w", err)
		}
		result.Status = store.HeartbeatStatusFailed
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, err, result.Error.Error(), attemptID, profileKey, createdTask.ID, "", runTag, "creating_run", attentionEligible, attentionEligible)
		return result, result.Error
	}

	result.RunID = run.ID
	config.LastExecution.RunID = run.ID
	_ = e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)
	if e.teamExecStore != nil {
		e.teamExecStore.SetRunningRunID(teamID, agentID, run.ID, run.Status, run.StartedAt)
	}
	e.appendAttempt(context.Background(), &store.HeartbeatAttempt{
		ID:              attemptID,
		TeamID:          teamID,
		AgentID:         agentID,
		ProfileKey:      profileKey,
		TaskID:          createdTask.ID,
		RunID:           run.ID,
		Tag:             runTag,
		Status:          store.HeartbeatStatusRunning,
		Phase:           "run_created",
		StartedAt:       startedAt.Format(time.RFC3339),
		AdmissionOrigin: admissionOrigin,
	})

	// Wait for completion asynchronously with cancellable context
	waitCtx, waitCancel := context.WithCancel(context.Background())
	waiterKey := heartbeatWaiterKey(teamID, agentID, run.ID)
	waiterID, started := e.trackWaiter(waiterKey, waitCancel)
	if !started {
		waitCancel()
		result.Status = store.HeartbeatStatusRunning
		return result, nil
	}
	if e.runRegistry != nil {
		ownerStartedAt, _ := parseOwnerRunStartedAt(run.StartedAt)
		e.runRegistry.RegisterOwnerRun(teamID, agentID, run.ID, ownerStartedAt, waitCancel, ownerRunObligationState(run.Status))
	}
	go func() {
		defer e.inflight.Done()
		defer e.releaseWaiter(waiterID, waiterKey)
		defer waitCancel()
		e.waitForCompletion(waitCtx, teamID, agentID, run.ID, attemptID, profileKey, createdTask.ID, runTag, startedAt, logPath, config.EffectiveTimeout())
	}()

	result.Status = store.HeartbeatStatusRunning
	return result, nil
}

// orchestratorWorkReferences attaches a WorkReference only for a configured
// finite delivery effort. An incomplete configured identity blocks dispatch;
// ordinary non-effort heartbeats remain unaffected.
func (e *Executor) orchestratorWorkReferences(ctx context.Context, team *store.Team, agentID, profileKey string, config *store.HeartbeatConfig) ([]*eventpb.WorkReference, error) {
	if team == nil {
		return nil, nil
	}
	contract := team.Contract()
	finiteIntent := (contract.Purpose == teamconfig.PurposeDelivery && contract.Lifetime == teamconfig.LifetimeFinite) ||
		(contract.Purpose == teamconfig.PurposeDelivery && (len(contract.EffortRefs) > 0 || strings.TrimSpace(contract.Coordination.LeadAgentID) != "")) ||
		(len(contract.EffortRefs) > 0 && contract.Lifetime == teamconfig.LifetimeFinite)
	if !finiteIntent {
		return nil, nil
	}
	if strings.TrimSpace(contract.Coordination.LeadAgentID) == "" {
		return nil, fmt.Errorf("team %q configures finite effort work without a declared leader", team.ID)
	}
	// Effort identity belongs to the configured coordinator. Other members of
	// the finite team can still receive ordinary bounded heartbeats (for
	// example, acceptance review) without being labeled as orchestrators.
	if contract.Coordination.LeadAgentID != agentID {
		return nil, nil
	}
	if contract.Purpose != teamconfig.PurposeDelivery || contract.Lifetime != teamconfig.LifetimeFinite {
		return nil, fmt.Errorf("team %q configures finite effort work without purpose=delivery and lifetime=finite", team.ID)
	}
	if len(contract.EffortRefs) != 1 || strings.TrimSpace(contract.EffortRefs[0]) == "" {
		return nil, fmt.Errorf("team %q must declare exactly one effort reference", team.ID)
	}
	if config == nil || strings.TrimSpace(config.ProfileKey) == "" || strings.TrimSpace(profileKey) == "" || config.ProfileKey != profileKey {
		return nil, fmt.Errorf("member %q heartbeat profile does not match the dispatched profile", agentID)
	}
	if e == nil || e.teamStore == nil || e.agentClient == nil {
		return nil, errors.New("team store or Agent Manager client is unavailable")
	}
	members, err := e.teamStore.GetMembers(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("reading team membership: %w", err)
	}
	memberMatches := false
	for _, member := range members {
		if member.AgentID == agentID && (member.Status == "" || member.Status == store.MemberStatusActive) {
			memberMatches = true
			break
		}
	}
	if !memberMatches {
		return nil, fmt.Errorf("member %q is not active in team %q", agentID, team.ID)
	}
	effortRef := strings.TrimSpace(contract.EffortRefs[0])
	enrollment, err := e.agentClient.GetEffortEnrollment(ctx, effortRef)
	if err != nil {
		return nil, fmt.Errorf("reading Agent Manager enrollment %q: %w", effortRef, err)
	}
	if enrollment == nil || enrollment.GetEffortRef() != effortRef {
		return nil, fmt.Errorf("Agent Manager did not return exact enrollment %q", effortRef)
	}
	if enrollment.GetWithdrawn() {
		return nil, fmt.Errorf("Agent Manager enrollment %q is withdrawn", effortRef)
	}
	if strings.TrimSpace(enrollment.GetTargetRevision()) == "" {
		return nil, fmt.Errorf("Agent Manager enrollment %q has no target revision", effortRef)
	}
	return []*eventpb.WorkReference{{
		Kind:         "effort",
		Id:           enrollment.GetEffortRef(),
		Revision:     enrollment.GetTargetRevision(),
		Relationship: "orchestrator",
		Verified:     true,
		Visibility:   eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC,
		State:        eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE,
	}}, nil
}

// recoverPriorFiniteOwner checks the exact run retained in this member's last
// heartbeat projection before a finite coordinator starts new work. The
// existing task/run and canonical work reference are the ownership proof; an
// unavailable or mismatched owner read is retained as uncertain and never
// converted into a fresh dispatch.
func (e *Executor) recoverPriorFiniteOwner(ctx context.Context, team *store.Team, agentID string, config *store.HeartbeatConfig, workReferences []*eventpb.WorkReference) (ExecutionResult, bool, error) {
	if config == nil || config.LastExecution == nil || config.LastExecution.RunID == "" {
		return ExecutionResult{}, false, nil
	}
	priorRunID := config.LastExecution.RunID
	retainUnverified := func(cause error) (ExecutionResult, bool, error) {
		e.retainPriorFiniteOwner(team.ID, agentID, priorRunID, cause)
		return ExecutionResult{}, true, NewDispatchUncertainError(fmt.Errorf("reconciling prior finite effort run %q: %w", priorRunID, cause))
	}
	if e.agentClient == nil {
		return retainUnverified(errors.New("Agent Manager client is unavailable"))
	}
	run, err := e.agentClient.GetRun(ctx, priorRunID)
	if err != nil {
		return retainUnverified(fmt.Errorf("reading exact prior run: %w", err))
	}
	if run == nil || run.ID != priorRunID {
		return retainUnverified(fmt.Errorf("Agent Manager did not return exact prior run %q", priorRunID))
	}
	owner, err := e.validatePriorFiniteOwner(ctx, team, agentID, config, run, workReferences)
	if err != nil {
		return retainUnverified(err)
	}
	if IsTerminalStatus(run.Status) {
		return ExecutionResult{}, false, nil
	}
	if e.teamExecStore == nil {
		return retainUnverified(errors.New("team execution owner is unavailable to retain the resumed run"))
	}
	e.teamExecStore.SetRunningRunID(team.ID, agentID, run.ID, run.Status, run.StartedAt)
	if !e.WatchRecoveredRunWithProjection(team.ID, agentID, run.ID, owner.profileKey, run.TaskID, run.Tag, run.Status, run.StartedAt, ownerTerminalTimestamp(run), heartbeatAdmissionOriginForRun(config, run.ID)) {
		return ExecutionResult{}, true, NewDispatchUncertainError(fmt.Errorf("reattaching completion owner for prior run %q failed", run.ID))
	}
	startedAt := time.Now().UTC()
	if parsed, ok := parseOwnerRunStartedAt(run.StartedAt); ok {
		startedAt = parsed
	}
	return ExecutionResult{TeamID: team.ID, AgentID: agentID, RunID: run.ID, Status: store.HeartbeatStatusRunning, StartedAt: startedAt}, true, nil
}

func (e *Executor) retainPriorFiniteOwner(teamID, agentID, runID string, cause error) error {
	if e.teamExecStore == nil {
		return NewDispatchUncertainError(fmt.Errorf("retaining prior finite effort run %q without a team execution owner: %w", runID, cause))
	}
	e.teamExecStore.SetRunningRunID(teamID, agentID, runID)
	return NewDispatchUncertainError(fmt.Errorf("retaining prior finite effort run %q until identity can be confirmed: %w", runID, cause))
}

type priorFiniteOwner struct {
	profileKey string
}

func (e *Executor) validatePriorFiniteOwner(ctx context.Context, team *store.Team, agentID string, config *store.HeartbeatConfig, run *Run, workReferences []*eventpb.WorkReference) (*priorFiniteOwner, error) {
	if team == nil || run == nil || config == nil || config.LastExecution == nil || run.ID != config.LastExecution.RunID {
		return nil, errors.New("prior run does not match the persisted heartbeat identity")
	}
	if e.teamStore == nil || run.TaskID == "" {
		return nil, errors.New("prior run has no persisted heartbeat attempt owner")
	}
	// LastExecution is the current member projection; the attempt ledger is the
	// durable source for the task/profile/run identity it projected. Avoid
	// imposing tag or task-title conventions that older valid runs may not have.
	attempts, err := e.teamStore.ListHeartbeatAttemptsByRun(ctx, run.ID)
	if err != nil {
		return nil, fmt.Errorf("reading prior heartbeat attempt: %w", err)
	}
	var owner *store.HeartbeatAttempt
	for i := range attempts {
		if attempts[i].RunID != run.ID {
			continue
		}
		if owner != nil && (owner.TaskID != attempts[i].TaskID || owner.ProfileKey != attempts[i].ProfileKey || owner.Tag != attempts[i].Tag) {
			return nil, errors.New("prior run has conflicting persisted heartbeat attempt identities")
		}
		owner = &attempts[i]
	}
	if owner == nil || owner.TeamID != team.ID || owner.AgentID != agentID || owner.TaskID != run.TaskID || owner.Tag != run.Tag || strings.TrimSpace(owner.ProfileKey) == "" {
		return nil, errors.New("prior run does not match the persisted heartbeat attempt owner")
	}
	profile, err := e.agentClient.ResolveProfile(ctx, owner.ProfileKey)
	if err != nil {
		return nil, fmt.Errorf("resolving prior heartbeat profile: %w", err)
	}
	if profile == nil || strings.TrimSpace(profile.GetId()) == "" || profile.GetProfileKey() != owner.ProfileKey || run.ProfileID != profile.GetId() {
		return nil, errors.New("prior run profile does not match the persisted heartbeat profile")
	}
	if !sameCanonicalEffortIdentity(run.WorkReferences, workReferences) {
		return nil, errors.New("prior run work reference does not identify the current canonical effort enrollment")
	}
	return &priorFiniteOwner{profileKey: owner.ProfileKey}, nil
}

func sameCanonicalEffortIdentity(raw json.RawMessage, expected []*eventpb.WorkReference) bool {
	if len(expected) == 0 || len(raw) == 0 {
		return false
	}
	var declarations []json.RawMessage
	if len(expected) != 1 || json.Unmarshal(raw, &declarations) != nil || len(declarations) != len(expected) {
		return false
	}
	for i, declaration := range declarations {
		var actual eventpb.WorkReference
		if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(declaration, &actual); err != nil {
			return false
		}
		// Enrollment revision is frozen historical run metadata. The stable
		// effort identity and verified coordinator qualification must still match
		// the current enrollment before this run can retain the coordinator slot.
		actual.Revision = expected[i].Revision
		if !proto.Equal(&actual, expected[i]) {
			return false
		}
	}
	return true
}

// WatchRecoveredRunWithProjection reattaches a completion waiter using the
// owner's terminal time and the persisted heartbeat admission origin when known.
func (e *Executor) WatchRecoveredRunWithProjection(teamID, agentID, runID, profileKey, taskID, runTag, ownerStatus, ownerStartedAt, ownerTerminalAt, admissionOrigin string) bool {
	if e.agentClient == nil || e.teamStore == nil || runID == "" || !e.isCurrentRunOwner(teamID, agentID, runID) {
		return false
	}
	observedAt := time.Now().UTC()
	startedAt, hasOwnerStart := parseOwnerRunStartedAt(ownerStartedAt)
	startedAtText := ""
	if hasOwnerStart {
		startedAtText = startedAt.Format(time.RFC3339)
	}
	attemptID := uuid.NewString()
	cfgCtx := context.Background()
	logPath := ""
	if config, err := e.teamStore.GetHeartbeatConfig(cfgCtx, teamID, agentID); err == nil && config != nil {
		if config.LastExecution != nil {
			last := config.LastExecution
			if last.RunID == runID && IsTerminalStatus(ownerStatus) &&
				(last.Status == store.HeartbeatStatusCompleted || last.Status == store.HeartbeatStatusFailed || last.Status == store.HeartbeatStatusCancelled) &&
				sameKnownRunEnd(last.EndedAt, ownerTerminalAt) {
				// The heartbeat result was durably projected before a crash, but
				// the retained queue slot may still belong to this exact run. Reuse
				// the canonical completion release without appending a duplicate
				// projection or attempt.
				e.completeTerminalRunOwner(teamID, agentID, runID, last.Status == store.HeartbeatStatusFailed, last.Error)
				return true
			}
			if last.RunID == runID {
				if last.StartedAt != "" {
					startedAtText = last.StartedAt
				}
				logPath = last.LogPath
				if admissionOrigin == "" {
					admissionOrigin = last.AdmissionOrigin
				}
			}
		}
		if config.LastExecution == nil {
			config.LastExecution = &store.HeartbeatExecResult{StartedAt: startedAtText}
		}
		if config.LastExecution.StartedAt == "" {
			config.LastExecution.StartedAt = startedAtText
		}
		if config.LastExecution.RunID != runID {
			config.LastExecution.AdmissionOrigin = ""
		}
		if config.LastExecution.AdmissionOrigin == "" {
			config.LastExecution.AdmissionOrigin = admissionOrigin
		}
		config.LastExecution.RunID = runID
		config.LastExecution.EndedAt = ""
		config.LastExecution.Phase = "recovered_run"
		config.LastExecution.Status = store.HeartbeatStatusRunning
		config.LastExecution.Error = ""
		if IsNeedsReviewStatus(ownerStatus) {
			config.LastExecution.Status = store.HeartbeatStatusNeedsReview
			config.LastExecution.Error = "agent-manager run is waiting for operator review"
		}
		_ = e.teamStore.SetHeartbeatConfig(cfgCtx, teamID, agentID, config)
	}
	if logPath == "" {
		logPath = e.teamStore.GetMemberLogPathForContext(cfgCtx, teamID, agentID, observedAt.Format("2006-01-02T15-04-05Z"))
	}
	timeout := 30 * time.Minute
	if cfg, err := e.teamStore.GetHeartbeatConfig(cfgCtx, teamID, agentID); err == nil && cfg != nil {
		timeout = cfg.EffectiveTimeout()
	}

	waitCtx, waitCancel := context.WithCancel(context.Background())
	waiterKey := heartbeatWaiterKey(teamID, agentID, runID)
	waiterID, started := e.trackWaiter(waiterKey, waitCancel)
	if !started {
		waitCancel()
		return false
	}
	if e.runRegistry != nil {
		state := ""
		if IsNeedsReviewStatus(ownerStatus) {
			state = ObligationNeedsReview
		} else if IsParkedStatus(ownerStatus) {
			state = ObligationPaused
		}
		e.runRegistry.RegisterRecovered(teamID, agentID, runID, startedAt, state, waitCancel)
	}
	go func() {
		defer e.inflight.Done()
		defer e.releaseWaiter(waiterID, waiterKey)
		defer waitCancel()
		e.waitForCompletion(waitCtx, teamID, agentID, runID, attemptID, profileKey, taskID, runTag, startedAt, logPath, timeout)
	}()
	return true
}

// WatchConfirmedStoppedRun attaches the normal terminal projector to a run
// whose remote stop was confirmed, including recovered owners that have no
// local waiter cancel function.
func (e *Executor) WatchConfirmedStoppedRun(teamID, agentID, runID string) bool {
	if e == nil || e.agentClient == nil || e.teamStore == nil || runID == "" || !e.isCurrentRunOwner(teamID, agentID, runID) {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	run, err := e.agentClient.GetRun(ctx, runID)
	cancel()
	if err != nil || run == nil || run.ID != runID {
		return false
	}
	profileKey := ""
	if owner, ok := e.teamExecStore.(interface {
		ProfileKeyForRun(teamID, agentID, runID string) (string, bool)
	}); ok {
		profileKey, _ = owner.ProfileKeyForRun(teamID, agentID, runID)
	}
	admissionOrigin := ""
	if config, readErr := e.teamStore.GetHeartbeatConfig(context.Background(), teamID, agentID); readErr == nil && config != nil && config.LastExecution != nil && config.LastExecution.RunID == runID {
		admissionOrigin = config.LastExecution.AdmissionOrigin
	}
	return e.WatchRecoveredRunWithProjection(teamID, agentID, runID, profileKey, run.TaskID, run.Tag, run.Status, run.StartedAt, ownerTerminalTimestamp(run), admissionOrigin)
}

func sameKnownRunEnd(projectedEndedAt, ownerTerminalAt string) bool {
	if strings.TrimSpace(projectedEndedAt) == "" || strings.TrimSpace(ownerTerminalAt) == "" {
		return false
	}
	projected, projectedErr := time.Parse(time.RFC3339Nano, projectedEndedAt)
	owner, ownerErr := time.Parse(time.RFC3339Nano, ownerTerminalAt)
	return projectedErr == nil && ownerErr == nil && projected.Equal(owner)
}

func ownerTerminalTimestamp(run *Run) string {
	if run == nil {
		return ""
	}
	for _, value := range []string{run.EndedAt, run.UpdatedAt} {
		if timestamp, ok := parseOwnerRunStartedAt(value); ok {
			return timestamp.Format(time.RFC3339Nano)
		}
	}
	return ""
}

func parseOwnerRunStartedAt(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	startedAt, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || startedAt.IsZero() {
		return time.Time{}, false
	}
	return startedAt.UTC(), true
}

func formatOptionalHeartbeatStartedAt(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func heartbeatWaiterKey(teamID, agentID, runID string) string {
	return teamID + "/" + agentID + "/" + runID
}

// BuildContext constructs team/agent context for external consumption.
func (e *Executor) BuildContext(ctx context.Context, teamID, agentID string) (string, error) {
	if e.promptBuilder == nil {
		return "", fmt.Errorf("prompt builder is not configured")
	}
	return e.promptBuilder.BuildContext(ctx, PromptBuildRequest{TeamID: teamID, AgentID: agentID})
}

// BuildPrompt constructs the full prompt for heartbeat execution.
func (e *Executor) BuildPrompt(ctx context.Context, teamID, agentID string) (string, error) {
	if e.promptBuilder == nil {
		return "", fmt.Errorf("prompt builder is not configured")
	}
	return e.promptBuilder.Build(ctx, PromptBuildRequest{
		TeamID:  teamID,
		AgentID: agentID,
	})
}

// BuildPromptStructured returns the prompt as structured sections.
func (e *Executor) BuildPromptStructured(ctx context.Context, teamID, agentID string) ([]PromptSection, error) {
	if e.promptBuilder == nil {
		return nil, fmt.Errorf("prompt builder is not configured")
	}
	return e.promptBuilder.BuildStructured(ctx, PromptBuildRequest{
		TeamID:  teamID,
		AgentID: agentID,
	})
}

// heartbeatAdmissionOriginForRun returns provenance only when LastExecution
// still refers to the exact run being observed. Empty and unknown values stay
// empty; timestamps, tags, and workload attribution are not origin evidence.
func heartbeatAdmissionOriginForRun(config *store.HeartbeatConfig, runID string) string {
	if config == nil || config.LastExecution == nil || runID == "" || config.LastExecution.RunID != runID {
		return ""
	}
	switch config.LastExecution.AdmissionOrigin {
	case AdmissionOriginScheduled, AdmissionOriginManual:
		return config.LastExecution.AdmissionOrigin
	default:
		return ""
	}
}

// waitForCompletion polls for run completion and updates config
func (e *Executor) waitForCompletion(ctx context.Context, teamID, agentID, runID, attemptID, profileKey, taskID, tag string, startedAt time.Time, logPath string, timeout time.Duration) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	run, err := e.agentClient.WaitForRun(timeoutCtx, runID, 5*time.Second)
	if errors.Is(err, context.Canceled) {
		// StopRunning cancels this waiter only after Agent Manager confirms its
		// stop operation. Re-read the exact owner before treating that local
		// cancellation as terminal; never infer a stopped run from ctx alone.
		ownerCtx, ownerCancel := context.WithTimeout(context.Background(), 5*time.Second)
		ownerRun, ownerErr := e.agentClient.GetRun(ownerCtx, runID)
		ownerCancel()
		switch {
		case ownerErr != nil:
			err = fmt.Errorf("completion wait cancelled and owner read failed: %w", ownerErr)
		case ownerRun == nil || ownerRun.ID != runID:
			err = fmt.Errorf("completion wait cancelled without exact owner run %s", runID)
		case !IsTerminalStatus(ownerRun.Status):
			err = fmt.Errorf("completion wait cancelled before owner run %s reached a terminal state", runID)
		default:
			run = ownerRun
			err = nil
		}
	}
	if err == nil && (run == nil || !IsTerminalStatus(run.Status)) {
		err = fmt.Errorf("completion observer returned without terminal owner evidence for run %s", runID)
	}
	e.projectRunCompletion(ctx, teamID, agentID, runID, run, err, attemptID, profileKey, taskID, tag, startedAt, logPath, "", true)
}

type retainedTerminalProjection uint8

const (
	retainedTerminalUnconfirmed retainedTerminalProjection = iota
	retainedTerminalPersisted
	retainedTerminalSuperseded
)

// ProjectRetainedTerminal applies the same heartbeat result projection used by
// the completion waiter to an exact terminal run already observed by the
// scheduled owner read. Persisted means this run's heartbeat result is durable;
// superseded means the exact owner is terminal but a newer member projection
// was preserved. Attempt and log writes remain best-effort. Neither outcome
// releases the team slot; the caller must release that exact run separately.
func (e *Executor) ProjectRetainedTerminal(teamID, agentID, runID, profileKey, taskID, tag, admissionOrigin string, run *Run) retainedTerminalProjection {
	if run == nil || run.ID != runID || !IsTerminalStatus(run.Status) || !e.isCurrentRunOwner(teamID, agentID, runID) {
		return retainedTerminalUnconfirmed
	}
	cfgCtx := context.Background()
	config, err := e.teamStore.GetHeartbeatConfig(cfgCtx, teamID, agentID)
	if err != nil || config == nil {
		return retainedTerminalUnconfirmed
	}
	if config.LastExecution != nil {
		last := config.LastExecution
		if last.RunID == runID && IsTerminalStatus(last.Status) && sameKnownRunEnd(last.EndedAt, ownerTerminalTimestamp(run)) {
			if e.runRegistry != nil {
				e.runRegistry.CompleteRun(teamID, agentID, runID, last.Status == store.HeartbeatStatusFailed, last.Error)
			}
			e.scheduleHandoffExtraction(teamID, agentID, runID, retainedRunEndedAt(run))
			return retainedTerminalPersisted
		}
		if last.RunID == runID {
			if last.StartedAt != "" {
				if startedAt, ok := parseOwnerRunStartedAt(last.StartedAt); ok {
					// Preserve the canonical owner's exact start projection.
					projection := e.projectRunCompletion(context.Background(), teamID, agentID, runID, run, nil, uuid.NewString(), profileKey, taskID, tag, startedAt, last.LogPath, admissionOrigin, false)
					if projection != retainedTerminalUnconfirmed {
						e.scheduleHandoffExtraction(teamID, agentID, runID, retainedRunEndedAt(run))
					}
					return projection
				}
			}
		}
	}
	startedAt, _ := parseOwnerRunStartedAt(run.StartedAt)
	logPath := e.teamStore.GetMemberLogPathForContext(cfgCtx, teamID, agentID, time.Now().UTC().Format("2006-01-02T15-04-05Z"))
	projection := e.projectRunCompletion(context.Background(), teamID, agentID, runID, run, nil, uuid.NewString(), profileKey, taskID, tag, startedAt, logPath, admissionOrigin, false)
	if projection != retainedTerminalUnconfirmed {
		e.scheduleHandoffExtraction(teamID, agentID, runID, retainedRunEndedAt(run))
	}
	return projection
}

func retainedRunEndedAt(run *Run) time.Time {
	if endedAt, ok := parseOwnerRunStartedAt(ownerTerminalTimestamp(run)); ok {
		return endedAt
	}
	return time.Now().UTC()
}

func (e *Executor) projectRunCompletion(ctx context.Context, teamID, agentID, runID string, run *Run, err error, attemptID, profileKey, taskID, tag string, startedAt time.Time, logPath, admissionOriginHint string, releaseOwner bool) retainedTerminalProjection {
	terminalEvidence := false
	outcomeFailed := false
	outcomeMessage := ""
	if e.runRegistry != nil {
		defer func() {
			if terminalEvidence {
				e.runRegistry.CompleteRun(teamID, agentID, runID, outcomeFailed, outcomeMessage)
			}
		}()
	}

	endedAt := time.Now().UTC()
	if ownerTerminalAt, ok := parseOwnerRunStartedAt(ownerTerminalTimestamp(run)); err == nil && ok {
		endedAt = ownerTerminalAt
	}
	endedAtText := endedAt.Format(time.RFC3339Nano)
	if !e.isCurrentRunOwner(teamID, agentID, runID) {
		return retainedTerminalUnconfirmed
	}
	if err != nil {
		e.markRunObserverUnavailable(teamID, agentID, runID)
	}

	// Use a background context for config updates since the parent ctx may be cancelled
	cfgCtx := context.Background()

	// Get current config
	config, configErr := e.teamStore.GetHeartbeatConfig(cfgCtx, teamID, agentID)
	if configErr != nil {
		return retainedTerminalUnconfirmed
	}
	if config == nil {
		return retainedTerminalUnconfirmed
	}
	if !e.isCurrentRunOwner(teamID, agentID, runID) {
		return retainedTerminalUnconfirmed
	}
	admissionOrigin := heartbeatAdmissionOriginForRun(config, runID)
	if admissionOrigin == "" {
		admissionOrigin = admissionOriginHint
	}
	// A newer projection can be written before the recovered queue owner is
	// replaced. The old waiter may still release its exact retained slot, but
	// it must not overwrite that independently newer member projection.
	projectionSuperseded := config.LastExecution != nil && config.LastExecution.RunID != "" && config.LastExecution.RunID != runID
	projectionStatus := store.HeartbeatStatusObserverUnavailable

	if err != nil {
		// A failed local observer says nothing definitive about owner-run
		// outcome. Keep the execution and its queue slot visible until a later
		// scheduled owner read can reattach or prove terminal state.
		status := store.HeartbeatStatusObserverUnavailable
		projectionStatus = status
		if !projectionSuperseded {
			config.LastExecution = &store.HeartbeatExecResult{
				StartedAt:       formatOptionalHeartbeatStartedAt(startedAt),
				EndedAt:         endedAtText,
				Status:          status,
				RunID:           runID,
				LogPath:         logPath,
				Error:           err.Error(),
				AdmissionOrigin: admissionOrigin,
			}
			config.RefreshHealth()
		}
		category := classifyHeartbeatError(err.Error())
		e.appendAttempt(cfgCtx, &store.HeartbeatAttempt{
			ID:              attemptID,
			TeamID:          teamID,
			AgentID:         agentID,
			ProfileKey:      profileKey,
			TaskID:          taskID,
			RunID:           runID,
			Tag:             tag,
			Status:          status,
			Phase:           "waiting_for_completion",
			StartedAt:       formatOptionalHeartbeatStartedAt(startedAt),
			EndedAt:         endedAtText,
			ErrorCategory:   category,
			Error:           err.Error(),
			Recovery:        "retry_owner_observation_on_next_schedule",
			AdmissionOrigin: admissionOrigin,
		})
	} else {
		status := store.HeartbeatStatusCompleted
		errMsg := ""
		if IsCancelledStatus(run.Status) {
			status = store.HeartbeatStatusCancelled
		} else if IsFailedStatus(run.Status) {
			status = store.HeartbeatStatusFailed
			errMsg = run.Error
		}
		projectionStatus = status
		terminalEvidence = true
		outcomeFailed = status == store.HeartbeatStatusFailed
		outcomeMessage = errMsg

		if !projectionSuperseded {
			config.LastExecution = &store.HeartbeatExecResult{
				StartedAt:       formatOptionalHeartbeatStartedAt(startedAt),
				EndedAt:         endedAtText,
				Status:          status,
				RunID:           runID,
				LogPath:         logPath,
				Error:           errMsg,
				AdmissionOrigin: admissionOrigin,
			}
			config.RecordHeartbeatStatus(status)
		}
		category := ""
		recovery := ""
		if errMsg != "" {
			category = classifyHeartbeatError(errMsg)
			recovery = recoveryForHeartbeatError(category)
		}
		attempt := &store.HeartbeatAttempt{
			ID:              attemptID,
			TeamID:          teamID,
			AgentID:         agentID,
			ProfileKey:      profileKey,
			TaskID:          taskID,
			RunID:           runID,
			Tag:             tag,
			Status:          status,
			Phase:           "run_terminal",
			StartedAt:       formatOptionalHeartbeatStartedAt(startedAt),
			EndedAt:         endedAtText,
			ErrorCategory:   category,
			Error:           errMsg,
			Recovery:        recovery,
			AdmissionOrigin: admissionOrigin,
		}
		if status == store.HeartbeatStatusFailed && !projectionSuperseded {
			e.attachFailureAttention(cfgCtx, teamID, agentID, config, attempt)
			attempt.AttentionEligible = true
		}
		e.appendAttempt(cfgCtx, attempt)
	}

	// Write log file
	logContent := fmt.Sprintf("Heartbeat execution for %s/%s\n", teamID, agentID)
	logContent += fmt.Sprintf("Started: %s\n", formatOptionalHeartbeatStartedAt(startedAt))
	logContent += fmt.Sprintf("Ended: %s\n", endedAt.Format(time.RFC3339))
	logContent += fmt.Sprintf("Run ID: %s\n", runID)
	logContent += fmt.Sprintf("Status: %s\n", projectionStatus)
	if err != nil {
		logContent += fmt.Sprintf("Error: %s\n", err.Error())
	} else if IsFailedStatus(run.Status) && run.Error != "" {
		logContent += fmt.Sprintf("Error: %s\n", run.Error)
	}

	// Ensure logs directory exists and write log
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err == nil {
		_ = os.WriteFile(logPath, []byte(logContent), 0o644)
	}

	if !projectionSuperseded {
		if persistErr := e.teamStore.SetHeartbeatConfig(cfgCtx, teamID, agentID, config); persistErr != nil {
			log.Printf("Warning: failed to persist heartbeat terminal projection for %s/%s run %s: %v", teamID, agentID, runID, persistErr)
			if terminalEvidence {
				// The owner run is terminal, but the durable heartbeat projection is
				// not. Keep the exact retained obligation so recovery can retry this
				// projection instead of releasing its slot.
				terminalEvidence = false
			}
			e.markRunObserverUnavailable(teamID, agentID, runID)
			return retainedTerminalUnconfirmed
		}
	}

	if err == nil {
		if releaseOwner {
			e.completeTerminalRunOwner(teamID, agentID, runID, outcomeFailed, outcomeMessage)
			e.extractAndStoreHandoff(ctx, teamID, agentID, runID, endedAt)
		} else if e.runRegistry != nil {
			e.runRegistry.CompleteRun(teamID, agentID, runID, outcomeFailed, outcomeMessage)
		}
		terminalEvidence = false
	}

	if err != nil {
		return retainedTerminalUnconfirmed
	}
	if projectionSuperseded {
		return retainedTerminalSuperseded
	}
	return retainedTerminalPersisted
}

func (e *Executor) completeTerminalRunOwner(teamID, agentID, runID string, failed bool, message string) {
	if runID == "" || !e.isCurrentRunOwner(teamID, agentID, runID) {
		return
	}
	if e.runRegistry != nil {
		e.runRegistry.CompleteRun(teamID, agentID, runID, failed, message)
	}
	if e.OnRunComplete != nil {
		e.OnRunComplete(teamID, agentID, runID)
	} else if completion, ok := e.teamExecStore.(interface {
		OnRunComplete(teamID, agentID, runID string)
	}); ok {
		completion.OnRunComplete(teamID, agentID, runID)
	}
}

func (e *Executor) markRunObserverUnavailable(teamID, agentID, runID string) {
	if e.runRegistry != nil {
		e.runRegistry.MarkRunObserverUnavailable(teamID, agentID, runID)
	}
	if marker, ok := e.teamExecStore.(interface {
		MarkWatcherUnavailableForRun(teamID, agentID, runID string)
	}); ok {
		marker.MarkWatcherUnavailableForRun(teamID, agentID, runID)
	} else if marker, ok := e.teamExecStore.(interface {
		MarkWatcherUnavailable(teamID, agentID string)
	}); ok {
		marker.MarkWatcherUnavailable(teamID, agentID)
	}
}

// extractAndStoreHandoff fetches run events and extracts/stores the handoff (best-effort).
func (e *Executor) extractAndStoreHandoff(ctx context.Context, teamID, agentID, runID string, endedAt time.Time) {
	if e.agentClient == nil {
		return
	}

	// Agent Manager persists the resolver's evidence-ranked selection on the
	// run. The completion notification can arrive just before that projection
	// is readable, so retry briefly before falling back to transcript parsing.
	// A missing/ambiguous selection remains observable and must not be mistaken
	// for a successful handoff.
	const resultReadAttempts = 10
	for attempt := 0; attempt < resultReadAttempts; attempt++ {
		run, err := e.agentClient.GetRun(ctx, runID)
		if err == nil && run != nil && run.Result != nil {
			if strings.EqualFold(strings.TrimSpace(run.Result.Selection.Status), "selected") && strings.TrimSpace(run.Result.FinalOutput) != "" {
				e.storeHandoff(ctx, teamID, agentID, runID, endedAt, run.Result.FinalOutput)
				return
			}
			if attempt == resultReadAttempts-1 {
				log.Printf("Warning: final-output resolver did not select a handoff (run %s: status=%s rule=%s)", runID, run.Result.Selection.Status, run.Result.Selection.Rule)
			}
		} else if err != nil && attempt == resultReadAttempts-1 {
			log.Printf("Warning: failed to read resolved handoff (run %s): %v", runID, err)
		}
		if attempt < resultReadAttempts-1 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
		}
	}

	// Compatibility fallback for older agent-manager responses that predate the
	// persisted result field. New responses take the resolver path above.
	if e.handoffExtractor == nil {
		return
	}

	eventsJSON, err := e.agentClient.GetRunEvents(ctx, runID, -1, 0)
	if err != nil {
		log.Printf("Warning: failed to fetch run events for handoff extraction (run %s): %v", runID, err)
		return
	}

	content, err := e.handoffExtractor.Extract(ctx, eventsJSON)
	if err != nil {
		log.Printf("Warning: handoff extraction failed (run %s): %v", runID, err)
		return
	}

	if content == "" {
		return
	}

	e.storeHandoff(ctx, teamID, agentID, runID, endedAt, content)
}

func (e *Executor) storeHandoff(ctx context.Context, teamID, agentID, runID string, endedAt time.Time, content string) {
	// Store last handoff for prompt injection
	if err := e.teamStore.SetLastHandoff(ctx, teamID, agentID, content); err != nil {
		log.Printf("Warning: failed to store last handoff (run %s): %v", runID, err)
	}

	// Append to team history
	entry := &store.HandoffEntry{
		AgentID:   agentID,
		RunID:     runID,
		Timestamp: endedAt.Format(time.RFC3339),
		Content:   content,
	}
	if err := e.teamStore.AppendHandoffHistory(ctx, teamID, entry); err != nil {
		log.Printf("Warning: failed to append handoff history (run %s): %v", runID, err)
	}
}

// updateConfigCancelled records a request proven not to have been dispatched,
// or a terminal owner cancellation. Cancellation preserves the consecutive
// definitive-failure count and never qualifies for recovery/attention.
func (e *Executor) updateConfigCancelled(ctx context.Context, teamID, agentID string, config *store.HeartbeatConfig, startedAt time.Time, detail, attemptID, profileKey, taskID, runID, tag, phase string) {
	endedAt := time.Now().UTC()
	config.LastExecution = &store.HeartbeatExecResult{
		StartedAt:       startedAt.Format(time.RFC3339),
		EndedAt:         endedAt.Format(time.RFC3339),
		Status:          store.HeartbeatStatusCancelled,
		Error:           detail,
		AdmissionOrigin: heartbeatAdmissionOrigin(ctx),
	}
	config.RecordHeartbeatStatus(store.HeartbeatStatusCancelled)
	persistCtx := context.Background()
	_ = e.teamStore.SetHeartbeatConfig(persistCtx, teamID, agentID, config)
	e.appendAttempt(persistCtx, &store.HeartbeatAttempt{
		ID:                  attemptID,
		TeamID:              teamID,
		AgentID:             agentID,
		ProfileKey:          profileKey,
		TaskID:              taskID,
		RunID:               runID,
		Tag:                 tag,
		Status:              store.HeartbeatStatusCancelled,
		Phase:               phase,
		StartedAt:           startedAt.Format(time.RFC3339),
		EndedAt:             endedAt.Format(time.RFC3339),
		Error:               detail,
		ConsecutiveFailures: config.ConsecutiveFailures,
		AttentionEligible:   false,
		AdmissionOrigin:     heartbeatAdmissionOrigin(ctx),
	})
}

func boundedHeartbeatReportContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
}

func isPreSendCancellation(err error) bool {
	return err != nil && IsDispatchRejected(err) && !IsDispatchUncertain(err) && errors.Is(err, context.Canceled)
}

// updateConfigFailed updates config with failed status
func (e *Executor) updateConfigFailed(ctx context.Context, teamID, agentID string, config *store.HeartbeatConfig, startedAt time.Time, cause error, errMsg, attemptID, profileKey, taskID, runID, tag, phase string, attentionEligible, countAsFailure bool) {
	endedAt := time.Now().UTC()
	config.LastExecution = &store.HeartbeatExecResult{
		StartedAt:       startedAt.Format(time.RFC3339),
		EndedAt:         endedAt.Format(time.RFC3339),
		Status:          store.HeartbeatStatusFailed,
		Error:           errMsg,
		AdmissionReason: admissionReasonFromError(cause),
		AdmissionOrigin: heartbeatAdmissionOrigin(ctx),
	}
	if countAsFailure {
		config.RecordHeartbeatStatus(store.HeartbeatStatusFailed)
	} else {
		config.RefreshHealth()
	}
	_ = e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)
	category := classifyHeartbeatFailure(cause, errMsg)
	recoveryOutcome := store.HeartbeatRecoveryNotRequested
	attempt := &store.HeartbeatAttempt{
		ID:                  attemptID,
		TeamID:              teamID,
		AgentID:             agentID,
		ProfileKey:          profileKey,
		TaskID:              taskID,
		RunID:               runID,
		Tag:                 tag,
		Status:              store.HeartbeatStatusFailed,
		Phase:               phase,
		StartedAt:           startedAt.Format(time.RFC3339),
		EndedAt:             endedAt.Format(time.RFC3339),
		ErrorCategory:       category,
		Error:               errMsg,
		AdmissionReason:     admissionReasonFromError(cause),
		AdmissionOrigin:     heartbeatAdmissionOrigin(ctx),
		Recovery:            recoveryForHeartbeatError(category),
		ConsecutiveFailures: config.ConsecutiveFailures,
		FailureEpisodeID:    heartbeatFailureEpisodeID(teamID, agentID, config),
		RecoveryOutcome:     recoveryOutcome,
		AttentionEligible:   attentionEligible,
	}
	e.attachFailureAttention(ctx, teamID, agentID, config, attempt)
	e.appendAttempt(ctx, attempt)
}

func classifyHeartbeatFailure(cause error, message string) string {
	if _, ok := AgentManagerAuthorizationStatus(cause); ok {
		return "agent_manager_authorization"
	}
	return classifyHeartbeatError(message)
}

func (e *Executor) attachFailureAttention(ctx context.Context, teamID, agentID string, config *store.HeartbeatConfig, attempt *store.HeartbeatAttempt) {
	if config == nil || attempt == nil {
		return
	}
	attempt.ConsecutiveFailures = config.ConsecutiveFailures
	attempt.FailureEpisodeID = heartbeatFailureEpisodeID(teamID, agentID, config)
	if attempt.RecoveryOutcome == "" {
		attempt.RecoveryOutcome = store.HeartbeatRecoveryNotRequested
	}
	if config.ConsecutiveFailures >= heartbeatAttentionFailureThreshold {
		e.mu.Lock()
		resolver := e.TeamReviewURL
		e.mu.Unlock()
		if resolver != nil {
			attempt.ReviewURL = strings.TrimSpace(resolver(ctx, teamID))
		}
	}
}

func (e *Executor) updateConfigApprovalPending(ctx context.Context, teamID, agentID string, config *store.HeartbeatConfig, startedAt time.Time, detail, attemptID, profileKey, taskID, runID, tag, phase string) {
	endedAt := time.Now().UTC()
	config.LastExecution = &store.HeartbeatExecResult{
		StartedAt:       startedAt.Format(time.RFC3339),
		EndedAt:         endedAt.Format(time.RFC3339),
		Status:          store.HeartbeatStatusApprovalPending,
		Phase:           phase,
		RunID:           runID,
		Error:           detail,
		AdmissionOrigin: heartbeatAdmissionOrigin(ctx),
	}
	config.RefreshHealth()
	_ = e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)
	e.appendAttempt(ctx, &store.HeartbeatAttempt{
		ID:              attemptID,
		TeamID:          teamID,
		AgentID:         agentID,
		ProfileKey:      profileKey,
		TaskID:          taskID,
		RunID:           runID,
		Tag:             tag,
		Status:          store.HeartbeatStatusApprovalPending,
		Phase:           phase,
		StartedAt:       startedAt.Format(time.RFC3339),
		EndedAt:         endedAt.Format(time.RFC3339),
		Error:           detail,
		AdmissionOrigin: heartbeatAdmissionOrigin(ctx),
	})
}

func (e *Executor) appendAttempt(ctx context.Context, attempt *store.HeartbeatAttempt) {
	if e.teamStore == nil || attempt == nil || attempt.TeamID == "" {
		return
	}
	if attempt.ID == "" {
		attempt.ID = uuid.NewString()
	}
	if attempt.Phase == "" {
		attempt.Phase = "unknown"
	}
	if err := e.teamStore.AppendHeartbeatAttempt(ctx, attempt.TeamID, attempt); err != nil {
		log.Printf("Warning: failed to append heartbeat attempt for %s/%s: %v", attempt.TeamID, attempt.AgentID, err)
	}
}

func classifyHeartbeatError(message string) string {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "invalid json request body"), strings.Contains(lower, "validation error"):
		return "contract_validation"
	case strings.Contains(lower, "agent client is not configured"):
		return "configuration"
	case strings.Contains(lower, "connection refused"), strings.Contains(lower, "no such host"), strings.Contains(lower, "deadline exceeded"):
		return "dependency_unavailable"
	case strings.Contains(lower, "not found"):
		return "missing_dependency"
	case strings.Contains(lower, "cancel"):
		return "cancelled"
	default:
		return "agent_run_failed"
	}
}

func recoveryForHeartbeatError(category string) string {
	switch category {
	case "contract_validation":
		return "fix_integration_contract"
	case "configuration":
		return "configure_agent_manager_client"
	case "dependency_unavailable":
		return "retry_when_dependency_healthy"
	case "missing_dependency":
		return "reconcile_missing_resource"
	case "cancelled":
		return "operator_cancelled"
	case "agent_manager_authorization":
		return "review_agent_manager_authorization"
	default:
		return "inspect_run_or_retry"
	}
}

// TriggerManual manually triggers a heartbeat execution
func (e *Executor) TriggerManual(ctx context.Context, teamID, agentID string) (*ExecutionResult, error) {
	// Get heartbeat config to get profile key
	config, err := e.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if err != nil {
		return nil, fmt.Errorf("getting heartbeat config: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("heartbeat config not found")
	}

	// Use config's profile key if set; otherwise pass empty string so
	// Execute() resolves the default based on the team's runtime mode.
	return e.Execute(ctx, teamID, agentID, config.ProfileKey)
}
