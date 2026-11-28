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
	for _, cell := range manifest.Cells {
		if !strings.Contains(cell.ReceiptRef, ".qemu.{target_ground}.<content-sha256>.json") {
			t.Errorf("%s receipt_ref %q must bind target ground and append-only content digest", cell.CaseID, cell.ReceiptRef)
		}
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

func TestQEMUProgramReadsFreshTargetReceiptsAndKeepsAssertionsPending(t *testing.T) {
	source, err := os.ReadFile(qemuProgramPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	wait := strings.Index(text, `scenario_to_cloud.operation.wait`)
	read := strings.Index(text, `scenario_to_cloud.operation.target_receipt`)
	classify := strings.Index(text, `if not observed["target_receipts"] or observed["target_receipt_error"]:`)
	pending := strings.Index(text, `cell["verdict"] = "pending_assertions"`)
	if wait < 0 || read < 0 || classify < 0 || pending < 0 || !(wait < read && read < classify && classify < pending) {
		t.Fatal("QEMU runner must read a fresh target receipt after wait, fail closed when missing, and keep per-cell assertions pending")
	}
	contractPath := "../../.vrooli/program-runtime/cloud-qemu-qualification.json"
	contract, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contract), `"id": "scenario-to-cloud/operation/target-receipt"`) {
		t.Fatal("QEMU contract must declare target receipt binding")
	}
}

func TestQEMUProgramResolvesComputeManagerIdentityBeforeDeploymentReads(t *testing.T) {
	source, err := os.ReadFile(qemuProgramPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	lookup := strings.Index(text, `scenario_to_cloud.qemu.get(instance_id=manager_instance_id)`)
	deploymentRead := strings.Index(text, `scenario_to_cloud.deployment.get(`)
	if lookup < 0 || deploymentRead < 0 || lookup > deploymentRead {
		t.Fatal("QEMU qualification must resolve the Compute Manager reference before reading a deployment")
	}
	if !strings.Contains(text, `target.get("computeManagerInstanceId")`) || !strings.Contains(text, `bound_manager_id != manager_instance_id`) {
		t.Fatal("QEMU qualification must bind deployment identity to the Compute Manager instance ID")
	}
	contract, err := os.ReadFile("../../.vrooli/program-runtime/cloud-qemu-qualification.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contract), `"id": "scenario-to-cloud/qemu/get"`) {
		t.Fatal("QEMU qualification contract must declare the generated manager-reference binding")
	}
}

func TestQEMUProgramAdmitsEveryMatrixLaneCellForItsGround(t *testing.T) {
	m := mustMatrix(t)
	source, err := os.ReadFile(qemuProgramPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if !strings.Contains(text, `TARGET_GROUND = "qemu"`) {
		t.Fatal("program must remain bound to the QEMU target ground")
	}
	if !strings.Contains(text, `if len(cases) != 1:`) {
		t.Fatal("one QEMU operation must not be attributed to multiple matrix cases without case-specific fixtures and assertions")
	}
	block := strings.SplitN(strings.SplitN(text, "QEMU_LANE_CASES = {\n", 2)[1], "\n}\nFAULT_CAPABILITIES", 2)
	if len(block) != 2 {
		t.Fatal("cannot locate QEMU_LANE_CASES")
	}
	rows := regexp.MustCompile(`(?m)^\s+"([^"]+)": (\[[^\n]*\]),$`).FindAllStringSubmatch(block[0], -1)
	got := map[Lane][]string{}
	for _, row := range rows {
		var cases []string
		if err := json.Unmarshal([]byte(row[2]), &cases); err != nil {
			t.Fatalf("parse QEMU %s cases: %v", row[1], err)
		}
		got[Lane(row[1])] = cases
	}
	if len(got) != len(m.LaneVocabulary) {
		t.Fatalf("QEMU program lane sets = %d, matrix vocabulary = %d", len(got), len(m.LaneVocabulary))
	}
	for _, lane := range m.LaneVocabulary {
		want := []string{}
		for _, c := range m.Cases {
			if c.Required && containsLane(c.Lanes, lane) && containsGround(m.TargetGroundsFor(c.ID, lane), TargetGroundQEMU) {
				want = append(want, c.ID)
			}
		}
		actual, ok := got[lane]
		if !ok {
			t.Errorf("QEMU runner missing lane %s", lane)
			continue
		}
		sort.Strings(want)
		sort.Strings(actual)
		if strings.Join(want, ",") != strings.Join(actual, ",") {
			t.Errorf("QEMU %s cases differ from ground-applicable matrix cells: got %v want %v", lane, actual, want)
		}
	}
}

// TestUnboundQEMUReceiptsCannotSatisfyGroundCells [REQ:STC-P0-041] P20-A06:
// historical preflight receipts lack an owned target generation, so none can
// satisfy either target-ground cell.
func TestUnboundQEMUReceiptsCannotSatisfyGroundCells(t *testing.T) {
	m, manifest := loadQEMULane(t)
	receipts, err := LoadEvidenceDir(filepath.Join("..", "..", "certification", "evidence"), m)
	if err != nil {
		t.Fatalf("load evidence: %v", err)
	}
	byCase := map[string]Receipt{}
	for _, r := range receipts {
		if r.Lane != LaneQEMU {
			continue
		}
		if r.Verdict == VerdictPassed {
			t.Fatalf("receipt %s claims a passed qemu cell without a target-bound run", r.Ref)
		}
		if r.Ref != r.CaseID && r.Ref != r.CaseID+".qemu" {
			t.Fatalf("receipt %s: qemu-lane receipts are stored as <CASE>.json or <CASE>.qemu.json", r.Ref)
		}
		if _, dup := byCase[r.CaseID]; dup {
			t.Fatalf("case %s holds two historical qemu-lane receipts", r.CaseID)
		}
		joined := strings.Join(r.Limitations, "\n")
		if !strings.Contains(joined, "EXT-06") || !strings.Contains(joined, "EXT-05") {
			t.Fatalf("receipt %s must name EXT-05 and EXT-06 in its limitations", r.Ref)
		}
		if r.Target.Ground != "" || r.Target.Generation != 0 {
			t.Fatalf("historical receipt %s unexpectedly claims an owned target: %+v", r.Ref, r.Target)
		}
		byCase[r.CaseID] = r
	}
	if len(byCase) != len(manifest.Cells) {
		t.Fatalf("%d historical qemu receipts, want one for each of %d lane cases", len(byCase), len(manifest.Cells))
	}
	readiness := Evaluate(m, receipts, testCandidate(), time.Now())
	if readiness.Ready {
		t.Fatalf("unbound qemu receipts must not certify")
	}
	qemuMissing := 0
	for _, cell := range readiness.MissingCells {
		if cell.Lane == LaneQEMU {
			qemuMissing++
		}
	}
	for _, cell := range readiness.SatisfiedCells {
		if cell.Lane == LaneQEMU {
			t.Fatalf("qemu cell %s/%s satisfied without a target-bound receipt", cell.CaseID, cell.TargetGround)
		}
	}
	if want := len(manifest.Cells) * len(m.TargetGroundVocabulary); qemuMissing != want {
		t.Fatalf("%d qemu ground cells missing, want %d because historical receipts are unbound", qemuMissing, want)
	}
}
