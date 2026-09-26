package passivefidelityqualification

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestValidateComposesCurrentManagedCrashAndSemanticsOwners(t *testing.T) {
	root := newEvidenceFixture(t)
	if err := Validate(root, "sha256:test-build"); err != nil {
		t.Fatalf("current passive-fidelity evidence rejected: %v", err)
	}
}

func TestValidateRejectsStaleBuildAndMissingSemanticAssertions(t *testing.T) {
	t.Run("stale managed build", func(t *testing.T) {
		root := newEvidenceFixture(t)
		if err := Validate(root, "sha256:other-build"); err == nil {
			t.Fatal("accepted evidence from another managed build")
		}
	})
	t.Run("missing core semantic", func(t *testing.T) {
		root := newEvidenceFixture(t)
		path := filepath.Join(root, EvidenceDir, "passive-fidelity-semantics-w188-test.json")
		evidence := repocontracttest.ReadJSONFileInto[semanticsEvidence](t, path)
		evidence.Cases[0].Assertions = []string{"one independent click effect"}
		testutil.WriteJSONFile(t, path, evidence)
		if err := Validate(root, "sha256:test-build"); err == nil {
			t.Fatal("accepted incomplete event semantics")
		}
	})
}

func TestValidateRejectsStorageEscapeAndDuplicateCrashRetry(t *testing.T) {
	t.Run("primary database write", func(t *testing.T) {
		root := newEvidenceFixture(t)
		path := filepath.Join(root, EvidenceDir, "passive-fidelity-managed-test.json")
		evidence := repocontracttest.ReadJSONFileInto[managedEvidence](t, path)
		evidence.OwnerReceipt.StorageIsolation.PrimaryRequestsDuringTestMode = 1
		testutil.WriteJSONFile(t, path, evidence)
		if err := Validate(root, "sha256:test-build"); err == nil {
			t.Fatal("accepted a managed cohort that wrote to primary storage")
		}
	})
	t.Run("duplicate after crash retry", func(t *testing.T) {
		root := newEvidenceFixture(t)
		path := filepath.Join(root, EvidenceDir, "passive-fidelity-process-crash-w188-test.json")
		evidence := repocontracttest.ReadJSONFileInto[crashEvidence](t, path)
		evidence.Owner.TotalAfterRetry++
		testutil.WriteJSONFile(t, path, evidence)
		if err := Validate(root, "sha256:test-build"); err == nil {
			t.Fatal("accepted duplicated crash-recovery event")
		}
	})
}

func newEvidenceFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	sourceRoot := filepath.Clean("../../../")
	allSources := append(append(append([]string{}, managedSources...), crashSources...), semanticsSources...)
	seen := make(map[string]bool)
	uniqueSources := make([]string, 0, len(allSources))
	for _, rel := range allSources {
		if seen[rel] {
			continue
		}
		seen[rel] = true
		uniqueSources = append(uniqueSources, rel)
	}
	testutil.CopyFiles(t, sourceRoot, root, uniqueSources)
	contractSHA, err := fileSHA(filepath.Join(root, "docs/internal/REFRACTOR_CONTRACT.json"))
	if err != nil {
		t.Fatal(err)
	}

	managed := managedEvidence{ContractRow: ContractRow, Result: "passed", SourceSHA256: fixtureHashes(t, root, managedSources)}
	managed.OwnerReceipt.ContractRow = ContractRow
	managed.OwnerReceipt.ManagedBuildIdentityBefore = "sha256:test-build"
	managed.OwnerReceipt.ManagedBuildIdentityAfter = "sha256:test-build"
	managed.OwnerReceipt.Actions = 10000
	managed.OwnerReceipt.FixtureEffects = 10000
	managed.OwnerReceipt.UniqueJournalIDs = 10000
	managed.OwnerReceipt.StrictlyIncreasingJournal = true
	managed.OwnerReceipt.AppliedInputReceipts = 10000
	managed.OwnerReceipt.StrictlyIncreasingApplied = true
	managed.OwnerReceipt.StorageIsolation.RoutedTestPool = true
	managed.OwnerReceipt.StorageIsolation.TestPoolRequests = 10
	managed.OwnerReceipt.StorageIsolation.TemporaryDatabase = true
	managedRawPath := ".vrooli/runtime/rehabilitation-evidence/passive-fidelity-managed-owner-test.json"
	testutil.WriteJSONFile(t, filepath.Join(root, managedRawPath), managed.OwnerReceipt)
	managed.OwnerArtifact = artifactRef{Path: managedRawPath, SHA256: fixtureSHA(t, filepath.Join(root, managedRawPath))}
	testutil.WriteJSONFile(t, filepath.Join(root, EvidenceDir, "passive-fidelity-managed-test.json"), managed)

	crash := crashEvidence{ContractRow: ContractRow, Result: "passed", Tests: "1/1", OwnerTest: "TestJournalSameIDRetryRecoversAcrossServiceProcessDeath", SourceSHA256: fixtureHashes(t, root, crashSources)}
	crash.Owner.Case = "recording-service-process-death"
	crash.Owner.ActionsBeforeCrash = 10000
	crash.Owner.CommittedBeforeAcknowledgment = true
	crash.Owner.ChildTerminatedAbruptly = true
	crash.Owner.ReopenedTotal = 10001
	crash.Owner.RetriedSameEventID = true
	crash.Owner.TotalAfterRetry = 10001
	crash.Owner.ExpectedPrefixIntactAndOrdered = true
	crashRawPath := ".vrooli/runtime/rehabilitation-evidence/passive-fidelity-crash-test.json"
	testutil.WriteJSONFile(t, filepath.Join(root, crashRawPath), crash.Owner)
	crash.OwnerArtifact = artifactRef{Path: crashRawPath, SHA256: fixtureSHA(t, filepath.Join(root, crashRawPath))}
	testutil.WriteJSONFile(t, filepath.Join(root, EvidenceDir, "passive-fidelity-process-crash-w188-test.json"), crash)

	semantics := semanticsEvidence{ContractRow: ContractRow, Result: "passed", Tests: "3/3", OwnerTests: []string{
		"[CRITICAL] should capture all core event types in single session",
		"[CRITICAL] should capture navigation events",
		"[CRITICAL] should continue capturing events after navigation",
	}, SourceSHA256: fixtureHashes(t, root, semanticsSources)}
	semantics.Cases = []struct {
		Case        string   `json:"case"`
		ActionTypes []string `json:"actionTypes"`
		Assertions  []string `json:"assertions"`
	}{
		{Case: "core-events", ActionTypes: []string{"click", "type", "scroll"}, Assertions: []string{"one independent click effect", "typed value and input selector retained", "positive scroll delta retained", "unique increasing sequence numbers"}},
		{Case: "navigation", ActionTypes: []string{"click", "navigate"}, Assertions: []string{"click triggering navigation retained", "navigation entry identifies /page-2"}},
		{Case: "capture-after-navigation", ActionTypes: []string{"click", "navigate"}, Assertions: []string{"pre-navigation click retained", "navigation completed", "post-navigation click retained"}},
	}
	semanticsRawPath := "playwright-driver/.vrooli/runtime/rehabilitation-evidence/passive-fidelity-semantics-test.jsonl"
	var rawSemantics []map[string]any
	for _, observation := range semantics.Cases {
		rawSemantics = append(rawSemantics, map[string]any{"case": observation.Case, "actionTypes": observation.ActionTypes, "assertions": observation.Assertions})
	}
	writeFixtureJSONLines(t, filepath.Join(root, semanticsRawPath), rawSemantics)
	semantics.OwnerArtifact = artifactRef{Path: semanticsRawPath, SHA256: fixtureSHA(t, filepath.Join(root, semanticsRawPath))}
	testutil.WriteJSONFile(t, filepath.Join(root, EvidenceDir, "passive-fidelity-semantics-w188-test.json"), semantics)
	if contractSHA == "" {
		t.Fatal("fixture contract digest is empty")
	}
	return root
}

func fixtureHashes(t *testing.T, root string, paths []string) map[string]string {
	t.Helper()
	result := make(map[string]string, len(paths))
	for _, rel := range paths {
		digest, err := fileSHA(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		result[rel] = digest
	}
	return result
}

func fixtureSHA(t *testing.T, path string) string {
	t.Helper()
	digest, err := fileSHA(path)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func writeFixtureJSONLines(t *testing.T, path string, values []map[string]any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	encoder := json.NewEncoder(file)
	for _, value := range values {
		if err := encoder.Encode(value); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
