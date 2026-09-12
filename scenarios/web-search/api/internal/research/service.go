package research

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"web-search/internal/capture"
	"web-search/internal/evidence"
	"web-search/internal/findings"
	"web-search/internal/research/agentmanager"
)

// FindingsService is the subset of the findings application surface the
// research domain depends on: capture (Add), gather (semantic + nearby reads),
// and reconcile (Supersede / Flag). Both internalfindings.Service and a test
// fake satisfy it.
type FindingsService interface {
	Add(ctx context.Context, in findings.NewFinding) (findings.Finding, error)
	Supersede(ctx context.Context, id, replacement, reason string) (findings.Finding, error)
	Flag(ctx context.Context, id, reason string) (findings.Finding, error)
}

// FindingsGatherer is the bounded semantic-gather seam: given a query it returns
// the findings semantically NEAR it, already capped by the caller's limit. The
// production impl wraps the findings semantic index + store; a test fake
// satisfies it. A nil Gatherer makes GatherRelatedFindings unavailable.
type FindingsGatherer interface {
	Gather(ctx context.Context, query string, limit int) ([]GatheredFinding, error)
}

// Deps wires the research service's seams. All are optional: a nil Searcher /
// Fetcher / Synthesizer makes L2 abstain (graceful when the fetch stack/ollama are
// down); a nil AgentManager makes L3 unavailable; a nil Findings disables
// capture. main.go constructs the production impls from env; tests inject fakes.
type Deps struct {
	EvidenceStore FindingsStore
	ReceiptStore  evidence.Repository
	Live          LiveSearch
	Now           func() time.Time
	Searcher      Searcher
	Fetcher       Fetcher
	Synthesizer   Synthesizer
	Findings      FindingsService
	Gatherer      FindingsGatherer
	AgentManager  agentmanager.Service
	Logger        *log.Logger

	// Excerpter selects what part of each fetched page the synthesis model
	// reads. Nil falls back to positional truncation (the legacy behavior);
	// main.go wires the relevance-aware excerpter unless the escape hatch
	// lever turns it off.
	Excerpter Excerpter

	// ConfidenceGate overrides HighConfidenceThreshold for the L3 reconcile
	// confidence gate. Zero (or any value outside (0,1]) keeps the default.
	ConfidenceGate float64
	// GatherCap overrides MaxGatherFindings, the hard cap on the bounded
	// GATHER sweep. Zero or negative keeps the default.
	GatherCap int
	// MaxResearchLoops overrides DefaultMaxResearchLoops, the iteration
	// budget written into the L3 task contract. Zero or negative keeps the
	// default.
	MaxResearchLoops int
	FetchConcurrency int
	// FetchPerHostConcurrency bounds simultaneous fetches to one hostname.
	// Zero keeps the conservative default of one.
	FetchPerHostConcurrency int
	AttemptOutbox           *capture.Repository
	OutcomeMetrics          *OutcomeMetrics
}

// Service orchestrates the L2 (synchronous fetch/read/synthesize) and L3
// (agent-manager run) research paths over its injected seams.
type Service struct {
	evidenceStore           FindingsStore
	receiptStore            evidence.Repository
	live                    LiveSearch
	now                     func() time.Time
	searcher                Searcher
	fetcher                 Fetcher
	synthesizer             Synthesizer
	excerpter               Excerpter
	findings                FindingsService
	gatherer                FindingsGatherer
	agentManager            agentmanager.Service
	logger                  *log.Logger
	confidenceGate          float64
	gatherCap               int
	maxLoops                int
	fetchConcurrency        int
	fetchPerHostConcurrency int
	attemptOutbox           *capture.Repository
	outcomeMetrics          *OutcomeMetrics
}

// NewService constructs the research service.
func NewService(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = log.Default()
	}
	gate := d.ConfidenceGate
	if math.IsNaN(gate) || gate <= 0 || gate > 1 {
		gate = HighConfidenceThreshold
	}
	gatherCap := d.GatherCap
	if gatherCap <= 0 || gatherCap > MaxGatherFindings {
		gatherCap = MaxGatherFindings
	}
	maxLoops := d.MaxResearchLoops
	if maxLoops <= 0 || maxLoops > DefaultMaxResearchLoops {
		maxLoops = DefaultMaxResearchLoops
	}
	fetchConcurrency := d.FetchConcurrency
	if fetchConcurrency <= 0 || fetchConcurrency > MaxTopN {
		fetchConcurrency = 4
	}
	fetchPerHostConcurrency := d.FetchPerHostConcurrency
	if fetchPerHostConcurrency <= 0 || fetchPerHostConcurrency > MaxTopN {
		fetchPerHostConcurrency = 1
	}
	excerpter := d.Excerpter
	if excerpter == nil {
		excerpter = PositionalExcerpter{}
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Service{
		evidenceStore: d.EvidenceStore, live: d.Live, now: d.Now,
		receiptStore:   d.ReceiptStore,
		searcher:       d.Searcher,
		fetcher:        d.Fetcher,
		synthesizer:    d.Synthesizer,
		excerpter:      excerpter,
		findings:       d.Findings,
		gatherer:       d.Gatherer,
		agentManager:   d.AgentManager,
		logger:         logger,
		confidenceGate: gate,
		gatherCap:      gatherCap,
		maxLoops:       maxLoops, fetchConcurrency: fetchConcurrency,
		fetchPerHostConcurrency: fetchPerHostConcurrency,
		attemptOutbox:           d.AttemptOutbox,
		outcomeMetrics:          d.OutcomeMetrics,
	}
}

// RunL2 runs the synchronous L2 fetch -> read -> cited synthesis pipeline.
// Untyped requests use one synthesis pass; declared multipart requests use one
// bounded pass per question. When the seams needed for L2 are not wired it
// abstains gracefully rather than erroring.
func (s *Service) RunL2(ctx context.Context, query string, topN int, capture bool) (L2Outcome, error) {
	return s.RunL2WithContract(ctx, query, topN, capture, nil)
}

func (s *Service) RunL2WithContract(ctx context.Context, query string, topN int, capture bool, questions []ResearchQuestion) (L2Outcome, error) {
	return s.RunL2WithContractAndParent(ctx, query, topN, capture, questions, "")
}

// RunL2WithContractAndParent runs L2 and records optional parent execution
// lineage on the durable attempt capture. Direct L2 callers leave parentRunID
// empty; delegated L3 child calls provide their Agent Manager run identity.
func (s *Service) RunL2WithContractAndParent(ctx context.Context, query string, topN int, capture bool, questions []ResearchQuestion, parentRunID string) (L2Outcome, error) {
	return s.runL2Attempt(ctx, query, topN, capture, questions, parentRunID, 0)
}

// RunL2WithPolicyAndParent applies the caller's evidence byte budget to the
// actual retained documents and model input. It is used by Answer after the
// shared policy has been validated.
func (s *Service) RunL2WithPolicyAndParent(ctx context.Context, policy EvidencePolicy, parentRunID string) (L2Outcome, error) {
	if err := policy.Validate(); err != nil {
		return L2Outcome{}, err
	}
	if policy.Effort != "l2" {
		return L2Outcome{}, fmt.Errorf("L2 policy effort must be l2")
	}
	return s.runL2Attempt(ctx, policy.Query, policy.TopN, policy.Capture, policy.Questions, parentRunID, policy.MaxEvidenceBytes)
}

func (s *Service) runL2Attempt(ctx context.Context, query string, topN int, capture bool, questions []ResearchQuestion, parentRunID string, maxEvidenceBytes int) (L2Outcome, error) {
	started := s.now()
	if s.searcher == nil || s.fetcher == nil || s.synthesizer == nil {
		out := L2Outcome{
			Brief:     Brief{Query: query, Level: LevelL2, Summary: abstainNote},
			Abstained: true,
		}
		s.captureAttempt(ctx, query, out, parentRunID)
		s.recordOutcomeMetric(started, out, questions)
		return out, nil
	}
	out, err := s.runL2(ctx, query, topN, capture, questions, maxEvidenceBytes)
	if err != nil {
		s.captureAttempt(ctx, query, out, parentRunID)
		s.recordOutcomeMetric(started, out, questions)
		return out, err
	}
	s.persistAssessments(ctx, out.Assessments)
	s.captureAttempt(ctx, query, out, parentRunID)
	s.recordOutcomeMetric(started, out, questions)
	return out, nil
}

func (s *Service) recordOutcomeMetric(started time.Time, out L2Outcome, questions []ResearchQuestion) {
	if s.outcomeMetrics == nil {
		return
	}
	supportedClaims, assessedClaims := 0, 0
	for _, assessment := range out.Assessments {
		assessedClaims++
		if assessment.Disposition == AssessmentSupported {
			supportedClaims++
		}
	}
	supportedQuestions, requiredQuestions := 0, 0
	for _, question := range questions {
		if question.Required {
			requiredQuestions++
		}
	}
	for _, coverage := range out.Coverage {
		if coverage.Status == "supported" {
			supportedQuestions++
		}
	}
	s.outcomeMetrics.Record(OutcomeMetric{
		At: started, SupportedClaims: supportedClaims, AssessedClaims: assessedClaims,
		SupportedQuestions: supportedQuestions, RequiredQuestions: requiredQuestions,
		FetchSuccesses: len(out.Excerpts), FetchAttempts: len(out.Excerpts) + len(out.FetchFailures),
		Calls: len(out.EvidenceReceiptIDs) + len(out.FetchFailures), Bytes: excerptBytes(out.Excerpts),
		Duration: s.now().Sub(started),
	})
}

func excerptBytes(excerpts []DocumentExcerpt) int {
	total := 0
	for _, excerpt := range excerpts {
		total += len([]byte(excerpt.Excerpt))
	}
	return total
}

func (s *Service) persistAssessments(ctx context.Context, assessments []ClaimAssessment) {
	writer, ok := s.receiptStore.(evidence.AssessmentWriter)
	if !ok {
		return
	}
	for i := range assessments {
		assessment := &assessments[i]
		refs, err := json.Marshal(assessment.Evidence)
		if err != nil {
			s.logger.Printf("research: marshal assessment %q: %v", assessment.ClaimID, err)
			continue
		}
		stored, err := writer.CreateAssessment(ctx, evidence.Assessment{ClaimID: assessment.ClaimID, Disposition: string(assessment.Disposition), PolicyRevision: assessment.PolicyRevision, Reason: assessment.Reason, EvidenceJSON: string(refs)})
		if err != nil {
			s.logger.Printf("research: persist assessment %q: %v", assessment.ClaimID, err)
			continue
		}
		assessment.AssessmentID = stored.AssessmentID
	}
}

func (s *Service) captureAttempt(ctx context.Context, query string, out L2Outcome, parentRunID ...string) {
	if s.attemptOutbox == nil {
		return
	}
	attemptID := uuid.NewString()
	parent := ""
	if len(parentRunID) > 0 {
		parent = strings.TrimSpace(parentRunID[0])
	}
	payload, err := json.Marshal(map[string]any{
		"attempt_id": attemptID, "task_id": "research:" + query, "operation": "research.l2",
		"context_key": "research-assurance-v1", "outcome": captureOutcome(out),
		"failure_fingerprint": captureFailure(out), "started_at": s.now().UTC().Format(time.RFC3339Nano),
		"finished_at": s.now().UTC().Format(time.RFC3339Nano), "task_started_at": s.now().UTC().Format(time.RFC3339Nano),
		"attempt_number": 1, "recall_status": "no_match", "provenance": "operator",
		"trigger": "research request", "approach": "bounded L2 evidence collection",
		"observation_id": attemptID + "-quality", "disposition": captureDisposition(out),
		"method_revision": "research-assurance-v1", "observed_at": s.now().UTC().Format(time.RFC3339Nano),
		"abstained": out.Abstained, "reason": out.AbstainReason,
		"evidence_refs": out.EvidenceReceiptIDs, "fetch_failures": out.FetchFailures,
		"parent_run_id": parent,
	})
	if err != nil {
		s.logger.Printf("[web-search] capture marshal failed: %v", err)
		return
	}
	if _, _, err := s.attemptOutbox.Enqueue(ctx, attemptID, "research:"+query, "research-attempt:"+attemptID, payload); err != nil {
		s.logger.Printf("[web-search] capture enqueue failed: %v", err)
	}
}

func (s *Service) captureL3Admission(ctx context.Context, query string, result agentmanager.RunResult) {
	if s.attemptOutbox == nil {
		return
	}
	attemptID := uuid.NewString()
	now := s.now().UTC().Format(time.RFC3339Nano)
	payload, err := json.Marshal(map[string]any{
		"attempt_id": attemptID, "task_id": "research:" + query, "operation": "research.l3",
		"context_key": "research-assurance-v1", "outcome": "unknown",
		"started_at": now, "finished_at": now, "task_started_at": now,
		"attempt_number": 1, "recall_status": "no_match", "provenance": "operator",
		"trigger": "research investigation", "approach": "bounded L3 delegated investigation",
		"observation_id": attemptID + "-quality", "disposition": "unknown",
		"method_revision": "research-assurance-v1", "observed_at": now,
		"run_id": result.RunID,
	})
	if err != nil {
		s.logger.Printf("[web-search] L3 capture marshal failed: %v", err)
		return
	}
	if _, _, err := s.attemptOutbox.Enqueue(ctx, attemptID, "research:"+query, "research-attempt:"+attemptID, payload); err != nil {
		s.logger.Printf("[web-search] L3 capture enqueue failed: %v", err)
	}
}

func captureOutcome(out L2Outcome) string {
	if out.Abstained || len(out.EvidenceReceiptIDs) == 0 || len(out.FetchFailures) > 0 {
		return "unknown"
	}
	return "verified_success"
}

func captureDisposition(out L2Outcome) string {
	if out.Abstained || len(out.EvidenceReceiptIDs) == 0 || len(out.FetchFailures) > 0 {
		return "unknown"
	}
	return "supported"
}
func captureFailure(out L2Outcome) string {
	if len(out.FetchFailures) > 0 {
		return "fetch_failure"
	}
	return ""
}

// CaptureStatus reports durable attempt delivery state without exposing
// payloads or transport details to callers.
func (s *Service) CaptureStatus(ctx context.Context) (capture.Counts, error) {
	if s.attemptOutbox == nil {
		return capture.Counts{}, fmt.Errorf("capture outbox is not configured")
	}
	return s.attemptOutbox.Counts(ctx)
}

func (s *Service) EvidenceReceipt(ctx context.Context, id string) (evidence.Receipt, error) {
	if s.receiptStore == nil {
		return evidence.Receipt{}, fmt.Errorf("evidence store is unavailable")
	}
	return s.receiptStore.GetReceipt(ctx, id)
}

func (s *Service) EvidencePassage(ctx context.Context, id string) (evidence.Passage, error) {
	if s.receiptStore == nil {
		return evidence.Passage{}, fmt.Errorf("evidence store is unavailable")
	}
	return s.receiptStore.GetPassage(ctx, id)
}

func (s *Service) EvidenceAssessment(ctx context.Context, id string) (evidence.Assessment, error) {
	reader, ok := s.receiptStore.(evidence.AssessmentReader)
	if !ok {
		return evidence.Assessment{}, fmt.Errorf("evidence assessment store is unavailable")
	}
	return reader.GetAssessment(ctx, id)
}
