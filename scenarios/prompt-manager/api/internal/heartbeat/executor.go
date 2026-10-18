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
	// MarkDispatchUncertain retains the slot when the owner may have accepted
	// a run that the caller cannot confirm.
	MarkDispatchUncertain(teamID, agentID, reason string)
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
	OnComplete       func(teamID, agentID string)
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
	return id, true
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

	// Generate log path
	timestamp := startedAt.Format("2006-01-02T15-04-05Z")
	logPath := e.teamStore.GetMemberLogPathForContext(ctx, teamID, agentID, timestamp)
	result.LogPath = logPath
	attemptID := uuid.NewString()

	// Update config to running
	config.LastExecution = &store.HeartbeatExecResult{
		StartedAt: startedAt.Format(time.RFC3339),
		Status:    store.HeartbeatStatusRunning,
		LogPath:   logPath,
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
				retained := NewDispatchUncertainError(fmt.Errorf("canceling unsent task dispatch: %w", cancelErr))
				result.Error = retained
				result.Status = store.HeartbeatStatusFailed
				e.teamExecStore.MarkDispatchUncertain(teamID, agentID, retained.Error())
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
				retained := NewDispatchUncertainError(fmt.Errorf("retaining capacity-deferred task intent: %w", deferErr))
				e.teamExecStore.MarkDispatchUncertain(teamID, agentID, retained.Error())
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
		uncertain := NewDispatchUncertainError(fmt.Errorf("persisting confirmed task stage: %w", err))
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
				retained := NewDispatchUncertainError(fmt.Errorf("retaining capacity-deferred run intent: %w", deferErr))
				e.teamExecStore.MarkDispatchUncertain(teamID, agentID, retained.Error())
				result.Error = retained
			}
			result.Status = store.HeartbeatStatusFailed
			e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, err, result.Error.Error(), attemptID, profileKey, createdTask.ID, "", runTag, "creating_run", true, true)
			return result, result.Error
		}
		attentionEligible := !IsDispatchUncertain(err)
		if !attentionEligible {
			// The owner may have accepted the run. Do not cancel the task and
			// do not release the slot: retain a visible, recoverable obligation.
			log.Printf("heartbeat dispatch outcome uncertain for %s/%s (task %s, key %s): %v", teamID, agentID, createdTask.ID, dispatchKey, err)
			if e.teamExecStore != nil {
				e.teamExecStore.MarkDispatchUncertain(teamID, agentID, err.Error())
			}
		} else {
			// Definitive refusal: no run was accepted, so the orphaned task is
			// safe to cancel and the slot is safe to release.
			if cancelErr := e.agentClient.CancelTask(ctx, createdTask.ID); cancelErr != nil {
				log.Printf("Warning: failed to cancel orphaned heartbeat task %s: %v", createdTask.ID, cancelErr)
			} else if deleteErr := e.agentClient.DeleteTask(ctx, createdTask.ID); deleteErr != nil {
				log.Printf("Warning: failed to delete cancelled heartbeat task %s: %v", createdTask.ID, deleteErr)
			}
		}
		result.Error = fmt.Errorf("creating run: %w", err)
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
		ID:         attemptID,
		TeamID:     teamID,
		AgentID:    agentID,
		ProfileKey: profileKey,
		TaskID:     createdTask.ID,
		RunID:      run.ID,
		Tag:        runTag,
		Status:     store.HeartbeatStatusRunning,
		Phase:      "run_created",
		StartedAt:  startedAt.Format(time.RFC3339),
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
	e.inflight.Add(1)
	go func() {
		defer e.inflight.Done()
		defer e.releaseWaiter(waiterID, waiterKey)
		defer waitCancel()
		e.waitForCompletion(waitCtx, teamID, agentID, run.ID, attemptID, profileKey, createdTask.ID, runTag, startedAt, logPath, config.EffectiveTimeout())
	}()

	result.Status = store.HeartbeatStatusRunning
	return result, nil
}

// WatchRecoveredRun re-attaches a completion waiter to a run that was not
// created in this process. It is used after Recover binds a retained uncertain
// dispatch to its confirmed owner run: without a waiter, the recovered
// obligation would hold its slot even after the owner reports terminal state.
//
// The waiter is the same path a live dispatch uses, so a recovered run is
// released (via OnComplete) exactly like a run started by Execute.
func (e *Executor) WatchRecoveredRun(teamID, agentID, runID, profileKey, taskID, runTag, ownerStatus, ownerStartedAt string) bool {
	if e.agentClient == nil || e.teamStore == nil || runID == "" {
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
	if config, err := e.teamStore.GetHeartbeatConfig(cfgCtx, teamID, agentID); err == nil && config != nil {
		if config.LastExecution == nil {
			config.LastExecution = &store.HeartbeatExecResult{StartedAt: startedAtText}
		}
		config.LastExecution.StartedAt = startedAtText
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
	logPath := e.teamStore.GetMemberLogPathForContext(cfgCtx, teamID, agentID, observedAt.Format("2006-01-02T15-04-05Z"))
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
	e.inflight.Add(1)
	go func() {
		defer e.inflight.Done()
		defer e.releaseWaiter(waiterID, waiterKey)
		defer waitCancel()
		e.waitForCompletion(waitCtx, teamID, agentID, runID, attemptID, profileKey, taskID, runTag, startedAt, logPath, timeout)
	}()
	return true
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

// waitForCompletion polls for run completion and updates config
func (e *Executor) waitForCompletion(ctx context.Context, teamID, agentID, runID, attemptID, profileKey, taskID, tag string, startedAt time.Time, logPath string, timeout time.Duration) {
	terminalEvidence := false
	outcomeFailed := false
	outcomeMessage := ""
	if e.runRegistry != nil {
		defer func() {
			if terminalEvidence {
				e.runRegistry.Complete(teamID, agentID, outcomeFailed, outcomeMessage)
			}
		}()
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	run, err := e.agentClient.WaitForRun(timeoutCtx, runID, 5*time.Second)
	if err == nil && (run == nil || !IsTerminalStatus(run.Status)) {
		err = fmt.Errorf("completion observer returned without terminal owner evidence for run %s", runID)
	}

	endedAt := time.Now().UTC()
	if err != nil {
		if e.runRegistry != nil {
			e.runRegistry.MarkObserverUnavailable(teamID, agentID)
		}
		if marker, ok := e.teamExecStore.(interface {
			MarkWatcherUnavailable(teamID, agentID string)
		}); ok {
			marker.MarkWatcherUnavailable(teamID, agentID)
		}
	}

	// Use a background context for config updates since the parent ctx may be cancelled
	cfgCtx := context.Background()

	// Get current config
	config, configErr := e.teamStore.GetHeartbeatConfig(cfgCtx, teamID, agentID)
	if configErr != nil {
		return
	}
	if config == nil {
		return
	}

	if err != nil {
		// A failed local observer says nothing definitive about owner-run
		// outcome. Keep the execution and its queue slot visible until a later
		// scheduled owner read can reattach or prove terminal state.
		status := store.HeartbeatStatusObserverUnavailable
		config.LastExecution = &store.HeartbeatExecResult{
			StartedAt: formatOptionalHeartbeatStartedAt(startedAt),
			EndedAt:   endedAt.Format(time.RFC3339),
			Status:    status,
			RunID:     runID,
			LogPath:   logPath,
			Error:     err.Error(),
		}
		config.RefreshHealth()
		category := classifyHeartbeatError(err.Error())
		e.appendAttempt(cfgCtx, &store.HeartbeatAttempt{
			ID:            attemptID,
			TeamID:        teamID,
			AgentID:       agentID,
			ProfileKey:    profileKey,
			TaskID:        taskID,
			RunID:         runID,
			Tag:           tag,
			Status:        status,
			Phase:         "waiting_for_completion",
			StartedAt:     formatOptionalHeartbeatStartedAt(startedAt),
			EndedAt:       endedAt.Format(time.RFC3339),
			ErrorCategory: category,
			Error:         err.Error(),
			Recovery:      "retry_owner_observation_on_next_schedule",
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
		terminalEvidence = true
		outcomeFailed = status == store.HeartbeatStatusFailed
		outcomeMessage = errMsg

		config.LastExecution = &store.HeartbeatExecResult{
			StartedAt: formatOptionalHeartbeatStartedAt(startedAt),
			EndedAt:   endedAt.Format(time.RFC3339),
			Status:    status,
			RunID:     runID,
			LogPath:   logPath,
			Error:     errMsg,
		}
		config.RecordHeartbeatStatus(status)
		category := ""
		recovery := ""
		if errMsg != "" {
			category = classifyHeartbeatError(errMsg)
			recovery = recoveryForHeartbeatError(category)
		}
		attempt := &store.HeartbeatAttempt{
			ID:            attemptID,
			TeamID:        teamID,
			AgentID:       agentID,
			ProfileKey:    profileKey,
			TaskID:        taskID,
			RunID:         runID,
			Tag:           tag,
			Status:        status,
			Phase:         "run_terminal",
			StartedAt:     formatOptionalHeartbeatStartedAt(startedAt),
			EndedAt:       endedAt.Format(time.RFC3339),
			ErrorCategory: category,
			Error:         errMsg,
			Recovery:      recovery,
		}
		if status == store.HeartbeatStatusFailed {
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
	logContent += fmt.Sprintf("Status: %s\n", config.LastExecution.Status)
	if config.LastExecution.Error != "" {
		logContent += fmt.Sprintf("Error: %s\n", config.LastExecution.Error)
	}

	// Ensure logs directory exists and write log
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err == nil {
		_ = os.WriteFile(logPath, []byte(logContent), 0o644)
	}

	_ = e.teamStore.SetHeartbeatConfig(cfgCtx, teamID, agentID, config)

	if terminalEvidence && e.runRegistry != nil {
		e.runRegistry.Complete(teamID, agentID, outcomeFailed, outcomeMessage)
		terminalEvidence = false
	}
	if err == nil && e.OnComplete != nil {
		e.OnComplete(teamID, agentID)
	}

	// Handoff resolution is optional post-processing. Notify the execution
	// owner as soon as terminal run state is durable, then allow shutdown to
	// cancel the bounded best-effort resolver instead of making completion wait
	// on a projection that may not exist.
	if err == nil {
		e.extractAndStoreHandoff(ctx, teamID, agentID, runID, endedAt)
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
		StartedAt: startedAt.Format(time.RFC3339),
		EndedAt:   endedAt.Format(time.RFC3339),
		Status:    store.HeartbeatStatusCancelled,
		Error:     detail,
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
		StartedAt: startedAt.Format(time.RFC3339),
		EndedAt:   endedAt.Format(time.RFC3339),
		Status:    store.HeartbeatStatusApprovalPending,
		Phase:     phase,
		RunID:     runID,
		Error:     detail,
	}
	config.RefreshHealth()
	_ = e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)
	e.appendAttempt(ctx, &store.HeartbeatAttempt{
		ID:         attemptID,
		TeamID:     teamID,
		AgentID:    agentID,
		ProfileKey: profileKey,
		TaskID:     taskID,
		RunID:      runID,
		Tag:        tag,
		Status:     store.HeartbeatStatusApprovalPending,
		Phase:      phase,
		StartedAt:  startedAt.Format(time.RFC3339),
		EndedAt:    endedAt.Format(time.RFC3339),
		Error:      detail,
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
