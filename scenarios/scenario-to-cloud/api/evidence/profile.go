package evidence

import (
	"fmt"
	"sort"

	"scenario-to-cloud/certification"
)

// ProfileCloudLaunchV1 is the cloud lifecycle capability profile: the
// release-lane cells of the certification matrix a governed publication must
// hold passed evidence for. It is defined without any packaging assumption
// (no installer, window, or desktop capability appears in it).
const ProfileCloudLaunchV1 = "cloud-launch-v1"

// CloudLaunchCases lists the matrix cases in the profile. Each case keeps the
// lanes the matrix declares for it.
var CloudLaunchCases = []string{
	"RELEASE-07", "RELEASE-08",
	"EDGE-07", "EDGE-08",
	"GOV-01", "GOV-02", "GOV-03", "GOV-04", "GOV-05", "GOV-06", "GOV-07", "GOV-08",
	"RUN-03", "RUN-04", "RUN-05",
	"DATA-04", "DATA-05",
}

// CellID names one (case, lane) cell.
type CellID struct {
	CaseID string             `json:"case_id"`
	Lane   certification.Lane `json:"lane"`
}

// String is the stable "<case>/<lane>" spelling used in ramp cell ids.
func (c CellID) String() string { return c.CaseID + "/" + string(c.Lane) }

// ParseCellID reverses String.
func ParseCellID(value string) (CellID, error) {
	for i := len(value) - 1; i >= 0; i-- {
		if value[i] == '/' {
			id := CellID{CaseID: value[:i], Lane: certification.Lane(value[i+1:])}
			if id.CaseID == "" || id.Lane == "" {
				break
			}
			return id, nil
		}
	}
	return CellID{}, fmt.Errorf("cell id %q must be <case>/<lane>", value)
}

// Profile is one capability profile: the required cells grouped by lane.
type Profile struct {
	ID             string
	MatrixRevision string
	Cells          []CellID
	Lanes          map[certification.Lane][]CellID
	cases          map[string]certification.Case
}

// LoadProfile builds the named profile from the embedded matrix. Only
// cloud-launch-v1 exists today; an unknown id is refused.
func LoadProfile(id string) (*Profile, error) {
	if id != ProfileCloudLaunchV1 {
		return nil, fmt.Errorf("unknown capability profile %q", id)
	}
	m, err := certification.LoadEmbedded()
	if err != nil {
		return nil, err
	}
	return profileFromMatrix(id, m, CloudLaunchCases)
}

func profileFromMatrix(id string, m *certification.Matrix, caseIDs []string) (*Profile, error) {
	p := &Profile{ID: id, MatrixRevision: m.MatrixRevision, Lanes: map[certification.Lane][]CellID{}, cases: map[string]certification.Case{}}
	for _, caseID := range caseIDs {
		c, ok := m.Case(caseID)
		if !ok {
			return nil, fmt.Errorf("profile %s names unknown matrix case %s", id, caseID)
		}
		if !c.Required {
			return nil, fmt.Errorf("profile %s names optional matrix case %s", id, caseID)
		}
		p.cases[caseID] = c
		for _, lane := range c.Lanes {
			cell := CellID{CaseID: caseID, Lane: lane}
			p.Cells = append(p.Cells, cell)
			p.Lanes[lane] = append(p.Lanes[lane], cell)
		}
	}
	sort.Slice(p.Cells, func(i, j int) bool {
		if p.Cells[i].CaseID != p.Cells[j].CaseID {
			return p.Cells[i].CaseID < p.Cells[j].CaseID
		}
		return p.Cells[i].Lane < p.Cells[j].Lane
	})
	for lane := range p.Lanes {
		cells := p.Lanes[lane]
		sort.Slice(cells, func(i, j int) bool { return cells[i].CaseID < cells[j].CaseID })
	}
	return p, nil
}

// Case returns the matrix row for a case in the profile.
func (p *Profile) Case(id string) (certification.Case, bool) {
	c, ok := p.cases[id]
	return c, ok
}

// Contains reports whether the cell is part of the profile.
func (p *Profile) Contains(cell CellID) bool {
	for _, candidate := range p.Cells {
		if candidate == cell {
			return true
		}
	}
	return false
}

// LaneOrder lists the lanes present in the profile in stable order.
func (p *Profile) LaneOrder() []certification.Lane {
	lanes := make([]certification.Lane, 0, len(p.Lanes))
	for lane := range p.Lanes {
		lanes = append(lanes, lane)
	}
	sort.Slice(lanes, func(i, j int) bool { return lanes[i] < lanes[j] })
	return lanes
}
