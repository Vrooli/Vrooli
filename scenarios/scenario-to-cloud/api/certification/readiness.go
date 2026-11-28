package certification

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Cell is one named (case, lane, target-ground) obligation with the reason it
// does not satisfy readiness, or the receipt that satisfied it.
type Cell struct {
	CaseID       string       `json:"case_id"`
	Lane         Lane         `json:"lane"`
	TargetGround TargetGround `json:"target_ground"`
	Reason       string       `json:"reason"`
	ReceiptRef   string       `json:"receipt_ref,omitempty"`
	// ObservedDigest is the release digest the best available receipt carries
	// when it differs from the candidate (stale cells only).
	ObservedDigest    string    `json:"observed_release_digest,omitempty"`
	ObservedCandidate Candidate `json:"observed_candidate,omitempty"`
}

// Readiness is the certification decision for one candidate.
type Readiness struct {
	Ready            bool      `json:"ready"`
	MatrixRevision   string    `json:"matrix_revision"`
	Candidate        Candidate `json:"candidate"`
	EvaluatedAt      time.Time `json:"evaluated_at"`
	RequiredCells    int       `json:"required_cells"`
	SatisfiedCells   []Cell    `json:"satisfied_cells"`
	MissingCells     []Cell    `json:"missing_cells"`
	FailedCells      []Cell    `json:"failed_cells"`
	StaleCells       []Cell    `json:"stale_cells"`
	UnavailableCells []Cell    `json:"unavailable_cells"`
	// Summary is the single next action for an operator.
	Summary string `json:"summary"`
}

// NotReadyCount returns the number of cells that block readiness.
func (r Readiness) NotReadyCount() int {
	return len(r.MissingCells) + len(r.FailedCells) + len(r.StaleCells) + len(r.UnavailableCells)
}

// Evaluate computes readiness of candidate against m using receipts.
//
// Precedence per required cell, considering only receipts whose case and lane
// match:
//  1. any failed receipt for the candidate digest -> failed
//  2. otherwise a passed receipt for the candidate digest -> satisfied
//  3. otherwise a skipped/unavailable/unsupported receipt for the candidate -> unavailable
//  4. otherwise any receipt for another digest -> stale
//  5. otherwise -> missing
//
// A failed receipt for the candidate is never masked by a later passed one:
// re-running until green is not a certification path, the failed receipt must
// be withdrawn by its owner with an explanation.
func Evaluate(m *Matrix, receipts []Receipt, candidate Candidate, now time.Time) Readiness {
	out := Readiness{
		MatrixRevision: m.MatrixRevision,
		Candidate:      candidate,
		EvaluatedAt:    now.UTC(),
	}
	byCell := indexValidReceipts(m, receipts)
	cells := m.RequiredCells()
	out.RequiredCells = len(cells)
	for _, id := range cells {
		out.addCellResult(evaluateCell(id, byCell[id], candidate))
	}
	for _, s := range [][]Cell{out.SatisfiedCells, out.MissingCells, out.FailedCells, out.StaleCells, out.UnavailableCells} {
		sortCells(s)
	}
	out.Ready = out.NotReadyCount() == 0 && out.RequiredCells > 0
	out.Summary = summarize(out)
	return out
}

type cellResult struct {
	cell Cell
	kind cellResultKind
}

type cellResultKind uint8

const (
	cellMissing cellResultKind = iota
	cellFailed
	cellSatisfied
	cellUnavailable
	cellStale
)

func indexValidReceipts(m *Matrix, receipts []Receipt) map[CellID][]Receipt {
	byCell := make(map[CellID][]Receipt)
	for _, receipt := range receipts {
		// Invalid receipts are ignored by the evaluator. LoadEvidenceDir returns
		// their validation error to file-based callers; this guard also keeps
		// direct callers fail-closed.
		if err := receipt.Validate(m); err != nil {
			continue
		}
		id := CellID{CaseID: receipt.CaseID, Lane: receipt.Lane, TargetGround: receipt.Target.Ground}
		byCell[id] = append(byCell[id], receipt)
	}
	return byCell
}

func evaluateCell(id CellID, receipts []Receipt, candidate Candidate) cellResult {
	cell := Cell{CaseID: id.CaseID, Lane: id.Lane, TargetGround: id.TargetGround}
	if !candidateIsComplete(candidate) {
		cell.Reason = "candidate must include valid release, configuration and closure sha256 digests"
		return cellResult{cell: cell, kind: cellMissing}
	}
	groups := groupCellReceipts(receipts, candidate)
	return resultFromReceiptGroups(cell, groups)
}

func candidateIsComplete(candidate Candidate) bool {
	return isSHA256Digest(candidate.ReleaseDigest) && isSHA256Digest(candidate.ConfigurationDigest) && isSHA256Digest(candidate.ClosureDigest)
}

type cellReceiptGroups struct {
	failed      *Receipt
	passed      *Receipt
	unavailable *Receipt
	other       *Receipt
}

func groupCellReceipts(receipts []Receipt, candidate Candidate) cellReceiptGroups {
	var groups cellReceiptGroups
	for i := range receipts {
		receipt := &receipts[i]
		if !sameCandidate(receipt.Candidate, candidate) {
			groups.other = laterReceipt(groups.other, receipt)
			continue
		}
		groups = addCandidateReceipt(groups, receipt)
	}
	return groups
}

func addCandidateReceipt(groups cellReceiptGroups, receipt *Receipt) cellReceiptGroups {
	switch receipt.Verdict {
	case VerdictFailed:
		groups.failed = laterReceipt(groups.failed, receipt)
	case VerdictPassed:
		groups.passed = laterReceipt(groups.passed, receipt)
	default:
		groups.unavailable = laterReceipt(groups.unavailable, receipt)
	}
	return groups
}

func laterReceipt(current, candidate *Receipt) *Receipt {
	if current == nil || candidate.ObservedAt.After(current.ObservedAt) {
		return candidate
	}
	return current
}

func resultFromReceiptGroups(cell Cell, groups cellReceiptGroups) cellResult {
	switch {
	case groups.failed != nil:
		cell.Reason, cell.ReceiptRef = "failed receipt for candidate", groups.failed.Ref
		return cellResult{cell: cell, kind: cellFailed}
	case groups.passed != nil:
		cell.Reason, cell.ReceiptRef = "passed", groups.passed.Ref
		return cellResult{cell: cell, kind: cellSatisfied}
	case groups.unavailable != nil:
		cell.Reason = fmt.Sprintf("receipt verdict %s for candidate", groups.unavailable.Verdict)
		cell.ReceiptRef = groups.unavailable.Ref
		return cellResult{cell: cell, kind: cellUnavailable}
	case groups.other != nil:
		cell.Reason = "evidence exists only for another release, configuration or closure candidate"
		cell.ReceiptRef = groups.other.Ref
		cell.ObservedDigest = groups.other.Candidate.ReleaseDigest
		cell.ObservedCandidate = groups.other.Candidate
		return cellResult{cell: cell, kind: cellStale}
	default:
		cell.Reason = "no receipt"
		return cellResult{cell: cell, kind: cellMissing}
	}
}

func (r *Readiness) addCellResult(result cellResult) {
	switch result.kind {
	case cellFailed:
		r.FailedCells = append(r.FailedCells, result.cell)
	case cellSatisfied:
		r.SatisfiedCells = append(r.SatisfiedCells, result.cell)
	case cellUnavailable:
		r.UnavailableCells = append(r.UnavailableCells, result.cell)
	case cellStale:
		r.StaleCells = append(r.StaleCells, result.cell)
	default:
		r.MissingCells = append(r.MissingCells, result.cell)
	}
}

func sameCandidate(left, right Candidate) bool {
	return left.ReleaseDigest == right.ReleaseDigest &&
		left.ConfigurationDigest == right.ConfigurationDigest &&
		left.ClosureDigest == right.ClosureDigest
}

func sortCells(cells []Cell) {
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].CaseID != cells[j].CaseID {
			return cells[i].CaseID < cells[j].CaseID
		}
		if cells[i].Lane != cells[j].Lane {
			return cells[i].Lane < cells[j].Lane
		}
		return cells[i].TargetGround < cells[j].TargetGround
	})
}

func summarize(r Readiness) string {
	if r.Ready {
		return fmt.Sprintf("ready: all %d required cells hold passed receipts for release %s", r.RequiredCells, r.Candidate.ReleaseDigest)
	}
	parts := []string{}
	if n := len(r.FailedCells); n > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", n))
	}
	if n := len(r.MissingCells); n > 0 {
		parts = append(parts, fmt.Sprintf("%d missing", n))
	}
	if n := len(r.StaleCells); n > 0 {
		parts = append(parts, fmt.Sprintf("%d stale", n))
	}
	if n := len(r.UnavailableCells); n > 0 {
		parts = append(parts, fmt.Sprintf("%d unavailable", n))
	}
	next := "collect passed receipts for the named cells"
	if len(r.FailedCells) > 0 {
		next = "resolve failed cells first; a failed receipt is never masked by a later pass"
	}
	return fmt.Sprintf("not_ready: %s of %d required cells; next: %s", strings.Join(parts, ", "), r.RequiredCells, next)
}

// Report evaluates candidate against the embedded matrix and the receipts in
// evidenceDir and returns the readiness decision as indented JSON. It is
// CLI-free so both an HTTP handler and a command can serve it unchanged.
func Report(evidenceDir string, candidate Candidate, now time.Time) ([]byte, error) {
	m, err := LoadEmbedded()
	if err != nil {
		return nil, err
	}
	receipts, err := LoadEvidenceDir(evidenceDir, m)
	if err != nil {
		return nil, err
	}
	readiness := Evaluate(m, receipts, candidate, now)
	return json.MarshalIndent(readiness, "", "  ")
}
