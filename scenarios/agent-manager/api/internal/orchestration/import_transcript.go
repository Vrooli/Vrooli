// Import transcript orchestration adopts external harness evidence safely.
package orchestration

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

// ImportTranscriptRequest adopts a newline-delimited harness transcript as a
// terminal read-only run. The caller's file is never parsed in place.
type ImportTranscriptRequest struct {
	Path            string
	RunnerType      domain.RunnerType
	Label           string
	SourceHarness   string
	SourceSessionID string
}

var importedTranscriptTaskID = uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/imported-transcripts"))

// ImportTranscript copies, parses, and persists an external transcript using
// the same parser and event store as restart recovery.
func (o *Orchestrator) ImportTranscript(ctx context.Context, req ImportTranscriptRequest) (*domain.Run, error) {
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return nil, fmt.Errorf("transcript path is required")
	}
	if req.SourceHarness != "" && req.SourceSessionID != "" {
		existing, err := o.runs.GetByImportProvenance(ctx, req.SourceHarness, req.SourceSessionID)
		if err != nil {
			return nil, fmt.Errorf("lookup imported transcript: %w", err)
		}
		if existing != nil {
			return existing, nil
		}
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
	goalID, goalStatus := transcriptGoalMetadata(source)
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind transcript after goal scan: %w", err)
	}
	root, err := o.resolveRunStateRoot(ctx)
	if err != nil {
		return nil, err
	}
	now := o.now()
	task, err := o.ensureImportedTranscriptTask(ctx)
	if err != nil {
		return nil, err
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: "agent-manager-imported", RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeImported, Status: domain.RunStatusFailed, ResolvedConfig: &domain.RunConfig{RunnerType: runnerType, ManifestIndexSnapshot: runsignal.CurrentCatalogSnapshot()}, ImportSourceHarness: req.SourceHarness, ImportSourceSessionID: req.SourceSessionID, ImportedAt: &now}
	run.GoalID, run.GoalStatus = goalID, goalStatus
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
	if factory, ok := parser.(runner.TranscriptParserFactory); ok {
		parser = factory.NewTranscriptParser()
	}
	if setter, ok := parser.(runner.TranscriptModelSetter); ok {
		setter.SetTranscriptModel(runTranscriptModel(run))
	}
	var terminal *runner.TranscriptTerminal
	_, terminal, err = runner.Consume(ctx, runner.ConsumeArgs{RunID: run.ID, Transcript: snapshot.TranscriptPath, ParseFn: func(id uuid.UUID, line string) runner.TranscriptParseResult {
		return parser.ParseTranscriptLine(id, line)
	}, EventSink: importEventSink{ctx: ctx, runID: run.ID, store: o.events}, OnAdvance: func(cursor, seq int64) error { run.TranscriptCursor, run.TranscriptLastSeq = cursor, seq; return nil }, OnSessionID: func(id string) error { run.SessionID = id; return nil }})
	if err != nil {
		return nil, fmt.Errorf("parse imported transcript: %w", err)
	}
	// Imported transcripts are historical evidence. Preserve their event window
	// as the run lifecycle so duration and time accounting describe the original
	// session rather than the instant it happened to be imported.
	if events, getErr := o.events.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1, Limit: 100000}); getErr != nil {
		return nil, fmt.Errorf("read imported event window: %w", getErr)
	} else if started, ended, ok := eventWindow(events); ok {
		run.StartedAt, run.EndedAt = &started, &ended
	} else {
		run.EndedAt, run.StartedAt = &now, &now
	}
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
	if err := o.projectInvocationReadModel(ctx, run); err != nil {
		return nil, fmt.Errorf("project imported transcript: %w", err)
	}
	return o.attachRunActions(ctx, run), nil
}

func transcriptGoalMetadata(source *os.File) (string, string) {
	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	for scanner.Scan() {
		var value any
		if json.Unmarshal(scanner.Bytes(), &value) != nil {
			continue
		}
		if condition, met, ok := findGoalStatus(value); ok {
			goalID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/goal/"+condition)).String()
			status := "unmet"
			if met {
				status = "met"
			}
			return goalID, status
		}
	}
	return "", ""
}

func findGoalStatus(value any) (string, bool, bool) {
	object, ok := value.(map[string]any)
	if !ok {
		if array, ok := value.([]any); ok {
			for _, child := range array {
				if condition, met, found := findGoalStatus(child); found {
					return condition, met, true
				}
			}
		}
		return "", false, false
	}
	if object["type"] == "goal_status" {
		condition, _ := object["condition"].(string)
		met, _ := object["met"].(bool)
		if strings.TrimSpace(condition) != "" {
			return condition, met, true
		}
	}
	if attachment, ok := object["attachment"].(map[string]any); ok && attachment["type"] == "goal_status" {
		condition, _ := attachment["condition"].(string)
		met, _ := attachment["met"].(bool)
		if strings.TrimSpace(condition) != "" {
			return condition, met, true
		}
	}
	for _, child := range object {
		if condition, met, ok := findGoalStatus(child); ok {
			return condition, met, true
		}
	}
	return "", false, false
}

func eventWindow(events []*domain.RunEvent) (time.Time, time.Time, bool) {
	var started, ended time.Time
	for _, item := range events {
		if item == nil || item.Timestamp.IsZero() {
			continue
		}
		if started.IsZero() || item.Timestamp.Before(started) {
			started = item.Timestamp
		}
		if ended.IsZero() || item.Timestamp.After(ended) {
			ended = item.Timestamp
		}
	}
	return started, ended, !started.IsZero() && !ended.IsZero()
}

// ensureImportedTranscriptTask creates one durable, clearly labelled task for
// external evidence. Runs still retain ExecutionModeImported and their import
// tag, so consumers can distinguish them from Agent Manager executions without
// inventing an invalid task-less run shape.
func (o *Orchestrator) ensureImportedTranscriptTask(ctx context.Context) (*domain.Task, error) {
	task, err := o.tasks.Get(ctx, importedTranscriptTaskID)
	if err != nil {
		return nil, fmt.Errorf("get imported transcript task: %w", err)
	}
	if task != nil {
		return task, nil
	}
	task = &domain.Task{
		ID:          importedTranscriptTaskID,
		Title:       "Imported external transcripts",
		Description: "Durable synthetic task that owns read-only imported coding-agent transcript runs.",
		ScopePath:   ".",
		CreatedBy:   "agent-manager-import",
	}
	if _, err := o.CreateTask(ctx, task); err != nil {
		// A concurrent importer may have won creation. Read once to preserve a
		// stable task rather than exposing an avoidable uniqueness error.
		existing, getErr := o.tasks.Get(ctx, importedTranscriptTaskID)
		if getErr == nil && existing != nil {
			return existing, nil
		}
		return nil, fmt.Errorf("create imported transcript task: %w", err)
	}
	return task, nil
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
