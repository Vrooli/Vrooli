package certification

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	qemuLanePath    = "../../certification/lanes/qemu.json"
	qemuProgramPath = "../../.vrooli/program-runtime/cloud-qemu-qualification.py"
)

func loadQEMULane(t *testing.T) (*Matrix, *LaneManifest) {
	t.Helper()
	m := mustMatrix(t)
	manifest, err := LoadLaneManifest(qemuLanePath, m)
	if err != nil {
		t.Fatalf("load qemu lane manifest: %v", err)
	}
	return m, manifest
}

// TestQEMULaneManifestMatchesMatrix [REQ:STC-P0-041] every matrix case whose
// lanes include qemu is a cell of the lane manifest and vice versa, so the
// lane cannot silently drop a required cell or invent one the matrix does
// not require.
func TestQEMULaneManifestMatchesMatrix(t *testing.T) {
	m, manifest := loadQEMULane(t)
	if manifest.Lane != LaneQEMU {
		t.Fatalf("lane = %s, want qemu", manifest.Lane)
	}
	want := []string{}
	for _, c := range m.Cases {
		for _, lane := range c.Lanes {
			if lane == LaneQEMU {
				want = append(want, c.ID)
			}
		}
	}
	got := manifest.CaseIDs()
	sort.Strings(want)
	sorted := append([]string(nil), got...)
	sort.Strings(sorted)
	if strings.Join(want, ",") != strings.Join(sorted, ",") {
		t.Fatalf("lane cells %v differ from matrix qemu cases %v", sorted, want)
	}
	if len(got) == 0 {
		t.Fatalf("the qemu lane must hold cells")
	}
}

// TestQEMULaneRefusesDrift proves the validator names a dropped cell and an
// undeclared fault instead of accepting them.
func TestQEMULaneRefusesDrift(t *testing.T) {
	m, manifest := loadQEMULane(t)
	dropped := *manifest
	dropped.Cells = manifest.Cells[1:]
	if err := dropped.Validate(m); err == nil || !strings.Contains(err.Error(), manifest.Cells[0].CaseID) {
		t.Fatalf("dropping %s must be refused by name, got %v", manifest.Cells[0].CaseID, err)
	}
	extra := *manifest
	extra.Cells = append(append([]LaneCell(nil), manifest.Cells...), LaneCell{CaseID: "AUTH-01", Fixture: "stateless-web", Architectures: []string{"amd64"}, ReceiptRef: "x"})
	if err := extra.Validate(m); err == nil || !strings.Contains(err.Error(), "AUTH-01") {
		t.Fatalf("a case without the qemu lane must be refused, got %v", err)
	}
	badFault := *manifest
	badFault.Cells = append([]LaneCell(nil), manifest.Cells...)
	badFault.Cells[0].Fault = &LaneFault{Capability: "network_partition_of_the_world"}
	if err := badFault.Validate(m); err == nil || !strings.Contains(err.Error(), "network_partition_of_the_world") {
		t.Fatalf("an undeclared fault capability must be refused, got %v", err)
	}
}

// TestQEMULaneProgramCaseListMatchesManifest keeps the qualification
// program's admission set equal to the lane manifest: the program refuses a
// case the lane does not require, and the lane cannot add a case the program
// would not admit.
func TestQEMULaneProgramCaseListMatchesManifest(t *testing.T) {
	_, manifest := loadQEMULane(t)
	source, err := os.ReadFile(qemuProgramPath)
	if err != nil {
		t.Fatalf("read program source: %v", err)
	}
	match := regexp.MustCompile(`(?m)^QEMU_CASES = (\[.*\])$`).FindSubmatch(source)
	if match == nil {
		t.Fatalf("program source must declare QEMU_CASES as a one-line JSON array")
	}
	var programCases []string
	if err := json.Unmarshal(match[1], &programCases); err != nil {
		t.Fatalf("parse QEMU_CASES: %v", err)
	}
	sort.Strings(programCases)
	want := manifest.CaseIDs()
	sort.Strings(want)
	if strings.Join(programCases, ",") != strings.Join(want, ",") {
		t.Fatalf("program QEMU_CASES %v differ from lane cells %v", programCases, want)
	}
	faultMatch := regexp.MustCompile(`(?m)^FAULT_CAPABILITIES = (\[.*\])$`).FindSubmatch(source)
	if faultMatch == nil {
		t.Fatalf("program source must declare FAULT_CAPABILITIES as a one-line JSON array")
	}
	var programFaults []string
	if err := json.Unmarshal(faultMatch[1], &programFaults); err != nil {
		t.Fatalf("parse FAULT_CAPABILITIES: %v", err)
	}
	wantFaults := []string{}
	for name := range manifest.FaultCapabilities {
		wantFaults = append(wantFaults, name)
	}
	sort.Strings(programFaults)
	sort.Strings(wantFaults)
	if strings.Join(programFaults, ",") != strings.Join(wantFaults, ",") {
		t.Fatalf("program FAULT_CAPABILITIES %v differ from lane %v", programFaults, wantFaults)
	}
}

// TestQEMULaneReceiptsAreHonestlyUnavailable [REQ:STC-P0-041] P20-A06: every
// qemu cell holds a receipt, none of them passes, every one names the
// external inputs that keep it open, and readiness for the candidate stays
// not_ready with the qemu cells listed as unavailable rather than missing.
func TestQEMULaneReceiptsAreHonestlyUnavailable(t *testing.T) {
	m, manifest := loadQEMULane(t)
	receipts, err := LoadEvidenceDir(filepath.Join("..", "..", "certification", "evidence"), m)
	if err != nil {
		t.Fatalf("load evidence: %v", err)
	}
	byCase := map[string]Receipt{}
	var candidate string
	for _, r := range receipts {
		if r.Lane != LaneQEMU {
			continue
		}
		if r.Verdict == VerdictPassed {
			t.Fatalf("receipt %s claims a passed qemu cell on a host without the lane tools", r.Ref)
		}
		if r.Ref != r.CaseID && r.Ref != r.CaseID+".qemu" {
			t.Fatalf("receipt %s: qemu-lane receipts are stored as <CASE>.json or <CASE>.qemu.json", r.Ref)
		}
		if _, dup := byCase[r.CaseID]; dup {
			t.Fatalf("case %s holds two qemu-lane receipts; keep one per cell", r.CaseID)
		}
		joined := strings.Join(r.Limitations, "\n")
		if !strings.Contains(joined, "EXT-06") || !strings.Contains(joined, "EXT-05") {
			t.Fatalf("receipt %s must name EXT-05 and EXT-06 in its limitations", r.Ref)
		}
		byCase[r.CaseID] = r
		candidate = r.Candidate.ReleaseDigest
	}
	for _, cell := range manifest.Cells {
		r, ok := byCase[cell.CaseID]
		if !ok {
			t.Fatalf("qemu cell %s has no receipt", cell.CaseID)
		}
		if r.Verdict != VerdictUnavailable {
			t.Fatalf("qemu cell %s verdict %s, want unavailable until the lane runs", cell.CaseID, r.Verdict)
		}
	}
	readiness := Evaluate(m, receipts, Candidate{ReleaseDigest: candidate}, time.Now())
	if readiness.Ready {
		t.Fatalf("unavailable qemu receipts must not certify")
	}
	// Receipts written by different owners may bind to different placeholder
	// digests; for any candidate a qemu cell is explicit (unavailable or
	// stale), never missing and never satisfied.
	explicit := map[string]bool{}
	for _, cells := range [][]Cell{readiness.UnavailableCells, readiness.StaleCells} {
		for _, cell := range cells {
			if cell.Lane == LaneQEMU {
				explicit[cell.CaseID] = true
			}
		}
	}
	for _, cell := range readiness.MissingCells {
		if cell.Lane == LaneQEMU {
			t.Fatalf("qemu cell %s is reported missing; it must be an explicit unavailable receipt", cell.CaseID)
		}
	}
	for _, cell := range readiness.SatisfiedCells {
		if cell.Lane == LaneQEMU {
			t.Fatalf("qemu cell %s is satisfied without the lane ever running", cell.CaseID)
		}
	}
	if len(explicit) != len(manifest.Cells) {
		t.Fatalf("%d qemu cells explicit, want %d", len(explicit), len(manifest.Cells))
	}
}
