package certification

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteReceiptAppendsValidatedGroundBoundEvidence(t *testing.T) {
	dir := t.TempDir()
	caseRow, ok := mustMatrix(t).Case("RUN-01")
	if !ok {
		t.Fatal("RUN-01 missing from matrix")
	}
	receipt := Receipt{
		CaseID:          "RUN-01",
		DriverRef:       "certification-assertion:v1:RUN-01:package:qemu",
		Verdict:         VerdictPassed,
		Lane:            LanePackage,
		RequirementRefs: append([]string(nil), caseRow.RequirementRefs...),
		Candidate: Candidate{
			ReleaseDigest:       "sha256:1111111111111111111111111111111111111111111111111111111111111111",
			ConfigurationDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			ClosureDigest:       "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		},
		Target: Target{
			Ground:       TargetGroundQEMU,
			MachineID:    "local-qemu:guest-7",
			Generation:   7,
			Architecture: "amd64",
			OS:           "ubuntu-24.04",
		},
		ValidationReceiptRef:  "test-genie:run-203",
		OperationRefs:         []string{"operation:ops-8"},
		ArtifactRefs:          []string{"target-receipt://ops-8/apply?fence=9", "assertion-artifact:certification-assertion:v1:RUN-01:package:qemu:sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		Assertions:            []string{caseRow.Assertion},
		ObservedAt:            time.Date(2026, 11, 15, 18, 0, 0, 0, time.UTC),
		ownerEvidenceVerified: true,
	}

	first, err := WriteReceipt(dir, receipt)
	if err != nil {
		t.Fatalf("write first receipt: %v", err)
	}
	if !strings.Contains(filepath.Base(first), "RUN-01.package.qemu.") {
		t.Fatalf("receipt path is not ground/cell identified: %s", first)
	}
	loaded, err := LoadEvidenceDir(dir, mustMatrix(t))
	if err != nil {
		t.Fatalf("load receipt: %v", err)
	}
	if len(loaded) != 1 || loaded[0].SchemaVersion != 2 || loaded[0].Target.Generation != 7 {
		t.Fatalf("loaded receipt lost target binding: %+v", loaded)
	}
	if _, err := WriteReceipt(dir, receipt); err == nil {
		t.Fatal("identical evidence must not replace or silently deduplicate an existing append-only receipt")
	}
}

func TestWriteReceiptRejectsUnboundPassedEvidenceAndUnsafeRef(t *testing.T) {
	dir := t.TempDir()
	receipt := Receipt{
		CaseID:     "RUN-01",
		Verdict:    VerdictPassed,
		Lane:       LanePackage,
		Candidate:  Candidate{ReleaseDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		ObservedAt: time.Now().UTC(),
	}
	if _, err := WriteReceipt(dir, receipt); err == nil || !strings.Contains(err.Error(), "schema v2 requires target.ground") {
		t.Fatalf("unbound pass must fail closed, got %v", err)
	}

	receipt.Ref = "../escape"
	receipt.Target = Target{Ground: TargetGroundQEMU, MachineID: "guest", Generation: 1, Architecture: "amd64", OS: "ubuntu-24.04"}
	receipt.DriverRef = "certification-assertion:v1:RUN-01:package:qemu"
	receipt.Candidate = Candidate{
		ReleaseDigest:       "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		ConfigurationDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		ClosureDigest:       "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	}
	receipt.ValidationReceiptRef = "test-genie:run:203"
	caseRow, _ := mustMatrix(t).Case("RUN-01")
	receipt.RequirementRefs = append([]string(nil), caseRow.RequirementRefs...)
	receipt.OperationRefs = []string{"operation:ops-8"}
	receipt.ArtifactRefs = []string{"target-receipt://ops-8/apply?fence=9", "assertion-artifact:certification-assertion:v1:RUN-01:package:qemu:sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	receipt.Assertions = []string{caseRow.Assertion}
	receipt.ownerEvidenceVerified = true
	if _, err := WriteReceipt(dir, receipt); err == nil || !strings.Contains(err.Error(), "safe filename") {
		t.Fatalf("unsafe evidence path must be refused, got %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("rejected receipt wrote evidence: entries=%v err=%v", entries, err)
	}
}

func TestWriteReceiptRejectsStructurallyValidButUnauthenticatedPass(t *testing.T) {
	cell := mustMatrix(t).RequiredCells()[0]
	result := syntheticDriverResults(t, []CellID{cell})[0]
	result.Verdict = VerdictPassed
	if _, err := ReceiptFromDriverResult(mustMatrix(t), result); err == nil || !strings.Contains(err.Error(), "authenticated target receipt") {
		t.Fatalf("structural fields were accepted without both owner proofs: %v", err)
	}
}
