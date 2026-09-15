package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"web-console/backends/opencode"
)

// openCodeRewindAdapter is the provider-owned control path. It uses the
// documented OpenCode session/revert API and verifies the returned session
// marker before the projection is allowed to move.
type openCodeRewindAdapter struct{ client opencode.Client }

func (a openCodeRewindAdapter) Provider() string { return "opencode" }

func (a openCodeRewindAdapter) Preflight(ctx context.Context, event ConversationEvent, preflight ConversationControlPreflight) (ConversationControlPreflight, error) {
	if a.client == nil || event.NativeProvenance == nil || event.NativeProvenance.SessionID == "" || event.NativeProvenance.MessageID == "" {
		preflight.Decision = ControlUnavailable
		return preflight, nil
	}
	if strings.TrimSpace(event.NativeProvenance.CompactionLineage) != "" {
		preflight.Decision = ControlUnavailable
		preflight.Consequences = append(preflight.Consequences, "The selected boundary crosses a compaction lineage; native rewind is refused until that lineage is resolved.")
		return preflight, nil
	}
	status, err := a.client.SessionStatus(ctx)
	if err != nil {
		return preflight, err
	}
	preflight.Decision = ControlSupported
	preflight.Capability.Available = true
	preflight.Capability.ReasonCode = ControlSupported
	preflight.ConfirmationKey = fmt.Sprintf("confirm-%d", time.Now().UnixNano())
	preflight.Consequences = []string{"OpenCode will revert from this native message boundary.", "Later projected messages will be reconciled from the native session.", "The current draft is preserved in Web Console."}
	if current, ok := status[event.NativeProvenance.SessionID]; ok && current.Type == "busy" {
		preflight.Consequences = append(preflight.Consequences, "The owner will abort the active turn before invoking native rewind.")
	}
	return preflight, nil
}

func (a openCodeRewindAdapter) Execute(ctx context.Context, event ConversationEvent, _ bool) (string, error) {
	if a.client == nil || event.NativeProvenance == nil {
		return "", fmt.Errorf("opencode native provenance is unavailable")
	}
	status, err := a.client.SessionStatus(ctx)
	if err != nil {
		return "", err
	}
	if current, ok := status[event.NativeProvenance.SessionID]; ok && current.Type == "busy" {
		if err := a.client.AbortSession(ctx, event.NativeProvenance.SessionID); err != nil {
			return "", err
		}
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			status, err = a.client.SessionStatus(ctx)
			if err != nil {
				return "", err
			}
			if status[event.NativeProvenance.SessionID].Type != "busy" {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if status[event.NativeProvenance.SessionID].Type == "busy" {
			return "", fmt.Errorf("opencode session remained busy after bounded abort")
		}
	}
	if err := a.client.RevertMessage(ctx, event.NativeProvenance.SessionID, event.NativeProvenance.MessageID, event.NativeProvenance.BoundaryID); err != nil {
		return "", err
	}
	marker := event.NativeProvenance.TurnID
	if marker == "" {
		marker = event.NativeProvenance.MessageID
	}
	return marker, nil
}

func (a openCodeRewindAdapter) Verify(ctx context.Context, event ConversationEvent, target string) (bool, error) {
	if a.client == nil || event.NativeProvenance == nil || strings.TrimSpace(target) == "" {
		return false, nil
	}
	sessions, err := a.client.ListSessions(ctx)
	if err != nil {
		return false, err
	}
	for _, session := range sessions {
		if session.ID == event.NativeProvenance.SessionID {
			return session.Revert != nil && session.Revert.MessageID == target, nil
		}
	}
	return false, nil
}
