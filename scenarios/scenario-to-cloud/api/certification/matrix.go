// Package certification owns the cloud certification matrix and the readiness
// decision derived from evidence receipts.
//
// The matrix is the plan's mandatory validation matrix (94 cases) copied into
// certification/matrix.json at the scenario root and embedded here. A required
// case is satisfied only when every lane it declares holds a passed receipt for
// the exact candidate release digest; anything else names the (case, lane)
// cell so certification cannot pass on missing, failed, stale or unavailable
// evidence.
package certification

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

//go:embed matrix.json
var embeddedMatrix []byte

// Lane names a certification lane. Values are the lane_vocabulary in the matrix.
type Lane string

// Lane vocabulary.
const (
	LanePackage           Lane = "package"
	LaneAPI               Lane = "api"
	LaneWorkflow          Lane = "workflow"
	LaneRelay             Lane = "relay"
	LaneQEMU              Lane = "qemu"
	LaneRealVPS           Lane = "real_vps"
	LaneBASUI             Lane = "bas_ui"
	LaneIndependentReview Lane = "independent_review"
	LaneSoak              Lane = "soak"
)

// Case is one row of the mandatory matrix with a stable ID.
type Case struct {
	ID              string   `json:"id"`
	Family          string   `json:"family"`
	PrimaryPhase    int      `json:"primary_phase"`
	Required        bool     `json:"required"`
	Lanes           []Lane   `json:"lanes"`
	LaneText        string   `json:"lane_text"`
	Trigger         string   `json:"trigger"`
	Assertion       string   `json:"assertion"`
	Dimensions      []string `json:"dimensions"`
	RequirementRefs []string `json:"requirement_refs"`
	Owner           string   `json:"owner"`
	OwnerNotes      string   `json:"owner_notes"`
	Evidence        string   `json:"evidence"`
}

// UnsupportedCell is a declared-unsupported combination with its policy reason.
type UnsupportedCell struct {
	ID             string `json:"id"`
	Dimension      string `json:"dimension"`
	Value          string `json:"value"`
	Classification string `json:"classification"`
	Reason         string `json:"reason"`
	PolicyRef      string `json:"policy_ref"`
}

// Source records where the matrix was copied from.
type Source struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	CaseCount int    `json:"case_count"`
}

// Matrix is the certification matrix document.
type Matrix struct {
	SchemaVersion       int               `json:"schema_version"`
	MatrixRevision      string            `json:"matrix_revision"`
	Status              string            `json:"status"`
	Source              Source            `json:"source"`
	LaneVocabulary      []Lane            `json:"lane_vocabulary"`
	DimensionVocabulary []string          `json:"dimension_vocabulary"`
	VerdictVocabulary   []Verdict         `json:"verdict_vocabulary"`
	ReadinessRule       string            `json:"readiness_rule"`
	Cases               []Case            `json:"cases"`
	UnsupportedCells    []UnsupportedCell `json:"unsupported_cells"`
}

// EmbeddedMatrixSHA256 returns the digest of the embedded matrix bytes.
func EmbeddedMatrixSHA256() string {
	sum := sha256.Sum256(embeddedMatrix)
	return hex.EncodeToString(sum[:])
}

// LoadEmbedded parses the matrix embedded in the binary.
func LoadEmbedded() (*Matrix, error) {
	return Parse(embeddedMatrix)
}

// LoadFile parses a matrix document from disk.
func LoadFile(path string) (*Matrix, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read matrix: %w", err)
	}
	return Parse(data)
}

// Parse decodes and validates a matrix document.
func Parse(data []byte) (*Matrix, error) {
	var m Matrix
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse matrix: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate checks structural invariants: unique IDs, lanes and dimensions from
// the declared vocabularies, at least one lane per case, requirement refs present.
func (m *Matrix) Validate() error {
	if m.SchemaVersion != 1 {
		return fmt.Errorf("matrix: unsupported schema_version %d", m.SchemaVersion)
	}
	if len(m.Cases) == 0 {
		return fmt.Errorf("matrix: no cases")
	}
	lanes := make(map[Lane]struct{}, len(m.LaneVocabulary))
	for _, l := range m.LaneVocabulary {
		lanes[l] = struct{}{}
	}
	dims := make(map[string]struct{}, len(m.DimensionVocabulary))
	for _, d := range m.DimensionVocabulary {
		dims[d] = struct{}{}
	}
	seen := make(map[string]struct{}, len(m.Cases))
	for _, c := range m.Cases {
		if c.ID == "" {
			return fmt.Errorf("matrix: case with empty id")
		}
		if _, dup := seen[c.ID]; dup {
			return fmt.Errorf("matrix: duplicate case id %q", c.ID)
		}
		seen[c.ID] = struct{}{}
		if len(c.Lanes) == 0 {
			return fmt.Errorf("matrix: case %s declares no lanes", c.ID)
		}
		for _, l := range c.Lanes {
			if _, ok := lanes[l]; !ok {
				return fmt.Errorf("matrix: case %s uses lane %q outside lane_vocabulary", c.ID, l)
			}
		}
		for _, d := range c.Dimensions {
			if _, ok := dims[d]; !ok {
				return fmt.Errorf("matrix: case %s uses dimension %q outside dimension_vocabulary", c.ID, d)
			}
		}
		if len(c.RequirementRefs) == 0 {
			return fmt.Errorf("matrix: case %s has no requirement_refs", c.ID)
		}
		if c.Owner == "" {
			return fmt.Errorf("matrix: case %s has empty owner (use \"unassigned\")", c.ID)
		}
	}
	return nil
}

// Case returns the case with the given ID.
func (m *Matrix) Case(id string) (Case, bool) {
	for _, c := range m.Cases {
		if c.ID == id {
			return c, true
		}
	}
	return Case{}, false
}

// RequiredCases returns the required cases in matrix order.
func (m *Matrix) RequiredCases() []Case {
	out := make([]Case, 0, len(m.Cases))
	for _, c := range m.Cases {
		if c.Required {
			out = append(out, c)
		}
	}
	return out
}

// RequiredCells enumerates every (case, lane) cell that must hold passing
// evidence, sorted by case then lane.
func (m *Matrix) RequiredCells() []CellID {
	var cells []CellID
	for _, c := range m.RequiredCases() {
		for _, l := range c.Lanes {
			cells = append(cells, CellID{CaseID: c.ID, Lane: l})
		}
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].CaseID != cells[j].CaseID {
			return cells[i].CaseID < cells[j].CaseID
		}
		return cells[i].Lane < cells[j].Lane
	})
	return cells
}

// CellID identifies one (case, lane) evidence cell.
type CellID struct {
	CaseID string `json:"case_id"`
	Lane   Lane   `json:"lane"`
}

func (c CellID) String() string { return c.CaseID + "/" + string(c.Lane) }
