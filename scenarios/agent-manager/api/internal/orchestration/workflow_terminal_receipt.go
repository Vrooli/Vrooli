package orchestration

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/maintenance"
	"github.com/google/uuid"
)

// RecoverTerminalAccounting recovers only original provider receipts. It never
// starts/tails a runner, changes lifecycle/finalization, or applies a sandbox.
func (l workflowChildLauncher) RecoverTerminalAccounting(ctx context.Context, id uuid.UUID) error {
	return l.recoverTerminalAccounting(ctx, id, maintenance.ExcludeExecutor)
}

func (l workflowChildLauncher) recoverTerminalAccounting(parent context.Context, id uuid.UUID, exclude func(context.Context, *domain.Run) error) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	// Continuation uses this same owner mutex. Refuse contention; do not race a
	// new invocation or hold a workflow worker indefinitely behind one.
	if !l.o.wakeMu.TryLock() {
		return fmt.Errorf("terminal receipt recovery conflicts with a continuation owner")
	}
	defer l.o.wakeMu.Unlock()
	run, err := l.o.runs.Get(ctx, id)
	if err != nil {
		return err
	}
	if run == nil || !run.Status.IsTerminal() || run.EndedAt == nil || run.SessionID == "" || run.TranscriptPath == "" || run.ResolvedConfig == nil || run.ExecutionMode.Normalized() != domain.ExecutionModeCodecPipe {
		return fmt.Errorf("original managed terminal run/session/transcript binding is required")
	}
	if l.o.events == nil || l.o.runners == nil || exclude == nil {
		return fmt.Errorf("terminal receipt recovery owners are unavailable")
	}
	const maxEvents = 4096
	retained, err := l.o.events.Get(ctx, id, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMetric, domain.EventTypeStatus}, Limit: maxEvents + 1})
	if err != nil {
		return err
	}
	if len(retained) > maxEvents {
		return fmt.Errorf("terminal receipt recovery exceeds bounded event capacity")
	}
	state, err := meteredWorkflowChildState(run, retained, l.o.now())
	if err != nil || (state.TokensKnown && state.ChargeMeasured) {
		return err
	}
	invocations := 0
	for _, item := range retained {
		if invocationreadmodel.BeginsRunInvocation(item) {
			invocations++
		}
	}
	if invocations > 1 {
		return fmt.Errorf("multiple retained invocations require original per-invocation receipt reconciliation")
	}
	if err := exclude(ctx, run); err != nil {
		return fmt.Errorf("terminal accounting executor exclusion: %w", err)
	}
	owned, err := l.o.runners.Get(run.ResolvedConfig.RunnerType)
	if err != nil {
		return err
	}
	factory, ok := owned.(runner.TranscriptParserFactory)
	if !ok {
		return fmt.Errorf("fresh canonical transcript parser unavailable")
	}
	parser := factory.NewTranscriptParser()
	if setter, ok := parser.(runner.TranscriptModelSetter); ok {
		setter.SetTranscriptModel(runTranscriptModel(run))
	}
	if setter, ok := parser.(runner.TranscriptBillingSetter); ok {
		setter.SetTranscriptBilling(run.Billing)
	}
	receipt, digest, err := readOriginalTerminalReceipt(ctx, run, parser)
	if err != nil {
		return err
	}
	verified, err := meteredWorkflowChildState(run, []*domain.RunEvent{receipt}, l.o.now())
	if err != nil || !verified.TokensKnown || !verified.ChargeMeasured {
		return fmt.Errorf("retained provider receipt is incomplete: %v", err)
	}
	current, err := l.o.runs.Get(ctx, id)
	if err != nil {
		return err
	}
	if current == nil || current.LifecycleVersion != run.LifecycleVersion || current.Status != run.Status || current.SessionID != run.SessionID || current.TranscriptPath != run.TranscriptPath || current.EndedAt == nil || !current.EndedAt.Equal(*run.EndedAt) {
		return fmt.Errorf("terminal receipt owner binding changed during recovery")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	receipt.ID = uuid.NewSHA1(id, []byte("terminal-receipt-v1:"+digest))
	receipt.RunID, receipt.Timestamp = id, *run.EndedAt
	for _, item := range retained {
		if item.ID == receipt.ID {
			return nil
		}
	}
	audit := domain.NewLogEvent(id, "info", fmt.Sprintf("recovered original terminal accounting: session %s transcript sha256:%s; run outcome and finalization unchanged", run.SessionID, digest))
	audit.ID = uuid.NewSHA1(id, []byte("terminal-receipt-audit-v1:"+digest))
	return l.o.events.Append(ctx, id, receipt, audit)
}

func readOriginalTerminalReceipt(ctx context.Context, run *domain.Run, parser runner.TranscriptParser) (*domain.RunEvent, string, error) {
	const maxBytes = 16 << 20
	f, err := os.Open(run.TranscriptPath)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return nil, "", err
	}
	if !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > maxBytes {
		return nil, "", fmt.Errorf("terminal transcript exceeds bounded regular-file capacity")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	after, err := os.Stat(run.TranscriptPath)
	if err != nil {
		return nil, "", err
	}
	if !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || len(data) != int(before.Size()) || data[len(data)-1] != '\n' {
		return nil, "", fmt.Errorf("terminal transcript changed or has an incomplete tail")
	}
	var receipt *domain.RunEvent
	sessionSeen, terminalCount, receiptCount := false, 0, 0
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		if !json.Valid(scanner.Bytes()) {
			return nil, "", fmt.Errorf("malformed retained transcript")
		}
		parsed := parser.ParseTranscriptLine(run.ID, scanner.Text())
		if parsed.Err != nil {
			return nil, "", parsed.Err
		}
		if parsed.SessionID != "" {
			if parsed.SessionID != run.SessionID {
				return nil, "", fmt.Errorf("retained transcript belongs to a different session")
			}
			sessionSeen = true
		}
		if parsed.Terminal != nil {
			terminalCount++
		}
		for _, item := range parsed.Events {
			if usage, ok := item.Data.(*domain.UsageEventData); ok {
				if usage.ReconciliationAuthority {
					receiptCount++
					copyUsage := *usage
					receipt = &domain.RunEvent{RunID: run.ID, EventType: domain.EventTypeMetric, Data: &copyUsage}
				} else if receipt != nil {
					return nil, "", fmt.Errorf("provider work follows the retained terminal receipt")
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}
	if !sessionSeen || terminalCount > 1 || receiptCount != 1 || receipt == nil {
		return nil, "", fmt.Errorf("exact single-invocation terminal receipt is unavailable")
	}
	return receipt, fmt.Sprintf("%x", sha256.Sum256(data)), nil
}
