package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"web-console/backends/codex"
)

type codexConversationAdapter struct{ owner *codex.ManagedOwner }

func (a codexConversationAdapter) Provider() string { return "codex" }

func (a codexConversationAdapter) Preflight(_ context.Context, event ConversationEvent, p ConversationControlPreflight) (ConversationControlPreflight, error) {
	if a.owner == nil || event.NativeProvenance == nil {
		p.Decision = ControlUnavailable
		return p, nil
	}
	if event.NativeProvenance.TurnID == "" || event.NativeProvenance.SessionID == "" || event.NativeProvenance.SessionID != a.owner.OriginalThreadID() && event.NativeProvenance.SessionID != a.owner.ThreadID() {
		p.Decision = ControlStaleTarget
		p.Consequences = []string{"The selected event is not bound to the managed Codex thread; projected history was preserved."}
		return p, nil
	}
	if strings.TrimSpace(event.NativeProvenance.CompactionLineage) != "" {
		p.Decision = ControlUnavailable
		p.Consequences = []string{"The selected boundary crosses a compaction lineage; native fork is refused."}
		return p, nil
	}
	if active := a.owner.ActiveTurnID(); active != "" && active == event.NativeProvenance.TurnID {
		p.Decision = ControlUnavailable
		p.Consequences = []string{"The selected turn is still active; native restore requires a completed turn boundary."}
		return p, nil
	}
	p.Decision = ControlSupported
	p.Capability.Available = true
	p.Capability.ReasonCode = ControlSupported
	p.Capability.LaunchMode = "codex_app_server"
	p.Capability.ControlMode = "native_capable"
	p.Capability.NativeOwner = a.owner.OwnerID()
	for i := range p.Capability.Options {
		p.Capability.Options[i].Supported = p.Capability.Options[i].ID == "restore_conversation"
	}
	p.ConfirmationKey = fmt.Sprintf("codex-confirm-%d", time.Now().UnixNano())
	p.Consequences = []string{"Codex will create a conversation branch through the selected completed turn.", "Workspace files remain unchanged.", "The original Codex branch remains recoverable.", "The current Web Console draft is preserved."}
	return p, nil
}

func (a codexConversationAdapter) Execute(ctx context.Context, event ConversationEvent, _ bool) (string, error) {
	if a.owner == nil || event.NativeProvenance == nil {
		return "", codex.ErrMissingThread
	}
	if active := a.owner.ActiveTurnID(); active != "" {
		if err := a.owner.InterruptActive(ctx); err != nil {
			return "", fmt.Errorf("interrupt active Codex turn: %w", err)
		}
	}
	return a.owner.ForkThrough(ctx, event.NativeProvenance.TurnID)
}

func (a codexConversationAdapter) Verify(ctx context.Context, event ConversationEvent, target string) (bool, error) {
	if a.owner == nil || target == "" || target != a.owner.ThreadID() {
		return false, nil
	}
	return a.owner.VerifyThread(ctx, target, event.NativeProvenance.TurnID)
}
