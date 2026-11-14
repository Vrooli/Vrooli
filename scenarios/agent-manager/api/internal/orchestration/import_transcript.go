// Import transcript orchestration adopts external harness evidence safely.
package orchestration

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/conversationpolicy"
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/repository"
	"agent-manager/internal/structuredresult"
	"agent-manager/internal/transcriptindex"

	"github.com/google/uuid"
)

// ImportTranscriptRequest captures one harness transcript file on demand.
// The file is read in place by the Transcript Indexer; Label applies when the
// transcript carries no harness title. SourceHarness and SourceSessionID are
// accepted for compatibility with callers but provenance now comes from the
// file itself.
type ImportTranscriptRequest struct {
	Path            string
	RunnerType      domain.RunnerType
	Label           string
	LabelSource     domain.RunLabelSource
	SourceHarness   string
	SourceSessionID string
}

var importedTranscriptTaskID = uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/imported-transcripts"))

// importTranscriptMaxLineBytes bounds the label/runner-detection line scanners.
// Codex rollouts legitimately embed >16MB lines (inline images, large tool
// output); the codex codec already tolerates 32–64MB elsewhere. At 16MB these
// heuristic scanners returned "token too long" from scanner.Err() and hard-
// failed the whole import of an otherwise parseable session, so a single large
// line silently dropped a real run from the evidence corpus. The event parse
// itself (runner.Consume) reads with an unbounded bufio.Reader and is unaffected.
const importTranscriptMaxLineBytes = 64 << 20

type transcriptLabelEvidence struct {
	HarnessTitle string
	UserPrompt   string
	Source       string
}

func readTranscriptLabelEvidence(source io.Reader) (transcriptLabelEvidence, error) {
	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 64*1024), importTranscriptMaxLineBytes)
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
			if candidate := findTranscriptString(value, transcriptUserText); !domain.IsInjectedContextText(candidate) {
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

// ImportTranscript captures one transcript file on demand through the
// Transcript Indexer — the same capture path as the background loop — and
// returns its run. A file inside the indexed roots needs no manual import;
// this serves files elsewhere and "capture it now".
func (o *Orchestrator) ImportTranscript(ctx context.Context, req ImportTranscriptRequest) (*domain.Run, error) {
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return nil, fmt.Errorf("transcript path is required")
	}
	if o.transcriptIndex == nil {
		return nil, domain.NewConfigMissingError("transcript_index", "transcript indexer is not configured", nil)
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("open transcript: %w", err)
	}
	// The caller's harness label is the import provenance visibility rules
	// match on; without one the runner names the format.
	harness := strings.TrimSpace(req.SourceHarness)
	switch req.RunnerType {
	case domain.RunnerTypeCodex:
		harness = cmpOr(harness, transcriptindex.HarnessCodex)
	case domain.RunnerTypeClaudeCode:
		harness = cmpOr(harness, transcriptindex.HarnessClaude)
	case "":
	default:
		if _, err := o.runners.Get(req.RunnerType); err != nil {
			return nil, fmt.Errorf("unsupported transcript format: %w", err)
		}
	}
	if transcriptindex.Family(harness) == transcriptindex.HarnessConfigured && req.RunnerType != "" {
		harness = string(req.RunnerType)
	}
	runID, err := o.transcriptIndex.IndexPath(ctx, path, harness)
	if errors.Is(err, transcriptindex.ErrExcluded) {
		return nil, newImportExcludedError(o.conversationPolicy.Evaluate(conversationpolicy.EnforceImport, importPolicyFields(req, req.RunnerType)))
	}
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(runID)
	if err != nil {
		return nil, err
	}
	run, err := o.runs.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, domain.NewNotFoundError("Run", id)
	}
	if label := strings.TrimSpace(req.Label); label != "" && run.LabelSource != domain.RunLabelSourceHarness &&
		(req.LabelSource == domain.RunLabelSourceManual || req.LabelSource == domain.RunLabelSourceDerived) {
		run.Label, run.LabelSource = shortenTranscriptLabel(label), req.LabelSource
		if err := o.runs.Update(ctx, run); err != nil {
			return nil, err
		}
	}
	return o.attachRunActions(ctx, run), nil
}

func cmpOr(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
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
		evidence, readErr := readTranscriptLabelEvidence(io.LimitReader(file, transcriptHeadBytes))
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

func transcriptGoalMetadata(source io.Reader, providers ...runner.GoalMarkerProvider) (string, string) {
	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	var preamble string
	for scanner.Scan() {
		if len(providers) > 0 && providers[0] != nil {
			if marker, ok := providers[0].GoalStatusFromTranscriptLine(scanner.Text()); ok {
				goalID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/goal/"+marker.Objective)).String()
				return goalID, string(marker.Status)
			}
		}
		var value any
		if json.Unmarshal(scanner.Bytes(), &value) != nil {
			continue
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
	scanner.Buffer(make([]byte, 64*1024), importTranscriptMaxLineBytes)
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

// redactMatchesExistingRun reports whether an already-imported run matches a
// `redact` rule at the import seam today. The re-import/rehydrate path has no
// fresh request to evaluate, so it derives the policy fields from the stored
// run. It does NOT record a counter (RecordImport already fired on the first
// import); it only answers "should replayed events be masked". Returns false
// when no masker is configured so rehydrate stays a cheap no-op by default.
func (o *Orchestrator) redactMatchesExistingRun(run *domain.Run) bool {
	if o.piiMasker == nil || !o.conversationPolicy.Enabled() || run == nil {
		return false
	}
	harness := strings.TrimSpace(run.ImportSourceHarness)
	if harness == "" && run.ResolvedConfig != nil {
		harness = string(run.ResolvedConfig.RunnerType)
	}
	fields := conversationpolicy.Fields{Harness: harness, Tag: strings.TrimSpace(run.Tag)}
	decision := o.conversationPolicy.Evaluate(conversationpolicy.EnforceImport, fields)
	return decision.Matched && decision.Action == conversationpolicy.ActionRedact
}

// ImportExcludedError reports that a visibility policy excluded a transcript at
// the import seam, so no run_events were written. It is a signal, not a
// failure: the sweep counts it as a policy exclusion rather than an error.
type ImportExcludedError struct {
	Decision conversationpolicy.Decision
}

func (e *ImportExcludedError) Error() string {
	return "conversation excluded by import policy: " + conversationpolicy.DescribeRule(e.Decision.Rule)
}

func newImportExcludedError(decision conversationpolicy.Decision) error {
	return &ImportExcludedError{Decision: decision}
}

// IsImportExcluded reports whether err is a policy-driven import exclusion.
func IsImportExcluded(err error) bool {
	var excluded *ImportExcludedError
	return errors.As(err, &excluded)
}

// importPolicyFields builds the policy match surface known at the import seam.
// Model is usually not yet resolved from the transcript here, so model-based
// rules are better expressed at the search seam; harness, tag, and scope are
// reliable at import (enough for "keep this harness's chats out entirely").
func importPolicyFields(req ImportTranscriptRequest, runnerType domain.RunnerType) conversationpolicy.Fields {
	harness := strings.TrimSpace(req.SourceHarness)
	if harness == "" {
		harness = string(runnerType)
	}
	tag := "agent-manager-imported"
	if label := strings.TrimSpace(req.Label); label != "" {
		tag = "agent-manager-imported-" + label
	}
	return conversationpolicy.Fields{
		Harness: harness,
		Tag:     tag,
	}
}

func importedRunLifecycleError(operation string) error {
	return domain.NewValidationErrorWithCode("run", "imported run cannot perform lifecycle operation: "+operation, domain.ErrCodePolicyScope)
}
