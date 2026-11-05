package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrMissingThread = errors.New("codex native thread is missing")
	ErrMissingTurn   = errors.New("codex completed turn boundary is missing")
	ErrInvalidFork   = errors.New("codex app-server returned an invalid fork")
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

func (o *ManagedOwner) Close() error {
	if o == nil || o.client == nil {
		return nil
	}
	return o.client.Close()
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
	if o == nil || strings.TrimSpace(turnID) == "" {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if threadID != "" && threadID != o.threadID && threadID != o.originalThreadID {
		return false
	}
	o.activeTurnID = ""
	o.lastVerifiedTurnID = turnID
	return true
}

func (o *ManagedOwner) VerifyThread(ctx context.Context, expected, requiredTurnID string) (bool, error) {
	if o == nil || o.client == nil {
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
	if o == nil || o.client == nil {
		return ErrClosed
	}
	return o.client.Respond(id, nil, &RPCError{Code: -32001, Message: reason})
}

// ForkThrough changes the active branch only after app-server returns a new
// thread. The source thread is never discarded.
func (o *ManagedOwner) ForkThrough(ctx context.Context, completedTurnID string) (string, error) {
	if o == nil || o.client == nil {
		return "", ErrClosed
	}
	if strings.TrimSpace(completedTurnID) == "" {
		return "", ErrMissingTurn
	}
	o.mu.Lock()
	defer o.mu.Unlock()
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
	if o == nil || o.client == nil {
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
	if o == nil || o.client == nil {
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
