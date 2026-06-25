package initiativereview

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/pathredact"
	"swarm-manager/internal/review"
	"swarm-manager/internal/storage"
)

// Decide flips the initiative from review_pending to a terminal status per
// the user's verdict, appending an audit record under review/decisions/.
// Only review_pending initiatives can be decided — any other state is an
// explicit error so double-decides and stale CLI calls can't mutate terminal
// state.
func (s *Service) Decide(ctx context.Context, initiativeName string, verdict Verdict, rationale, decidedBy string) (DecideResponse, error) {
	init, err := s.initStore.Load(initiativeName)
	if err != nil {
		return DecideResponse{}, err
	}
	if init.Status != initiatives.InitiativeStatusReviewPending {
		return DecideResponse{}, fmt.Errorf("initiative status is %q; decide requires %q", init.Status, initiatives.InitiativeStatusReviewPending)
	}
	target := verdict.TargetStatus()
	if target == "" {
		return DecideResponse{}, fmt.Errorf("invalid verdict %q", verdict)
	}

	priorStatus := init.Status
	decidedAt := s.clock().UTC().Format(time.RFC3339)

	init.Status = target
	init.Updated = decidedAt
	if err := s.initStore.Save(init); err != nil {
		return DecideResponse{}, fmt.Errorf("save initiative: %w", err)
	}

	latest, _, _ := review.LoadLatestRound(s.initStore.InitDir(initiativeName))
	latestRound := 0
	if latest != nil {
		latestRound = latest.RoundNum
	}

	// Decision record is supplementary audit; failure to persist it logs
	// but does not roll back the status flip (mirrors backlog review_decide).
	if writeErr := s.writeDecision(initiativeName, DecisionRecord{
		Verdict:     string(verdict),
		Status:      target,
		Rationale:   strings.TrimSpace(rationale),
		DecidedBy:   strings.TrimSpace(decidedBy),
		DecidedAt:   decidedAt,
		PriorStatus: priorStatus,
		Round:       latestRound,
	}); writeErr != nil {
		slog.Warn("initiative review: persist decision record", "initiative", initiativeName, "err", writeErr)
	}

	return DecideResponse{
		Initiative: initiativeName,
		Verdict:    string(verdict),
		Status:     target,
		Rationale:  strings.TrimSpace(rationale),
		DecidedAt:  decidedAt,
	}, nil
}

// writeDecision persists a DecisionRecord under review/decisions/.
// Timestamp+verdict in the filename preserves chronological sort and lets
// operators grep the audit log without parsing JSON.
func (s *Service) writeDecision(initiativeName string, rec DecisionRecord) error {
	dir := filepath.Join(s.initStore.InitDir(initiativeName), "review", "decisions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create decisions dir: %w", err)
	}
	safeTS := strings.ReplaceAll(strings.ReplaceAll(rec.DecidedAt, ":", ""), "-", "")
	filename := fmt.Sprintf("%s-%s.json", safeTS, rec.Verdict)
	path := filepath.Join(dir, filename)
	value := any(rec)
	if redacted, changed, err := pathredact.NewForArtifactPath(path).RedactJSONValue(rec); err != nil {
		return fmt.Errorf("redact decision: %w", err)
	} else if changed {
		value = redacted
	}
	return storage.WriteJSONAtomic(path, value)
}

// ListDecisions returns all decision records for an initiative, newest-first,
// for audit/CLI consumers. Missing directory → empty slice, not an error.
func (s *Service) ListDecisions(initiativeName string) ([]DecisionRecord, error) {
	dir := filepath.Join(s.initStore.InitDir(initiativeName), "review", "decisions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read decisions dir: %w", err)
	}
	out := make([]DecisionRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var rec DecisionRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DecidedAt > out[j].DecidedAt })
	return out, nil
}
