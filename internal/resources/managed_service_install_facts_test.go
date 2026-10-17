package resources

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestDiscardStagedArtifactPreservesActiveOrUncertainResource(t *testing.T) {
	for _, test := range []struct {
		name           string
		state          string
		ownershipError bool
		want           string
	}{
		{"active selected version", "live", false, "vrooli resource stop fixture"},
		{"active older version", "older", false, "vrooli resource stop fixture"},
		{"unreadable ownership", "live", true, "ownership"},
		{"malformed state", "malformed", false, "state"},
	} {
		t.Run(test.name, func(t *testing.T) {
			controller, path, supervisor := discardArtifactFixture(t)
			withOwnershipFacts(t, func(int) (map[string]string, error) {
				if test.ownershipError {
					return nil, errors.New("ownership inspection denied")
				}
				return map[string]string{managedServiceOwnershipTokenEnv: "owned"}, nil
			}, executableAt("/unrelated"))
			state := ManagedServiceState{InstanceID: "owned-fixture", PID: os.Getpid(), ArtifactPath: path, ArtifactVersion: "1.0.0", ArtifactSHA256: strings.Repeat("a", 64), OwnershipTokenHash: managedServiceTokenHash("owned")}
			if test.state == "older" {
				state.ArtifactPath = filepath.Join(filepath.Dir(path), "old-version", "server")
				state.ArtifactVersion = "0.9.0"
			}
			if err := supervisor.writeState(state); err != nil {
				t.Fatal(err)
			}
			if test.state == "malformed" {
				if err := os.WriteFile(supervisor.statePath, []byte("{broken"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			var progress bytes.Buffer
			err := controller.DiscardStagedArtifact("fixture", &progress)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Errorf("DiscardStagedArtifact() = %v, want refusal containing %q", err, test.want)
			}
			if progress.Len() != 0 {
				t.Errorf("destructive progress before refusal: %q", progress.String())
			}
			for _, file := range []string{path, installFactsPath(path)} {
				body, err := os.ReadFile(file)
				if err != nil || string(body) != "keep" {
					t.Errorf("artifact evidence changed: %s = %q, %v", file, body, err)
				}
			}
		})
	}
}

func TestDiscardStagedArtifactAllowsStoppedOrAbsentResource(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent", true: "stopped"}[recorded], func(t *testing.T) {
			controller, path, supervisor := discardArtifactFixture(t)
			if recorded {
				if err := supervisor.writeState(ManagedServiceState{InstanceID: "stopped-fixture", PID: 0, ArtifactPath: path, ArtifactSHA256: strings.Repeat("a", 64), OwnershipTokenHash: managedServiceTokenHash("owned"), ArtifactVersion: "0.9.0"}); err != nil {
					t.Fatal(err)
				}
			}
			if err := controller.DiscardStagedArtifact("fixture", nil); err != nil {
				t.Fatal(err)
			}
			for _, file := range []string{path, installFactsPath(path)} {
				if _, err := os.Stat(file); !os.IsNotExist(err) {
					t.Errorf("stopped artifact not removed: %s: %v", file, err)
				}
			}
		})
	}
}

func discardArtifactFixture(t *testing.T) (*Controller, string, *ManagedServiceSupervisor) {
	t.Helper()
	root, home := t.TempDir(), t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", t.TempDir())
	manifest := testresource.ResourceManifest("fixture", testresource.WithResourceDriver("managed-service"))
	manifest.ManagedService.Acquisition = &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: "https://example.invalid/server", SHA256: strings.Repeat("a", 64)}}}
	testresource.WriteResourceManifest(t, root, "fixture", manifest)
	controller := NewController(root, home)
	loaded, err := controller.ResourceManifest("fixture")
	if err != nil {
		t.Fatal(err)
	}
	path, err := managedServiceArtifactPath(controller, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{path, installFactsPath(path)} {
		if err := os.WriteFile(file, []byte("keep"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	supervisor, _, err := managedServiceSupervisorFor("fixture")
	if err != nil {
		t.Fatal(err)
	}
	return controller, path, supervisor
}

func TestDiscardStagedArtifactRechecksAfterConcurrentLifecycle(t *testing.T) {
	controller, path, supervisor := discardArtifactFixture(t)
	withOwnershipFacts(t, func(int) (map[string]string, error) {
		return map[string]string{managedServiceOwnershipTokenEnv: "owned"}, nil
	}, executableAt("/unrelated"))
	release, err := acquireManagedServiceLifecycleLock(context.Background(), "fixture")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	done := make(chan error, 1)
	go func() { done <- controller.DiscardStagedArtifact("fixture", nil) }()
	select {
	case err := <-done:
		t.Fatalf("discard escaped an active lifecycle transaction: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err := supervisor.writeState(ManagedServiceState{InstanceID: "owned-fixture", PID: os.Getpid(), ArtifactPath: path, ArtifactVersion: "1.0.0", ArtifactSHA256: strings.Repeat("a", 64), OwnershipTokenHash: managedServiceTokenHash("owned")}); err != nil {
		t.Fatal(err)
	}
	release()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "vrooli resource stop fixture") {
			t.Fatalf("discard failed to recheck newly active service: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("discard did not return after lifecycle release")
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "keep" {
		t.Fatalf("newly active artifact changed: %q, %v", body, err)
	}
}

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

func TestWithOperatorAccelPreferenceCopiesFactsAndAddsPreference(t *testing.T) {
	original := binaryfetch.Facts{"os": "linux"}
	got := withOperatorAccelPreference(original, "force_cpu")
	if got["os"] != "linux" || got[binaryfetch.FactOperatorAccelPreference] != "force_cpu" {
		t.Fatalf("enriched facts = %#v", got)
	}
	got["os"] = "changed"
	if original["os"] != "linux" {
		t.Fatal("fact enrichment mutated the caller's map")
	}
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
