// Responsibility: reconcile bounded historical transcript evidence without execution or accounting effects.
package orchestration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

type TranscriptReplayRange struct {
	FromByte, ThroughByte int64
	SHA256                string
}

type TranscriptReplayReceipt struct {
	FromByte, ThroughByte                                        int64
	SourceSHA256                                                 string
	RecordsParsed, EventsInserted, EventsReplayed, EventsIgnored int64
	EventsWithPublicationTime                                    int64
	InsertedEventIDs                                             []string
}

// replayTranscriptEvidence operates only on complete records already behind
// the run's committed cursor. It does not invoke execution/accounting hooks,
// update run state, or rewind source metadata. Store occurrence IDs deduplicate
// existing evidence, preserving the originally stored payload on replay.
func (r *Reconciler) replayTranscriptEvidence(ctx context.Context, run *domain.Run, selected TranscriptReplayRange) (*TranscriptReplayReceipt, error) {
	if selected.FromByte < 0 || selected.ThroughByte <= selected.FromByte || selected.ThroughByte-selected.FromByte > 1<<20 || selected.ThroughByte > run.TranscriptCursor {
		return nil, domain.NewValidationError("transcriptRange", "select at most 1 MiB of complete records within committed stdout")
	}
	wanted, err := hex.DecodeString(selected.SHA256)
	if err != nil || len(wanted) != sha256.Size {
		return nil, domain.NewValidationError("transcriptSHA256", "exact source SHA-256 is required")
	}
	if run.ExecutionMode.Normalized() != domain.ExecutionModeCodecPipe || run.ResolvedConfig == nil || run.ResolvedConfig.RunnerType != domain.RunnerTypeCodex || r.events == nil {
		return nil, domain.NewValidationError("executionMode", "historical tool evidence reconciliation requires managed Codex codec-pipe stdout")
	}
	parser, path, state, err := r.recoveryParser(ctx, run)
	if err != nil {
		return nil, err
	}
	if parser == nil || state == nil || state.Meta.RunID != run.ID.String() || filepath.Clean(path) != filepath.Clean(state.TranscriptPath) {
		return nil, domain.NewValidationError("transcript", "retained managed stdout binding is unavailable")
	}
	factory, ok := parser.(runner.TranscriptParserFactory)
	if !ok {
		return nil, domain.NewValidationError("transcript", "fresh independent transcript parser is required")
	}
	parser = factory.NewTranscriptParser()
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if selected.FromByte > 0 {
		var boundary [1]byte
		if _, err = file.ReadAt(boundary[:], selected.FromByte-1); err != nil || boundary[0] != '\n' {
			return nil, domain.NewValidationError("transcriptRange", "range must start at a complete-record boundary")
		}
	}
	data := make([]byte, selected.ThroughByte-selected.FromByte)
	if _, err = io.ReadFull(io.NewSectionReader(file, selected.FromByte, int64(len(data))), data); err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	if !bytes.Equal(digest[:], wanted) || data[len(data)-1] != '\n' {
		return nil, domain.NewValidationError("transcriptRange", "source digest or complete-record boundary does not match")
	}
	receipt := &TranscriptReplayReceipt{FromByte: selected.FromByte, ThroughByte: selected.ThroughByte, SourceSHA256: hex.EncodeToString(digest[:])}
	var evidence []*domain.RunEvent
	publicationTime := map[string]bool{}
	offset := selected.FromByte
	for _, record := range bytes.SplitAfter(data, []byte{'\n'}) {
		if len(record) == 0 {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Only self-contained command records are independently interpretable
		// mid-stream. Message/partial tool-block context cannot be invented.
		var wire struct {
			Type string `json:"type"`
			Item struct {
				ID      string `json:"id"`
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"item"`
		}
		if err := json.Unmarshal(record, &wire); err != nil || wire.Item.ID == "" || wire.Item.Type != "command_execution" || wire.Item.Command == "" || (wire.Type != "item.started" && wire.Type != "item.completed") {
			return nil, domain.NewValidationError("transcriptRange", "select self-contained Codex command_execution records; context-dependent records are unsupported")
		}
		parsed := parser.ParseTranscriptLine(run.ID, string(record))
		if parsed.Err != nil {
			return nil, parsed.Err
		}
		if len(parsed.Events) == 0 {
			return nil, domain.NewValidationError("transcriptRange", "selected record produced no independent tool evidence")
		}
		runner.AssignTranscriptEventIDs(run.ID, path, offset, string(record), parsed.Events)
		receipt.RecordsParsed++
		for _, event := range parsed.Events {
			if event == nil {
				continue
			}
			switch event.EventType {
			case domain.EventTypeToolCall, domain.EventTypeToolResult:
				event.KeepStoredOnReplay = true
				if !parsed.Timestamp.IsZero() {
					event.Timestamp = parsed.Timestamp
				} else {
					publicationTime[event.ID.String()] = true
				}
				evidence = append(evidence, event)
			default:
				receipt.EventsIgnored++
			}
		}
		if len(evidence) > 128 {
			return nil, domain.NewValidationError("transcriptRange", "select at most 128 tool facts per reconciliation")
		}
		offset += int64(len(record))
	}
	// Appends are allowed while the committed prefix remains unchanged. A
	// replaced/truncated source or edited selected bytes must not be published.
	current, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	verify := make([]byte, len(data))
	if _, err = file.ReadAt(verify, selected.FromByte); err != nil {
		return nil, err
	}
	if !os.SameFile(before, current) || !bytes.Equal(data, verify) {
		return nil, fmt.Errorf("historical transcript source changed before reconciliation")
	}
	if len(evidence) > 0 {
		// Deliberately use the evidence store, not an execution sink: historical
		// tool/message repair must not consume budgets or rerun lifecycle hooks.
		if err = r.events.Append(ctx, run.ID, evidence...); err != nil {
			return nil, err
		}
		for _, event := range evidence {
			if event.PersistenceReplay {
				receipt.EventsReplayed++
			} else {
				receipt.EventsInserted++
				receipt.InsertedEventIDs = append(receipt.InsertedEventIDs, event.ID.String())
				if publicationTime[event.ID.String()] {
					receipt.EventsWithPublicationTime++
				}
				if r.broadcaster != nil {
					r.broadcaster.BroadcastEvent(ctx, event)
				}
			}
		}
	}
	return receipt, nil
}

func transcriptReplayMessage(receipt *TranscriptReplayReceipt) string {
	return fmt.Sprintf("historical tool evidence reconciled: inserted=%d replayed=%d ignored=%d; execution, accounting and cursors unchanged", receipt.EventsInserted, receipt.EventsReplayed, receipt.EventsIgnored)
}
