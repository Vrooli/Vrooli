package planlog

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"plan-manager/internal/httpc"
	planmodel "plan-manager/internal/planmodel"

	"github.com/vrooli/api-core/discovery"
)

const (
	scenarioQASystem           = "scenario-qa"
	swarmManagerSystem         = "swarm-manager"
	promptManagerSystem        = "prompt-manager"
	promptManagerTeamAPI       = "/api/v1/teams/%s/knowledge"
	promptManagerBugCaptureAPI = "/api/v1/teams/%s/bugs/capture"
	swarmRecordsAPI            = "/api/v1/records"

	bugWriterSkillID = "report-bug"
	// Retained only for the legacy lookup helper until its callers are removed.
	// New capture forwarding never uses these classifications.
	recordScenario   = "plan-manager"
	recordKind       = "execute"
	recordOutcome    = "shipped"
	planlogMarkerKey = "planlog-entry"
)

// URLResolver resolves local scenario API URLs. Production uses
// api-core/discovery; tests inject a static resolver.
type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

// NewScenarioQABugReporter returns the live scenario-qa bug reporter. It writes
// through Prompt Manager's team knowledge API because bug-inbox is currently a
// Prompt Manager knowledge topic, not a dedicated scenario-qa API.
func NewScenarioQABugReporter(doer httpc.Doer, resolver URLResolver) BugReporter {
	return &scenarioQABugReporter{doer: defaultDoer(doer), resolver: defaultResolver(resolver)}
}

// NewSwarmRecordWriter returns the live Swarm Manager record writer.
func NewSwarmRecordWriter(doer httpc.Doer, resolver URLResolver) RecordWriter {
	return &swarmRecordWriter{doer: defaultDoer(doer), resolver: defaultResolver(resolver)}
}

func defaultDoer(doer httpc.Doer) httpc.Doer {
	if doer != nil {
		return doer
	}
	return http.DefaultClient
}

func defaultResolver(resolver URLResolver) URLResolver {
	if resolver != nil {
		return resolver
	}
	return discovery.NewResolver(discovery.ResolverConfig{})
}

type scenarioQABugReporter struct {
	doer     httpc.Doer
	resolver URLResolver
}

func (r *scenarioQABugReporter) FileBug(ctx context.Context, entry Entry) (DownstreamRef, error) {
	ref := DownstreamRef{System: scenarioQASystem, Kind: "bug_report"}
	baseURL, err := r.resolve(ctx, promptManagerSystem)
	if err != nil {
		return ref, ErrDownstreamUnavailable{System: scenarioQASystem, Reason: err.Error()}
	}
	result, err := r.capture(ctx, baseURL, entry)
	if err != nil {
		return ref, err
	}
	ref.Reference = firstNonEmpty(result.Knowledge.ID, result.DraftID)
	ref.Capture = planmodel.CaptureDisposition{State: result.Disposition, DraftID: result.DraftID, Needs: result.Needs, Invalid: result.Invalid, Warnings: result.Warnings, NextAction: result.NextAction}
	ref.Detail = "scenario-qa disposition: " + result.Disposition
	return ref, nil
}

func (r *scenarioQABugReporter) resolve(ctx context.Context, scenario string) (string, error) {
	baseURL, err := r.resolver.ResolveScenarioURLDefault(ctx, scenario)
	if err != nil {
		return "", fmt.Errorf("resolve %s URL: %w", scenario, err)
	}
	return strings.TrimRight(baseURL, "/"), nil
}

func (r *scenarioQABugReporter) lookup(ctx context.Context, baseURL, topic string) (string, bool, error) {
	values := url.Values{}
	values.Set("topic", topic)
	values.Set("last", "1")
	endpoint := baseURL + fmt.Sprintf(promptManagerTeamAPI, scenarioQASystem) + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", false, err
	}
	resp, err := r.doer.Do(req)
	if err != nil {
		return "", false, ErrDownstreamUnavailable{System: scenarioQASystem, Reason: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return "", false, ErrDownstreamUnavailable{System: scenarioQASystem, Reason: statusDetail(resp)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, fmt.Errorf("scenario-qa bug lookup rejected: %s", statusDetail(resp))
	}
	var out knowledgeListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", false, fmt.Errorf("decode scenario-qa bug lookup: %w", err)
	}
	for _, entry := range out.Entries {
		if entry.Topic == topic && strings.TrimSpace(entry.ID) != "" {
			return entry.ID, true, nil
		}
	}
	return "", false, nil
}

func (r *scenarioQABugReporter) capture(ctx context.Context, baseURL string, entry Entry) (bugCaptureResponse, error) {
	payload := bugCaptureRequest{Title: entry.Title, SignalType: entry.Bug.SignalType, Severity: entry.Bug.Severity, Repro: entry.Bug.Repro, Expected: entry.Bug.Expected, Actual: entry.Bug.Actual, Description: entry.Bug.Description, Context: entry.Bug.Context, HonestyFlags: entry.Bug.HonestyFlags, IdempotencyKey: entry.ID}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return bugCaptureResponse{}, err
	}
	endpoint := baseURL + fmt.Sprintf(promptManagerBugCaptureAPI, scenarioQASystem)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return bugCaptureResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vrooli-Attribution", bugAttributionHeader())
	resp, err := r.doer.Do(req)
	if err != nil {
		return bugCaptureResponse{}, ErrDownstreamUnavailable{System: scenarioQASystem, Reason: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return bugCaptureResponse{}, ErrDownstreamUnavailable{System: scenarioQASystem, Reason: statusDetail(resp)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return bugCaptureResponse{}, fmt.Errorf("scenario-qa bug capture rejected: %s", statusDetail(resp))
	}
	var out bugCaptureResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return bugCaptureResponse{}, fmt.Errorf("decode scenario-qa bug capture: %w", err)
	}
	if out.Disposition != "published" && out.Disposition != "draft" {
		return bugCaptureResponse{}, fmt.Errorf("scenario-qa bug capture returned unknown disposition %q", out.Disposition)
	}
	return out, nil
}

type swarmRecordWriter struct {
	doer     httpc.Doer
	resolver URLResolver
}

func (w *swarmRecordWriter) WriteRecord(ctx context.Context, entry Entry) (DownstreamRef, error) {
	ref := DownstreamRef{System: swarmManagerSystem, Kind: "record"}
	baseURL, err := w.resolve(ctx)
	if err != nil {
		return ref, ErrDownstreamUnavailable{System: swarmManagerSystem, Reason: err.Error()}
	}
	result, err := w.capture(ctx, baseURL, entry)
	if err != nil {
		return ref, err
	}
	ref.Reference = result.Record.ID
	ref.Capture = planmodel.CaptureDisposition{State: result.Disposition, DraftID: result.DraftID, Needs: result.Needs, Invalid: result.Invalid, Warnings: result.Warnings, NextAction: result.NextAction}
	ref.Detail = "swarm-manager disposition: " + result.Disposition
	return ref, nil
}

func (w *swarmRecordWriter) resolve(ctx context.Context) (string, error) {
	baseURL, err := w.resolver.ResolveScenarioURLDefault(ctx, swarmManagerSystem)
	if err != nil {
		return "", fmt.Errorf("resolve swarm-manager URL: %w", err)
	}
	return strings.TrimRight(baseURL, "/"), nil
}

func (w *swarmRecordWriter) lookup(ctx context.Context, baseURL, marker string) (string, bool, error) {
	values := url.Values{}
	values.Set("scenario", recordScenario)
	values.Set("kind", recordKind)
	endpoint := baseURL + swarmRecordsAPI + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", false, err
	}
	resp, err := w.doer.Do(req)
	if err != nil {
		return "", false, ErrDownstreamUnavailable{System: swarmManagerSystem, Reason: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return "", false, ErrDownstreamUnavailable{System: swarmManagerSystem, Reason: statusDetail(resp)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, fmt.Errorf("swarm-manager record lookup rejected: %s", statusDetail(resp))
	}
	var out recordListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", false, fmt.Errorf("decode swarm-manager record lookup: %w", err)
	}
	for _, rec := range out.Records {
		if strings.Contains(rec.Trigger, marker) && strings.TrimSpace(rec.ID) != "" {
			return rec.ID, true, nil
		}
	}
	return "", false, nil
}

func (w *swarmRecordWriter) capture(ctx context.Context, baseURL string, entry Entry) (recordCaptureResponse, error) {
	payload := recordCaptureRequest{Kind: entry.Record.Kind, Scenario: entry.Record.Scenario, Trigger: entry.Record.Trigger, Approach: entry.Record.Approach, Evidence: entry.Record.Evidence, Outcome: entry.Record.Outcome, CreatedBy: entry.Record.CreatedBy, IdempotencyKey: entry.ID}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return recordCaptureResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+swarmRecordsAPI+"/capture", &body)
	if err != nil {
		return recordCaptureResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.doer.Do(req)
	if err != nil {
		return recordCaptureResponse{}, ErrDownstreamUnavailable{System: swarmManagerSystem, Reason: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return recordCaptureResponse{}, ErrDownstreamUnavailable{System: swarmManagerSystem, Reason: statusDetail(resp)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return recordCaptureResponse{}, fmt.Errorf("swarm-manager record capture rejected: %s", statusDetail(resp))
	}
	var out recordCaptureResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return recordCaptureResponse{}, fmt.Errorf("decode swarm-manager record capture: %w", err)
	}
	if out.Disposition != "published" && out.Disposition != "draft" {
		return recordCaptureResponse{}, fmt.Errorf("swarm-manager record capture returned unknown disposition %q", out.Disposition)
	}
	return out, nil
}

type knowledgeAddRequest struct {
	Topic      string `json:"topic"`
	Content    string `json:"content"`
	CallerNote string `json:"caller_note,omitempty"`
	Source     string `json:"source,omitempty"`
}

type knowledgeListResponse struct {
	Entries []knowledgeEntryResponse `json:"entries"`
}

type knowledgeEntryResponse struct {
	ID    string `json:"id"`
	Topic string `json:"topic"`
}

type bugCaptureRequest struct {
	Title          string            `json:"title"`
	SignalType     string            `json:"signal_type"`
	Severity       string            `json:"severity"`
	Repro          []string          `json:"repro"`
	Expected       string            `json:"expected"`
	Actual         string            `json:"actual"`
	Description    string            `json:"description"`
	Context        map[string]string `json:"context"`
	HonestyFlags   []string          `json:"honesty_flags"`
	IdempotencyKey string            `json:"idempotency_key"`
}

type bugCaptureResponse struct {
	Disposition string                        `json:"disposition"`
	DraftID     string                        `json:"draft_id"`
	Knowledge   knowledgeEntryResponse        `json:"knowledge"`
	Needs       []string                      `json:"needs"`
	Invalid     []planmodel.CaptureDiagnostic `json:"invalid"`
	Warnings    []string                      `json:"warnings"`
	NextAction  []string                      `json:"next_action"`
}

type recordCreateRequest struct {
	Kind      string `json:"kind"`
	Scenario  string `json:"scenario"`
	Trigger   string `json:"trigger"`
	Approach  string `json:"approach"`
	Evidence  string `json:"evidence"`
	Outcome   string `json:"outcome"`
	CreatedBy string `json:"created_by"`
}

type recordCaptureRequest struct {
	Kind           string `json:"kind"`
	Scenario       string `json:"scenario"`
	Trigger        string `json:"trigger"`
	Approach       string `json:"approach"`
	Evidence       string `json:"evidence"`
	Outcome        string `json:"outcome"`
	CreatedBy      string `json:"created_by"`
	IdempotencyKey string `json:"idempotency_key"`
}

type recordCaptureResponse struct {
	Disposition string                        `json:"disposition"`
	DraftID     string                        `json:"draft_id"`
	Record      recordResponse                `json:"record"`
	Needs       []string                      `json:"needs"`
	Invalid     []planmodel.CaptureDiagnostic `json:"invalid"`
	Warnings    []string                      `json:"warnings"`
	NextAction  []string                      `json:"next_action"`
}

type recordListResponse struct {
	Records []recordResponse `json:"records"`
}

type recordEnvelope struct {
	Record recordResponse `json:"record"`
}

type recordResponse struct {
	ID      string `json:"id"`
	Trigger string `json:"trigger"`
}

func bugTopic(entry Entry) string {
	return "bug-inbox/code-defect/" + slugify(entry.Title, entry.ID)
}

func slugify(title, fallback string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug := strings.Trim(re.ReplaceAllString(title, "-"), "-")
	if slug == "" {
		slug = "planlog-entry"
	}
	parts := strings.Split(slug, "-")
	if len(parts) > 6 {
		slug = strings.Join(parts[:6], "-")
	}
	return slug + "-" + shortID(fallback)
}

func shortID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func bugContent(entry Entry) string {
	observed := observedDate(entry)
	severity := bugSeverity(entry.Severity)
	command := yamlStringOrNull(entry.SourceCommand)
	return fmt.Sprintf(`---
severity: %s
reporter: plan-manager
reporter_team: plan-manager
observed_at: %s
context:
  scenario: plan-manager
  skill: null
  member: null
  command: %s
repro:
  - "Plan Manager log entry %s recorded this bug_report."
expected: "Plan execution should proceed without this defect."
actual: %s
description: |
%s
honesty_flags: [minimal-context]
---

## What you were trying to do

Plan Manager was forwarding a bug_report entry from its execution log.

## What happened

%s

## Why this looks like a bug

The entry was explicitly filed as a Plan Manager bug_report and forwarded through the scenario-qa intake contract.
`,
		severity,
		observed,
		command,
		entry.ID,
		yamlQuote(firstNonEmpty(entry.Title, "Plan Manager bug_report")),
		indentBlock(bugDescription(entry), 2),
		provenanceBlock(entry),
	)
}

func bugDescription(entry Entry) string {
	parts := []string{}
	if strings.TrimSpace(entry.Detail) != "" {
		parts = append(parts, strings.TrimSpace(entry.Detail))
	}
	if len(entry.Evidence) > 0 {
		parts = append(parts, "Evidence:\n- "+strings.Join(entry.Evidence, "\n- "))
	}
	parts = append(parts, provenanceBlock(entry))
	return strings.Join(parts, "\n\n")
}

func bugSeverity(sev Severity) string {
	switch sev {
	case planmodel.LogSeverityCritical:
		return "blocker"
	case planmodel.LogSeverityHigh, planmodel.LogSeverityMedium:
		return "major"
	default:
		return "minor"
	}
}

func observedDate(entry Entry) string {
	if ts, err := time.Parse(time.RFC3339Nano, entry.CreatedAt); err == nil {
		return ts.UTC().Format("2006-01-02")
	}
	return time.Now().UTC().Format("2006-01-02")
}

func planlogMarker(entryID string) string {
	return "[" + planlogMarkerKey + ":" + strings.TrimSpace(entryID) + "]"
}

func recordApproach(entry Entry) string {
	if strings.TrimSpace(entry.Detail) != "" {
		return strings.TrimSpace(entry.Detail) + "\n\n" + provenanceBlock(entry)
	}
	return provenanceBlock(entry)
}

func recordEvidence(entry Entry) string {
	parts := []string{}
	if len(entry.Evidence) > 0 {
		parts = append(parts, strings.Join(entry.Evidence, "\n"))
	}
	parts = append(parts, provenanceBlock(entry))
	return strings.Join(parts, "\n\n")
}

func provenanceBlock(entry Entry) string {
	lines := []string{
		"Plan Manager provenance:",
		"- entry_id: " + nullish(entry.ID),
		"- plan_id: " + nullish(entry.PlanID),
		"- execution_id: " + nullish(entry.ExecutionID),
		"- phase_id: " + nullish(entry.PhaseID),
		"- promoted_from_id: " + nullish(entry.PromotedFromID),
		"- attribution_run_id: " + nullish(entry.AttributionRunID),
		"- source_command: " + nullish(entry.SourceCommand),
	}
	return strings.Join(lines, "\n")
}

func bugAttributionHeader() string {
	team := scenarioQASystem
	source := bugWriterSkillID
	payload := map[string]any{
		"kind":            "writer-skill",
		"member_id":       nil,
		"team_id":         team,
		"run_id":          nil,
		"spawn_origin":    "unknown",
		"source_skill_id": source,
	}
	b, _ := json.Marshal(payload)
	return base64.StdEncoding.EncodeToString(b)
}

func statusDetail(resp *http.Response) string {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	msg := strings.TrimSpace(string(data))
	if msg == "" {
		return resp.Status
	}
	return resp.Status + ": " + msg
}

func yamlStringOrNull(s string) string {
	if strings.TrimSpace(s) == "" {
		return "null"
	}
	return yamlQuote(s)
}

func yamlQuote(s string) string {
	b, _ := json.Marshal(strings.TrimSpace(s))
	return string(b)
}

func indentBlock(s string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func nullish(s string) string {
	if strings.TrimSpace(s) == "" {
		return "null"
	}
	return strings.TrimSpace(s)
}
