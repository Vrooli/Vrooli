// This file issues authoritative terminal usage receipts for workflow child runs.
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

// RecoverTerminalAccounting recovers terminal usage from the harness's own
// retained transcript. It never starts/tails a runner, changes lifecycle or
// finalization, or applies a sandbox.
func (l workflowChildLauncher) RecoverTerminalAccounting(ctx context.Context, id uuid.UUID) error {
	return l.o.recoverTerminalAccounting(ctx, id, maintenance.ExcludeExecutor)
}

func (l workflowChildLauncher) recoverTerminalAccounting(parent context.Context, id uuid.UUID, exclude func(context.Context, *domain.Run) error) error {
	return l.o.recoverTerminalAccounting(parent, id, exclude)
}

// recoverTerminalAccounting appends the usage receipt a terminal run's live
// stream never recorded, read back from the harness's own transcript: the
// harness's final receipt when it wrote one, or, for a final turn the harness
// never closed, the receipt its parser can prove complete. Unknown stays
// unknown; the run's outcome and finalization are never changed.
func (o *Orchestrator) recoverTerminalAccounting(parent context.Context, id uuid.UUID, exclude func(context.Context, *domain.Run) error) error {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	// Continuation uses this same owner mutex. Refuse contention; do not race a
	// new invocation or hold a workflow worker indefinitely behind one.
	if !o.wakeMu.TryLock() {
		return fmt.Errorf("terminal receipt recovery conflicts with a continuation owner")
	}
	defer o.wakeMu.Unlock()
	run, err := o.runs.Get(ctx, id)
	if err != nil {
		return err
	}
	if run == nil || !run.Status.IsTerminal() || run.EndedAt == nil || run.SessionID == "" || run.ResolvedConfig == nil {
		return fmt.Errorf("original managed terminal run/session binding is required")
	}
	mode := run.ExecutionMode.Normalized()
	if mode != domain.ExecutionModeCodecPipe && mode != domain.ExecutionModeInteractive {
		return fmt.Errorf("terminal receipt recovery requires a managed codec-pipe or interactive run")
	}
	transcriptPath := run.TranscriptPath
	if transcriptPath == "" && mode == domain.ExecutionModeInteractive && run.ResolvedConfig.RunnerType == domain.RunnerTypeClaudeCode {
		// Interactive claude writes its pinned native transcript under the
		// user's project directory; recovery resolves it the same way.
		transcriptPath, _ = findClaudeNativeTranscript(run.SessionID)
	}
	if transcriptPath == "" {
		return fmt.Errorf("original transcript binding is required")
	}
	if o.events == nil || o.runners == nil || exclude == nil {
		return fmt.Errorf("terminal receipt recovery owners are unavailable")
	}
	const maxEvents = 4096
	retained, err := o.events.Get(ctx, id, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMetric, domain.EventTypeStatus}, Limit: maxEvents + 1})
	if err != nil {
		return err
	}
	if len(retained) > maxEvents {
		return fmt.Errorf("terminal receipt recovery exceeds bounded event capacity")
	}
	state, err := meteredWorkflowChildState(run, retained, o.now())
	if err != nil || (state.TokensKnown && state.ChargeMeasured) {
		return err
	}
	if state.TokensKnown && latestUsageRecovered(retained) {
		// Already recovered; this run's charge basis cannot be measured (for
		// example unpriced), and re-reading the transcript cannot change that.
		return nil
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
	owned, err := o.runners.Get(run.ResolvedConfig.RunnerType)
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
	receipt, digest, err := readTranscriptTerminalReceipt(ctx, run, transcriptPath, parser, mode == domain.ExecutionModeInteractive)
	if err != nil {
		return err
	}
	combined := append(append(make([]*domain.RunEvent, 0, len(retained)+len(receipt)), retained...), receipt...)
	verified, err := meteredWorkflowChildState(run, combined, o.now())
	if err != nil || !verified.TokensKnown {
		return fmt.Errorf("retained provider receipt is incomplete: %v", err)
	}
	current, err := o.runs.Get(ctx, id)
	if err != nil {
		return err
	}
	if current == nil || current.LifecycleVersion != run.LifecycleVersion || current.Status != run.Status || current.SessionID != run.SessionID || current.TranscriptPath != run.TranscriptPath || current.EndedAt == nil || !current.EndedAt.Equal(*run.EndedAt) {
		return fmt.Errorf("terminal receipt owner binding changed during recovery")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	for i, item := range receipt {
		key := "terminal-receipt-v1:" + digest
		if i > 0 {
			key = fmt.Sprintf("%s:%d", key, i)
		}
		item.ID = uuid.NewSHA1(id, []byte(key))
		item.RunID, item.Timestamp = id, *run.EndedAt
	}
	for _, item := range retained {
		if item.ID == receipt[0].ID {
			return nil
		}
	}
	audit := domain.NewLogEvent(id, "info", fmt.Sprintf("recovered terminal accounting (%s): session %s transcript sha256:%s; run outcome and finalization unchanged", receiptSource(receipt), run.SessionID, digest))
	audit.ID = uuid.NewSHA1(id, []byte("terminal-receipt-audit-v1:"+digest))
	return o.events.Append(ctx, id, append(receipt, audit)...)
}

// latestUsageRecovered reports whether the newest usage fact was itself
// recovered from a transcript rather than carried by the live stream.
func latestUsageRecovered(events []*domain.RunEvent) bool {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i] == nil {
			continue
		}
		if usage, ok := events[i].Data.(*domain.UsageEventData); ok {
			return usage.ReconciliationSource != ""
		}
	}
	return false
}

func receiptSource(receipt []*domain.RunEvent) string {
	for _, item := range receipt {
		if usage, ok := item.Data.(*domain.UsageEventData); ok && usage.ReconciliationSource != "" {
			return usage.ReconciliationSource
		}
	}
	return domain.ReconciliationSourceTranscriptRecovery
}

// readTranscriptTerminalReceipt re-parses a retained transcript with a fresh
// parser and returns the usage receipt that covers its final turn. A codec-pipe
// transcript is one invocation carrying exactly one harness receipt. An
// interactive transcript spans turns: its last harness receipt counts only when
// no provider work follows it; otherwise the parser may close the final turn if
// it can prove no request was outstanding when the transcript ends.
func readTranscriptTerminalReceipt(ctx context.Context, run *domain.Run, path string, parser runner.TranscriptParser, multiTurn bool) ([]*domain.RunEvent, string, error) {
	const maxBytes = 64 << 20
	f, err := os.Open(path)
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
	after, err := os.Stat(path)
	if err != nil {
		return nil, "", err
	}
	if !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || len(data) != int(before.Size()) || data[len(data)-1] != '\n' {
		return nil, "", fmt.Errorf("terminal transcript changed or has an incomplete tail")
	}
	var receipt *domain.RunEvent
	sessionSeen, terminalCount, receiptCount, workAfterReceipt := false, 0, 0, false
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
			usage, ok := item.Data.(*domain.UsageEventData)
			if !ok {
				continue
			}
			if usage.ReconciliationAuthority {
				receiptCount++
				copyUsage := *usage
				copyUsage.ReconciliationSource = domain.ReconciliationSourceTranscriptRecovery
				receipt = &domain.RunEvent{RunID: run.ID, EventType: domain.EventTypeMetric, Data: &copyUsage}
				workAfterReceipt = false
				continue
			}
			if receipt != nil && !multiTurn {
				return nil, "", fmt.Errorf("provider work follows the retained terminal receipt")
			}
			workAfterReceipt = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	if !sessionSeen {
		return nil, "", fmt.Errorf("retained transcript never names the run's session")
	}
	if !multiTurn {
		if terminalCount > 1 || receiptCount != 1 || receipt == nil {
			return nil, "", fmt.Errorf("exact single-invocation terminal receipt is unavailable")
		}
		return []*domain.RunEvent{receipt}, digest, nil
	}
	if receipt != nil && !workAfterReceipt {
		return []*domain.RunEvent{receipt}, digest, nil
	}
	if finalizer, ok := parser.(runner.TranscriptInterruptedTurnFinalizer); ok {
		if closing, ok := finalizer.FinalizeInterruptedTurn(run.ID, *run.EndedAt); ok && len(closing) > 0 {
			return closing, digest, nil
		}
	}
	return nil, "", fmt.Errorf("no harness receipt covers the transcript's final turn")
}
