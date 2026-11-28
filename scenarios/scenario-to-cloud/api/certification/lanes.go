package certification

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// LaneFault is the declared fault for one lane cell.
type LaneFault struct {
	Capability string `json:"capability"`
	Point      string `json:"point"`
	Kind       string `json:"kind"`
}

// LaneCell is one required (case, lane) cell with its fixture and fault.
type LaneCell struct {
	CaseID           string     `json:"case_id"`
	Family           string     `json:"family"`
	Fixture          string     `json:"fixture"`
	Fault            *LaneFault `json:"fault"`
	Architectures    []string   `json:"architectures"`
	RepeatAfterReset bool       `json:"repeat_after_reset"`
	RequirementRefs  []string   `json:"requirement_refs"`
	ReceiptRef       string     `json:"receipt_ref"`
}

// LaneManifest is one certification lane's required cells, prerequisites and
// budgets (certification/lanes/<lane>.json). The matrix decides which cases
// belong to a lane; the manifest says how each cell is exercised.
type LaneManifest struct {
	SchemaVersion         int                            `json:"schema_version"`
	Lane                  Lane                           `json:"lane"`
	MatrixRevision        string                         `json:"matrix_revision"`
	Provider              string                         `json:"provider"`
	OperatorPrerequisites []map[string]any               `json:"operator_prerequisites"`
	FaultCapabilities     map[string]LaneFaultCapability `json:"fault_capabilities"`
	Budgets               map[string]any                 `json:"budgets"`
	Cells                 []LaneCell                     `json:"cells"`
}

// LaneFaultCapability is a fault the lane may inject, with its declared
// injection points and blast radius.
type LaneFaultCapability struct {
	Points      []string `json:"points"`
	Kind        string   `json:"kind"`
	BlastRadius string   `json:"blast_radius"`
}

// LoadLaneManifest reads and validates one lane manifest against m: every
// cell must be a matrix case that declares the lane, every matrix case that
// declares the lane must be a cell, and every cell fault must name a declared
// capability.
func LoadLaneManifest(path string, m *Matrix) (*LaneManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read lane manifest: %w", err)
	}
	var manifest LaneManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse lane manifest: %w", err)
	}
	if err := manifest.Validate(m); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// Validate checks the manifest against the matrix.
func (l *LaneManifest) Validate(m *Matrix) error {
	if l.SchemaVersion != 1 {
		return fmt.Errorf("lane manifest: unsupported schema_version %d", l.SchemaVersion)
	}
	if l.MatrixRevision != m.MatrixRevision {
		return fmt.Errorf("lane manifest: matrix_revision %q differs from matrix %q", l.MatrixRevision, m.MatrixRevision)
	}
	if !matrixHasLane(m, l.Lane) {
		return fmt.Errorf("lane manifest: unknown lane %q", l.Lane)
	}
	expected := casesForLane(m, l.Lane)
	seen, err := validateLaneCells(l, expected)
	if err != nil {
		return err
	}
	missing := missingLaneCases(expected, seen)
	if len(missing) > 0 {
		return fmt.Errorf("lane manifest: matrix cases with lane %s absent from the manifest: %v", l.Lane, missing)
	}
	return nil
}

func matrixHasLane(m *Matrix, want Lane) bool {
	for _, lane := range m.LaneVocabulary {
		if lane == want {
			return true
		}
	}
	return false
}

func casesForLane(m *Matrix, lane Lane) map[string]bool {
	expected := map[string]bool{}
	for _, c := range m.Cases {
		for _, caseLane := range c.Lanes {
			if caseLane == lane {
				expected[c.ID] = true
			}
		}
	}
	return expected
}

func validateLaneCells(manifest *LaneManifest, expected map[string]bool) (map[string]bool, error) {
	seen := map[string]bool{}
	for _, cell := range manifest.Cells {
		if seen[cell.CaseID] {
			return nil, fmt.Errorf("lane manifest: case %s listed twice", cell.CaseID)
		}
		seen[cell.CaseID] = true
		if err := validateLaneCell(manifest, cell, expected); err != nil {
			return nil, err
		}
	}
	return seen, nil
}

func validateLaneCell(manifest *LaneManifest, cell LaneCell, expected map[string]bool) error {
	if !expected[cell.CaseID] {
		return fmt.Errorf("lane manifest: case %s does not declare lane %s in the matrix", cell.CaseID, manifest.Lane)
	}
	if cell.Fixture == "" || len(cell.Architectures) == 0 || cell.ReceiptRef == "" {
		return fmt.Errorf("lane manifest: case %s needs fixture, architectures and receipt_ref", cell.CaseID)
	}
	return validateLaneFault(manifest, cell)
}

func validateLaneFault(manifest *LaneManifest, cell LaneCell) error {
	if cell.Fault == nil {
		return nil
	}
	capability, ok := manifest.FaultCapabilities[cell.Fault.Capability]
	if !ok {
		return fmt.Errorf("lane manifest: case %s names undeclared fault capability %q", cell.CaseID, cell.Fault.Capability)
	}
	if cell.Fault.Point != "" && !contains(capability.Points, cell.Fault.Point) {
		return fmt.Errorf("lane manifest: case %s fault point %q is not declared for capability %q", cell.CaseID, cell.Fault.Point, cell.Fault.Capability)
	}
	return nil
}

func missingLaneCases(expected, seen map[string]bool) []string {
	missing := []string{}
	for id := range expected {
		if !seen[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
	}
	return missing
}

// CaseIDs returns the cell case IDs in manifest order.
func (l *LaneManifest) CaseIDs() []string {
	out := make([]string, 0, len(l.Cells))
	for _, cell := range l.Cells {
		out = append(out, cell.CaseID)
	}
	return out
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
