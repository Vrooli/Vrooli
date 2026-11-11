package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	ErrMissingThread     = errors.New("codex native thread is missing")
	ErrMissingTurn       = errors.New("codex completed turn boundary is missing")
	ErrInvalidFork       = errors.New("codex app-server returned an invalid fork")
	ErrGenerationChanged = errors.New("codex native session generation changed")
	ErrTurnChanged       = errors.New("codex active turn changed")
	ErrTurnBusy          = errors.New("codex native thread is busy")
)

// ManagedOwner is the single live owner for one Web Console-managed Codex
// thread. It deliberately does not inspect rollout files; those remain audit
// and recovery evidence only.
type ManagedOwner struct {
	client             *Client
	mu                 sync.RWMutex
	ownerID            string
	threadID           string
	originalThreadID   string
	lastVerifiedTurnID string
	activeTurnID       string
	closeOnce          sync.Once
	closing            atomic.Bool
	closeDone          chan struct{}
	closeErr           error
}

func NewManagedOwner(client *Client, ownerID, threadID string) (*ManagedOwner, error) {
	if client == nil {
		return nil, ErrClosed
	}
	if strings.TrimSpace(ownerID) == "" || strings.TrimSpace(threadID) == "" {
		return nil, ErrMissingThread
	}
	return &ManagedOwner{client: client, ownerID: ownerID, threadID: threadID, originalThreadID: threadID}, nil
}

// StartManaged establishes the complete managed lifecycle: spawn the owned
// app-server, initialize the client, announce readiness, and create one
// persisted native thread. Any failure closes the process before returning.
func StartManaged(ctx context.Context, executable, ownerID, clientName, clientVersion, cwd string, args ...string) (*ManagedOwner, error) {
	return StartManagedWithEnv(ctx, executable, ownerID, clientName, clientVersion, cwd, nil, args...)
}

func StartManagedWithEnv(ctx context.Context, executable, ownerID, clientName, clientVersion, cwd string, env []string, args ...string) (*ManagedOwner, error) {
	// The provider is owned by the Web Console session, not by the HTTP/RPC
	// request that created it. A request context is normally canceled as soon
	// as Create returns; binding exec.CommandContext to it would silently kill
	// the managed app-server immediately after a successful handshake.
	client, err := StartInDirWithEnv(context.WithoutCancel(ctx), executable, cwd, env, args...)
	if err != nil {
		return nil, err
	}
	if err := client.Initialize(ctx, clientName, clientVersion); err != nil {
		_ = client.Close()
		return nil, err
	}
	thread, err := client.StartThread(ctx, cwd)
	if err != nil || strings.TrimSpace(thread.ID()) == "" {
		_ = client.Close()
		if err != nil {
			return nil, err
		}
		return nil, ErrMissingThread
	}
	owner, err := NewManagedOwner(client, ownerID, thread.ID())
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	return owner, nil
}

func ResumeManaged(ctx context.Context, executable, ownerID, clientName, clientVersion, threadID string, args ...string) (*ManagedOwner, error) {
	return ResumeManagedInDir(ctx, executable, "", ownerID, clientName, clientVersion, threadID, args...)
}

func ResumeManagedInDir(ctx context.Context, executable, cwd, ownerID, clientName, clientVersion, threadID string, args ...string) (*ManagedOwner, error) {
	return ResumeManagedInDirWithEnv(ctx, executable, cwd, ownerID, clientName, clientVersion, threadID, nil, args...)
}

func ResumeManagedInDirWithEnv(ctx context.Context, executable, cwd, ownerID, clientName, clientVersion, threadID string, env []string, args ...string) (*ManagedOwner, error) {
	if strings.TrimSpace(threadID) == "" {
		return nil, ErrMissingThread
	}
	client, err := StartInDirWithEnv(context.WithoutCancel(ctx), executable, cwd, env, args...)
	if err != nil {
		return nil, err
	}
	if err := client.Initialize(ctx, clientName, clientVersion); err != nil {
		_ = client.Close()
		return nil, err
	}
	thread, err := client.ResumeThread(ctx, threadID)
	if err != nil || (thread.ID() != "" && thread.ID() != threadID) {
		_ = client.Close()
		if err != nil {
			return nil, err
		}
		return nil, ErrInvalidFork
	}
	return NewManagedOwner(client, ownerID, threadID)
}

func (o *ManagedOwner) Close() error { return o.CloseContext(context.Background()) }

// BeginRetirement synchronously fences raw owner references before starting
// the single transport retirement. Registries use it when marking an owner
// retiring so stale rewind/control adapters cannot admit fresh work.
func (o *ManagedOwner) BeginRetirement() {
	if o == nil || o.client == nil {
		return
	}
	o.closeOnce.Do(func() {
		o.closing.Store(true)
		o.closeDone = make(chan struct{})
		go func() { o.closeErr = o.client.Close(); close(o.closeDone) }()
	})
}

// CloseContext waits for the owned transport/process retirement. A canceled
// caller does not discard that owner or start another close; a later call can
// observe the same retirement result. It never reports success before Close ends.
func (o *ManagedOwner) CloseContext(ctx context.Context) error {
	if o == nil || o.client == nil {
		return nil
	}
	o.BeginRetirement()
	select {
	case <-o.closeDone:
		return o.closeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (o *ManagedOwner) OwnerID() string  { o.mu.RLock(); defer o.mu.RUnlock(); return o.ownerID }
func (o *ManagedOwner) ThreadID() string { o.mu.RLock(); defer o.mu.RUnlock(); return o.threadID }
func (o *ManagedOwner) OriginalThreadID() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.originalThreadID
}

func (o *ManagedOwner) LastVerifiedTurnID() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.lastVerifiedTurnID
}

// RestoreLineage reapplies durable branch identity after a Web Console
// restart. The active provider thread is resumed first; only persisted,
// previously verified lineage is allowed to restore the original branch
// reference and last completed boundary.
func (o *ManagedOwner) RestoreLineage(originalThreadID, lastVerifiedTurnID string) error {
	if o == nil {
		return ErrClosed
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if strings.TrimSpace(originalThreadID) != "" {
		o.originalThreadID = originalThreadID
	}
	if strings.TrimSpace(lastVerifiedTurnID) != "" {
		o.lastVerifiedTurnID = lastVerifiedTurnID
	}
	return nil
}

func (o *ManagedOwner) ActiveTurnID() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.activeTurnID
}

func (o *ManagedOwner) ProviderVersion() string {
	if o == nil || o.client == nil {
		return ""
	}
	return o.client.ServerVersion()
}

func (o *ManagedOwner) MarkTurnCompleted(threadID, turnID string) bool {
	if o == nil || o.closing.Load() || strings.TrimSpace(turnID) == "" {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closing.Load() || threadID != o.threadID || (o.activeTurnID != "" && o.activeTurnID != turnID) {
		return false
	}
	o.activeTurnID = ""
	o.lastVerifiedTurnID = turnID
	return true
}

// MarkTurnStarted records provider-originated turns, including goal
// continuations. A delayed start for an already completed turn must not
// resurrect it; an authoritative read reconciles any otherwise stale cache.
func (o *ManagedOwner) MarkTurnStarted(threadID, turnID string) bool {
	if o == nil || o.closing.Load() || strings.TrimSpace(turnID) == "" {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closing.Load() || threadID != o.threadID || turnID == o.lastVerifiedTurnID {
		return false
	}
	o.activeTurnID = turnID
	return true
}

func (o *ManagedOwner) VerifyThread(ctx context.Context, expected, requiredTurnID string) (bool, error) {
	if o == nil || o.client == nil || o.closing.Load() {
		return false, ErrClosed
	}
	if strings.TrimSpace(expected) == "" {
		return false, ErrMissingThread
	}
	thread, err := o.client.ReadThread(ctx, expected)
	if err != nil {
		return false, err
	}
	if thread.Thread.ID != expected {
		return false, nil
	}
	if strings.TrimSpace(requiredTurnID) == "" {
		return true, nil
	}
	for _, turn := range thread.Thread.Turns {
		if turn.ID == requiredTurnID && (turn.Status == "" || turn.Status == "completed") {
			return true, nil
		}
	}
	return false, nil
}

// Notifications exposes the owned app-server stream to the session adapter.
// The adapter may project events, but it cannot replace or mutate the owner.
func (o *ManagedOwner) Notifications() <-chan Message {
	if o == nil || o.client == nil {
		return nil
	}
	return o.client.Notifications()
}

func (o *ManagedOwner) Requests() <-chan Message {
	if o == nil || o.client == nil {
		return nil
	}
	return o.client.Requests()
}

// Done reports provider-process loss for the registry recovery path.
func (o *ManagedOwner) Done() <-chan struct{} {
	if o == nil || o.client == nil {
		return nil
	}
	return o.client.Done()
}

func (o *ManagedOwner) RejectRequest(id json.RawMessage, reason string) error {
	if o == nil || o.client == nil || o.closing.Load() {
		return ErrClosed
	}
	return o.client.Respond(id, nil, &RPCError{Code: -32001, Message: reason})
}

// ForkThrough changes the active branch only after app-server returns a new
// thread. The source thread is never discarded.
func (o *ManagedOwner) ForkThrough(ctx context.Context, completedTurnID string) (string, error) {
	if o == nil || o.client == nil || o.closing.Load() {
		return "", ErrClosed
	}
	if strings.TrimSpace(completedTurnID) == "" {
		return "", ErrMissingTurn
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closing.Load() {
		return "", ErrClosed
	}
	result, err := o.client.ForkThread(ctx, o.threadID, completedTurnID)
	if err != nil {
		return "", fmt.Errorf("fork through %s: %w", completedTurnID, err)
	}
	if strings.TrimSpace(result.ID()) == "" || result.ID() == o.threadID {
		return "", ErrInvalidFork
	}
	o.threadID = result.ID()
	o.lastVerifiedTurnID = completedTurnID
	return result.ID(), nil
}

func (o *ManagedOwner) Interrupt(ctx context.Context, turnID string) error {
	if o == nil || o.client == nil || o.closing.Load() {
		return ErrClosed
	}
	o.mu.RLock()
	threadID := o.threadID
	o.mu.RUnlock()
	if strings.TrimSpace(threadID) == "" {
		return ErrMissingThread
	}
	if strings.TrimSpace(turnID) == "" {
		return ErrMissingTurn
	}
	return o.client.InterruptTurn(ctx, threadID, turnID)
}

func (o *ManagedOwner) InterruptActive(ctx context.Context) error {
	if o == nil {
		return ErrClosed
	}
	turnID := o.ActiveTurnID()
	if turnID == "" {
		return nil
	}
	return o.Interrupt(ctx, turnID)
}

// StartTurn submits one user turn to the currently owned native thread. The
// returned ID is provider-owned and is the only valid boundary for later
// control operations.
func (o *ManagedOwner) StartTurn(ctx context.Context, text string) (string, error) {
	if o == nil || o.client == nil || o.closing.Load() {
		return "", ErrClosed
	}
	if strings.TrimSpace(text) == "" {
		return "", errors.New("codex turn text is required")
	}
	o.mu.RLock()
	threadID := o.threadID
	o.mu.RUnlock()
	result, err := o.client.StartTurn(ctx, threadID, text)
	if err != nil {
		return "", fmt.Errorf("start Codex turn: %w", err)
	}
	if strings.TrimSpace(result.ID()) == "" {
		return "", ErrMissingTurn
	}
	o.mu.Lock()
	o.activeTurnID = result.ID()
	o.mu.Unlock()
	return result.ID(), nil
}

// DeliverPrompt binds delivery to this live owner while holding the generation
// lock across the native RPC. Idle delivery starts a turn; busy delivery uses
// Codex's expected-turn CAS steering operation. If the owner exits or its turn
// changes, the native request fails and no terminal input path is involved.
func (o *ManagedOwner) DeliverPrompt(ctx context.Context, expectedThreadID, expectedTurnID, text string) (turnID, delivery string, err error) {
	if o == nil || o.client == nil || o.closing.Load() {
		return "", "", ErrClosed
	}
	if strings.TrimSpace(text) == "" {
		return "", "", errors.New("codex prompt text is required")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closing.Load() {
		return "", "", ErrClosed
	}
	if expectedThreadID == "" || expectedThreadID != o.threadID {
		return "", "", ErrGenerationChanged
	}
	if expectedTurnID == "" {
		// A delayed notification cannot veto an idle observation. Recheck the
		// same provider thread at dispatch; only a live idle thread may start.
		state := readNativeThreadStatus(ctx, o.client, o.threadID)
		if state.Status == "active" {
			return "", "", ErrTurnBusy
		}
		if state.Status != "idle" {
			return "", "", errors.New("native Codex thread is not confirmed idle")
		}
		result, callErr := o.client.StartTurn(ctx, o.threadID, text)
		if callErr != nil {
			return "", "", fmt.Errorf("start native Codex turn: %w", callErr)
		}
		turnID = result.ID()
		delivery = "started"
	} else {
		// Provider-originated notifications can lag the authoritative state read.
		// The provider's expected-turn CAS decides steering, not that cache.
		result, callErr := o.client.SteerTurn(ctx, o.threadID, expectedTurnID, text)
		if callErr != nil {
			return "", "", fmt.Errorf("steer native Codex turn: %w", callErr)
		}
		turnID = result.ID()
		delivery = "steered"
	}
	if strings.TrimSpace(turnID) == "" {
		return "", "", ErrMissingTurn
	}
	if delivery == "started" {
		o.activeTurnID = turnID
	}
	return turnID, delivery, nil
}

type GoalMutationOrigin string

const (
	GoalOriginUserDirected GoalMutationOrigin = "user"
	GoalOriginAutomatic    GoalMutationOrigin = "automatic"
)

type GoalResult struct {
	ThreadID  string `json:"threadId"`
	Objective string `json:"objective"`
	Status    string `json:"status"`
}

// SetGoal updates Codex's native thread goal while holding the same owner
// generation lock used for turn delivery. It has no terminal input fallback.
func (o *ManagedOwner) SetGoal(ctx context.Context, expectedThreadID, objective, status string, origin GoalMutationOrigin, tokenBudget *int64) (GoalResult, error) {
	if o == nil || o.client == nil || o.closing.Load() {
		return GoalResult{}, ErrClosed
	}
	if origin != GoalOriginUserDirected && origin != GoalOriginAutomatic {
		return GoalResult{}, errors.New("native goal mutation origin is required")
	}
	if strings.TrimSpace(status) == "" || (status != "active" && status != "paused" && status != "blocked" && status != "complete") {
		return GoalResult{}, errors.New("native goal status is invalid")
	}
	if status == "active" && strings.TrimSpace(objective) == "" {
		return GoalResult{}, errors.New("active native goal objective is required")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closing.Load() {
		return GoalResult{}, ErrClosed
	}
	if expectedThreadID == "" || expectedThreadID != o.threadID {
		return GoalResult{}, ErrGenerationChanged
	}
	expectedObjective := objective
	if origin == GoalOriginAutomatic {
		current, err := o.client.ReadGoal(ctx, expectedThreadID)
		if err != nil {
			return GoalResult{}, fmt.Errorf("read existing native goal before automatic mutation: %w", err)
		}
		if current == nil || (objective != "" && current.Objective != objective) {
			return GoalResult{}, errors.New("automatic native goal mutation requires the unchanged existing objective")
		}
		expectedObjective = current.Objective
	}
	params := map[string]any{"threadId": o.threadID, "status": status, "origin": origin}
	// Automatic lifecycle work never sends replacement text. The native owner
	// lock serializes our mutations, while omission also preserves a goal edited
	// by another provider actor between this read and set. Such a race fails the
	// returned-objective check instead of overwriting the user's newer intent.
	if objective != "" && origin == GoalOriginUserDirected {
		params["objective"] = objective
	}
	if tokenBudget != nil {
		params["tokenBudget"] = *tokenBudget
	}
	var result struct {
		Goal GoalResult `json:"goal"`
	}
	if err := o.client.Call(ctx, "thread/goal/set", params, &result); err != nil {
		return GoalResult{}, fmt.Errorf("set native Codex goal: %w", err)
	}
	if result.Goal.ThreadID != expectedThreadID || result.Goal.Status != status || strings.TrimSpace(result.Goal.Objective) == "" || (expectedObjective != "" && result.Goal.Objective != expectedObjective) {
		return GoalResult{}, errors.New("Codex native goal response did not confirm the requested thread, status and objective")
	}
	return result.Goal, nil
}
