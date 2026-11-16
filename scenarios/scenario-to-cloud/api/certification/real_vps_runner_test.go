package certification

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const realVPSRunnerPath = "../../.vrooli/program-runtime/cloud-real-vps-qualification.py"

// TestRealVPSRunnerCaseSetsMatchMatrix keeps runner admission tied to every
// lane cell that the matrix explicitly assigns to the real_vps ground.
func TestRealVPSRunnerCaseSetsMatchMatrix(t *testing.T) {
	matrix := mustMatrix(t)
	source, err := os.ReadFile(realVPSRunnerPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if !strings.Contains(text, "REAL_VPS_LANE_CASES = {\n") {
		t.Fatal("runner must declare REAL_VPS_LANE_CASES")
	}
	block := strings.SplitN(strings.SplitN(text, "REAL_VPS_LANE_CASES = {\n", 2)[1], "\n}\nFAULT_CAPABILITIES", 2)
	if len(block) != 2 {
		t.Fatal("cannot locate real VPS lane case map")
	}
	rows := regexp.MustCompile(`(?m)^\s+"([^"]+)": (\[[^\n]*\]),$`).FindAllStringSubmatch(block[0], -1)
	got := map[Lane][]string{}
	for _, row := range rows {
		var cases []string
		if err := json.Unmarshal([]byte(row[2]), &cases); err != nil {
			t.Fatalf("parse %s cases: %v", row[1], err)
		}
		got[Lane(row[1])] = cases
	}
	if len(got) != len(matrix.LaneVocabulary) {
		t.Fatalf("runner lane sets = %d, matrix vocabulary = %d", len(got), len(matrix.LaneVocabulary))
	}
	for _, lane := range matrix.LaneVocabulary {
		want := []string{}
		for _, c := range matrix.Cases {
			if !c.Required || !containsLane(c.Lanes, lane) {
				continue
			}
			if containsGround(matrix.TargetGroundsFor(c.ID, lane), TargetGroundRealVPS) {
				want = append(want, c.ID)
			}
		}
		actual, ok := got[lane]
		if !ok {
			t.Errorf("runner missing lane %s", lane)
			continue
		}
		sort.Strings(want)
		sort.Strings(actual)
		if strings.Join(want, ",") != strings.Join(actual, ",") {
			t.Errorf("runner %s cases differ from matrix: got %v want %v", lane, actual, want)
		}
	}
}

// TestRealVPSRunnerFailsClosed verifies that the independently governed VPS
// runner cannot admit production, QEMU, target creation, or fabricated passes.
func TestRealVPSRunnerFailsClosed(t *testing.T) {
	data, err := os.ReadFile(realVPSRunnerPath)
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	required := []string{`TARGET_GROUND = "real_vps"`, `OWNED_TARGET_PREFIX = "disposable-vps:"`, `environment_ref.startswith("cl2-staging:")`, `if not authorization_ref:`, `if len(cases) != 1:`, `scenario_to_cloud.deployment.apply`, `scenario_to_cloud.operation.target_receipt`, `validate_target_receipt(row`, `"pending_assertions"`, `"pending"`}
	for _, needle := range required {
		if !strings.Contains(source, needle) {
			t.Errorf("runner missing fail-closed contract %q", needle)
		}
	}
	if strings.Contains(source, `"verdict": "passed"`) || strings.Contains(source, `certification.WriteReceipt(`) {
		t.Fatal("runner must not manufacture or persist passed receipts")
	}
}

// TestRealVPSRunnerReadsFreshTargetReceiptBeforeClaimingVerifiedJourney
// keeps operation/step receipt reads distinct from the pending per-cell
// assertion state. Receipt collection follows the operation wait, and missing
// receipts fail closed.
func TestRealVPSRunnerReadsFreshTargetReceiptBeforeClaimingVerifiedJourney(t *testing.T) {
	data, err := os.ReadFile(realVPSRunnerPath)
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	wait := strings.Index(source, `scenario_to_cloud.operation.wait`)
	read := strings.Index(source, `scenario_to_cloud.operation.target_receipt`)
	classify := strings.Index(source, `if not observed["target_receipts"] or observed["target_receipt_error"]:`)
	pending := strings.Index(source, `cell["verdict"] = "pending_assertions"`)
	if wait < 0 || read < 0 || classify < 0 || pending < 0 || !(wait < read && read < classify && classify < pending) {
		t.Fatal("runner must read and validate a fresh target receipt after wait, fail closed when absent, and preserve pending per-cell assertions")
	}
	contract, err := os.ReadFile("../../.vrooli/program-runtime/cloud-real-vps-qualification.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contract), `"id": "scenario-to-cloud/operation/target-receipt"`) {
		t.Fatal("runner contract must declare its target-owner receipt read binding")
	}
}

// TestRealVPSRunnerRequiresBridgeIdentityFence ensures the target identity
// validator is called before plan/apply and checks all immutable enrollment
// identity fields. Behavioral cases execute the same Python helper below.
func TestRealVPSRunnerRequiresBridgeIdentityFence(t *testing.T) {
	data, err := os.ReadFile(realVPSRunnerPath)
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	required := []string{
		`def validate_bridge_target(target, expected_machine_id, expected_generation):`,
		`target.get("transport") or "") != "bridge"`,
		`target.get("machineId") or ""`,
		`target.get("nodeId") or ""`,
		`target.get("enrollmentGeneration")`,
		`observed_generation != expected_generation`,
		`validate_bridge_target(target_ref_data, target_ref, expected_target_generation)`,
	}
	for _, needle := range required {
		if !strings.Contains(source, needle) {
			t.Errorf("runner missing Bridge identity fence %q", needle)
		}
	}
	fence := strings.Index(source, `validate_bridge_target(target_ref_data`)
	plan := strings.Index(source, `scenario_to_cloud.deployment.plan`)
	if fence < 0 || plan < 0 || fence > plan {
		t.Fatal("Bridge identity fence must execute before compiling a plan")
	}
	if strings.Contains(source, `target.get("transport") or "") != "ssh"`) {
		t.Fatal("SSH transport must not be accepted for real-VPS qualification")
	}
}
