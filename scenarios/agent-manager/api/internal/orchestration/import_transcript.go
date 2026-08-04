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
	"sort"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/runstate"
	"agent-manager/internal/transcriptredact"

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
	detection, err := o.detectTranscriptRunner(source, req.RunnerType)
	if err != nil {
		return nil, err
	}
	runnerType := detection.RunnerType
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
	var markerProvider runner.GoalMarkerProvider
	if provider, ok := parserRunner.(runner.GoalMarkerProvider); ok {
		markerProvider = provider
	}
	goalID, goalStatus := transcriptGoalMetadata(source, markerProvider)
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
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: "agent-manager-imported", RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeImported, Status: domain.RunStatusUnknown, ResolvedConfig: &domain.RunConfig{RunnerType: runnerType, ManifestIndexSnapshot: runsignal.CurrentCatalogSnapshot(), TranscriptCodec: string(runnerType), TranscriptCodecScore: detection.Score}, ImportSourceHarness: req.SourceHarness, ImportSourceSessionID: req.SourceSessionID, ImportedAt: &now}
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
	if setter, ok := parser.(runner.TranscriptRetentionSetter); ok {
		setter.SetTranscriptRetention(true)
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
		// Missing terminal evidence is an honest unknown outcome, not a
		// provider failure. This preserves imported sessions for analysis
		// without inventing a terminal result.
		run.Status = domain.RunStatusUnknown
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

// rehydrateImportedTranscript restores the append-only source stream for an
// imported run whose normalized events were evicted before imported retention
// was exempted. The run-state transcript is the durable copy created during
// import, so replay does not need to reach back into a provider archive or
// create a duplicate run identity.
func (o *Orchestrator) rehydrateImportedTranscript(ctx context.Context, run *domain.Run) error {
	if run == nil || run.ExecutionMode.Normalized() != domain.ExecutionModeImported || strings.TrimSpace(run.TranscriptPath) == "" {
		return nil
	}
	if _, err := os.Stat(run.TranscriptPath); err != nil {
		return fmt.Errorf("imported transcript is unavailable: %w", err)
	}
	if run.ResolvedConfig == nil {
		return fmt.Errorf("imported run has no transcript codec")
	}
	owned, err := o.runners.Get(run.ResolvedConfig.RunnerType)
	if err != nil {
		return fmt.Errorf("load imported transcript codec: %w", err)
	}
	parser, ok := owned.(runner.TranscriptParser)
	if !ok {
		return fmt.Errorf("transcript codec %q has no parser", run.ResolvedConfig.RunnerType)
	}
	if factory, ok := parser.(runner.TranscriptParserFactory); ok {
		parser = factory.NewTranscriptParser()
	}
	if setter, ok := parser.(runner.TranscriptModelSetter); ok {
		setter.SetTranscriptModel(runTranscriptModel(run))
	}
	if setter, ok := parser.(runner.TranscriptRetentionSetter); ok {
		setter.SetTranscriptRetention(true)
	}
	var terminal *runner.TranscriptTerminal
	_, terminal, err = runner.Consume(ctx, runner.ConsumeArgs{
		RunID:      run.ID,
		Transcript: run.TranscriptPath,
		ParseFn: func(id uuid.UUID, line string) runner.TranscriptParseResult {
			return parser.ParseTranscriptLine(id, line)
		},
		EventSink: importEventSink{ctx: ctx, runID: run.ID, store: o.events},
		OnAdvance: func(cursor, seq int64) error {
			run.TranscriptCursor, run.TranscriptLastSeq = cursor, seq
			return nil
		},
		OnSessionID: func(id string) error { run.SessionID = id; return nil },
	})
	if err != nil {
		return fmt.Errorf("rehydrate imported transcript: %w", err)
	}
	if terminal != nil && terminal.Success {
		run.Status = domain.RunStatusComplete
		run.ExitCode = intPtr(terminal.ExitCode)
	}
	run.UpdatedAt = o.now()
	return o.runs.Update(ctx, run)
}

func transcriptGoalMetadata(source *os.File, providers ...runner.GoalMarkerProvider) (string, string) {
	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	var preamble string
	for scanner.Scan() {
		if len(providers) > 0 && providers[0] != nil {
			if condition, met, ok := providers[0].GoalStatusFromTranscriptLine(scanner.Text()); ok {
				goalID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/goal/"+condition)).String()
				status := "unmet"
				if met {
					status = "met"
				}
				return goalID, status
			}
		}
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
		if preamble == "" {
			preamble = findGoalPreamble(value)
		}
	}
	if preamble != "" {
		goalID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/goal-preamble/"+preamble)).String()
		return goalID, "unmet"
	}
	return "", ""
}

// findGoalPreamble recovers a stable cohort key when an external transcript
// has no explicit goal_status record. Only a normalized hash is returned;
// prompt text never enters the run metadata or analytical read model.
func findGoalPreamble(value any) string {
	object, ok := value.(map[string]any)
	if !ok {
		if array, ok := value.([]any); ok {
			for _, child := range array {
				if preamble := findGoalPreamble(child); preamble != "" {
					return preamble
				}
			}
		}
		return ""
	}
	role, _ := object["role"].(string)
	if strings.EqualFold(strings.TrimSpace(role), "user") {
		for _, key := range []string{"content", "message", "text", "prompt"} {
			if text := normalizeGoalText(object[key]); len(text) >= 40 {
				return text
			}
		}
	}
	for _, child := range object {
		if preamble := findGoalPreamble(child); preamble != "" {
			return preamble
		}
	}
	return ""
}

func normalizeGoalText(value any) string {
	var text string
	switch value := value.(type) {
	case string:
		text = value
	case []any:
		parts := make([]string, 0, len(value))
		for _, child := range value {
			if part := normalizeGoalText(child); part != "" {
				parts = append(parts, part)
			}
		}
		text = strings.Join(parts, " ")
	case map[string]any:
		for _, key := range []string{"text", "content", "value"} {
			if part := normalizeGoalText(value[key]); part != "" {
				text = part
				break
			}
		}
	}
	return strings.Join(strings.Fields(text), " ")
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

type transcriptDetection struct {
	RunnerType domain.RunnerType
	Score      float64
}

func (o *Orchestrator) detectTranscriptRunner(source *os.File, requested domain.RunnerType) (transcriptDetection, error) {
	if requested != "" {
		if _, err := o.runners.Get(requested); err != nil {
			return transcriptDetection{}, fmt.Errorf("unsupported transcript format: %w", err)
		}
		return transcriptDetection{RunnerType: requested, Score: 1}, nil
	}
	scanner := bufio.NewScanner(source)
	// Archived harness records can contain a large compacted JSON payload on a
	// single line. Detection must inspect those records instead of failing at
	// Scanner's 64 KiB default before codec scoring begins.
	scanner.Buffer(make([]byte, 64*1024), 16<<20)
	type candidateScore struct {
		typeName domain.RunnerType
		score    float64
	}
	scores := map[domain.RunnerType]float64{}
	lines := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines++
		if lines > 200 {
			break
		}
		for _, candidate := range o.runners.List() {
			factory, ok := candidate.(runner.TranscriptParserFactory)
			if !ok {
				continue
			}
			result := factory.NewTranscriptParser().ParseTranscriptLine(uuid.New(), line)
			if result.Err != nil {
				continue
			}
			for _, event := range result.Events {
				if event == nil {
					continue
				}
				switch event.EventType {
				case domain.EventTypeToolCall:
					scores[candidate.Type()]++
					scores[candidate.Type()] += 3
				case domain.EventTypeToolResult:
					scores[candidate.Type()] += 3
				case domain.EventTypeMessage:
					scores[candidate.Type()]++
				}
			}
			if result.Terminal != nil {
				scores[candidate.Type()] += 10
			}
			if result.SessionID != "" {
				scores[candidate.Type()] += 3
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return transcriptDetection{}, fmt.Errorf("read transcript: %w", err)
	}
	ranked := make([]candidateScore, 0, len(scores))
	for typeName, score := range scores {
		if score > 0 {
			ranked = append(ranked, candidateScore{typeName: typeName, score: score})
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].typeName < ranked[j].typeName
		}
		return ranked[i].score > ranked[j].score
	})
	if len(ranked) == 0 {
		return transcriptDetection{}, fmt.Errorf("unsupported transcript format: no typed events, terminal, or session identity found")
	}
	if len(ranked) > 1 && ranked[0].score == ranked[1].score {
		return transcriptDetection{}, fmt.Errorf("ambiguous transcript format: %s and %s both scored %.0f", ranked[0].typeName, ranked[1].typeName, ranked[0].score)
	}
	return transcriptDetection{RunnerType: ranked[0].typeName, Score: ranked[0].score}, nil
}

type importEventSink struct {
	ctx   context.Context
	runID uuid.UUID
	store interface {
		Append(context.Context, uuid.UUID, ...*domain.RunEvent) error
	}
}

func (s importEventSink) Emit(event *domain.RunEvent) error {
	if event != nil {
		if message, ok := event.Data.(*domain.MessageEventData); ok && (strings.EqualFold(message.Role, "user") || strings.EqualFold(message.Role, "operator")) {
			copy := *message
			copy.Content = transcriptredact.Redact(copy.Content)
			event = &domain.RunEvent{ID: event.ID, RunID: event.RunID, EventType: event.EventType, Timestamp: event.Timestamp, Data: &copy}
		}
	}
	return s.store.Append(s.ctx, s.runID, event)
}
func (s importEventSink) Close() error { return nil }

func importedRunLifecycleError(operation string) error {
	return domain.NewValidationErrorWithCode("run", "imported run cannot perform lifecycle operation: "+operation, domain.ErrCodePolicyScope)
}
