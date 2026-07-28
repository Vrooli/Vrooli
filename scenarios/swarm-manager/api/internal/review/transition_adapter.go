package review

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"swarm-manager/internal/transitionrunner"

	"google.golang.org/protobuf/types/known/structpb"
)

// RegisterTransitionAdapter contributes the review-owned snapshots and ledger
// projections. It deliberately contains no workflow start or collection code.
func (s *Service) RegisterTransitionAdapter(registrar transitionrunner.Registrar) {
	registrar.RegisterInput("work.review", s.buildReviewInput)
	registrar.RegisterInput("review.evidence_request", s.buildEvidenceRequestInput)
	registrar.RegisterApply("apply_review_outcome", s.applyReviewOutcome)
	registrar.RegisterApply("apply_review_evidence_request", s.applyEvidenceRequestOutcome)
}

func reviewSubject(kind, name string, round int) string {
	return kind + "/" + name + "/" + strconv.Itoa(round)
}

func reviewThreadSubject(kind, name string, round int, thread string) string {
	return reviewSubject(kind, name, round) + "/" + thread
}

func parseReviewSubject(subject string, thread bool) (string, string, int, string, error) {
	parts := strings.Split(strings.TrimSpace(subject), "/")
	want := 3
	if thread {
		want = 4
	}
	if len(parts) != want || parts[0] == "" || parts[1] == "" || (thread && parts[3] == "") {
		return "", "", 0, "", fmt.Errorf("invalid review subject reference %q", subject)
	}
	round, err := strconv.Atoi(parts[2])
	if err != nil || round < 1 {
		return "", "", 0, "", fmt.Errorf("invalid review round in subject reference %q", subject)
	}
	threadID := ""
	if thread {
		threadID = parts[3]
	}
	return parts[0], parts[1], round, threadID, nil
}

// buildReviewInput reprojects one review round from its persisted request
// snapshot. The snapshot is stored rather than recomputed because it carries
// GCT results and baseline diffs captured at start; re-gathering them would
// produce a different version on every rebuild and reject every apply.
func (s *Service) buildReviewInput(_ context.Context, subject string) (transitionrunner.Snapshot, error) {
	kind, name, roundNum, _, err := parseReviewSubject(subject, false)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	round, err := LoadRound(s.resolveItemDir(kind, name), roundNum)
	if err != nil || round == nil {
		return transitionrunner.Snapshot{}, fmt.Errorf("load review round: %w", err)
	}
	if len(round.AgentWorkflowSnapshot) == 0 {
		return transitionrunner.Snapshot{}, fmt.Errorf("review round %d has no persisted request snapshot", roundNum)
	}
	return reviewRoundSnapshot(kind, name, round.ExecutionID, round.AgentWorkflowSnapshot)
}

// reviewRoundSnapshot is the single projection used by both the start and the
// apply-time rebuild, so the two can never drift.
func reviewRoundSnapshot(kind, name, executionID string, raw json.RawMessage) (transitionrunner.Snapshot, error) {
	var snapshot map[string]any
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return transitionrunner.Snapshot{}, fmt.Errorf("decode independent review snapshot: %w", err)
	}
	version := snapshotVersion(raw)
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": kind, "name": name, "executionId": executionID, "version": version}, "snapshot": snapshot})
	if err != nil {
		return transitionrunner.Snapshot{}, fmt.Errorf("build independent review input: %w", err)
	}
	return transitionrunner.Snapshot{Input: input, EntityVersion: version}, nil
}

// buildEvidenceRequestInput reprojects one evidence request. Every field it
// needs already lives on the persisted thread, so no extra state is stored.
func (s *Service) buildEvidenceRequestInput(_ context.Context, subject string) (transitionrunner.Snapshot, error) {
	kind, name, roundNum, threadID, err := parseReviewSubject(subject, true)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	round, err := LoadRound(s.resolveItemDir(kind, name), roundNum)
	if err != nil || round == nil {
		return transitionrunner.Snapshot{}, fmt.Errorf("load review round: %w", err)
	}
	for _, thread := range round.RequestThreads {
		if thread.ID != threadID {
			continue
		}
		request := ""
		if len(thread.Messages) > 0 {
			request = thread.Messages[0].Content
		}
		return evidenceRequestSnapshot(kind, name, round.ExecutionID, roundNum, threadID, request, thread.EvidenceID)
	}
	return transitionrunner.Snapshot{}, fmt.Errorf("review thread %q not found", threadID)
}

// evidenceRequestSnapshot is shared by the start path and the apply-time
// rebuild so both derive the same version from the same durable thread fields.
func evidenceRequestSnapshot(kind, name, executionID string, roundNum int, threadID, request, evidenceID string) (transitionrunner.Snapshot, error) {
	snapshot := map[string]any{"round": roundNum, "thread": threadID, "request": request, "evidenceId": evidenceID}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return transitionrunner.Snapshot{}, fmt.Errorf("encode evidence request snapshot: %w", err)
	}
	digest := sha256.Sum256(append([]byte(threadID+"\x00"), raw...))
	version := "sha256:" + hex.EncodeToString(digest[:])
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": kind, "name": name, "executionId": executionID + "/" + threadID, "version": version}, "snapshot": snapshot})
	if err != nil {
		return transitionrunner.Snapshot{}, fmt.Errorf("build evidence request input: %w", err)
	}
	return transitionrunner.Snapshot{Input: input, EntityVersion: version}, nil
}

func (s *Service) startReviewTransition(ctx context.Context, params startReviewParams) error {
	roundNum, err := NextRoundNumber(params.ItemDir)
	if err != nil {
		return fmt.Errorf("determine next round: %w", err)
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("encode independent review snapshot: %w", err)
	}
	version := snapshotVersion(encoded)
	// Persist the request snapshot with the round before starting. The runner's
	// registered builder reads it back to construct the input, so the round has
	// to be on disk first.
	round := Round{RoundNum: roundNum, GeneratedAt: time.Now().UTC().Format(time.RFC3339), ExecutionID: params.ExecutionID, Status: RoundStatusGathering, Evidence: []EvidenceItem{}, AgentWorkflowVersion: version, AgentWorkflowSnapshot: encoded}
	if err := SaveRound(params.ItemDir, round); err != nil {
		return fmt.Errorf("save review round: %w", err)
	}
	started, err := s.transitionRunner.StartWith(ctx, "work.review", reviewSubject(params.BacklogKind, params.BacklogName, roundNum), transitionrunner.PreparedInput{FirstRunNodeID: "review", Activity: &transitionrunner.Activity{OwnerType: "backlog", OwnerKind: params.BacklogKind, OwnerName: params.BacklogName, OwnerTitle: params.ItemTitle, Purpose: "review"}})
	if err != nil {
		return fmt.Errorf("start independent review transition: %w", err)
	}
	round.AgentWorkflowExecutionID, round.AgentWorkflowDefinition = started.ExecutionID, started.DefinitionDigest
	if len(started.Attempts) > 0 {
		round.RunID = started.Attempts[0].RunID
	}
	if err := SaveRound(params.ItemDir, round); err != nil {
		return fmt.Errorf("save review round run association: %w", err)
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitReviewStarted(params.ExecutionID, roundNum)
	}
	return nil
}

func snapshotVersion(raw []byte) string { // stable snapshot version keeps runner idempotency exact.
	// The existing workflow path uses SHA-256; preserve its observable version format.
	digest := sha256.Sum256(raw)
	return fmt.Sprintf("sha256:%x", digest)
}

func (s *Service) applyReviewOutcome(ctx context.Context, subject string, outcome transitionrunner.Outcome) error {
	kind, name, roundNum, _, err := parseReviewSubject(subject, false)
	if err != nil {
		return err
	}
	itemDir := s.resolveItemDir(kind, name)
	round, err := LoadRound(itemDir, roundNum)
	if err != nil || round == nil {
		return fmt.Errorf("load review round: %w", err)
	}
	FinalizeRoundFromResult(round, outcome.Result, outcome.Name)
	if err := SaveRound(itemDir, *round); err != nil {
		return err
	}
	if s.onRoundTerminal != nil {
		s.onRoundTerminal(ctx, kind, name, *round)
	}
	s.notifyRoundTerminal(ctx, kind, name, *round)
	return nil
}

func (s *Service) applyEvidenceRequestOutcome(_ context.Context, subject string, outcome transitionrunner.Outcome) error {
	kind, name, roundNum, threadID, err := parseReviewSubject(subject, true)
	if err != nil {
		return err
	}
	itemDir := s.resolveItemDir(kind, name)
	round, err := LoadRound(itemDir, roundNum)
	if err != nil || round == nil {
		return fmt.Errorf("load review round: %w", err)
	}
	for i := range round.RequestThreads {
		thread := &round.RequestThreads[i]
		if thread.ID == threadID {
			var result struct {
				Summary  string         `json:"summary"`
				Evidence []EvidenceItem `json:"evidence"`
			}
			_ = json.Unmarshal(outcome.Result, &result)
			if strings.TrimSpace(result.Summary) == "" {
				result.Summary = firstNonEmptyString(outcome.TerminalCode, "Evidence request did not complete.")
			}
			round.Evidence = append(round.Evidence, result.Evidence...)
			ids := make([]string, 0, len(result.Evidence))
			for _, evidence := range result.Evidence {
				if evidence.ID != "" {
					ids = append(ids, evidence.ID)
				}
			}
			thread.Messages = append(thread.Messages, RequestMessage{Role: "assistant", Content: result.Summary, Timestamp: s.now().Format(time.RFC3339), AddedEvidenceIDs: ids})
			if outcome.Name == "fulfilled" {
				thread.Status = "fulfilled"
			}
			return SaveRound(itemDir, *round)
		}
	}
	return fmt.Errorf("review thread %q not found", threadID)
}
