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
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/repository"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/runstate"
	"agent-manager/internal/structuredresult"
	"agent-manager/internal/transcriptredact"

	"github.com/google/uuid"
)

// ImportTranscriptRequest adopts a newline-delimited harness transcript as a
// terminal read-only run. The caller's file is never parsed in place.
type ImportTranscriptRequest struct {
	Path            string
	RunnerType      domain.RunnerType
	Label           string
	LabelSource     domain.RunLabelSource
	SourceHarness   string
	SourceSessionID string
}

var importedTranscriptTaskID = uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/imported-transcripts"))

type transcriptLabelEvidence struct {
	HarnessTitle string
	UserPrompt   string
	Source       string
}

func readTranscriptLabelEvidence(source *os.File) (transcriptLabelEvidence, error) {
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return transcriptLabelEvidence{}, err
	}
	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 64*1024), 16<<20)
	var evidence transcriptLabelEvidence
	const maxSourceBytes = 64 * 1024
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(evidence.Source) < maxSourceBytes {
			remaining := maxSourceBytes - len(evidence.Source)
			if len(line) > remaining {
				line = line[:remaining]
			}
			evidence.Source += string(line) + "\n"
		}
		var value any
		if json.Unmarshal(scanner.Bytes(), &value) != nil {
			continue
		}
		if evidence.HarnessTitle == "" {
			evidence.HarnessTitle = findTranscriptString(value, func(object map[string]any) (string, bool) {
				typeName, _ := object["type"].(string)
				if typeName != "ai-title" {
					return "", false
				}
				title, _ := object["aiTitle"].(string)
				return strings.TrimSpace(title), strings.TrimSpace(title) != ""
			})
		}
		if evidence.UserPrompt == "" {
			// Keep scanning past harness-injected preambles. Codex writes its
			// instructions block as the first user-role record, so accepting
			// the first user text verbatim labelled thousands of runs with the
			// same AGENTS.md boilerplate instead of the operator's request.
			if candidate := findTranscriptString(value, transcriptUserText); !isInjectedContextText(candidate) {
				evidence.UserPrompt = candidate
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return transcriptLabelEvidence{}, err
	}
	return evidence, nil
}

func findTranscriptString(value any, match func(map[string]any) (string, bool)) string {
	switch item := value.(type) {
	case map[string]any:
		if result, ok := match(item); ok {
			return result
		}
		for _, child := range item {
			if result := findTranscriptString(child, match); result != "" {
				return result
			}
		}
	case []any:
		for _, child := range item {
			if result := findTranscriptString(child, match); result != "" {
				return result
			}
		}
	}
	return ""
}

// injectedContextFilePrefixes are instruction files harnesses inline into the
// first user-role record. The marker is always at the start of the text.
var injectedContextFilePrefixes = []string{
	"# agents.md",
	"# claude.md",
	"agents.md instructions",
	"claude.md instructions",
}

// isInjectedContextText reports whether a candidate user message is harness-
// injected context (an instructions block, environment dump, or command
// envelope) rather than something a person typed. Such text is unusable as a
// run label: it is identical across every session the harness started.
//
// Empty text counts as injected so callers keep scanning for a real message.
func isInjectedContextText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}
	// Harness context blocks open with a tag: <user_instructions>,
	// <environment_context>, <local-command-caveat>, <command-name>, ...
	if strings.HasPrefix(trimmed, "<") {
		if end := strings.IndexByte(trimmed, '>'); end > 1 && end <= 64 {
			return true
		}
	}
	lowered := strings.ToLower(trimmed)
	for _, prefix := range injectedContextFilePrefixes {
		if strings.HasPrefix(lowered, prefix) {
			return true
		}
	}
	return false
}

func transcriptUserText(object map[string]any) (string, bool) {
	if role, _ := object["role"].(string); !strings.EqualFold(strings.TrimSpace(role), "user") {
		if typeName, _ := object["type"].(string); typeName != "user_message" {
			return "", false
		}
	}
	for _, key := range []string{"content", "message", "text", "prompt"} {
		if text := normalizeTranscriptText(object[key]); text != "" {
			return shortenTranscriptLabel(text), true
		}
	}
	return "", false
}

func normalizeTranscriptText(value any) string {
	switch item := value.(type) {
	case string:
		return strings.Join(strings.Fields(item), " ")
	case []any:
		parts := make([]string, 0, len(item))
		for _, child := range item {
			if text := normalizeTranscriptText(child); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, " ")
	case map[string]any:
		for _, key := range []string{"text", "content", "value", "message"} {
			if text := normalizeTranscriptText(item[key]); text != "" {
				return text
			}
		}
	}
	return ""
}

func shortenTranscriptLabel(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 140 {
		return value[:137] + "…"
	}
	return value
}

func (o *Orchestrator) resolveImportedLabel(ctx context.Context, req ImportTranscriptRequest, runnerType domain.RunnerType, evidence transcriptLabelEvidence) (string, domain.RunLabelSource) {
	if evidence.HarnessTitle != "" {
		return evidence.HarnessTitle, domain.RunLabelSourceHarness
	}
	if req.LabelSource == domain.RunLabelSourceManual && strings.TrimSpace(req.Label) != "" {
		return shortenTranscriptLabel(req.Label), domain.RunLabelSourceManual
	}
	if req.LabelSource == domain.RunLabelSourceDerived && strings.TrimSpace(req.Label) != "" {
		return shortenTranscriptLabel(req.Label), domain.RunLabelSourceDerived
	}
	if evidence.UserPrompt != "" {
		return evidence.UserPrompt, domain.RunLabelSourceDerived
	}

	if o.labelGenerator != nil {
		response, err := o.labelGenerator.Extract(ctx, structuredresult.ExtractRequest{
			Source:      evidence.Source,
			Schema:      json.RawMessage(`{"type":"string","minLength":1,"maxLength":160}`),
			Instruction: "Create a concise human-readable title for this coding-agent session. Return only the title.",
		})
		if err == nil {
			var generated string
			if json.Unmarshal(response.Candidate, &generated) == nil && strings.TrimSpace(generated) != "" {
				return shortenTranscriptLabel(generated), domain.RunLabelSourceGenerated
			}
		}
	}

	// A provider outage must not make historical evidence disappear, so the run
	// still gets a deterministic identifying label. The source is placeholder,
	// not generated: no provider wrote this text and it says nothing about the
	// work, so consumers can find these and regenerate them later.
	identifier := strings.TrimSpace(req.SourceSessionID)
	if identifier == "" {
		identifier = "unknown"
	}
	return fmt.Sprintf("%s session %s", runnerType, identifier), domain.RunLabelSourcePlaceholder
}

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
	goalID, _ := transcriptGoalMetadata(source, markerProvider)
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind transcript after goal scan: %w", err)
	}
	labelEvidence, err := readTranscriptLabelEvidence(source)
	if err != nil {
		return nil, fmt.Errorf("read transcript label evidence: %w", err)
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind transcript after label scan: %w", err)
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
	run.Label, run.LabelSource = o.resolveImportedLabel(ctx, req, runnerType, labelEvidence)
	run.GoalID = goalID
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
	}, EventSink: importEventSink{ctx: ctx, runID: run.ID, store: o.events}, OnAdvance: func(cursor, seq int64) error { run.TranscriptCursor, run.TranscriptLastSeq = cursor, seq; return nil }, OnSessionID: func(id string) error { run.SessionID = id; return nil }, OnLabel: func(label string, source domain.RunLabelSource) error {
		if source == domain.RunLabelSourceHarness && strings.TrimSpace(label) != "" {
			run.Label, run.LabelSource = strings.TrimSpace(label), source
		}
		return nil
	}})
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

// LabelBackfillResult reports a label-only repair of imported runs. The
// updater seam intentionally changes no run identity or attribution columns.
type LabelBackfillResult struct {
	Scanned  int `json:"scanned"`
	Updated  int `json:"updated"`
	Skipped  int `json:"skipped"`
	Missing  int `json:"missing"`
	Failures int `json:"failures"`
}

type SubjectBackfillResult struct {
	Scanned  int `json:"scanned"`
	Updated  int `json:"updated"`
	Skipped  int `json:"skipped"`
	Missing  int `json:"missing"`
	Failures int `json:"failures"`
}

// BackfillRunSubjects projects subjects directly from retained invocation
// facts. It never replays or parses raw transcripts, and it updates only the
// derived subject column so historical runs become queryable without
// rewriting identity, result, or attribution fields.
func (o *Orchestrator) BackfillRunSubjects(ctx context.Context) (*SubjectBackfillResult, error) {
	updater, ok := o.runs.(repository.RunSubjectUpdater)
	if !ok {
		return nil, fmt.Errorf("run repository does not support subject-only updates")
	}
	if o.invocationReadModel == nil {
		return nil, fmt.Errorf("invocation read model is not configured")
	}
	runs, err := o.runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 100000}})
	if err != nil {
		return nil, fmt.Errorf("list runs for subject backfill: %w", err)
	}
	result := &SubjectBackfillResult{}
	for _, run := range runs {
		if run == nil {
			continue
		}
		result.Scanned++
		facts, factsErr := o.invocationReadModel.Facts(ctx, run.ID.String())
		if factsErr != nil {
			result.Failures++
			continue
		}
		// Areas come from retained tool-call events, which facts do not carry.
		// A run whose events aged out still yields its tool subject.
		events, eventsErr := o.events.Get(ctx, run.ID, event.GetOptions{})
		if eventsErr != nil {
			events = nil
		}
		subject := invocationreadmodel.DeriveRunSubject(facts, events)
		if len(subject) == 0 {
			result.Missing++
			continue
		}
		if err := updater.UpdateRunSubject(ctx, run.ID, subject); err != nil {
			result.Failures++
			continue
		}
		result.Updated++
	}
	result.Skipped = result.Scanned - result.Updated - result.Missing - result.Failures
	return result, nil
}

// BackfillImportedRunLabels recovers labels for every unlabelled run: imported
// runs from their retained transcript, native runs from their own task title.
// It never invokes the generator: legacy backfills are limited to harness or
// derived evidence, as required for a cost-free and auditable repair.
func (o *Orchestrator) BackfillImportedRunLabels(ctx context.Context) (*LabelBackfillResult, error) {
	updater, ok := o.runs.(repository.RunLabelUpdater)
	if !ok {
		return nil, fmt.Errorf("run repository does not support label-only updates")
	}
	runs, err := o.runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 100000}})
	if err != nil {
		return nil, fmt.Errorf("list imported runs for label backfill: %w", err)
	}
	transcriptPaths := indexExternalTranscriptPaths(runs)
	result := &LabelBackfillResult{}
	for _, run := range runs {
		if run == nil {
			continue
		}
		if run.ExecutionMode.Normalized() != domain.ExecutionModeImported {
			o.backfillNativeRunLabel(ctx, updater, run, result)
			continue
		}
		result.Scanned++
		transcriptPath := strings.TrimSpace(run.TranscriptPath)
		if transcriptPath == "" || !fileExists(transcriptPath) {
			transcriptPath = transcriptPaths[run.ImportSourceSessionID]
		}
		if transcriptPath == "" {
			result.Missing++
			if updateErr := updater.UpdateRunLabel(ctx, run.ID, legacyFallbackLabel(run), domain.RunLabelSourceDerived); updateErr != nil {
				result.Failures++
			} else {
				result.Updated++
			}
			continue
		}
		file, openErr := os.Open(transcriptPath)
		if openErr != nil {
			result.Missing++
			if updateErr := updater.UpdateRunLabel(ctx, run.ID, legacyFallbackLabel(run), domain.RunLabelSourceDerived); updateErr != nil {
				result.Failures++
			} else {
				result.Updated++
			}
			continue
		}
		evidence, readErr := readTranscriptLabelEvidence(file)
		_ = file.Close()
		if readErr != nil {
			result.Failures++
			if updateErr := updater.UpdateRunLabel(ctx, run.ID, legacyFallbackLabel(run), domain.RunLabelSourceDerived); updateErr != nil {
				result.Failures++
			} else {
				result.Updated++
			}
			continue
		}
		label, source := evidence.HarnessTitle, domain.RunLabelSourceHarness
		if label == "" {
			label, source = evidence.UserPrompt, domain.RunLabelSourceDerived
		}
		if label == "" {
			// Nothing in the transcript names the work: no harness title and no
			// user message that was not harness-injected context. Fall back to
			// the same deterministic form the import path uses, and keep the
			// source honest — this label was not derived from session content
			// and no provider generated it.
			label = legacyFallbackLabel(run)
			source = domain.RunLabelSourcePlaceholder
		}
		label = shortenTranscriptLabel(label)
		if label == "" {
			result.Missing++
			continue
		}
		if run.Label == label && run.LabelSource == source {
			result.Skipped++
			continue
		}
		if updateErr := updater.UpdateRunLabel(ctx, run.ID, label, source); updateErr != nil {
			result.Failures++
			continue
		}
		result.Updated++
	}
	return result, nil
}

// backfillNativeRunLabel labels runs Agent Manager executed itself. Unlike
// imported runs — which all share one synthetic task — a native run owns its
// task, so the task title is the best available name for it. The source is
// derived because the label comes from the run's own record rather than from
// session content or an author writing a run label.
func (o *Orchestrator) backfillNativeRunLabel(ctx context.Context, updater repository.RunLabelUpdater, run *domain.Run, result *LabelBackfillResult) {
	if strings.TrimSpace(run.Label) != "" {
		return
	}
	result.Scanned++
	task, err := o.tasks.Get(ctx, run.TaskID)
	if err != nil || task == nil {
		result.Failures++
		return
	}
	label := shortenTranscriptLabel(task.Title)
	if label == "" {
		result.Missing++
		return
	}
	if err := updater.UpdateRunLabel(ctx, run.ID, label, domain.RunLabelSourceDerived); err != nil {
		result.Failures++
		return
	}
	result.Updated++
}

func legacyFallbackLabel(run *domain.Run) string {
	if run == nil {
		return "Imported run"
	}
	harness := strings.TrimSpace(run.ImportSourceHarness)
	harness = strings.TrimPrefix(harness, "resource:")
	if harness == "" {
		harness = "external"
	}
	if session := strings.TrimSpace(run.ImportSourceSessionID); session != "" {
		return shortenTranscriptLabel(fmt.Sprintf("%s session %s", harness, session))
	}
	return "Imported run " + run.ID.String()[:8]
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// indexExternalTranscriptPaths recovers the source file for legacy imported
// rows that predate transcript_path persistence. The index is built once per
// backfill, not once per run, and only from the governed durable-data roots.
func indexExternalTranscriptPaths(runs []*domain.Run) map[string]string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	roots := make(map[string]string)
	for _, run := range runs {
		if run == nil || run.ImportSourceSessionID == "" {
			continue
		}
		switch {
		case strings.Contains(run.ImportSourceHarness, "codex"):
			roots[filepath.Join(home, ".codex", "sessions")] = "codex"
		case strings.Contains(run.ImportSourceHarness, "claude"):
			roots[filepath.Join(home, ".claude", "projects")] = "claude"
		}
	}
	index := make(map[string]string)
	for root := range roots {
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry == nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
				return walkErr
			}
			file, openErr := os.Open(path)
			if openErr != nil {
				return nil
			}
			sessionID := transcriptSessionID(file)
			_ = file.Close()
			if sessionID != "" {
				index[sessionID] = path
			}
			return nil
		})
	}
	return index
}

func transcriptSessionID(file *os.File) string {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return ""
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	for lines := 0; scanner.Scan() && lines < 80; lines++ {
		var value any
		if json.Unmarshal(scanner.Bytes(), &value) != nil {
			continue
		}
		if id := findTranscriptString(value, func(object map[string]any) (string, bool) {
			if typeName, _ := object["type"].(string); typeName == "session_meta" {
				if id, ok := object["id"].(string); ok && strings.TrimSpace(id) != "" {
					return strings.TrimSpace(id), true
				}
			}
			for _, key := range []string{"session_id", "sessionId", "thread_id", "threadId"} {
				if id, ok := object[key].(string); ok && strings.TrimSpace(id) != "" {
					return strings.TrimSpace(id), true
				}
			}
			return "", false
		}); id != "" {
			return id
		}
	}
	return ""
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
