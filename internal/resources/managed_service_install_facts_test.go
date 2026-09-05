package resources

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// Feature: a changed host is diagnosed as a changed host
//
//	As an operator whose GPU appeared after boot
//	I want "the host changed, re-acquire" instead of "checksum mismatch"
//	So that a resource with intact bytes and a moved host has a named cause and
//	a command that fixes it.

func stagedFactsArtifact(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "reranker_linux_amd64")
	if err := os.WriteFile(path, []byte("staged bytes"), 0o755); err != nil {
		t.Fatalf("write staged artifact: %v", err)
	}
	return path
}

// Scenario: an artifact staged under different facts reports needs_reacquire.
func TestCheckFactDriftNamesBothFactSetsAndTheRemediation(t *testing.T) {
	// Given an artifact staged on a host with no CUDA device
	path := stagedFactsArtifact(t)
	installFacts := binaryfetch.Facts{"os": "linux", "arch": "amd64", "accel.backends": "cpu"}
	cpuTarget := binaryfetch.AcquisitionTarget{ArtifactSHA256: strings.Repeat("a", 64)}
	if err := writeInstallFacts(path, "reranker", installFacts, cpuTarget, resourcedeployment.ServiceArtifact{}, time.Now()); err != nil {
		t.Fatalf("writeInstallFacts: %v", err)
	}

	// When the host later reports CUDA and the resolver selects the GPU target
	currentFacts := binaryfetch.Facts{"os": "linux", "arch": "amd64", "accel.backends": "cuda,cpu", "accel.cuda_compute": "8.9"}
	gpuTarget := binaryfetch.AcquisitionTarget{ArtifactSHA256: strings.Repeat("b", 64)}
	err := checkFactDrift(path, "reranker", currentFacts, gpuTarget)

	// Then it is fact drift, not a corrupt artifact
	if !errors.Is(err, ErrFactDrift) {
		t.Fatalf("checkFactDrift() = %v, want ErrFactDrift", err)
	}
	var drift *FactDriftError
	if !errors.As(err, &drift) {
		t.Fatalf("checkFactDrift() = %T, want *FactDriftError", err)
	}
	// And the message names what actually changed on the host
	message := err.Error()
	for _, want := range []string{"accel.backends", `"cpu"`, `"cuda,cpu"`, "the host changed"} {
		if !strings.Contains(message, want) {
			t.Fatalf("message = %q, want it to contain %q", message, want)
		}
	}
	// And the message never blames the bytes
	if strings.Contains(strings.ToLower(message), "corrupt") && !strings.Contains(message, "not corrupt") {
		t.Fatalf("message = %q, want it to say the bytes are not corrupt", message)
	}
	// And it carries the exact command that repairs it
	if got := drift.Remediation(); got != "vrooli resource install reranker --reacquire" {
		t.Fatalf("Remediation() = %q, want the reacquire command", got)
	}
	if !strings.Contains(message, drift.Remediation()) {
		t.Fatalf("message = %q, want it to contain the remediation command", message)
	}
}

// Scenario: unchanged facts are not drift, so a digest mismatch stays a digest
// mismatch.
func TestCheckFactDriftIgnoresAnUnchangedHost(t *testing.T) {
	// Given an artifact staged under a fact set
	path := stagedFactsArtifact(t)
	facts := binaryfetch.Facts{"os": "linux", "arch": "amd64", "accel.backends": "cuda,cpu"}
	target := binaryfetch.AcquisitionTarget{ArtifactSHA256: strings.Repeat("c", 64)}
	if err := writeInstallFacts(path, "reranker", facts, target, resourcedeployment.ServiceArtifact{}, time.Now()); err != nil {
		t.Fatalf("writeInstallFacts: %v", err)
	}

	// When the resolver selects the same target on the same host
	// Then there is no drift, and a corrupt file will surface as a checksum
	// mismatch through the existing path
	if err := checkFactDrift(path, "reranker", facts, target); err != nil {
		t.Fatalf("checkFactDrift() = %v, want nil for an unchanged host", err)
	}
}

// Scenario: an artifact with no record is not drift.
//
// Every artifact staged before this record existed has none, and absence of
// evidence must not be reported as a changed host.
func TestCheckFactDriftTreatsAMissingRecordAsNoEvidence(t *testing.T) {
	// Given a staged artifact with no install-facts sidecar
	path := stagedFactsArtifact(t)

	// When the drift check runs against a different target
	err := checkFactDrift(path, "reranker", binaryfetch.Facts{"os": "linux"}, binaryfetch.AcquisitionTarget{ArtifactSHA256: strings.Repeat("d", 64)})
	// Then nothing is claimed
	if err != nil {
		t.Fatalf("checkFactDrift() = %v, want nil when there is no record to compare against", err)
	}
}

// Scenario: the record round-trips and lives beside the artifact.
func TestInstallFactsRoundTripBesideTheArtifact(t *testing.T) {
	// Given a staged artifact and its recorded facts
	path := stagedFactsArtifact(t)
	facts := binaryfetch.Facts{"os": "linux", "arch": "amd64", "accel.cuda_compute": "8.9"}
	target := binaryfetch.AcquisitionTarget{ArtifactSHA256: strings.Repeat("e", 64), Layout: "file"}
	if err := writeInstallFacts(path, "whisper", facts, target, resourcedeployment.ServiceArtifact{Version: "1.9.2"}, time.Now()); err != nil {
		t.Fatalf("writeInstallFacts: %v", err)
	}

	// When the record is read back
	record, ok, err := readInstallFacts(path)
	if err != nil || !ok {
		t.Fatalf("readInstallFacts() = ok %v err %v", ok, err)
	}

	// Then every field survives
	if record.Resource != "whisper" || record.ArtifactSHA256 != target.ArtifactSHA256 || record.Layout != "file" || record.Version != "1.9.2" {
		t.Fatalf("record = %+v, want the written values", record)
	}
	if record.Facts["accel.cuda_compute"] != "8.9" {
		t.Fatalf("record facts = %v, want the accelerator fact preserved", record.Facts)
	}
	// And the record sits beside the artifact, so removing the artifact
	// directory removes it too and a stale record cannot outlive the bytes
	if got := filepath.Dir(installFactsPath(path)); got != filepath.Dir(path) {
		t.Fatalf("record dir = %q, want %q", got, filepath.Dir(path))
	}
}

// Scenario: the drift message enumerates every kind of fact change.
func TestChangedFactsDescribesAppearanceDisappearanceAndChange(t *testing.T) {
	// Given a fact set where one value moved, one appeared and one vanished
	installed := map[string]string{"accel.backends": "cpu", "gone": "yes"}
	current := map[string]string{"accel.backends": "cuda,cpu", "accel.cuda_compute": "8.9"}

	// When the difference is rendered
	changes := changedFacts(installed, current)

	// Then each kind of change is named, in a stable order
	joined := strings.Join(changes, " | ")
	for _, want := range []string{
		`accel.backends was "cpu" and is now "cuda,cpu"`,
		`accel.cuda_compute was absent and is now "8.9"`,
		`gone was "yes" and is now absent`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("changes = %q, want it to contain %q", joined, want)
		}
	}
}
