package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/review"
)

const ReviewEvidenceMigrationKey = "review-evidence/v1"

// ReviewRoundSource is one decoded legacy review projection. Directory
// traversal intentionally stays outside the evidence package so imports can
// be bounded to an explicit fixture or operator-selected source set.
type ReviewRoundSource struct {
	Kind  string
	Name  string
	Round review.Round
}

// ImportReviewRounds projects a bounded set of legacy review rounds and
// persists the source/projection parity audit that permits ledger reads. It is
// deterministic and idempotent, making it suitable for a one-shot command and
// for fixture validation without touching a live backlog store.
func (l *Ledger) ImportReviewRounds(ctx context.Context, sources []ReviewRoundSource) (MigrationAudit, error) {
	ordered := append([]ReviewRoundSource(nil), sources...)
	sort.Slice(ordered, func(i, j int) bool {
		left := fmt.Sprintf("%s/%s/%09d", ordered[i].Kind, ordered[i].Name, ordered[i].Round.RoundNum)
		right := fmt.Sprintf("%s/%s/%09d", ordered[j].Kind, ordered[j].Name, ordered[j].Round.RoundNum)
		return left < right
	})
	sourceHash, projectionHash := sha256.New(), sha256.New()
	sourceCount, projectionCount := 0, 0
	for _, source := range ordered {
		if strings.TrimSpace(source.Kind) == "" || strings.TrimSpace(source.Name) == "" || source.Round.RoundNum <= 0 {
			return MigrationAudit{}, fmt.Errorf("review evidence import requires kind, name, and positive round number")
		}
		for _, item := range source.Round.Evidence {
			_, _ = sourceHash.Write([]byte(source.Kind + "\x00" + source.Name + "\x00" + fmt.Sprintf("%d", source.Round.RoundNum) + "\x00" + item.ID + "\x00" + item.CriterionID + "\x00" + item.Settlement + "\n"))
			_, _ = projectionHash.Write([]byte(fmt.Sprintf("backlog/%s/%s/work.review/%d/%s\n", source.Kind, source.Name, source.Round.RoundNum, item.ID)))
			sourceCount++
		}
		if err := l.ImportReviewRound(ctx, source.Kind, source.Name, source.Round); err != nil {
			return MigrationAudit{}, err
		}
		count, err := l.countImportedReviewEvidence(ctx, source.Kind, source.Name, source.Round.RoundNum)
		if err != nil {
			return MigrationAudit{}, err
		}
		projectionCount += count
	}
	audit := MigrationAudit{Key: ReviewEvidenceMigrationKey, SourceDigest: hex.EncodeToString(sourceHash.Sum(nil)), ProjectionDigest: hex.EncodeToString(projectionHash.Sum(nil)), SourceCount: sourceCount, ProjectionCount: projectionCount}
	if err := l.RecordMigrationAudit(ctx, audit); err != nil {
		return MigrationAudit{}, err
	}
	return audit, nil
}

func (l *Ledger) countImportedReviewEvidence(ctx context.Context, kind, name string, roundNum int) (int, error) {
	if l == nil || l.db == nil {
		return 0, nil
	}
	attemptRef := fmt.Sprintf("backlog/%s/%s/work.review/%d", kind, name, roundNum)
	var count int
	err := l.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM evidence_links JOIN evidence_observations ON evidence_observations.id = evidence_links.observation_id WHERE attempt_ref = ? AND producer = 'swarm-review' AND source_system = 'work.review' AND action IN ('settled', 'refuted', 'unavailable')`, attemptRef).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count imported review evidence: %w", err)
	}
	return count, nil
}

// ImportReviewRound replays a legacy review projection into the canonical
// ledger. It is bounded to one already-decoded round; callers own directory
// traversal and parity auditing. Replaying is safe because ledger IDs are
// deterministic and Record is idempotent.
func (l *Ledger) ImportReviewRound(ctx context.Context, kind, name string, round review.Round) error {
	attemptRef := fmt.Sprintf("backlog/%s/%s/work.review/%d", kind, name, round.RoundNum)
	runID := review.ExecutionIDFromSnapshot(round.AgentWorkflowSnapshot)
	if runID == "" {
		runID = "legacy:" + attemptRef
	}
	observedAt := time.Now().UTC()
	if parsed, err := time.Parse(time.RFC3339, round.GeneratedAt); err == nil {
		observedAt = parsed
	}
	for _, item := range round.Evidence {
		criterionID := strings.TrimSpace(item.CriterionID)
		if criterionID == "" {
			criterionID = "legacy-unbound:" + item.ID
		}
		subjectID := fmt.Sprintf("%s/%s/%s", kind, name, criterionID)
		if err := l.Record(ctx, Observation{ID: attemptRef + "/" + item.ID, Producer: "swarm-review", SourceSystem: "work.review", RunID: runID, SubjectKind: "criterion", SubjectID: subjectID, Action: item.Settlement, Confidence: "reported", Title: item.Title, Description: item.Description, ObservedAt: observedAt}, attemptRef); err != nil {
			return err
		}
		if !item.Verified {
			continue
		}
		verifiedAt := observedAt
		if parsed, err := time.Parse(time.RFC3339, item.VerifiedAt); err == nil {
			verifiedAt = parsed
		}
		if err := l.Record(ctx, Observation{ID: attemptRef + "/" + item.ID + "/operator_verified", Producer: "swarm-review", SourceSystem: "work.review", RunID: runID, SubjectKind: "criterion", SubjectID: subjectID, Action: "operator_verified", Confidence: "operator_verified", Title: item.Title, Description: item.Description, Actor: "unknown-legacy", Reason: "Imported legacy review verification flag.", ObservedAt: verifiedAt}, attemptRef); err != nil {
			return err
		}
	}
	return nil
}
