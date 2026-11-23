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
	"github.com/vrooli/cli-core/agenttranscript"
)

// ImportTranscriptRequest captures one harness transcript file on demand.
// The file is read in place by the Transcript Indexer; Label applies when the
// transcript carries no harness title. SourceHarness can add caller-declared
// scope to the import policy decision; durable provenance still comes from
// the file itself. SourceSessionID is accepted for compatibility but does not
// override file identity.
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

// transcriptLabelEvidence is what a transcript offers for naming its run.
// Source is the excerpt a title generator reads when neither a harness title
// nor an operator prompt exists: one plain-text line per conversation message,
// in transcript order.
type transcriptLabelEvidence struct {
	HarnessTitle string
	UserPrompt   string
	Source       string
}

// A title needs the task and the agent's first replies. Transcripts open with
// tens of kilobytes of setup (session metadata, tool schemas, instruction
// files), so the excerpt keeps message text only and stops at titleSourceBytes,
// about 1,500 tokens.
const (
	titleSourceBytes     = 6 * 1024
	titleSourceItemBytes = 600
)

func agentManagerTranscriptEvidenceOptions(markers runner.GoalMarkerProvider, maxLineBytes, goalMaxLineBytes int, labels, goals bool) agenttranscript.TranscriptEvidenceOptions {
	options := agenttranscript.TranscriptEvidenceOptions{
		ExtractLabels:     labels,
		MaxLineBytes:      maxLineBytes,
		GoalMaxLineBytes:  goalMaxLineBytes,
		TitleSourceBytes:  titleSourceBytes,
		TitleItemBytes:    titleSourceItemBytes,
		UserPromptBytes:   140,
		IsInjectedContext: domain.IsInjectedContextText,
	}
	if goals {
		options.GoalPreambleMinBytes = 40
		if markers != nil {
			options.ParseGoalStatusLine = func(line string) (string, string, bool) {
				marker, ok := markers.GoalStatusFromTranscriptLine(line)
				return marker.Objective, string(marker.Status), ok
			}
		}
	}
	return options
}

func extractAgentManagerTranscriptEvidence(source io.Reader, markers runner.GoalMarkerProvider, maxLineBytes, goalMaxLineBytes int, labels, goals bool) (agenttranscript.TranscriptEvidence, error) {
	return agenttranscript.ExtractTranscriptEvidence(source, agentManagerTranscriptEvidenceOptions(markers, maxLineBytes, goalMaxLineBytes, labels, goals))
}

func transcriptLabelEvidenceFromShared(evidence agenttranscript.TranscriptEvidence) transcriptLabelEvidence {
	return transcriptLabelEvidence{HarnessTitle: evidence.HarnessTitle, UserPrompt: evidence.UserPrompt, Source: evidence.TitleSource}
}

func readTranscriptLabelEvidence(source io.Reader) (transcriptLabelEvidence, error) {
	evidence, err := extractAgentManagerTranscriptEvidence(source, nil, importTranscriptMaxLineBytes, importTranscriptMaxLineBytes, true, false)
	if err != nil {
		return transcriptLabelEvidence{}, err
	}
	return transcriptLabelEvidenceFromShared(evidence), nil
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

	// With no conversation text there is nothing to name; a generated title
	// would only restate the setup boilerplate.
	if o.labelGenerator != nil && strings.TrimSpace(evidence.Source) != "" {
		response, err := o.labelGenerator.Extract(ctx, structuredresult.ExtractRequest{
			Source:        evidence.Source,
			Schema:        json.RawMessage(`{"type":"string","minLength":1,"maxLength":160}`),
			Instruction:   "Create a concise human-readable title naming the task in this coding-agent session excerpt. Leave out session IDs and agent names. Return only the title.",
			PrivacyClass:  structuredresult.RunOutputPrivacyClass,
			PrivacySource: "agent_manager.imported_agent_transcript",
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
	if policyHarness := strings.TrimSpace(req.SourceHarness); policyHarness != "" {
		ctx = context.WithValue(ctx, importPolicyHarnessContextKey{}, policyHarness)
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

// AssignRunWorkload sets explicit attribution on an existing run without
// rewriting lifecycle state or inferring anything from the run identity.
func (o *Orchestrator) AssignRunWorkload(ctx context.Context, id uuid.UUID, workload domain.WorkloadRef) (*domain.Run, error) {
	updater, ok := o.runs.(repository.RunWorkloadUpdater)
	if !ok {
		return nil, fmt.Errorf("run repository does not support workload assignment")
	}
	workload.Key = strings.TrimSpace(workload.Key)
	workload.Instance = strings.TrimSpace(workload.Instance)
	if !workload.Kind.IsValid() || workload.Key == "" {
		return nil, fmt.Errorf("valid workload kind and non-empty workload key are required")
	}
	if err := updater.UpdateRunWorkload(ctx, id, workload); err != nil {
		return nil, err
	}
	run, err := o.runs.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, domain.NewNotFoundError("Run", id)
	}
	return run, nil
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
	var markers runner.GoalMarkerProvider
	if len(providers) > 0 {
		markers = providers[0]
	}
	evidence, _ := extractAgentManagerTranscriptEvidence(source, markers, 4<<20, 4<<20, false, true)
	return transcriptGoalMetadataFromShared(evidence)
}

func transcriptGoalMetadataFromShared(evidence agenttranscript.TranscriptEvidence) (string, string) {
	if evidence.GoalMarkerFound {
		goalID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/goal/"+evidence.GoalMarkerObjective)).String()
		return goalID, evidence.GoalMarkerStatus
	}
	if evidence.GoalPreamble != "" {
		goalID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/goal-preamble/"+evidence.GoalPreamble)).String()
		return goalID, "unmet"
	}
	return "", ""
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

func (o *Orchestrator) detectTranscriptRunner(source io.Reader, requested domain.RunnerType) (transcriptDetection, error) {
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
