package certification

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Verdict is the outcome recorded on an evidence receipt.
type Verdict string

// Verdict vocabulary. Only VerdictPassed can satisfy a cell.
const (
	VerdictPassed      Verdict = "passed"
	VerdictFailed      Verdict = "failed"
	VerdictSkipped     Verdict = "skipped"
	VerdictUnavailable Verdict = "unavailable"
	VerdictUnsupported Verdict = "unsupported"
)

// Candidate identifies the exact release under certification.
type Candidate struct {
	ReleaseDigest       string `json:"release_digest"`
	ConfigurationDigest string `json:"configuration_digest"`
	ClosureDigest       string `json:"closure_digest"`
}

// Target identifies the machine a receipt was observed on.
type Target struct {
	MachineID    string `json:"machine_id"`
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

// Receipt is one evidence package for one (case, lane) cell. It follows the
// plan's evidence package schema; artifact references are opaque owner-scoped
// IDs, never local paths.
type Receipt struct {
	SchemaVersion        int       `json:"schema_version"`
	CaseID               string    `json:"case_id"`
	Verdict              Verdict   `json:"verdict"`
	Lane                 Lane      `json:"lane"`
	RequirementRefs      []string  `json:"requirement_refs"`
	Candidate            Candidate `json:"candidate"`
	Target               Target    `json:"target"`
	OperationRefs        []string  `json:"operation_refs"`
	ValidationReceiptRef string    `json:"validation_receipt_ref"`
	ArtifactRefs         []string  `json:"artifact_refs"`
	Assertions           []string  `json:"assertions"`
	ObservedAt           time.Time `json:"observed_at"`
	Limitations          []string  `json:"limitations"`

	// Ref is the receipt's own identity (file basename without extension when
	// loaded from a directory). It is reported back in readiness cells.
	Ref string `json:"-"`
}

// Validate checks a receipt against the vocabularies of m.
func (r Receipt) Validate(m *Matrix) error {
	if r.SchemaVersion != 1 {
		return fmt.Errorf("receipt %s: unsupported schema_version %d", r.Ref, r.SchemaVersion)
	}
	if r.CaseID == "" {
		return fmt.Errorf("receipt %s: empty case_id", r.Ref)
	}
	if _, ok := m.Case(r.CaseID); !ok {
		return fmt.Errorf("receipt %s: unknown case_id %q", r.Ref, r.CaseID)
	}
	switch r.Verdict {
	case VerdictPassed, VerdictFailed, VerdictSkipped, VerdictUnavailable, VerdictUnsupported:
	default:
		return fmt.Errorf("receipt %s: unknown verdict %q", r.Ref, r.Verdict)
	}
	laneKnown := false
	for _, l := range m.LaneVocabulary {
		if l == r.Lane {
			laneKnown = true
			break
		}
	}
	if !laneKnown {
		return fmt.Errorf("receipt %s: unknown lane %q", r.Ref, r.Lane)
	}
	if r.Verdict == VerdictPassed && strings.TrimSpace(r.Candidate.ReleaseDigest) == "" {
		return fmt.Errorf("receipt %s: passed verdict without candidate.release_digest", r.Ref)
	}
	if r.ObservedAt.IsZero() {
		return fmt.Errorf("receipt %s: observed_at is required", r.Ref)
	}
	return nil
}

// LoadEvidenceDir reads every *.json receipt in dir. A missing directory is
// treated as empty evidence, not an error: absence of evidence is a valid,
// explicitly not-ready state.
func LoadEvidenceDir(dir string, m *Matrix) ([]Receipt, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read evidence dir: %w", err)
	}
	var receipts []Receipt
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read receipt %s: %w", e.Name(), err)
		}
		r, err := ParseReceipt(data, m)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		r.Ref = strings.TrimSuffix(e.Name(), ".json")
		receipts = append(receipts, r)
	}
	sort.Slice(receipts, func(i, j int) bool { return receipts[i].Ref < receipts[j].Ref })
	return receipts, nil
}

// ParseReceipt decodes and validates one receipt.
func ParseReceipt(data []byte, m *Matrix) (Receipt, error) {
	var r Receipt
	if err := json.Unmarshal(data, &r); err != nil {
		return Receipt{}, fmt.Errorf("parse receipt: %w", err)
	}
	if err := r.Validate(m); err != nil {
		return Receipt{}, err
	}
	return r, nil
}
