package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"web-console/backends/codex"
	"web-console/internal/sessionstore"
)

// ControlReason is stable API vocabulary. UI copy is selected from these
// codes; server prose is diagnostic only.
type ControlReason string

const (
	ControlSupported            ControlReason = "supported"
	ControlUnsupported          ControlReason = "unsupported"
	ControlUnavailable          ControlReason = "unavailable"
	ControlStaleTarget          ControlReason = "stale_target"
	ControlAlreadyAtTarget      ControlReason = "already_at_target"
	ControlConfirmationRequired ControlReason = "confirmation_required"
	ControlFailedPreserved      ControlReason = "failed_preserved"
	ControlFailedUncertain      ControlReason = "failed_uncertain"
)

type HarnessCapability struct {
	Provider             string                        `json:"provider"`
	Operations           []string                      `json:"operations"`
	VersionRange         string                        `json:"versionRange"`
	RequiresProvenance   bool                          `json:"requiresProvenance"`
	Destructive          bool                          `json:"destructive"`
	VerificationStrength string                        `json:"verificationStrength"`
	Available            bool                          `json:"available"`
	ReasonCode           ControlReason                 `json:"reasonCode"`
	LaunchMode           string                        `json:"launchMode,omitempty"`
	ControlMode          string                        `json:"controlMode,omitempty"`
	NativeOwner          string                        `json:"nativeOwner,omitempty"`
	Options              []ConversationOperationOption `json:"options"`
}

// ConversationOperationOption is the provider-neutral UI contract. Effects
// are explicit so a client never has to infer whether an action changes the
// branch or workspace from a label such as "rewind".
type ConversationOperationOption struct {
	ID                   string   `json:"id"`
	Label                string   `json:"label"`
	Supported            bool     `json:"supported"`
	Default              bool     `json:"default"`
	ChangesConversation  bool     `json:"changesConversation"`
	ChangesWorkspace     bool     `json:"changesWorkspace"`
	CreatesBranch        bool     `json:"createsBranch"`
	RequiresConfirmation bool     `json:"requiresConfirmation"`
	Boundary             string   `json:"boundary"`
	Consequences         []string `json:"consequences"`
	UnavailableReason    string   `json:"unavailableReason,omitempty"`
}

// NativeRewindAdapter is the owner seam for a harness that can prove rewind.
// It is intentionally narrower than terminal input: implementations must use
// native identifiers and return a verified postcondition before projection
// reconciliation is permitted.
type NativeRewindAdapter interface {
	Provider() string
	Preflight(context.Context, ConversationEvent, ConversationControlPreflight) (ConversationControlPreflight, error)
	Execute(context.Context, ConversationEvent, bool) (string, error)
	Verify(context.Context, ConversationEvent, string) (bool, error)
}

type ConversationControlPreflight struct {
	OperationID     string            `json:"operationId"`
	Operation       string            `json:"operation"`
	SessionID       string            `json:"sessionId"`
	EventID         string            `json:"eventId"`
	Sequence        int64             `json:"sequence"`
	Capability      HarnessCapability `json:"capability"`
	Decision        ControlReason     `json:"decision"`
	ConfirmationKey string            `json:"confirmationKey,omitempty"`
	TargetDigest    string            `json:"targetDigest"`
	Consequences    []string          `json:"consequences"`
	ExpiresAt       time.Time         `json:"expiresAt"`
}

type conversationControlRequest struct {
	OperationID     string `json:"operationId,omitempty"`
	Operation       string `json:"operation,omitempty"`
	EventID         string `json:"eventId"`
	ConfirmationKey string `json:"confirmationKey"`
	PreserveDraft   bool   `json:"preserveDraft"`
}

type conversationControlOutcome struct {
	OperationID string        `json:"operationId"`
	SessionID   string        `json:"sessionId"`
	EventID     string        `json:"eventId"`
	State       string        `json:"state"`
	ReasonCode  ControlReason `json:"reasonCode"`
	Detail      string        `json:"detail"`
	TerminalURL string        `json:"terminalUrl,omitempty"`
}

type conversationControlState struct {
	mu          sync.Mutex
	preflights  map[string]ConversationControlPreflight
	adapters    map[string]NativeRewindAdapter
	codexOwners map[string]*codex.ManagedOwner
}

func (s *Server) controlReceipt(ctx context.Context, operationID string) (conversationControlOutcome, bool) {
	if s == nil || s.db == nil || strings.TrimSpace(operationID) == "" {
		return conversationControlOutcome{}, false
	}
	var out conversationControlOutcome
	err := s.db.QueryRowContext(ctx, `SELECT operation_id, session_id, event_id, state, reason_code, detail FROM conversation_control_receipts WHERE operation_id = ?`, operationID).
		Scan(&out.OperationID, &out.SessionID, &out.EventID, &out.State, &out.ReasonCode, &out.Detail)
	return out, err == nil
}

func (s *Server) beginControlReceipt(ctx context.Context, operationID, sessionID, eventID string) bool {
	if s == nil || s.db == nil {
		return true
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO conversation_control_receipts(operation_id, session_id, event_id, state, reason_code, detail, created_at, updated_at) VALUES (?, ?, ?, 'executing', 'supported', 'Native rewind is executing.', ?, ?)`, operationID, sessionID, eventID, now, now)
	if err != nil {
		return false
	}
	inserted, err := result.RowsAffected()
	return err == nil && inserted == 1
}

func (s *Server) completeControlReceipt(ctx context.Context, out conversationControlOutcome) {
	if s == nil || s.db == nil || out.OperationID == "" {
		return
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE conversation_control_receipts SET state = ?, reason_code = ?, detail = ?, updated_at = ? WHERE operation_id = ?`, out.State, out.ReasonCode, out.Detail, time.Now().UTC().Format(time.RFC3339Nano), out.OperationID)
}

// reconcileOrphanedControlReceipts runs once during server startup. An
// executing receipt cannot still have an in-memory provider owner after a
// process restart, so it must be surfaced as uncertain rather than replayed
// as success or silently left executing forever.
func (s *Server) reconcileOrphanedControlReceipts(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `UPDATE conversation_control_receipts
		SET state = 'failed-uncertain', reason_code = 'failed_uncertain',
		    detail = 'The Web Console restarted while native control was executing; provider state requires reconciliation.',
		    updated_at = ?
		WHERE state = 'executing'`, now)
	return err
}

func (s *Server) controlState() *conversationControlState {
	s.conversationControlMu.Lock()
	defer s.conversationControlMu.Unlock()
	if s.conversationControl == nil {
		s.conversationControl = &conversationControlState{preflights: make(map[string]ConversationControlPreflight), adapters: make(map[string]NativeRewindAdapter), codexOwners: make(map[string]*codex.ManagedOwner)}
	}
	return s.conversationControl
}

// registerManagedCodexOwner is the only path that can grant native-capable
// mode. The caller must have already completed the app-server handshake and
// verified the native thread; persisting that result makes restart behavior
// explicit and prevents a second writer from being attached by accident.
func (s *Server) registerManagedCodexOwner(ctx context.Context, webSessionID string, owner *codex.ManagedOwner, version, transport, descriptorJSON string) error {
	if s == nil || owner == nil || strings.TrimSpace(webSessionID) == "" {
		return fmt.Errorf("managed Codex owner is incomplete")
	}
	state := s.controlState()
	state.mu.Lock()
	if existing := state.codexOwners[webSessionID]; existing != nil && existing != owner {
		state.mu.Unlock()
		return fmt.Errorf("managed Codex session already has a different owner")
	}
	state.codexOwners[webSessionID] = owner
	state.mu.Unlock()
	if s.sessionStore == nil {
		return nil
	}
	if err := s.sessionStore.UpdateAgentInfo(ctx, webSessionID, sessionstore.AgentInfo{
		AgentType: sessionstore.AgentCodex, LaunchMode: sessionstore.LaunchModeCodexAppServer,
		ControlMode: sessionstore.ControlModeNativeCapable, NativeOwner: owner.OwnerID(),
		NativeTransport: transport, ProviderVersion: version, NativeThreadID: owner.ThreadID(),
		LastVerifiedTurnID: owner.LastVerifiedTurnID(), LaunchDescriptorJSON: descriptorJSON,
	}); err != nil {
		state.mu.Lock()
		delete(state.codexOwners, webSessionID)
		state.mu.Unlock()
		return fmt.Errorf("persist managed Codex ownership: %w", err)
	}
	return nil
}

func (s *Server) nativeRewindAdapter(sessionID, provider string) NativeRewindAdapter {
	if strings.Contains(strings.ToLower(provider), "codex") {
		state := s.controlState()
		state.mu.Lock()
		defer state.mu.Unlock()
		if owner := state.codexOwners[sessionID]; owner != nil {
			return codexConversationAdapter{owner: owner}
		}
	}
	if strings.Contains(strings.ToLower(provider), "opencode") && s.opencodeWatcher != nil {
		if client := s.opencodeWatcher.nativeClient(); client != nil {
			return openCodeRewindAdapter{client: client}
		}
	}
	return nil
}

func conversationCapabilities(provider string) HarnessCapability {
	switch {
	case strings.Contains(provider, "claude"):
		provider = "claude"
	case strings.Contains(provider, "codex"):
		provider = "codex"
	case strings.Contains(provider, "grok"):
		provider = "grok"
	case strings.Contains(provider, "opencode"):
		provider = "opencode"
	}
	capability := HarnessCapability{Provider: provider, Operations: []string{"restore_conversation"}, VersionRange: "verified-native-only", RequiresProvenance: true, Destructive: false, VerificationStrength: "native-postcondition", Available: false, ReasonCode: ControlUnsupported}
	capability.Options = []ConversationOperationOption{
		{ID: "restore_conversation", Label: "Restore conversation", Default: true, ChangesConversation: true, CreatesBranch: true, RequiresConfirmation: true, Boundary: "through", Consequences: []string{"Files and workspace state remain unchanged.", "The original conversation branch remains available when the provider supports forking."}},
		{ID: "fork_conversation", Label: "Fork conversation", ChangesConversation: true, CreatesBranch: true, RequiresConfirmation: true, Boundary: "through", Consequences: []string{"The selected conversation becomes a new branch; files remain unchanged."}, UnavailableReason: "The active harness has not verified an explicit fork operation."},
		{ID: "restore_code", Label: "Restore code", ChangesWorkspace: true, RequiresConfirmation: true, Boundary: "provider_defined", Consequences: []string{"Workspace files would change."}, UnavailableReason: "Code restoration is not verified for this harness."},
		{ID: "restore_conversation_and_code", Label: "Restore conversation and code", ChangesConversation: true, ChangesWorkspace: true, CreatesBranch: true, RequiresConfirmation: true, Boundary: "provider_defined", Consequences: []string{"Conversation and workspace files would change."}, UnavailableReason: "Combined restoration is not verified for this harness."},
		{ID: "summarize_from_message", Label: "Summarize from message", ChangesConversation: true, RequiresConfirmation: true, Boundary: "through", Consequences: []string{"Conversation history would be summarized."}, UnavailableReason: "Summarization from a message is not verified for this harness."},
	}
	if provider == "opencode" {
		capability.VersionRange = ">=1.18.30 <1.19.0"
	}
	if provider != "claude" && provider != "codex" && provider != "grok" && provider != "opencode" {
		capability.Provider = "unknown"
	}
	return capability
}

func legacyCapability(provider string) HarnessCapability {
	c := conversationCapabilities(provider)
	c.LaunchMode = "terminal_pty"
	c.ControlMode = "transcript_only"
	c.Available = false
	c.ReasonCode = ControlUnsupported
	for i := range c.Options {
		c.Options[i].Supported = false
		c.Options[i].UnavailableReason = "This session is terminal-owned; use the terminal recovery path."
	}
	return c
}

func writeControlJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Server) handleConversationControlPreflight(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimSpace(mux.Vars(r)["id"])
	var req conversationControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.EventID) == "" {
		writeControlJSON(w, http.StatusBadRequest, map[string]any{"decision": ControlUnavailable, "reasonCode": "invalid_request"})
		return
	}
	operationID := fmt.Sprintf("rewind-%d", time.Now().UnixNano())
	managedLive := false
	if s.managedCodex != nil {
		_, managedErr := s.managedCodex.Get(r.Context(), sessionID)
		managedLive = managedErr == nil
	}
	legacyLive := false
	if s.sessions != nil {
		_, legacyLive = s.sessions.Get(sessionID)
	}
	if !legacyLive && !managedLive {
		writeControlJSON(w, http.StatusOK, ConversationControlPreflight{OperationID: operationID, SessionID: sessionID, EventID: req.EventID, Decision: ControlUnavailable, Capability: conversationCapabilities("unknown"), Consequences: []string{"Live session ownership could not be established."}, ExpiresAt: time.Now().Add(2 * time.Minute)})
		return
	}
	if !sessionStillOwned(s, sessionID) {
		writeControlJSON(w, http.StatusOK, ConversationControlPreflight{OperationID: operationID, SessionID: sessionID, EventID: req.EventID, Decision: ControlUnavailable, Capability: conversationCapabilities("unknown"), Consequences: []string{"The session is not live or is no longer owned by this Web Console instance."}, ExpiresAt: time.Now().Add(2 * time.Minute)})
		return
	}
	event, found := s.conversations.GetEvent(r.Context(), sessionID, strings.TrimSpace(req.EventID))
	if !found {
		writeControlJSON(w, http.StatusOK, ConversationControlPreflight{OperationID: operationID, SessionID: sessionID, EventID: req.EventID, Decision: ControlStaleTarget, Capability: conversationCapabilities("unknown"), Consequences: []string{"The selected projection event no longer exists."}, ExpiresAt: time.Now().Add(2 * time.Minute)})
		return
	}
	provider := event.Source
	if provider == "" {
		provider = "unknown"
	}
	capability := conversationCapabilities(provider)
	// Historical rows have no mode metadata. Treat them conservatively as the
	// existing terminal path; native capability requires a session-specific
	// persisted handshake, never merely an installed app-server binary.
	if s.sessionStore != nil && strings.Contains(strings.ToLower(provider), "codex") {
		if meta, err := s.sessionStore.Get(r.Context(), sessionID); err == nil {
			if meta.LaunchMode != "codex_app_server" || meta.ControlMode != "native_capable" || meta.NativeOwner == "" {
				capability = legacyCapability(provider)
			} else {
				capability.LaunchMode = string(meta.LaunchMode)
				capability.ControlMode = string(meta.ControlMode)
				capability.NativeOwner = meta.NativeOwner
			}
		}
	}
	digest := sha256.Sum256([]byte(strings.Join([]string{sessionID, event.ID, fmt.Sprint(event.Sequence), event.Text}, "\x00")))
	preflight := ConversationControlPreflight{OperationID: operationID, Operation: req.Operation, SessionID: sessionID, EventID: event.ID, Sequence: event.Sequence, Capability: capability, Decision: ControlUnavailable, TargetDigest: hex.EncodeToString(digest[:]), Consequences: []string{"No verified native rewind adapter is registered for this harness.", "The projected transcript will remain unchanged.", "Open Terminal to use the harness-owned recovery path."}, ExpiresAt: time.Now().Add(2 * time.Minute)}
	if preflight.Operation == "" || preflight.Operation == "rewind" {
		preflight.Operation = "restore_conversation"
	}
	adapter := s.nativeRewindAdapter(sessionID, provider)
	if req.Operation != "" && req.Operation != "restore_conversation" && req.Operation != "rewind" {
		preflight.Decision = ControlUnsupported
		preflight.Consequences = []string{"This operation is not advertised by the active harness."}
		adapter = nil
	}
	if event.NativeProvenance == nil {
		preflight.Decision = ControlUnavailable
		preflight.Consequences = append(preflight.Consequences, "This message has no harness-owned native provenance.")
	} else if adapter != nil && preflight.Decision != ControlUnsupported {
		var err error
		preflight, err = adapter.Preflight(r.Context(), event, preflight)
		if err != nil {
			preflight.Decision = ControlUnavailable
			preflight.Consequences = append(preflight.Consequences, "The native owner could not be reached; projected history was preserved.")
		}
	} else {
		preflight.Decision = ControlUnsupported
	}
	state := s.controlState()
	state.mu.Lock()
	state.preflights[operationID] = preflight
	if adapter != nil {
		state.adapters[operationID] = adapter
	}
	state.mu.Unlock()
	writeControlJSON(w, http.StatusOK, preflight)
}

func (s *Server) handleConversationControlExecute(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimSpace(mux.Vars(r)["id"])
	var req conversationControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.EventID) == "" {
		writeControlJSON(w, http.StatusBadRequest, map[string]any{"state": "failed-preserved", "reasonCode": "invalid_request"})
		return
	}
	if previous, found := s.controlReceipt(r.Context(), req.OperationID); found {
		// A completed receipt is a replay. An executing receipt belongs to a
		// concurrent request; returning it is safer than invoking the provider
		// twice and gives the caller a durable in-progress state.
		writeControlJSON(w, http.StatusOK, previous)
		return
	}
	state := s.controlState()
	state.mu.Lock()
	preflight, ok := state.preflights[req.OperationID]
	if ok && time.Now().After(preflight.ExpiresAt) {
		delete(state.preflights, req.OperationID)
		ok = false
	}
	state.mu.Unlock()
	outcome := conversationControlOutcome{OperationID: req.OperationID, SessionID: sessionID, EventID: req.EventID, State: "failed-preserved", ReasonCode: ControlUnsupported, Detail: "No verified native rewind adapter is available; projected history was preserved."}
	state.mu.Lock()
	adapter := state.adapters[req.OperationID]
	state.mu.Unlock()
	if !ok || preflight.SessionID != sessionID || preflight.EventID != req.EventID || (req.Operation != "" && req.Operation != preflight.Operation && !(req.Operation == "rewind" && preflight.Operation == "restore_conversation")) {
		outcome.ReasonCode = ControlStaleTarget
		outcome.Detail = "A fresh preflight and confirmation binding are required; projected history was preserved."
	} else if preflight.Decision != ControlSupported || adapter == nil {
		outcome.ReasonCode = preflight.Decision
		if outcome.ReasonCode == "" {
			outcome.ReasonCode = ControlUnsupported
		}
		outcome.Detail = "The selected operation is not supported by the active harness; projected history was preserved."
	} else if req.ConfirmationKey == "" || req.ConfirmationKey != preflight.ConfirmationKey {
		outcome.ReasonCode = ControlConfirmationRequired
		outcome.Detail = "A matching confirmation is required; projected history was preserved."
	} else if preflight.Decision == ControlSupported && adapter != nil {
		event, found := s.conversations.GetEvent(r.Context(), sessionID, req.EventID)
		if !found {
			outcome.ReasonCode = ControlStaleTarget
			outcome.Detail = "The selected native target no longer exists; projected history was preserved."
		} else if !sessionStillOwned(s, sessionID) {
			outcome.ReasonCode = ControlFailedPreserved
			outcome.Detail = "Live session ownership could not be revalidated; projected history was preserved."
		} else if !sessionStillOwned(s, sessionID) {
			outcome.ReasonCode = ControlStaleTarget
			outcome.Detail = "Live session ownership changed after preflight; projected history was preserved."
		} else {
			digest := sha256.Sum256([]byte(strings.Join([]string{sessionID, event.ID, fmt.Sprint(event.Sequence), event.Text}, "\x00")))
			if hex.EncodeToString(digest[:]) != preflight.TargetDigest {
				outcome.ReasonCode = ControlStaleTarget
				outcome.Detail = "The projected target changed after preflight; projected history was preserved."
				s.completeControlReceipt(r.Context(), outcome)
				writeControlJSON(w, http.StatusOK, outcome)
				return
			}
			// Claim the durable receipt only after all request, confirmation,
			// liveness, and target-digest checks pass. Invalid retries must not
			// consume an operation ID that a valid retry still needs.
			if !s.beginControlReceipt(r.Context(), req.OperationID, sessionID, req.EventID) {
				if existing, found := s.controlReceipt(r.Context(), req.OperationID); found {
					writeControlJSON(w, http.StatusOK, existing)
				} else {
					outcome.ReasonCode = ControlFailedPreserved
					outcome.Detail = "The rewind operation could not be recorded before side effects; projected history was preserved."
					writeControlJSON(w, http.StatusOK, outcome)
				}
				return
			}
			target, err := adapter.Execute(r.Context(), event, req.PreserveDraft)
			if err != nil {
				outcome.ReasonCode = ControlFailedUncertain
				outcome.State = "failed-uncertain"
				outcome.Detail = "The native owner did not confirm rewind; reconciliation is required before changing projected history."
			} else {
				verified, verifyErr := adapter.Verify(r.Context(), event, target)
				if verifyErr != nil || !verified {
					outcome.ReasonCode = ControlFailedUncertain
					outcome.State = "failed-uncertain"
					outcome.Detail = "The native owner did not report the selected target after rewind; projected history was preserved."
				} else {
					if strings.Contains(strings.ToLower(event.Source), "codex") && s.managedCodex != nil && event.NativeProvenance != nil {
						if err := s.managedCodex.recordFork(r.Context(), sessionID, event.NativeProvenance.SessionID, target, event.NativeProvenance.TurnID); err != nil {
							outcome.ReasonCode = ControlFailedUncertain
							outcome.State = "failed-uncertain"
							outcome.Detail = "Native rewind was verified but branch metadata could not be persisted; projected history was preserved for recovery."
							s.completeControlReceipt(r.Context(), outcome)
							writeControlJSON(w, http.StatusOK, outcome)
							return
						}
					}
					if err := s.conversations.TruncateSessionAfter(r.Context(), sessionID, event.Sequence); err != nil {
						outcome.ReasonCode = ControlFailedUncertain
						outcome.State = "failed-uncertain"
						outcome.Detail = "Native rewind was verified but projection reconciliation failed; projected history was preserved for recovery."
					} else {
						outcome.State = "succeeded"
						outcome.ReasonCode = ControlSupported
						outcome.Detail = "Native rewind verified and the active conversation projection was reconciled; workspace files were unchanged."
						if s.opencodeWatcher != nil && event.NativeProvenance != nil {
							s.opencodeWatcher.reconcileNativeSession(r.Context(), event.NativeProvenance.SessionID, sessionID)
						}
					}
				}
			}
		}
	}
	s.completeControlReceipt(r.Context(), outcome)
	writeControlJSON(w, http.StatusOK, outcome)
}

func sessionStillOwned(s *Server, sessionID string) bool {
	if s == nil {
		return false
	}
	if s.sessions != nil {
		if _, live := s.sessions.Get(sessionID); live {
			return true
		}
	}
	if s.managedCodex != nil {
		_, live := s.managedCodex.Get(context.Background(), sessionID)
		return live == nil
	}
	return false
}
