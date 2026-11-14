// Responsibility: retry interruptions only on the retained invocation and owner.
package interactive

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

func sameRecoveryInvocation(current, binding *domain.Run) bool {
	if current == nil || current.ID != binding.ID || current.Status != domain.RunStatusRunning ||
		current.WebConsoleSessionID != binding.WebConsoleSessionID || current.SessionID != binding.SessionID ||
		current.LifecycleVersion != binding.LifecycleVersion {
		return false
	}
	if (current.InteractiveInvocationStartedAt == nil) != (binding.InteractiveInvocationStartedAt == nil) {
		return false
	}
	return current.InteractiveInvocationStartedAt == nil || current.InteractiveInvocationStartedAt.Equal(*binding.InteractiveInvocationStartedAt)
}

func (c *Coordinator) recordInterruptionRecovery(ctx context.Context, runID uuid.UUID, message string) {
	if c.deps.NewSink == nil {
		return
	}
	if events := c.deps.NewSink(ctx, runID); events != nil {
		_ = events.Emit(domain.NewLogEvent(runID, "warn", message))
		_ = events.Close()
	}
}

// recoverInterruptedSession treats a screen as a retry hint, never as native
// ownership proof. Native mutations stay on the receiving owner API; legacy
// providers retain their existing verified prompt path.
func (c *Coordinator) recoverInterruptedSession(ctx context.Context, binding *domain.Run, initial webconsole.SessionInfo, interruption ResumableInterruption) bool {
	reader, ok := c.deps.Runs.(interface {
		Get(context.Context, uuid.UUID) (*domain.Run, error)
	})
	if !ok || ctx.Err() != nil {
		return false
	}
	current, err := reader.Get(ctx, binding.ID)
	if err != nil || !sameRecoveryInvocation(current, binding) {
		return false
	}
	session, err := c.deps.Sessions.GetSession(ctx, binding.WebConsoleSessionID)
	if err != nil || session.ID != initial.ID || session.LaunchMode != initial.LaunchMode || session.NativeOwner != initial.NativeOwner || session.NativeThreadID != initial.NativeThreadID {
		return false
	}
	screen, err := c.deps.Sessions.Screen(ctx, binding.WebConsoleSessionID, true)
	if err != nil || ctx.Err() != nil {
		return false
	}
	// Screen/RPC reads can block while Stop or a new invocation wins.
	fresh, err := reader.Get(ctx, binding.ID)
	if err != nil || !sameRecoveryInvocation(fresh, binding) || ctx.Err() != nil {
		return false
	}
	current = fresh
	if current.ResolvedConfig == nil {
		return false
	}
	observed, detected := ClassifyScreenInterruption(screen)
	stillInterrupted := detected && observed == interruption
	codex := current.ResolvedConfig != nil && current.ResolvedConfig.RunnerType == domain.RunnerTypeCodex
	if session.LaunchMode == "codex_app_server" && !codex {
		return false
	}
	if codex {
		if session.LaunchMode != "codex_app_server" || session.ControlMode != "native_capable" || session.NativeOwner == "" || binding.SessionID == "" || session.NativeThreadID != binding.SessionID {
			return false
		}
		probe, ok := c.deps.Sessions.(webconsole.NativeAgentProbe)
		if !ok {
			return false
		}
		native, err := probe.ObserveNativeAgent(ctx, binding.WebConsoleSessionID, binding.SessionID)
		if err != nil || native.Generation != initial.NativeOwner || (native.State != webconsole.NativeAgentIdle && native.State != webconsole.NativeAgentBusy) || ctx.Err() != nil {
			return false
		}
		// Revalidate durable cancellation/invocation after the native read too.
		fresh, err = reader.Get(ctx, binding.ID)
		if err != nil || !sameRecoveryInvocation(fresh, binding) || ctx.Err() != nil {
			return false
		}
		if !stillInterrupted || native.State == webconsole.NativeAgentBusy {
			return true
		}
		hasGoal := fresh.ResolvedConfig != nil && strings.TrimSpace(fresh.ResolvedConfig.Until) != ""
		if interruption.Kind == "goal_stalled" && !hasGoal {
			return false
		}
		if hasGoal {
			// A stale footer is not authority to resume a goal. Readback after a
			// lost acknowledgement may already prove the requested active state.
			if !native.GoalAvailable || !native.HasGoal {
				return false
			}
			switch native.GoalStatus {
			case "active", "complete", "blocked":
				return true
			case "paused": // Only a currently paused goal may be resumed.
			default:
				return false
			}
			controller, ok := c.deps.Sessions.(webconsole.NativeGoalController)
			if !ok {
				return false
			}
			status, err := controller.SetNativeGoalForSession(ctx, binding.WebConsoleSessionID, binding.SessionID, initial.NativeOwner, runner.NativeGoalObjectiveFor(domain.RunnerTypeCodex, fresh.ResolvedConfig.Until), "active", webconsole.NativeGoalAutomatic, nil)
			return err == nil && ctx.Err() == nil && status == "active"
		}
		sender, ok := c.deps.Sessions.(webconsole.NativeOwnerPromptSender)
		if !ok {
			return false
		}
		// Generic continue has no authoritative goal-state readback. Permit one
		// automatic prompt per invocation, using the existing durable receipt
		// owner. A later retry needs a fresh explicit continuation invocation;
		// neither coordinator restart nor receipt expiry grants replay.
		receipts, ok := c.deps.RecoveryReceipts.(repository.ContinuationIdempotencyRepository)
		if !ok {
			return false
		}
		key := fmt.Sprintf("interruption-prompt/%s/%d", binding.ID, binding.LifecycleVersion)
		invocation := ""
		if binding.InteractiveInvocationStartedAt != nil {
			invocation = binding.InteractiveInvocationStartedAt.UTC().Format(time.RFC3339Nano)
		}
		hash := sha256.Sum256([]byte(binding.WebConsoleSessionID + "\n" + binding.SessionID + "\n" + initial.NativeOwner + "\n" + invocation))
		receipt, acquired, err := receipts.ReserveContinuation(ctx, key, binding.ID, hash[:], 24*time.Hour)
		if err != nil {
			return false
		}
		if !acquired {
			return receipt != nil && receipt.Status == domain.IdempotencyStatusComplete
		}
		submission, err := sender.SendPromptForNativeOwner(ctx, binding.WebConsoleSessionID, binding.SessionID, initial.NativeOwner, "continue", interactiveSource(fresh))
		receiptCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if errors.Is(err, webconsole.ErrNativePromptRefused) {
			_ = receipts.FailContinuation(receiptCtx, key, binding.ID, hash[:])
			return false
		}
		if err != nil || ctx.Err() != nil || !submission.Verified {
			return false
		}
		return receipts.CompleteContinuation(receiptCtx, key, binding.ID, hash[:]) == nil
	}
	if !stillInterrupted {
		return true
	}
	if interruption.Kind == "goal_stalled" {
		return false
	}
	submission, err := c.deps.Sessions.SendPrompt(ctx, binding.WebConsoleSessionID, "continue", interactiveSource(current))
	return err == nil && ctx.Err() == nil && submission.Verified
}
