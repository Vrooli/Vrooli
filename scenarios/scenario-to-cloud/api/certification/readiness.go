package certification

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Cell is one named (case, lane) cell with the reason it does not satisfy
// readiness, or the receipt that satisfied it.
type Cell struct {
	CaseID     string `json:"case_id"`
	Lane       Lane   `json:"lane"`
	Reason     string `json:"reason"`
	ReceiptRef string `json:"receipt_ref,omitempty"`
	// ObservedDigest is the release digest the best available receipt carries
	// when it differs from the candidate (stale cells only).
	ObservedDigest string `json:"observed_release_digest,omitempty"`
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
	digest := strings.TrimSpace(candidate.ReleaseDigest)
	byCell := make(map[CellID][]Receipt)
	for _, r := range receipts {
		id := CellID{CaseID: r.CaseID, Lane: r.Lane}
		byCell[id] = append(byCell[id], r)
	}
	cells := m.RequiredCells()
	out.RequiredCells = len(cells)
	for _, id := range cells {
		cell := Cell{CaseID: id.CaseID, Lane: id.Lane}
		if digest == "" {
			cell.Reason = "candidate release digest is empty; no receipt can be bound to it"
			out.MissingCells = append(out.MissingCells, cell)
			continue
		}
		var failed, passed, unavailable, other *Receipt
		for i := range byCell[id] {
			r := &byCell[id][i]
			if r.Candidate.ReleaseDigest != digest {
				if other == nil || r.ObservedAt.After(other.ObservedAt) {
					other = r
				}
				continue
			}
			switch r.Verdict {
			case VerdictFailed:
				if failed == nil || r.ObservedAt.After(failed.ObservedAt) {
					failed = r
				}
			case VerdictPassed:
				if passed == nil || r.ObservedAt.After(passed.ObservedAt) {
					passed = r
				}
			default:
				if unavailable == nil || r.ObservedAt.After(unavailable.ObservedAt) {
					unavailable = r
				}
			}
		}
		switch {
		case failed != nil:
			cell.Reason = "failed receipt for candidate"
			cell.ReceiptRef = failed.Ref
			out.FailedCells = append(out.FailedCells, cell)
		case passed != nil:
			cell.Reason = "passed"
			cell.ReceiptRef = passed.Ref
			out.SatisfiedCells = append(out.SatisfiedCells, cell)
		case unavailable != nil:
			cell.Reason = fmt.Sprintf("receipt verdict %s for candidate", unavailable.Verdict)
			cell.ReceiptRef = unavailable.Ref
			out.UnavailableCells = append(out.UnavailableCells, cell)
		case other != nil:
			cell.Reason = "evidence exists only for another release digest"
			cell.ReceiptRef = other.Ref
			cell.ObservedDigest = other.Candidate.ReleaseDigest
			out.StaleCells = append(out.StaleCells, cell)
		default:
			cell.Reason = "no receipt"
			out.MissingCells = append(out.MissingCells, cell)
		}
	}
	for _, s := range [][]Cell{out.SatisfiedCells, out.MissingCells, out.FailedCells, out.StaleCells, out.UnavailableCells} {
		sortCells(s)
	}
	out.Ready = out.NotReadyCount() == 0 && out.RequiredCells > 0
	out.Summary = summarize(out)
	return out
}

func sortCells(cells []Cell) {
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].CaseID != cells[j].CaseID {
			return cells[i].CaseID < cells[j].CaseID
		}
		return cells[i].Lane < cells[j].Lane
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
