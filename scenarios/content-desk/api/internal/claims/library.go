package claims

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	KindQuantitative     = "quantitative"
	KindExistence        = "existence"
	KindStatus           = "status"
	KindCapability       = "capability"
	KindNovelty          = "novelty"
	EvidenceKindCitation = "citation"
	EvidenceKindCheck    = "check"
	StateAsserted        = "asserted"
	StateVerified        = "verified"
	StateStale           = "stale"
)

var (
	ErrCheckRequired = errors.New("claim kind requires a re-runnable check")
	ErrInvalidAnchor = errors.New("citation anchor is outside draft body")
)

type (
	Claim    struct{ ID, Statement, Kind, VerificationStatus string }
	Evidence struct {
		Kind, Reference, Command, ExpectedResult, LastResult string
		LastRunAt                                            time.Time
		ObservedAt                                           time.Time
	}
	Citation struct {
		DraftID, ClaimID string
		Start, End       int
	}
	TextSpan struct {
		Start, End int
		ClaimID    string
	}
	Proposal struct {
		ID, DraftID, Statement, Status string
		CreatedAt, DecidedAt            time.Time
	}
)

type (
	SQLExecutor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	Library interface {
		Create(context.Context, Claim, Evidence) (Claim, error)
		List(context.Context) ([]Claim, error)
		ListForDraft(context.Context, string) ([]Claim, error)
		Cite(context.Context, Citation, string) error
		CitingDrafts(context.Context, string) ([]string, error)
		Verify(context.Context, string) (Claim, error)
		Sweep(context.Context) ([]Claim, error)
		ExpireNovelty(context.Context, time.Time, time.Duration) ([]Claim, error)
		Coverage(context.Context, string, string) ([]TextSpan, []TextSpan, error)
		ExtractProposals(context.Context, string, string) ([]Proposal, error)
		ListProposals(context.Context, string) ([]Proposal, error)
		DecideProposal(context.Context, string, string) (Proposal, error)
	}
	library struct {
		db     SQLExecutor
		runner Runner
	}
)

// ExtractProposals creates review-only proposals from declarative sentences.
// It is intentionally deterministic and local: an unavailable assistant must
// leave the draft workable, and extraction cannot satisfy the evidence gate.
func (l *library) ExtractProposals(ctx context.Context, draftID, body string) ([]Proposal, error) {
	if draftID == "" { return nil, fmt.Errorf("draft id is required") }
	for _, statement := range extractionCandidates(body) {
		_, err := l.db.ExecContext(ctx, `INSERT INTO claim_proposals (id, draft_id, statement, status, created_at) VALUES (?, ?, ?, 'proposed', ?) ON CONFLICT(draft_id, statement) DO NOTHING`, uuid.NewString(), draftID, statement, time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil { return nil, err }
	}
	return l.ListProposals(ctx, draftID)
}

func extractionCandidates(body string) []string {
	var out []string
	start := 0
	for i, r := range body {
		if r != '.' && r != '!' && r != '?' { continue }
		candidate := strings.TrimSpace(body[start:i+1])
		start = i + 1
		if len(candidate) >= 12 { out = append(out, candidate) }
	}
	if tail := strings.TrimSpace(body[start:]); len(tail) >= 12 { out = append(out, tail) }
	return out
}

func (l *library) ListProposals(ctx context.Context, draftID string) ([]Proposal, error) {
	rows, err := l.db.QueryContext(ctx, `SELECT id, draft_id, statement, status, created_at, COALESCE(decided_at,'') FROM claim_proposals WHERE draft_id = ? ORDER BY created_at, id`, draftID)
	if err != nil { return nil, err }
	defer rows.Close()
	var proposals []Proposal
	for rows.Next() {
		var proposal Proposal; var createdAt, decidedAt string
		if err := rows.Scan(&proposal.ID, &proposal.DraftID, &proposal.Statement, &proposal.Status, &createdAt, &decidedAt); err != nil { return nil, err }
		proposal.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); if err != nil { return nil, err }
		if decidedAt != "" { proposal.DecidedAt, err = time.Parse(time.RFC3339Nano, decidedAt); if err != nil { return nil, err } }
		proposals = append(proposals, proposal)
	}
	return proposals, rows.Err()
}

func (l *library) DecideProposal(ctx context.Context, id, status string) (Proposal, error) {
	if status != "accepted" && status != "rejected" { return Proposal{}, fmt.Errorf("proposal status %q is invalid", status) }
	now := time.Now().UTC()
	result, err := l.db.ExecContext(ctx, `UPDATE claim_proposals SET status = ?, decided_at = ? WHERE id = ? AND status = 'proposed'`, status, now.Format(time.RFC3339Nano), id)
	if err != nil { return Proposal{}, err }; affected, err := result.RowsAffected(); if err != nil { return Proposal{}, err }; if affected != 1 { return Proposal{}, fmt.Errorf("proposal %q is not pending", id) }
	row := l.db.QueryRowContext(ctx, `SELECT id, draft_id, statement, status, created_at, decided_at FROM claim_proposals WHERE id = ?`, id)
	var proposal Proposal; var createdAt, decidedAt string
	if err = row.Scan(&proposal.ID, &proposal.DraftID, &proposal.Statement, &proposal.Status, &createdAt, &decidedAt); err != nil { return Proposal{}, err }
	proposal.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); if err != nil { return Proposal{}, err }; proposal.DecidedAt, err = time.Parse(time.RFC3339Nano, decidedAt)
	return proposal, err
}

func NewLibrary(db SQLExecutor, runner Runner) Library { return &library{db: db, runner: runner} }

func (l *library) Create(ctx context.Context, claim Claim, evidence Evidence) (Claim, error) {
	if claim.ID == "" {
		claim.ID = uuid.NewString()
	}
	if claim.VerificationStatus == "" {
		claim.VerificationStatus = StateAsserted
	}
	if requiresCheck(claim.Kind) && evidence.Kind != EvidenceKindCheck {
		return Claim{}, ErrCheckRequired
	}
	if evidence.Kind != EvidenceKindCitation && evidence.Kind != EvidenceKindCheck {
		return Claim{}, fmt.Errorf("unsupported evidence kind %q", evidence.Kind)
	}
	if claim.Kind == KindNovelty && evidence.ObservedAt.IsZero() {
		return Claim{}, fmt.Errorf("novelty claim requires dated prior-art evidence")
	}
	if _, err := l.db.ExecContext(ctx, `INSERT INTO claims (id, statement, kind, verification_status, created_at) VALUES (?, ?, ?, ?, ?)`, claim.ID, claim.Statement, claim.Kind, claim.VerificationStatus, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return Claim{}, err
	}
	_, err := l.db.ExecContext(ctx, `INSERT INTO claim_evidence (id, claim_id, kind, reference, command, expected_result) VALUES (?, ?, ?, ?, ?, ?)`, uuid.NewString(), claim.ID, evidence.Kind, evidence.Reference, evidence.Command, evidence.ExpectedResult)
	if err != nil {
		return Claim{}, err
	}
	if claim.Kind == KindNovelty {
		if _, err := l.db.ExecContext(ctx, `INSERT INTO claim_novelty_evidence (claim_id, observed_at) VALUES (?, ?)`, claim.ID, evidence.ObservedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return Claim{}, err
		}
	}
	return claim, nil
}

// Coverage returns cited spans and the complementary text intervals. It is
// mechanical evidence for review, not a claim-classification or approval rule.
func (l *library) Coverage(ctx context.Context, draftID, body string) ([]TextSpan, []TextSpan, error) {
	rows, err := l.db.QueryContext(ctx, `SELECT claim_id, span_start, span_end FROM claim_citations WHERE draft_id = ? ORDER BY span_start, span_end, claim_id`, draftID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var supported []TextSpan
	for rows.Next() {
		var span TextSpan
		if err := rows.Scan(&span.ClaimID, &span.Start, &span.End); err != nil {
			return nil, nil, err
		}
		if span.Start < 0 || span.End > len(body) || span.End <= span.Start {
			return nil, nil, ErrInvalidAnchor
		}
		supported = append(supported, span)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	coveredUntil := 0
	var uncovered []TextSpan
	for _, span := range supported {
		if span.Start > coveredUntil {
			uncovered = append(uncovered, TextSpan{Start: coveredUntil, End: span.Start})
		}
		if span.End > coveredUntil {
			coveredUntil = span.End
		}
	}
	if coveredUntil < len(body) {
		uncovered = append(uncovered, TextSpan{Start: coveredUntil, End: len(body)})
	}
	return supported, uncovered, nil
}

// Sweep re-runs every check-backed claim. The returned claims include the
// current state, so scheduler callers can immediately report newly stale facts.
func (l *library) Sweep(ctx context.Context) ([]Claim, error) {
	rows, err := l.db.QueryContext(ctx, `SELECT DISTINCT claim_id FROM claim_evidence WHERE kind = 'check' ORDER BY claim_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	claims := make([]Claim, 0, len(ids))
	for _, id := range ids {
		claim, err := l.Verify(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("verify claim %q: %w", id, err)
		}
		claims = append(claims, claim)
	}
	return claims, nil
}

// ExpireNovelty removes verification from claims whose dated prior-art search
// is older than maxAge. Time alone changes novelty evidence; other claims are
// governed by their explicit checks.
func (l *library) ExpireNovelty(ctx context.Context, now time.Time, maxAge time.Duration) ([]Claim, error) {
	cutoff := now.UTC().Add(-maxAge).Format(time.RFC3339Nano)
	rows, err := l.db.QueryContext(ctx, `SELECT c.id, c.statement, c.kind, c.verification_status FROM claims c JOIN claim_novelty_evidence n ON n.claim_id = c.id WHERE n.observed_at < ? AND c.verification_status <> ? ORDER BY c.id`, cutoff, StateAsserted)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var expired []Claim
	for rows.Next() {
		var claim Claim
		if err := rows.Scan(&claim.ID, &claim.Statement, &claim.Kind, &claim.VerificationStatus); err != nil {
			return nil, err
		}
		expired = append(expired, claim)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range expired {
		if _, err := l.db.ExecContext(ctx, `UPDATE claims SET verification_status = ? WHERE id = ?`, StateAsserted, expired[i].ID); err != nil {
			return nil, err
		}
		expired[i].VerificationStatus = StateAsserted
	}
	return expired, nil
}

func requiresCheck(kind string) bool {
	return kind == KindQuantitative || kind == KindExistence || kind == KindStatus
}

func (l *library) List(ctx context.Context) ([]Claim, error) {
	rows, err := l.db.QueryContext(ctx, `SELECT id, statement, kind, verification_status FROM claims ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Claim
	for rows.Next() {
		var claim Claim
		if err := rows.Scan(&claim.ID, &claim.Statement, &claim.Kind, &claim.VerificationStatus); err != nil {
			return nil, err
		}
		out = append(out, claim)
	}
	return out, rows.Err()
}

func (l *library) ListForDraft(ctx context.Context, draftID string) ([]Claim, error) {
	rows, err := l.db.QueryContext(ctx, `SELECT c.id, c.statement, c.kind, c.verification_status FROM claims c JOIN claim_citations cc ON cc.claim_id = c.id WHERE cc.draft_id = ? ORDER BY c.id`, draftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Claim
	for rows.Next() {
		var claim Claim
		if err := rows.Scan(&claim.ID, &claim.Statement, &claim.Kind, &claim.VerificationStatus); err != nil {
			return nil, err
		}
		out = append(out, claim)
	}
	return out, rows.Err()
}

func (l *library) Cite(ctx context.Context, citation Citation, body string) error {
	if citation.Start < 0 || citation.End <= citation.Start || citation.End > len(body) {
		return ErrInvalidAnchor
	}
	_, err := l.db.ExecContext(ctx, `INSERT INTO claim_citations (draft_id, claim_id, span_start, span_end) VALUES (?, ?, ?, ?)`, citation.DraftID, citation.ClaimID, citation.Start, citation.End)
	return err
}

func (l *library) CitingDrafts(ctx context.Context, claimID string) ([]string, error) {
	rows, err := l.db.QueryContext(ctx, `SELECT DISTINCT draft_id FROM claim_citations WHERE claim_id = ? ORDER BY draft_id`, claimID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var drafts []string
	for rows.Next() {
		var draft string
		if err := rows.Scan(&draft); err != nil {
			return nil, err
		}
		drafts = append(drafts, draft)
	}
	return drafts, rows.Err()
}

func (l *library) Verify(ctx context.Context, claimID string) (Claim, error) {
	var claim Claim
	var evidence Evidence
	var raw sql.NullString
	err := l.db.QueryRowContext(ctx, `SELECT c.id, c.statement, c.kind, c.verification_status, e.kind, e.reference, e.command, e.expected_result, e.last_result, e.last_run_at FROM claims c JOIN claim_evidence e ON e.claim_id = c.id WHERE c.id = ? AND e.kind = 'check' LIMIT 1`, claimID).Scan(&claim.ID, &claim.Statement, &claim.Kind, &claim.VerificationStatus, &evidence.Kind, &evidence.Reference, &evidence.Command, &evidence.ExpectedResult, &evidence.LastResult, &raw)
	if err != nil {
		return Claim{}, err
	}
	result, err := l.runner.Run(ctx, EvidenceCheck{Command: evidence.Command, ExpectedResult: evidence.ExpectedResult})
	if err != nil {
		return Claim{}, err
	}
	claim.VerificationStatus = StateStale
	if result.Matches {
		claim.VerificationStatus = StateVerified
	}
	_, err = l.db.ExecContext(ctx, `UPDATE claims SET verification_status = ? WHERE id = ?`, claim.VerificationStatus, claim.ID)
	if err != nil {
		return Claim{}, err
	}
	_, err = l.db.ExecContext(ctx, `UPDATE claim_evidence SET last_result = ?, last_run_at = ? WHERE claim_id = ? AND kind = 'check'`, result.ActualResult, time.Now().UTC().Format(time.RFC3339Nano), claim.ID)
	if err != nil {
		return Claim{}, err
	}
	return claim, nil
}
