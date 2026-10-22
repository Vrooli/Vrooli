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

	"github.com/robfig/cron/v3"
	repocontract "github.com/vrooli/repo-contract-go"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// HeartbeatExecutor defines the execution behavior used by the scheduler.
type HeartbeatExecutor interface {
	Execute(ctx context.Context, teamID, agentID, profileKey string) (*ExecutionResult, error)
}

// HeartbeatConfigStore defines access to heartbeat configurations for scheduling.
type HeartbeatConfigStore interface {
	GetHeartbeatConfig(ctx context.Context, teamID, agentID string) (*store.HeartbeatConfig, error)
}

type TeamStateStore interface {
	Get(ctx context.Context, teamID string) (*store.Team, error)
}

type TeamMembershipStore interface {
	GetMembers(ctx context.Context, teamID string) ([]store.TeamMemberRelation, error)
}

// ScheduledHeartbeat represents a scheduled heartbeat with its timer
type ScheduledHeartbeat struct {
	TeamID    string
	AgentID   string
	Schedule  string
	EntryID   cron.EntryID
	NextRun   time.Time
	cancelCtx context.CancelFunc
}

// Scheduler manages cron-based heartbeat execution
type Scheduler struct {
	mu             sync.RWMutex
	lifecycleMu    sync.Mutex
	cron           *cron.Cron
	scheduled      map[string]*ScheduledHeartbeat // key: teamID/agentID
	executor       HeartbeatExecutor
	running        bool
	startError     string
	lastCallback   time.Time
	profileKey     string
	agentClient    AgentClient
	configStore    HeartbeatConfigStore
	teamStore      TeamStateStore
	teamExecStore  *TeamExecutionStore
	controlStore   *HeartbeatControlStore
	admissionMu    sync.Mutex
	wakeSignals    WakeSignalProvider
	admissionState wakeAdmissionStateStore
}

// NewScheduler creates a new heartbeat scheduler
func NewScheduler(executor HeartbeatExecutor, agentClient AgentClient, configStore HeartbeatConfigStore, teamExecStore *TeamExecutionStore) *Scheduler {
	var teamStore TeamStateStore
	if teams, ok := configStore.(TeamStateStore); ok {
		teamStore = teams
	}
	var wakeSignals WakeSignalProvider
	var admissionState wakeAdmissionStateStore
	if teams, ok := configStore.(*store.FileTeamStore); ok {
		wakeSignals = FileWakeSignalProvider{Store: teams}
		admissionState = teams
	}
	return &Scheduler{
		cron:           cron.New(),
		scheduled:      make(map[string]*ScheduledHeartbeat),
		executor:       executor,
		profileKey:     "prompt-manager-heartbeat",
		agentClient:    agentClient,
		configStore:    configStore,
		teamStore:      teamStore,
		teamExecStore:  teamExecStore,
		wakeSignals:    wakeSignals,
		admissionState: admissionState,
	}
}

// SetControlStore attaches the heartbeat engagement guard. The scheduler
// treats a paused gate as "do not schedule/start" without mutating heartbeat
// configs, so resume can restore the same enabled configs later.
func (s *Scheduler) SetControlStore(controlStore *HeartbeatControlStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.controlStore = controlStore
}

// Start begins the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// Resolve installed profiles without invoking the installer. A
	// scheduler that cannot prove its profile keys is not safe to start.
	if err := s.resolveStartupProfiles(ctx); err != nil {
		s.startError = err.Error()
		return fmt.Errorf("resolve heartbeat profiles: %w", err)
	}

	s.cron.Start()
	s.running = true
	s.startError = ""
	log.Println("Heartbeat scheduler started")

	return nil
}

// Stop halts the scheduler
func (s *Scheduler) Stop() {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	s.mu.Lock()

	if !s.running {
		s.mu.Unlock()
		return
	}

	// Cancel all scheduled heartbeats
	for key, sh := range s.scheduled {
		s.cron.Remove(sh.EntryID)
		if sh.cancelCtx != nil {
			sh.cancelCtx()
		}
		delete(s.scheduled, key)
	}

	ctx := s.cron.Stop()
	s.running = false
	s.mu.Unlock()
	<-ctx.Done()
	log.Println("Heartbeat scheduler stopped")
}

// Schedule adds or updates a heartbeat schedule
func (s *Scheduler) Schedule(teamID, agentID, schedule string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.controlStore != nil {
		if paused, err := s.controlStore.AllowStart(context.Background(), teamID); err != nil {
			if errors.Is(err, ErrHeartbeatPaused) {
				log.Printf("Heartbeat schedule held for %s/%s: %s", teamID, agentID, paused.Message)
				return nil
			}
			return err
		}
	}

	key := makeKey(teamID, agentID)

	// Remove existing schedule if any
	if existing, ok := s.scheduled[key]; ok {
		s.cron.Remove(existing.EntryID)
		if existing.cancelCtx != nil {
			existing.cancelCtx()
		}
		delete(s.scheduled, key)
	}

	// Parse and validate cron expression (standard 5-field: minute hour dom month dow)
	_, err := cronParser.Parse(schedule)
	if err != nil {
		return err
	}

	// Create context for this heartbeat
	ctx, cancel := context.WithCancel(context.Background())

	// Add to cron
	entryID, err := s.cron.AddFunc(schedule, func() {
		s.mu.Lock()
		s.lastCallback = time.Now().UTC()
		s.mu.Unlock()
		s.executeHeartbeat(ctx, teamID, agentID)
	})
	if err != nil {
		cancel()
		return err
	}

	// Get next run time
	entry := s.cron.Entry(entryID)
	nextRun := entry.Next

	s.scheduled[key] = &ScheduledHeartbeat{
		TeamID:    teamID,
		AgentID:   agentID,
		Schedule:  schedule,
		EntryID:   entryID,
		NextRun:   nextRun,
		cancelCtx: cancel,
	}

	log.Printf("Scheduled heartbeat for %s/%s with schedule %s, next run: %s",
		teamID, agentID, schedule, nextRun.Format(time.RFC3339))

	return nil
}

// Unschedule removes a heartbeat schedule
func (s *Scheduler) Unschedule(teamID, agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeKey(teamID, agentID)
	if sh, ok := s.scheduled[key]; ok {
		s.cron.Remove(sh.EntryID)
		if sh.cancelCtx != nil {
			sh.cancelCtx()
		}
		delete(s.scheduled, key)
		log.Printf("Unscheduled heartbeat for %s/%s", teamID, agentID)
	}
}

// GetNextRun returns the next scheduled execution time
func (s *Scheduler) GetNextRun(teamID, agentID string) *time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := makeKey(teamID, agentID)
	if sh, ok := s.scheduled[key]; ok {
		entry := s.cron.Entry(sh.EntryID)
		return &entry.Next
	}
	return nil
}

// cronParser is the shared parser matching the cron instance (standard 5-field).
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// GetNextRuns returns up to count future execution times for the given heartbeat,
// computed by iterating the cron schedule forward from now.
func (s *Scheduler) GetNextRuns(teamID, agentID string, count int) []time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := makeKey(teamID, agentID)
	sh, ok := s.scheduled[key]
	if !ok || count <= 0 {
		return nil
	}

	schedule, err := cronParser.Parse(sh.Schedule)
	if err != nil {
		return nil
	}

	runs := make([]time.Time, 0, count)
	t := time.Now()
	for i := 0; i < count; i++ {
		t = schedule.Next(t)
		if t.IsZero() {
			break
		}
		runs = append(runs, t)
	}
	return runs
}

// ScheduledRun is one scheduled heartbeat and its next execution time.
type ScheduledRun struct {
	TeamID  string
	AgentID string
	NextRun time.Time
}

// ListScheduled returns every scheduled heartbeat with its next execution time.
func (s *Scheduler) ListScheduled() []ScheduledRun {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runs := make([]ScheduledRun, 0, len(s.scheduled))
	for _, sh := range s.scheduled {
		entry := s.cron.Entry(sh.EntryID)
		runs = append(runs, ScheduledRun{TeamID: sh.TeamID, AgentID: sh.AgentID, NextRun: entry.Next})
	}
	return runs
}

// IsScheduled checks if a heartbeat is scheduled
func (s *Scheduler) IsScheduled(teamID, agentID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := makeKey(teamID, agentID)
	_, ok := s.scheduled[key]
	return ok
}

// executeHeartbeat runs a heartbeat execution
func (s *Scheduler) executeHeartbeat(ctx context.Context, teamID, agentID string) {
	log.Printf("Executing heartbeat for %s/%s", teamID, agentID)
	config, profileKey, allowed := s.heartbeatAdmissionConfig(ctx, teamID, agentID)
	if !allowed {
		return
	}

	// Route through the team execution store so the configured queue policy is enforced.
	// The ordinary wake gate wraps the admission operation, so an unchanged
	// signal never buys an agent run while legacy and explicitly-always configs
	// retain the historical behavior.
	// Persist trigger provenance separately from the owner caller mode. The
	// executor can later honor this schedule's eligibility contract without
	// guessing from the service credential or replaying during startup.
	dispatchCtx := withHeartbeatAdmissionOrigin(ctx, AdmissionOriginScheduled)
	terminalReconciled := false
	if s.teamExecStore != nil {
		// Scheduled retries are admission operations too. Serialize them with
		// one another and with ordinary wake admission so concurrent cron
		// callbacks cannot race a retained request into duplicate waiters.
		s.admissionMu.Lock()
		retryOutcome, retryErr := s.teamExecStore.retryUnavailableWatcher(dispatchCtx, teamID, agentID)
		retryKind := "owner observation"
		retried := retryOutcome != watcherRetryNone
		if !retried {
			retried, retryErr = s.teamExecStore.RetryApprovalPending(dispatchCtx, teamID, agentID)
			retryKind = "approval"
		}
		if !retried {
			retried, retryErr = s.teamExecStore.RetryAdmissionDeferred(dispatchCtx, teamID, agentID)
			retryKind = "capacity admission"
		}
		s.admissionMu.Unlock()
		if retryErr != nil && retried {
			log.Printf("Heartbeat %s retry for %s/%s remains unresolved: %v", retryKind, teamID, agentID, retryErr)
			return
		}
		terminalReconciled = retryOutcome == watcherRetryTerminalReconciled
		if retried && !terminalReconciled {
			return
		}
		if terminalReconciled {
			// Terminal projection can update auto-pause, member configuration, or
			// team state. Re-read the ordinary gates before any new admission.
			config, profileKey, allowed = s.heartbeatAdmissionConfig(ctx, teamID, agentID)
			if !allowed {
				return
			}
		}
	}
	execute := func() error {
		if s.teamExecStore != nil {
			result, err := s.teamExecStore.Enqueue(dispatchCtx, teamID, agentID, profileKey)
			if err != nil {
				if IsMemberAlreadyQueued(err) {
					log.Printf("Heartbeat skipped for %s/%s: already queued or running", teamID, agentID)
				}
				if !IsMemberAlreadyQueued(err) {
					log.Printf("Heartbeat execution failed for %s/%s: %v", teamID, agentID, err)
				}
				return err
			}
			log.Printf("Heartbeat enqueued for %s/%s: status=%s, position=%d",
				teamID, agentID, result.Status, result.Position)
			return nil
		}

		// Fallback: direct execution (no team execution store)
		result, err := s.executor.Execute(dispatchCtx, teamID, agentID, profileKey)
		if err != nil {
			log.Printf("Heartbeat execution failed for %s/%s: %v", teamID, agentID, err)
			return err
		}

		log.Printf("Heartbeat execution completed for %s/%s, run ID: %s, status: %s",
			teamID, agentID, result.RunID, result.Status)
		return nil
	}
	if config != nil && config.LastExecution != nil && config.LastExecution.Status == store.HeartbeatStatusApprovalPending && config.LastExecution.Phase == "creating_task" {
		// A typed task-stage approval response proves the owner created nothing.
		// Re-admit on the next cron tick even when ordinary wake signals are quiet.
		_ = execute()
		return
	}
	s.runOrdinaryWake(ctx, teamID, agentID, config, execute)
}

// heartbeatAdmissionConfig evaluates the same live team, member, control, and
// heartbeat configuration gates before initial admission and after terminal
// recovery. A terminal failure may auto-pause the member, so recovered state
// must never reuse the pre-reconciliation snapshot.
func (s *Scheduler) heartbeatAdmissionConfig(ctx context.Context, teamID, agentID string) (*store.HeartbeatConfig, string, bool) {
	if s.teamStore != nil {
		team, err := s.teamStore.Get(ctx, teamID)
		if err != nil {
			log.Printf("Heartbeat execution aborted for %s/%s: team lookup failed: %v", teamID, agentID, err)
			return nil, "", false
		}
		if !team.Enabled || team.Archived {
			log.Printf("Heartbeat execution skipped for %s/%s: team disabled or archived", teamID, agentID)
			return nil, "", false
		}
		if members, ok := s.teamStore.(TeamMembershipStore); ok {
			roster, err := members.GetMembers(ctx, teamID)
			if err != nil {
				log.Printf("Heartbeat execution aborted for %s/%s: roster lookup failed: %v", teamID, agentID, err)
				return nil, "", false
			}
			found := false
			for _, member := range roster {
				if member.AgentID == agentID && (member.Status == "" || member.Status == store.MemberStatusActive) {
					found = true
					break
				}
			}
			if !found {
				log.Printf("Heartbeat execution skipped for %s/%s: member is no longer active on the team", teamID, agentID)
				return nil, "", false
			}
		}
	}

	if s.controlStore != nil {
		if paused, err := s.controlStore.AllowStart(ctx, teamID); err != nil {
			if errors.Is(err, ErrHeartbeatPaused) {
				log.Printf("Heartbeat execution skipped for %s/%s: %s", teamID, agentID, paused.Message)
				return nil, "", false
			}
			log.Printf("Heartbeat execution gate failed for %s/%s: %v", teamID, agentID, err)
			return nil, "", false
		}
	}

	var profileKey string
	var config *store.HeartbeatConfig
	if s.configStore != nil {
		var err error
		config, err = s.configStore.GetHeartbeatConfig(ctx, teamID, agentID)
		if err != nil {
			log.Printf("Heartbeat execution aborted for %s/%s: %v", teamID, agentID, err)
			return nil, "", false
		}
		if config == nil {
			log.Printf("Heartbeat execution aborted for %s/%s: config not found", teamID, agentID)
			return nil, "", false
		}
		if !config.Enabled {
			log.Printf("Heartbeat execution skipped for %s/%s: config disabled", teamID, agentID)
			return nil, "", false
		}
		profileKey = config.ProfileKey
	}
	return config, profileKey, true
}

// runOrdinaryWake applies only to ordinary heartbeats. It holds a process-local
// admission lock across the signal check and enqueue so two cron callbacks
// cannot consume the same change concurrently. Runtime evidence is persisted
// separately from heartbeat.json and is best-effort: any inability to prove a
// signal or persist its baseline admits work rather than suppressing it.
func (s *Scheduler) runOrdinaryWake(ctx context.Context, teamID, agentID string, config *store.HeartbeatConfig, execute func() error) {
	if config == nil {
		_ = execute()
		return
	}
	policy := config.WakeAdmission
	if policy == nil || policy.Mode != teamconfig.WakeAdmissionOnChange || s.wakeSignals == nil || s.admissionState == nil {
		_ = execute()
		return
	}

	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()

	previous, err := s.admissionState.GetHeartbeatAdmissionState(ctx, teamID, agentID)
	if err != nil {
		log.Printf("Heartbeat wake admission unavailable for %s/%s; admitting work: state read failed: %v", teamID, agentID, err)
		_ = execute()
		return
	}
	signal, err := s.wakeSignals.Snapshot(ctx, teamID, agentID, policy.ChangeSources)
	if err != nil {
		log.Printf("Heartbeat wake admission unavailable for %s/%s; admitting work: signal read failed: %v", teamID, agentID, err)
		if previous == nil {
			previous = &store.HeartbeatAdmissionState{Version: 1}
		}
		recordWakeAdmissionFailOpen(previous, "signal-read", time.Now())
		if persistErr := s.admissionState.SetHeartbeatAdmissionState(ctx, teamID, agentID, previous); persistErr != nil {
			log.Printf("Heartbeat wake admission fail-open telemetry persistence failed for %s/%s: %v", teamID, agentID, persistErr)
		}
		_ = execute()
		return
	}
	decision := evaluateWakeAdmission(policy, previous, signal)
	if decision.Decision == wakeDecisionQuiet {
		recordWakeAdmission(previous, signal, decision, time.Now())
		if err := s.admissionState.SetHeartbeatAdmissionState(ctx, teamID, agentID, previous); err != nil {
			log.Printf("Heartbeat wake admission state update failed for %s/%s: %v", teamID, agentID, err)
		}
		log.Printf("Heartbeat skipped for %s/%s: wake signal unchanged", teamID, agentID)
		return
	}

	if err := execute(); err != nil {
		// Do not consume a change when the actual queue/execute operation failed.
		return
	}
	if previous == nil {
		previous = &store.HeartbeatAdmissionState{Version: 1}
	}
	recordWakeAdmission(previous, signal, decision, time.Now())
	if err := s.admissionState.SetHeartbeatAdmissionState(ctx, teamID, agentID, previous); err != nil {
		log.Printf("Heartbeat wake admission state persistence failed for %s/%s: %v", teamID, agentID, err)
	}
}

// resolveStartupProfiles qualifies the existing declaration/installation pair.
// Local overrides and source drift remain with AM's installer; startup refuses
// to overwrite them. Historical run filtering only requires exact identity.
func (s *Scheduler) resolveStartupProfiles(ctx context.Context) error {
	if s.agentClient == nil {
		return fmt.Errorf("agent profile reader unavailable")
	}
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return err
	}
	for _, filename := range []string{"heartbeat.json", "heartbeat-single-process.json"} {
		source := filepath.Join(".vrooli", "agent-manager", filename)
		body, err := os.ReadFile(filepath.Join(root, "scenarios", "prompt-manager", source))
		if err != nil {
			return err
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			return err
		}
		var version string
		if err := json.Unmarshal(raw["schemaVersion"], &version); err != nil || version != "agent-profile/v1" {
			return fmt.Errorf("unsupported profile source version: %s", source)
		}
		delete(raw, "schemaVersion")
		encoded, err := json.Marshal(raw)
		if err != nil {
			return err
		}
		expected := &AgentProfile{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(encoded, expected); err != nil {
			return err
		}
		installed, err := s.agentClient.ResolveProfile(ctx, expected.ProfileKey)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(body)
		hash := hex.EncodeToString(sum[:])
		if installed == nil || installed.OwnerScenario != "prompt-manager" || installed.SourcePath != filepath.ToSlash(source) || installed.SourceHash != hash || installed.LastAppliedHash != hash || installed.LocalOverride {
			return fmt.Errorf("profile %q installation provenance or override unavailable", expected.ProfileKey)
		}
		// AM normalizes absent network policy to localhost, tool enforcement to
		// enforced, and sandbox lifecycle/acceptance to empty typed messages.
		if expected.NetworkAccess == domainpb.NetworkAccess_NETWORK_ACCESS_UNSPECIFIED {
			expected.NetworkAccess = domainpb.NetworkAccess_NETWORK_ACCESS_LOCALHOST
		}
		if expected.ToolRestrictionPolicy == "" {
			expected.ToolRestrictionPolicy = "enforced"
		}
		if expected.SandboxConfig != nil {
			if expected.SandboxConfig.NetworkMode == domainpb.NetworkAccess_NETWORK_ACCESS_UNSPECIFIED {
				expected.SandboxConfig.NetworkMode = domainpb.NetworkAccess_NETWORK_ACCESS_LOCALHOST
			}
			if expected.SandboxConfig.Lifecycle == nil {
				expected.SandboxConfig.Lifecycle = &domainpb.SandboxLifecycleConfig{}
			}
			if expected.SandboxConfig.Acceptance == nil {
				expected.SandboxConfig.Acceptance = &domainpb.SandboxAcceptanceConfig{}
			}
		}
		actual := proto.Clone(installed).(*AgentProfile)
		actual.Id = ""
		actual.OwnerScenario = ""
		actual.SourcePath = ""
		actual.SourceHash = ""
		actual.LastAppliedHash = ""
		actual.SourceUpdatedAt = nil
		actual.CreatedAt = nil
		actual.UpdatedAt = nil
		actual.LocalOverride = false
		if !proto.Equal(actual, expected) {
			return fmt.Errorf("profile %q installed execution contract differs from declaration", expected.ProfileKey)
		}
	}
	return nil
}

// SchedulerState separates registration, running cron and observed callback.
type SchedulerState struct {
	Running      bool       `json:"running"`
	StartError   string     `json:"start_error,omitempty"`
	Registered   int        `json:"registered"`
	LastCallback *time.Time `json:"last_callback,omitempty"`
}

func (s *Scheduler) State() SchedulerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := SchedulerState{Running: s.running, StartError: s.startError, Registered: len(s.scheduled)}
	if !s.lastCallback.IsZero() {
		last := s.lastCallback
		state.LastCallback = &last
	}
	return state
}

type declaredProfile struct {
	ProfileKey string `json:"profileKey"`
}

type DeclaredProfileKeys struct {
	MultiProcess  string
	SingleProcess string
}

// LoadDeclaredProfileKeys reads the scenario-owned profile declarations from
// the repository. The declarations remain the only source of their keys.
func LoadDeclaredProfileKeys() (DeclaredProfileKeys, error) {
	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return DeclaredProfileKeys{}, fmt.Errorf("resolve repository root: %w", err)
	}
	return loadDeclaredProfileKeys(repoRoot)
}

func loadDeclaredProfileKeys(repoRoot string) (DeclaredProfileKeys, error) {
	load := func(filename string) (string, error) {
		path := filepath.Join(repoRoot, "scenarios", "prompt-manager", ".vrooli", "agent-manager", filename)
		payload, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", path, err)
		}
		var declaration declaredProfile
		if err := json.Unmarshal(payload, &declaration); err != nil {
			return "", fmt.Errorf("decode %s: %w", path, err)
		}
		if strings.TrimSpace(declaration.ProfileKey) == "" {
			return "", fmt.Errorf("profileKey is empty in %s", path)
		}
		return declaration.ProfileKey, nil
	}

	multi, err := load("heartbeat.json")
	if err != nil {
		return DeclaredProfileKeys{}, err
	}
	single, err := load("heartbeat-single-process.json")
	if err != nil {
		return DeclaredProfileKeys{}, err
	}
	return DeclaredProfileKeys{MultiProcess: multi, SingleProcess: single}, nil
}

// DefaultProfileKeyForRuntimeMode returns the portable profile appropriate for
// the team's interaction mode.
func DefaultProfileKeyForRuntimeMode(runtimeMode string) (string, error) {
	keys, err := LoadDeclaredProfileKeys()
	if err != nil {
		return "", err
	}
	if runtimeMode == teamconfig.RuntimeModeSingleProcess {
		return keys.SingleProcess, nil
	}
	return keys.MultiProcess, nil
}

// makeKey creates a unique key for a team/agent combination
func makeKey(teamID, agentID string) string {
	return teamID + "/" + agentID
}
