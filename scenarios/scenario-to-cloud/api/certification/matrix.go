// Package certification owns the cloud certification matrix and the readiness
// decision derived from evidence receipts.
//
// The matrix is the plan's mandatory validation matrix (94 cases) copied into
// certification/matrix.json at the scenario root and embedded here. Every
// required case/lane pair explicitly applies to QEMU and a real VPS. Readiness
// needs a separate schema-v2 receipt for each target ground, bound to the exact
// candidate release, configuration and closure digests; missing, failed,
// stale or unavailable ground evidence keeps certification open.
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

// TargetGround names one independent disposable proving ground. Every
// required case/lane obligation must hold on every ground in the matrix map.
type TargetGround string

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

const (
	TargetGroundQEMU    TargetGround = "qemu"
	TargetGroundRealVPS TargetGround = "real_vps"
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

// GroundApplicability binds one mandatory case/lane pair to the target grounds
// where it must run. The matrix must enumerate every pair explicitly; there
// is no implicit default.
type GroundApplicability struct {
	CaseID  string         `json:"case_id"`
	Lane    Lane           `json:"lane"`
	Grounds []TargetGround `json:"grounds"`
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
	SchemaVersion          int                   `json:"schema_version"`
	MatrixRevision         string                `json:"matrix_revision"`
	Status                 string                `json:"status"`
	Source                 Source                `json:"source"`
	LaneVocabulary         []Lane                `json:"lane_vocabulary"`
	TargetGroundVocabulary []TargetGround        `json:"target_ground_vocabulary"`
	GroundApplicability    []GroundApplicability `json:"ground_applicability"`
	DimensionVocabulary    []string              `json:"dimension_vocabulary"`
	VerdictVocabulary      []Verdict             `json:"verdict_vocabulary"`
	ReadinessRule          string                `json:"readiness_rule"`
	Cases                  []Case                `json:"cases"`
	UnsupportedCells       []UnsupportedCell     `json:"unsupported_cells"`
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
	lanes, grounds, dims, err := m.validationVocabularies()
	if err != nil {
		return err
	}
	if err := m.validateCases(lanes, dims); err != nil {
		return err
	}
	return m.validateGroundApplicability(grounds)
}

func (m *Matrix) validationVocabularies() (map[Lane]struct{}, map[TargetGround]struct{}, map[string]struct{}, error) {
	lanes := make(map[Lane]struct{}, len(m.LaneVocabulary))
	for _, l := range m.LaneVocabulary {
		lanes[l] = struct{}{}
	}
	grounds := make(map[TargetGround]struct{}, len(m.TargetGroundVocabulary))
	for _, ground := range m.TargetGroundVocabulary {
		if ground == "" {
			return nil, nil, nil, fmt.Errorf("matrix: target ground vocabulary contains an empty value")
		}
		if _, dup := grounds[ground]; dup {
			return nil, nil, nil, fmt.Errorf("matrix: duplicate target ground %q", ground)
		}
		grounds[ground] = struct{}{}
	}
	if len(grounds) == 0 {
		return nil, nil, nil, fmt.Errorf("matrix: no target grounds declared")
	}
	dims := make(map[string]struct{}, len(m.DimensionVocabulary))
	for _, d := range m.DimensionVocabulary {
		dims[d] = struct{}{}
	}
	return lanes, grounds, dims, nil
}

func (m *Matrix) validateCases(lanes map[Lane]struct{}, dims map[string]struct{}) error {
	seen := make(map[string]struct{}, len(m.Cases))
	for _, c := range m.Cases {
		if c.ID == "" {
			return fmt.Errorf("matrix: case with empty id")
		}
		if _, dup := seen[c.ID]; dup {
			return fmt.Errorf("matrix: duplicate case id %q", c.ID)
		}
		seen[c.ID] = struct{}{}
		if err := validateMatrixCase(c, lanes, dims); err != nil {
			return err
		}
	}
	return nil
}

func validateMatrixCase(c Case, lanes map[Lane]struct{}, dims map[string]struct{}) error {
	if err := validateCaseLanes(c, lanes); err != nil {
		return err
	}
	return validateCaseDetails(c, dims)
}

func validateCaseLanes(c Case, lanes map[Lane]struct{}) error {
	if len(c.Lanes) == 0 {
		return fmt.Errorf("matrix: case %s declares no lanes", c.ID)
	}
	for _, l := range c.Lanes {
		if _, ok := lanes[l]; !ok {
			return fmt.Errorf("matrix: case %s uses lane %q outside lane_vocabulary", c.ID, l)
		}
	}
	return nil
}

func validateCaseDetails(c Case, dims map[string]struct{}) error {
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
	return nil
}

func (m *Matrix) validateGroundApplicability(grounds map[TargetGround]struct{}) error {
	// Ground applicability is a closed map over every required case/lane pair.
	// In particular, a new case or lane cannot silently inherit a default that
	// bypasses QEMU or real-VPS evidence.
	expectedPairs, err := m.expectedGroundApplicabilityPairs()
	if err != nil {
		return err
	}
	seenPairs := make(map[LaneCellID]struct{}, len(m.GroundApplicability))
	for _, applicability := range m.GroundApplicability {
		pair := LaneCellID{CaseID: applicability.CaseID, Lane: applicability.Lane}
		if _, ok := expectedPairs[pair]; !ok {
			return fmt.Errorf("matrix: ground applicability names non-required cell %s/%s", applicability.CaseID, applicability.Lane)
		}
		if _, dup := seenPairs[pair]; dup {
			return fmt.Errorf("matrix: duplicate ground applicability for %s/%s", applicability.CaseID, applicability.Lane)
		}
		seenPairs[pair] = struct{}{}
		seenGrounds := make(map[TargetGround]struct{}, len(applicability.Grounds))
		for _, ground := range applicability.Grounds {
			if _, ok := grounds[ground]; !ok {
				return fmt.Errorf("matrix: %s/%s names unknown target ground %q", applicability.CaseID, applicability.Lane, ground)
			}
			if _, dup := seenGrounds[ground]; dup {
				return fmt.Errorf("matrix: %s/%s repeats target ground %q", applicability.CaseID, applicability.Lane, ground)
			}
			seenGrounds[ground] = struct{}{}
		}
		if len(seenGrounds) != len(grounds) {
			return fmt.Errorf("matrix: %s/%s must explicitly require every target ground", applicability.CaseID, applicability.Lane)
		}
	}
	return m.validateEveryRequiredPair(expectedPairs, seenPairs)
}

func (m *Matrix) expectedGroundApplicabilityPairs() (map[LaneCellID]struct{}, error) {
	expectedPairs := make(map[LaneCellID]struct{})
	for _, c := range m.RequiredCases() {
		for _, lane := range c.Lanes {
			pair := LaneCellID{CaseID: c.ID, Lane: lane}
			if _, dup := expectedPairs[pair]; dup {
				return nil, fmt.Errorf("matrix: case %s repeats lane %s", c.ID, lane)
			}
			expectedPairs[pair] = struct{}{}
		}
	}
	return expectedPairs, nil
}

func (m *Matrix) validateEveryRequiredPair(expectedPairs, seenPairs map[LaneCellID]struct{}) error {
	if len(seenPairs) != len(expectedPairs) {
		missing := make([]string, 0)
		for pair := range expectedPairs {
			if _, ok := seenPairs[pair]; !ok {
				missing = append(missing, pair.CaseID+"/"+string(pair.Lane))
			}
		}
		sort.Strings(missing)
		return fmt.Errorf("matrix: ground applicability missing required cells: %v", missing)
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

// RequiredCells enumerates every (case, lane, target ground) cell that must
// hold passing evidence, sorted by case, lane, then target ground.
func (m *Matrix) RequiredCells() []CellID {
	var cells []CellID
	for _, c := range m.RequiredCases() {
		for _, l := range c.Lanes {
			for _, ground := range m.TargetGroundsFor(c.ID, l) {
				cells = append(cells, CellID{CaseID: c.ID, Lane: l, TargetGround: ground})
			}
		}
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].CaseID != cells[j].CaseID {
			return cells[i].CaseID < cells[j].CaseID
		}
		if cells[i].Lane != cells[j].Lane {
			return cells[i].Lane < cells[j].Lane
		}
		return cells[i].TargetGround < cells[j].TargetGround
	})
	return cells
}

// TargetGroundsFor returns the explicitly declared grounds for one required
// case/lane pair. Invalid matrices return no grounds; Parse rejects them.
func (m *Matrix) TargetGroundsFor(caseID string, lane Lane) []TargetGround {
	for _, applicability := range m.GroundApplicability {
		if applicability.CaseID == caseID && applicability.Lane == lane {
			return append([]TargetGround(nil), applicability.Grounds...)
		}
	}
	return nil
}

// LaneCellID identifies a case/lane pair before target grounds are expanded.
type LaneCellID struct {
	CaseID string `json:"case_id"`
	Lane   Lane   `json:"lane"`
}

// CellID identifies one (case, lane) evidence cell.
type CellID struct {
	CaseID       string       `json:"case_id"`
	Lane         Lane         `json:"lane"`
	TargetGround TargetGround `json:"target_ground"`
}

func (c CellID) String() string {
	return c.CaseID + "/" + string(c.Lane) + "/" + string(c.TargetGround)
}
