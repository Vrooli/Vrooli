package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"web-console/backends/codex"
	sessionsH "web-console/handlers/sessions"
	"web-console/internal/backend"
	"web-console/internal/config"
	"web-console/internal/sessionstore"
)

type managedCodexEntry struct {
	session sessionsH.Session
	owner   *codex.ManagedOwner
}

type managedCodexRegistry struct {
	mu        sync.RWMutex
	store     sessionstore.Store
	entries   map[string]managedCodexEntry
	register  func(context.Context, string, *codex.ManagedOwner, string, string, string) error
	eventSink func(string, codex.NativeEvent)
	start     func(context.Context, string, string, string, string, string, []string, []string) (*codex.ManagedOwner, error)
	resume    func(context.Context, string, string, string, string, string, string, []string, []string) (*codex.ManagedOwner, error)
}

func newManagedCodexRegistry(store sessionstore.Store) *managedCodexRegistry {
	return &managedCodexRegistry{store: store, entries: make(map[string]managedCodexEntry), start: func(ctx context.Context, executable, ownerID, clientName, clientVersion, cwd string, env, args []string) (*codex.ManagedOwner, error) {
		return codex.StartManagedWithEnv(ctx, executable, ownerID, clientName, clientVersion, cwd, env, args...)
	}, resume: func(ctx context.Context, executable, ownerID, clientName, clientVersion, threadID, cwd string, env, args []string) (*codex.ManagedOwner, error) {
		return codex.ResumeManagedInDirWithEnv(ctx, executable, cwd, ownerID, clientName, clientVersion, threadID, env, args...)
	}}
}

func codexLaunchArgs(d sessionstore.LaunchDescriptor) []string {
	args := []string{"app-server", "--stdio"}
	config := func(key, value string) {
		if strings.TrimSpace(value) != "" {
			args = append(args, "-c", key+"=\""+strings.ReplaceAll(value, "\"", "\\\"")+"\"")
		}
	}
	config("model", d.Model)
	config("profile", d.Profile)
	config("approval_policy", d.ApprovalPolicy)
	config("sandbox_mode", d.Sandbox)
	if strings.TrimSpace(d.ApprovalPolicy) == "" {
		args = append(args, "-c", `approval_policy="never"`)
	}
	return args
}

func codexLaunchEnv(d sessionstore.LaunchDescriptor) []string {
	keys := make([]string, 0, len(d.Environment))
	for key := range d.Environment {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		env = append(env, key+"="+d.Environment[key])
	}
	return env
}

func (r *managedCodexRegistry) setRegistrar(fn func(context.Context, string, *codex.ManagedOwner, string, string, string) error) {
	r.register = fn
}

func (r *managedCodexRegistry) setEventSink(fn func(string, codex.NativeEvent)) {
	r.eventSink = fn
}

func (r *managedCodexRegistry) watch(id string, owner *codex.ManagedOwner) {
	if owner == nil || r.eventSink == nil {
		return
	}
	go func() {
		defer r.markUnavailable(id, owner)
		notifications := owner.Notifications()
		requests := owner.Requests()
		for notifications != nil || requests != nil {
			select {
			case message, ok := <-notifications:
				if !ok {
					notifications = nil
					continue
				}
				event, mapped := codex.MapNotification(message)
				if mapped {
					if event.Kind == "turn/completed" && owner.MarkTurnCompleted(event.ThreadID, event.TurnID) && r.store != nil {
						_ = r.store.UpdateAgentInfo(context.Background(), id, sessionstore.AgentInfo{NativeThreadID: owner.ThreadID(), LastVerifiedTurnID: owner.LastVerifiedTurnID()})
					}
					r.eventSink(id, event)
				}
			case request, ok := <-requests:
				if !ok {
					requests = nil
					continue
				}
				_ = owner.RejectRequest(request.ID, "approval policy does not permit interactive requests")
			}
		}
	}()
}

func (r *managedCodexRegistry) markUnavailable(id string, owner *codex.ManagedOwner) {
	r.mu.Lock()
	entry, ok := r.entries[id]
	if ok && entry.owner == owner {
		delete(r.entries, id)
	}
	r.mu.Unlock()
	if ok && entry.owner == owner && r.store != nil {
		_ = r.store.UpdateAgentInfo(context.Background(), id, sessionstore.AgentInfo{ControlMode: sessionstore.ControlModeRecoveryOnly, NativeOwner: ""})
	}
}

func (r *managedCodexRegistry) Create(ctx context.Context, in sessionsH.CreateInput) (sessionsH.Session, error) {
	if !strings.EqualFold(strings.TrimSpace(in.AgentType), string(sessionstore.AgentCodex)) {
		return sessionsH.Session{}, fmt.Errorf("managed Codex launch requires agent_type codex")
	}
	raw := strings.TrimSpace(in.LaunchDescriptorJSON)
	if raw == "" {
		raw = fmt.Sprintf(`{"agent":"codex","launchMode":"codex_app_server"}`)
	}
	d, err := sessionstore.ParseLaunchDescriptor(raw)
	if err != nil || d.LaunchMode != string(sessionstore.LaunchModeCodexAppServer) {
		return sessionsH.Session{}, fmt.Errorf("invalid managed Codex launch descriptor: %w", err)
	}
	cwd := strings.TrimSpace(in.WorkingDir)
	if cwd == "" {
		cwd = config.ResolveWorkingDir()
	}
	if len(d.WorkspaceRoots) > 0 && (len(d.WorkspaceRoots) != 1 || strings.TrimSpace(d.WorkspaceRoots[0]) != cwd) {
		return sessionsH.Session{}, fmt.Errorf("managed Codex supports one workspace root matching working_dir")
	}
	id := uuid.NewString()
	ownerID := "web-console:" + id
	executable := strings.TrimSpace(os.Getenv("CODEX_EXECUTABLE"))
	if executable == "" {
		executable = "codex"
	}
	start := r.start
	if start == nil {
		return sessionsH.Session{}, fmt.Errorf("managed Codex owner starter is not configured")
	}
	owner, err := start(ctx, executable, ownerID, "web-console", "1.0.0", cwd, codexLaunchEnv(d), codexLaunchArgs(d))
	if err != nil {
		return sessionsH.Session{}, fmt.Errorf("start managed Codex: %w", err)
	}
	version := strings.TrimSpace(os.Getenv("CODEX_APP_SERVER_VERSION"))
	if ownerVersion := strings.TrimSpace(owner.ProviderVersion()); ownerVersion != "" {
		version = ownerVersion
	}
	if version == "" {
		version = "unknown"
	}
	createdAt := time.Now().UTC()
	if r.store != nil {
		if err := r.store.Save(ctx, sessionstore.Metadata{ID: id, Backend: backend.Persistent, Cols: uint16(in.Cols), Rows: uint16(in.Rows), Created: createdAt, Detached: true, AgentType: sessionstore.AgentCodex, LaunchMode: sessionstore.LaunchModeCodexAppServer, ControlMode: sessionstore.ControlModeUnknown, LaunchDescriptorJSON: raw, CWD: cwd}); err != nil {
			_ = owner.Close()
			return sessionsH.Session{}, err
		}
	}
	if r.register != nil {
		if err := r.register(ctx, id, owner, version, "stdio", raw); err != nil {
			_ = owner.Close()
			return sessionsH.Session{}, err
		}
	} else if r.store != nil {
		if err := r.store.UpdateAgentInfo(ctx, id, sessionstore.AgentInfo{NativeOwner: owner.OwnerID(), NativeTransport: "stdio", ProviderVersion: version, NativeThreadID: owner.ThreadID(), ControlMode: sessionstore.ControlModeNativeCapable}); err != nil {
			_ = owner.Close()
			return sessionsH.Session{}, err
		}
	}
	if r.store != nil {
		origin := sessionstore.Origin(in.Origin)
		if origin == "" {
			origin = sessionstore.OriginProgrammatic
		}
		if err := r.store.SetProvenance(ctx, id, origin, in.Owner, in.DisplayLabel); err != nil {
			_ = owner.Close()
			_ = r.store.Delete(ctx, id)
			return sessionsH.Session{}, err
		}
	}
	s := sessionsH.Session{ID: id, CreatedAt: createdAt.Format(time.RFC3339), Cols: in.Cols, Rows: in.Rows, Backend: string(backend.Persistent), SurvivesRestart: true, Origin: in.Origin, Owner: in.Owner, DisplayLabel: in.DisplayLabel, LaunchMode: string(sessionstore.LaunchModeCodexAppServer), ControlMode: string(sessionstore.ControlModeNativeCapable), NativeOwner: owner.OwnerID(), NativeTransport: "stdio", ProviderVersion: version, NativeThreadID: owner.ThreadID()}
	r.mu.Lock()
	r.entries[id] = managedCodexEntry{session: s, owner: owner}
	r.mu.Unlock()
	r.watch(id, owner)
	return s, nil
}

func (r *managedCodexRegistry) List(ctx context.Context) ([]sessionsH.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]sessionsH.Session, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e.session)
	}
	return out, nil
}

func (r *managedCodexRegistry) Get(ctx context.Context, id string) (sessionsH.Session, error) {
	r.mu.RLock()
	e, ok := r.entries[id]
	r.mu.RUnlock()
	if !ok {
		return sessionsH.Session{}, fmt.Errorf("managed Codex session %q not found", id)
	}
	return e.session, nil
}

func (r *managedCodexRegistry) owner(id string) (*codex.ManagedOwner, bool) {
	r.mu.RLock()
	entry, ok := r.entries[id]
	r.mu.RUnlock()
	return entry.owner, ok && entry.owner != nil
}

func (r *managedCodexRegistry) recordFork(ctx context.Context, id, originalThreadID, activeThreadID, turnID string) error {
	if r.store != nil {
		if err := r.store.UpdateAgentInfo(ctx, id, sessionstore.AgentInfo{NativeThreadID: activeThreadID, LastVerifiedTurnID: turnID, ForkedFromNativeSession: originalThreadID}); err != nil {
			return err
		}
	}
	r.mu.Lock()
	if entry, ok := r.entries[id]; ok {
		entry.session.NativeThreadID = activeThreadID
		entry.session.LastVerifiedTurnID = turnID
		entry.session.ForkedFromNativeSession = originalThreadID
		r.entries[id] = entry
	}
	r.mu.Unlock()
	return nil
}

func (r *managedCodexRegistry) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	e, ok := r.entries[id]
	if ok {
		delete(r.entries, id)
	}
	r.mu.Unlock()
	if !ok {
		return fmt.Errorf("managed Codex session %q not found", id)
	}
	if err := e.owner.Close(); err != nil {
		return err
	}
	if r.store != nil {
		_ = r.store.Delete(ctx, id)
	}
	return nil
}

func (s *Server) captureManagedCodexEvent(sessionID string, native codex.NativeEvent) {
	if s == nil || s.conversations == nil || native.Partial || strings.TrimSpace(native.Text) == "" {
		return
	}
	s.managedCodex.mu.RLock()
	entry, ok := s.managedCodex.entries[sessionID]
	s.managedCodex.mu.RUnlock()
	if !ok || entry.owner == nil {
		return
	}
	threadID := native.ThreadID
	if threadID == "" {
		threadID = entry.owner.ThreadID()
	}
	if threadID != entry.owner.ThreadID() && threadID != entry.owner.OriginalThreadID() {
		return
	}
	role := native.Role
	if role != "user" {
		role = "assistant"
	}
	event, result := s.conversations.AppendNativeEvent(context.Background(), sessionID, "codex_app_server", role, native.Text, &NativeProvenance{
		Provider: "codex", SessionID: threadID, TurnID: native.TurnID, MessageID: native.MessageID, BoundaryID: native.BoundaryID, CompactionLineage: native.CompactionLineage,
	})
	if result.Appended && !result.Duplicate {
		s.publishConversationEvent(event)
	}
}

// Recover resumes only rows that contain an explicit managed mode and native
// thread id. A failed resume is downgraded to recovery_only; no rollout scan
// or newest-file heuristic is allowed to select a replacement thread.
func (r *managedCodexRegistry) Recover(ctx context.Context) error {
	if r.store == nil || r.resume == nil {
		return nil
	}
	rows, err := r.store.List(ctx)
	if err != nil {
		return err
	}
	executable := strings.TrimSpace(os.Getenv("CODEX_EXECUTABLE"))
	if executable == "" {
		executable = "codex"
	}
	for _, meta := range rows {
		if meta.LaunchMode != sessionstore.LaunchModeCodexAppServer || meta.NativeThreadID == "" {
			continue
		}
		descriptor, descriptorErr := sessionstore.ParseLaunchDescriptor(meta.LaunchDescriptorJSON)
		if descriptorErr != nil {
			_ = r.store.UpdateAgentInfo(ctx, meta.ID, sessionstore.AgentInfo{ControlMode: sessionstore.ControlModeRecoveryOnly})
			continue
		}
		owner, resumeErr := r.resume(ctx, executable, "web-console:"+meta.ID, "web-console", meta.ProviderVersion, meta.NativeThreadID, meta.CWD, codexLaunchEnv(descriptor), codexLaunchArgs(descriptor))
		if resumeErr != nil {
			_ = r.store.UpdateAgentInfo(ctx, meta.ID, sessionstore.AgentInfo{ControlMode: sessionstore.ControlModeRecoveryOnly, NativeOwner: ""})
			continue
		}
		if err := owner.RestoreLineage(meta.ForkedFromNativeSession, meta.LastVerifiedTurnID); err != nil {
			_ = owner.Close()
			_ = r.store.UpdateAgentInfo(ctx, meta.ID, sessionstore.AgentInfo{ControlMode: sessionstore.ControlModeRecoveryOnly, NativeOwner: ""})
			continue
		}
		version := meta.ProviderVersion
		if ownerVersion := strings.TrimSpace(owner.ProviderVersion()); ownerVersion != "" {
			version = ownerVersion
		}
		if r.register != nil && r.register(ctx, meta.ID, owner, version, meta.NativeTransport, meta.LaunchDescriptorJSON) != nil {
			_ = owner.Close()
			continue
		}
		s := sessionsH.Session{ID: meta.ID, CreatedAt: meta.Created.UTC().Format(time.RFC3339), Cols: int(meta.Cols), Rows: int(meta.Rows), Backend: string(backend.Persistent), SurvivesRestart: true, Origin: string(meta.Origin), Owner: meta.Owner, DisplayLabel: meta.DisplayLabel, LaunchMode: string(meta.LaunchMode), ControlMode: string(sessionstore.ControlModeNativeCapable), NativeOwner: owner.OwnerID(), NativeTransport: meta.NativeTransport, ProviderVersion: version, NativeThreadID: owner.ThreadID()}
		r.mu.Lock()
		r.entries[meta.ID] = managedCodexEntry{session: s, owner: owner}
		r.mu.Unlock()
		r.watch(meta.ID, owner)
	}
	return nil
}

var _ sessionsH.ManagedCodexService = (*managedCodexRegistry)(nil)
