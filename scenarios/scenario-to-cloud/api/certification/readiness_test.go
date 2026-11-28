package certification

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	candidateDigest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	otherDigest     = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
)

var testNow = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

func testCandidate() Candidate {
	return Candidate{ReleaseDigest: candidateDigest, ConfigurationDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", ClosureDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}
}

func mustMatrix(t *testing.T) *Matrix {
	t.Helper()
	m, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("load embedded: %v", err)
	}
	return m
}

func receipt(ref, caseID string, lane Lane, verdict Verdict, digest string) Receipt {
	m, err := LoadEmbedded()
	if err != nil {
		panic(err)
	}
	caseRow, ok := m.Case(caseID)
	if !ok {
		panic("unknown test case " + caseID)
	}
	driverRef, err := driverRefForMatrix(m, CellID{CaseID: caseID, Lane: lane, TargetGround: TargetGroundQEMU})
	if err != nil {
		panic(err)
	}
	return Receipt{
		SchemaVersion:        2,
		CaseID:               caseID,
		DriverRef:            driverRef,
		Verdict:              verdict,
		Lane:                 lane,
		RequirementRefs:      append([]string(nil), caseRow.RequirementRefs...),
		Candidate:            Candidate{ReleaseDigest: digest, ConfigurationDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", ClosureDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
		Target:               Target{Ground: TargetGroundQEMU, MachineID: "fixture-machine", Generation: 1, Architecture: "amd64", OS: "ubuntu-24.04"},
		OperationRefs:        []string{"operation:test"},
		ValidationReceiptRef: "test-genie:run-001",
		ArtifactRefs:         []string{"target-receipt://test/step?fence=1", "assertion-artifact:" + driverRef + ":sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		Assertions:           []string{caseRow.Assertion},
		ObservedAt:           testNow,
		Ref:                  ref,
	}
}

func cellIndex(cells []Cell) map[string]Cell {
	out := make(map[string]Cell, len(cells))
	for _, c := range cells {
		out[c.CaseID+"/"+string(c.Lane)+"/"+string(c.TargetGround)] = c
	}
	return out
}

func TestV2ReceiptRequiresExactMatrixRequirementsAndOperationReference(t *testing.T) {
	m := mustMatrix(t)
	r := receipt("scoped", "AUTH-01", LanePackage, VerdictPassed, candidateDigest)
	if err := r.Validate(m); err != nil {
		t.Fatalf("complete receipt rejected: %v", err)
	}
	if len(r.RequirementRefs) < 2 {
		t.Fatalf("AUTH-01 needs at least two matrix refs to prove order, got %v", r.RequirementRefs)
	}
	r.RequirementRefs[0], r.RequirementRefs[1] = r.RequirementRefs[1], r.RequirementRefs[0]
	if err := r.Validate(m); err == nil || !strings.Contains(err.Error(), "requirement_refs must exactly match") {
		t.Fatalf("reordered matrix requirements accepted: %v", err)
	}
	r = receipt("missing-op", "AUTH-01", LanePackage, VerdictPassed, candidateDigest)
	r.OperationRefs = nil
	if err := r.Validate(m); err == nil || !strings.Contains(err.Error(), "requires at least one operation_ref") {
		t.Fatalf("receipt without operation reference accepted: %v", err)
	}
	r = receipt("wrong-validation-source", "AUTH-01", LanePackage, VerdictPassed, candidateDigest)
	r.ValidationReceiptRef = "run:001"
	if err := r.Validate(m); err == nil || !strings.Contains(err.Error(), "must name a Test Genie receipt") {
		t.Fatalf("receipt with an untyped validation reference accepted: %v", err)
	}
	r = receipt("wrong-operation-ref", "AUTH-01", LanePackage, VerdictPassed, candidateDigest)
	r.OperationRefs = []string{"other:op-1"}
	if err := r.Validate(m); err == nil || !strings.Contains(err.Error(), "operation:<id>") {
		t.Fatalf("receipt with an untyped operation reference accepted: %v", err)
	}
}

// TestEmptyEvidenceIsNotReadyAndNamesEveryCase [REQ:STC-P0-040] (P02-A05,
// GOV-01): with no evidence every one of the 94 required cases is named.
func TestEmptyEvidenceIsNotReadyAndNamesEveryCase(t *testing.T) {
	m := mustMatrix(t)
	r := Evaluate(m, nil, testCandidate(), testNow)
	if r.Ready {
		t.Fatalf("empty evidence must not be ready")
	}
	named := map[string]bool{}
	for _, c := range r.MissingCells {
		named[c.CaseID] = true
	}
	if len(named) != expectedCaseCount {
		t.Fatalf("expected all %d cases named as missing, got %d", expectedCaseCount, len(named))
	}
	if len(r.MissingCells) != r.RequiredCells {
		t.Fatalf("expected %d missing cells, got %d", r.RequiredCells, len(r.MissingCells))
	}
	if len(r.FailedCells)+len(r.StaleCells)+len(r.UnavailableCells)+len(r.SatisfiedCells) != 0 {
		t.Fatalf("empty evidence must only produce missing cells: %+v", r)
	}
}

// TestFailedReceiptIsListedUnderFailed [REQ:STC-P0-040] (P02-A04): a
// deliberately failed receipt records a failed cell, and a later passed
// receipt for the same cell does not mask it.
func TestFailedReceiptIsListedUnderFailed(t *testing.T) {
	m := mustMatrix(t)
	receipts := []Receipt{
		receipt("run-04-package-fail", "RUN-04", LanePackage, VerdictFailed, candidateDigest),
		receipt("run-04-package-pass", "RUN-04", LanePackage, VerdictPassed, candidateDigest),
	}
	receipts[1].ObservedAt = testNow.Add(time.Hour)
	r := Evaluate(m, receipts, testCandidate(), testNow)
	if r.Ready {
		t.Fatalf("must not be ready with a failed cell")
	}
	failed := cellIndex(r.FailedCells)
	cell, ok := failed["RUN-04/package/qemu"]
	if !ok {
		t.Fatalf("RUN-04/package/qemu not listed under failed: %+v", r.FailedCells)
	}
	if cell.ReceiptRef != "run-04-package-fail" {
		t.Fatalf("failed cell must reference the failed receipt, got %q", cell.ReceiptRef)
	}
	if _, masked := cellIndex(r.SatisfiedCells)["RUN-04/package/qemu"]; masked {
		t.Fatalf("a later passed receipt must not mask a failed receipt for the same candidate")
	}
	if _, stillMissing := cellIndex(r.MissingCells)["RUN-04/package/qemu"]; stillMissing {
		t.Fatalf("failed cell must not also be reported as missing")
	}
}

// TestReceiptForAnotherReleaseDoesNotSatisfy [REQ:STC-P0-040] (GOV-02):
// evidence bound to another release digest is stale, not satisfying.
func TestReceiptForAnotherReleaseDoesNotSatisfy(t *testing.T) {
	m := mustMatrix(t)
	receipts := []Receipt{
		receipt("gov-02-api-other", "GOV-02", LaneAPI, VerdictPassed, otherDigest),
	}
	r := Evaluate(m, receipts, testCandidate(), testNow)
	if r.Ready {
		t.Fatalf("must not be ready")
	}
	if _, ok := cellIndex(r.SatisfiedCells)["GOV-02/api/qemu"]; ok {
		t.Fatalf("receipt for another digest must not satisfy the cell")
	}
	stale, ok := cellIndex(r.StaleCells)["GOV-02/api/qemu"]
	if !ok {
		t.Fatalf("expected GOV-02/api/qemu under stale cells: %+v", r.StaleCells)
	}
	if stale.ObservedDigest != otherDigest {
		t.Fatalf("stale cell must report the observed digest, got %q", stale.ObservedDigest)
	}
	if _, ok := cellIndex(r.MissingCells)["GOV-02/api/qemu"]; ok {
		t.Fatalf("stale cell must not also be reported as missing")
	}
}

func TestReceiptForDifferentConfigurationDoesNotSatisfy(t *testing.T) {
	m := mustMatrix(t)
	rcpt := receipt("gov-02-api-other-config", "GOV-02", LaneAPI, VerdictPassed, candidateDigest)
	rcpt.Candidate.ConfigurationDigest = "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	r := Evaluate(m, []Receipt{rcpt}, testCandidate(), testNow)
	cell, ok := cellIndex(r.StaleCells)["GOV-02/api/qemu"]
	if !ok || cell.ObservedCandidate.ConfigurationDigest != rcpt.Candidate.ConfigurationDigest {
		t.Fatalf("configuration-mismatched receipt must be stale and identify the observed candidate: %+v", r.StaleCells)
	}
}

// TestAllLanesSatisfiedRemovesCaseFromMissing [REQ:STC-P0-040]: a case leaves
// the missing set only when every declared lane holds a passed receipt for
// the candidate.
func TestAllLanesSatisfiedRemovesCaseFromMissing(t *testing.T) {
	m := mustMatrix(t)
	c, ok := m.Case("PLAN-05")
	if !ok {
		t.Fatalf("PLAN-05 missing from matrix")
	}
	if len(c.Lanes) < 2 {
		t.Fatalf("test needs a multi-lane case, PLAN-05 has %v", c.Lanes)
	}
	var receipts []Receipt
	for i, lane := range c.Lanes {
		if i == len(c.Lanes)-1 {
			break // leave the last lane unsatisfied first
		}
		receipts = append(receipts, receipt("plan-05-"+string(lane), "PLAN-05", lane, VerdictPassed, candidateDigest))
	}
	partial := Evaluate(m, receipts, testCandidate(), testNow)
	missing := cellIndex(partial.MissingCells)
	lastLane := c.Lanes[len(c.Lanes)-1]
	if _, ok := missing["PLAN-05/"+string(lastLane)+"/qemu"]; !ok {
		t.Fatalf("unsatisfied lane %s must remain missing", lastLane)
	}
	for _, lane := range c.Lanes[:len(c.Lanes)-1] {
		if _, ok := missing["PLAN-05/"+string(lane)+"/qemu"]; ok {
			t.Fatalf("satisfied lane %s must not be missing", lane)
		}
	}

	receipts = append(receipts, receipt("plan-05-"+string(lastLane), "PLAN-05", lastLane, VerdictPassed, candidateDigest))
	full := Evaluate(m, receipts, testCandidate(), testNow)
	for _, lane := range c.Lanes {
		if _, missing := cellIndex(full.MissingCells)["PLAN-05/"+string(lane)+"/qemu"]; missing {
			t.Fatalf("PLAN-05/%s/qemu must not be missing once its QEMU receipt passed", lane)
		}
	}
	if got := len(cellIndex(full.SatisfiedCells)); got != len(c.Lanes) {
		t.Fatalf("expected %d satisfied cells, got %d", len(c.Lanes), got)
	}
	if full.Ready {
		t.Fatalf("one satisfied case must not make the whole matrix ready")
	}
}

func TestEveryLaneCellNeedsBothGrounds(t *testing.T) {
	m := mustMatrix(t)
	c, ok := m.Case("AUTH-01")
	if !ok {
		t.Fatal("AUTH-01 missing from matrix")
	}
	var receipts []Receipt
	for _, lane := range c.Lanes {
		receipts = append(receipts, receipt("auth-01-"+string(lane)+"-qemu", c.ID, lane, VerdictPassed, candidateDigest))
	}
	r := Evaluate(m, receipts, testCandidate(), testNow)
	for _, lane := range c.Lanes {
		key := c.ID + "/" + string(lane) + "/qemu"
		if _, ok := cellIndex(r.SatisfiedCells)[key]; !ok {
			t.Fatalf("QEMU-ground receipt did not satisfy %s: %+v", key, r.SatisfiedCells)
		}
		vpsKey := c.ID + "/" + string(lane) + "/real_vps"
		if _, ok := cellIndex(r.MissingCells)[vpsKey]; !ok {
			t.Fatalf("missing VPS-ground receipt did not leave %s missing: %+v", vpsKey, r.MissingCells)
		}
	}
	if r.Ready {
		t.Fatal("QEMU-ground receipts alone must not certify the matrix")
	}

	for _, lane := range c.Lanes {
		rcpt := receipt("auth-01-"+string(lane)+"-vps", c.ID, lane, VerdictPassed, candidateDigest)
		rcpt.Target.Ground = TargetGroundRealVPS
		rcpt.DriverRef, _ = driverRefForMatrix(m, CellID{CaseID: c.ID, Lane: lane, TargetGround: TargetGroundRealVPS})
		rcpt.ArtifactRefs[1] = "assertion-artifact:" + rcpt.DriverRef + ":sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		receipts = append(receipts, rcpt)
	}
	r = Evaluate(m, receipts, testCandidate(), testNow)
	for _, lane := range c.Lanes {
		key := c.ID + "/" + string(lane) + "/real_vps"
		if _, ok := cellIndex(r.SatisfiedCells)[key]; !ok {
			t.Fatalf("VPS-ground receipt did not satisfy %s: %+v", key, r.SatisfiedCells)
		}
	}
}

func TestPassedReceiptRequiresGroundBoundEvidence(t *testing.T) {
	m := mustMatrix(t)
	r := receipt("auth-01-package-qemu", "AUTH-01", LanePackage, VerdictPassed, candidateDigest)
	if err := r.Validate(m); err != nil {
		t.Fatalf("complete target-bound receipt should validate: %v", err)
	}
	r.Target.Generation = 0
	if err := r.Validate(m); err == nil {
		t.Fatal("passed receipt without target generation must be rejected")
	}
	r = receipt("auth-01-package-qemu", "AUTH-01", LanePackage, VerdictPassed, candidateDigest)
	r.Assertions = nil
	if err := r.Validate(m); err == nil {
		t.Fatal("passed receipt without assertions must be rejected")
	}
	r = receipt("auth-01-package-qemu", "AUTH-01", LanePackage, VerdictPassed, candidateDigest)
	r.Candidate.ClosureDigest = "sha256:closure"
	if err := r.Validate(m); err == nil {
		t.Fatal("passed receipt with placeholder closure digest must be rejected")
	}
}

func TestLegacyReceiptCannotSatisfyTargetGround(t *testing.T) {
	m := mustMatrix(t)
	r := receipt("legacy-auth-01", "AUTH-01", LanePackage, VerdictPassed, candidateDigest)
	r.SchemaVersion = 1
	r.Target.Ground = ""
	r.Target.Generation = 0
	readiness := Evaluate(m, []Receipt{r}, testCandidate(), testNow)
	if readiness.Ready {
		t.Fatal("legacy ungrounded receipt must not certify the new target-ground profile")
	}
	if _, missing := cellIndex(readiness.MissingCells)["AUTH-01/package/qemu"]; !missing {
		t.Fatal("legacy receipt without a target ground must leave the QEMU cell missing")
	}
	if _, missing := cellIndex(readiness.MissingCells)["AUTH-01/package/real_vps"]; !missing {
		t.Fatal("legacy receipt without a target ground must leave the VPS cell missing")
	}
}

// TestUnavailableVerdictIsNamedNotPassed [REQ:STC-P0-040] (GOV-07): a
// skipped/unavailable/unsupported receipt for the candidate is reported as
// unavailable and never counts as passed.
func TestUnavailableVerdictIsNamedNotPassed(t *testing.T) {
	m := mustMatrix(t)
	receipts := []Receipt{
		receipt("ops-07-real-vps", "OPS-07", LaneRealVPS, VerdictUnavailable, candidateDigest),
		receipt("ops-07-qemu", "OPS-07", LaneQEMU, VerdictUnsupported, candidateDigest),
	}
	r := Evaluate(m, receipts, testCandidate(), testNow)
	unavailable := cellIndex(r.UnavailableCells)
	for _, key := range []string{"OPS-07/real_vps/qemu", "OPS-07/qemu/qemu"} {
		if _, ok := unavailable[key]; !ok {
			t.Fatalf("expected %s under unavailable cells: %+v", key, r.UnavailableCells)
		}
	}
	if _, ok := cellIndex(r.MissingCells)["OPS-07/qemu/real_vps"]; !ok {
		t.Fatalf("a QEMU-ground receipt cannot satisfy the VPS-ground cell: %+v", r.MissingCells)
	}
	if len(r.SatisfiedCells) != 0 {
		t.Fatalf("no cell may be satisfied by a non-passed verdict")
	}
}

func TestEmptyCandidateDigestCannotBeReady(t *testing.T) {
	m := mustMatrix(t)
	receipts := []Receipt{receipt("x", "AUTH-01", LanePackage, VerdictPassed, "")}
	if err := receipts[0].Validate(m); err == nil {
		t.Fatalf("passed receipt without release digest must be rejected")
	}
	r := Evaluate(m, nil, Candidate{}, testNow)
	if r.Ready || len(r.MissingCells) != r.RequiredCells {
		t.Fatalf("empty candidate must name every cell as missing")
	}
}

// TestReportLoadsEvidenceDirAndRefusesMissing [REQ:STC-P0-040] exercises the
// JSON report path end to end from a directory of receipts.
func TestReportLoadsEvidenceDirAndRefusesMissing(t *testing.T) {
	dir := t.TempDir()
	r := receipt("", "AUTH-01", LanePackage, VerdictFailed, candidateDigest)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "auth-01-package.json"), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err := Report(dir, testCandidate(), testNow)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	var decoded Readiness
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if decoded.Ready {
		t.Fatalf("report must not be ready")
	}
	failed := cellIndex(decoded.FailedCells)
	if cell, ok := failed["AUTH-01/package/qemu"]; !ok || cell.ReceiptRef != "auth-01-package" {
		t.Fatalf("expected AUTH-01/package/qemu failed with receipt ref from filename, got %+v", decoded.FailedCells)
	}
	if decoded.NotReadyCount() != decoded.RequiredCells {
		t.Fatalf("every required cell must be accounted for: %d vs %d", decoded.NotReadyCount(), decoded.RequiredCells)
	}

	// A missing evidence directory is explicit absence, not an error.
	out, err = Report(filepath.Join(dir, "does-not-exist"), testCandidate(), testNow)
	if err != nil {
		t.Fatalf("missing dir must be treated as empty evidence: %v", err)
	}
	if err := json.Unmarshal(out, &decoded); err != nil || decoded.Ready {
		t.Fatalf("missing dir must yield not ready (err=%v ready=%v)", err, decoded.Ready)
	}
}

func TestLoadEvidenceDirRejectsMalformedReceipt(t *testing.T) {
	m := mustMatrix(t)
	dir := t.TempDir()
	bad := `{"schema_version":1,"case_id":"NOPE-99","verdict":"passed","lane":"package","candidate":{"release_digest":"sha256:x"},"observed_at":"2026-09-09T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte(bad), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadEvidenceDir(dir, m); err == nil {
		t.Fatalf("unknown case id must be rejected, not silently dropped")
	}
}

// TestScenarioEvidenceDirectoryIsEmptyAtPhaseTwo records the honest baseline:
// the checked-in evidence directory holds no receipts, so the matrix reports
// every cell missing (P02-A01).
func TestScenarioEvidenceDirectoryIsEmptyAtPhaseTwo(t *testing.T) {
	m := mustMatrix(t)
	receipts, err := LoadEvidenceDir(filepath.Join("..", "..", "certification", "evidence"), m)
	if err != nil {
		t.Fatalf("load scenario evidence: %v", err)
	}
	r := Evaluate(m, receipts, testCandidate(), testNow)
	if r.Ready {
		t.Fatalf("checked-in evidence must not certify an arbitrary candidate")
	}
}
