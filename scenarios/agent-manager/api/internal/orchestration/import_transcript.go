// Import transcript orchestration adopts external harness evidence safely.
package orchestration

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/runreport"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

// ImportTranscriptRequest adopts a newline-delimited harness transcript as a
// terminal read-only run. The caller's file is never parsed in place.
type ImportTranscriptRequest struct {
	Path       string
	RunnerType domain.RunnerType
	Label      string
}

// ImportTranscript copies, parses, and persists an external transcript using
// the same parser and event store as restart recovery.
func (o *Orchestrator) ImportTranscript(ctx context.Context, req ImportTranscriptRequest) (*domain.Run, error) {
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return nil, fmt.Errorf("transcript path is required")
	}
	source, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open transcript: %w", err)
	}
	defer source.Close()
	runnerType, err := o.detectTranscriptRunner(source, req.RunnerType)
	if err != nil {
		return nil, err
	}
	parserRunner, err := o.runners.Get(runnerType)
	if err != nil {
		return nil, fmt.Errorf("unsupported transcript format: %w", err)
	}
	parser, ok := parserRunner.(runner.TranscriptParser)
	if !ok {
		return nil, fmt.Errorf("unsupported transcript format: runner %q has no transcript parser", runnerType)
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind transcript: %w", err)
	}
	root, err := o.resolveRunStateRoot(ctx)
	if err != nil {
		return nil, err
	}
	now := o.now()
	run := &domain.Run{ID: uuid.New(), Tag: "agent-manager-imported", RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeImported, Status: domain.RunStatusFailed, ResolvedConfig: &domain.RunConfig{RunnerType: runnerType, ManifestIndexSnapshot: runreport.CurrentCatalogSnapshot()}}
	if strings.TrimSpace(req.Label) != "" {
		run.Tag = "agent-manager-imported-" + strings.TrimSpace(req.Label)
	}
	state, err := runstate.Open(run.ID, runstate.OpenOptions{RootDir: root, RunnerType: runnerType, WorkingDir: filepath.Dir(path), StartedAt: now, OnWrite: func() { o.recordRunStateWrite(ctx) }})
	if err != nil {
		return nil, fmt.Errorf("create imported run state: %w", err)
	}
	snapshot := state.Snapshot()
	if _, err := io.Copy(state.TranscriptWriter(), source); err != nil {
		_ = state.Close()
		return nil, fmt.Errorf("copy transcript: %w", err)
	}
	if err := state.Close(); err != nil {
		return nil, err
	}
	run.TranscriptPath = snapshot.TranscriptPath
	if err := o.runs.Create(ctx, run); err != nil {
		return nil, err
	}
	var terminal *runner.TranscriptTerminal
	_, terminal, err = runner.Consume(ctx, runner.ConsumeArgs{RunID: run.ID, Transcript: snapshot.TranscriptPath, ParseFn: func(id uuid.UUID, line string) runner.TranscriptParseResult {
		return parser.ParseTranscriptLine(id, line)
	}, EventSink: importEventSink{ctx: ctx, runID: run.ID, store: o.events}, OnAdvance: func(cursor, seq int64) error { run.TranscriptCursor, run.TranscriptLastSeq = cursor, seq; return nil }, OnSessionID: func(id string) error { run.SessionID = id; return nil }})
	if err != nil {
		return nil, fmt.Errorf("parse imported transcript: %w", err)
	}
	run.EndedAt, run.StartedAt = &now, &now
	if terminal != nil && terminal.Success {
		run.Status = domain.RunStatusComplete
		run.ExitCode = intPtr(terminal.ExitCode)
	} else {
		run.Status = domain.RunStatusFailed
		run.ErrorMsg = "no terminal signal in imported transcript"
	}
	run.UpdatedAt = now
	if err := o.runs.Update(ctx, run); err != nil {
		return nil, err
	}
	return o.attachRunActions(ctx, run), nil
}

func (o *Orchestrator) detectTranscriptRunner(source *os.File, requested domain.RunnerType) (domain.RunnerType, error) {
	if requested != "" {
		if _, err := o.runners.Get(requested); err != nil {
			return "", fmt.Errorf("unsupported transcript format: %w", err)
		}
		return requested, nil
	}
	scanner := bufio.NewScanner(source)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		for _, candidate := range o.runners.List() {
			parser, ok := candidate.(runner.TranscriptParser)
			if !ok {
				continue
			}
			result := parser.ParseTranscriptLine(uuid.New(), line)
			if result.Err == nil && (len(result.Events) > 0 || result.Terminal != nil || result.SessionID != "") {
				return candidate.Type(), nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read transcript: %w", err)
	}
	return "", fmt.Errorf("unsupported transcript format")
}

type importEventSink struct {
	ctx   context.Context
	runID uuid.UUID
	store interface {
		Append(context.Context, uuid.UUID, ...*domain.RunEvent) error
	}
}

func (s importEventSink) Emit(event *domain.RunEvent) error {
	return s.store.Append(s.ctx, s.runID, event)
}
func (s importEventSink) Close() error { return nil }

func importedRunLifecycleError(operation string) error {
	return domain.NewValidationErrorWithCode("run", "imported run cannot perform lifecycle operation: "+operation, domain.ErrCodePolicyScope)
}
