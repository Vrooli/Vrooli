package evidencecompletenessqualification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/browser-automation-studio/internal/testutil"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/proto"
)

func TestValidateRequiresCurrentSourcesExactOwnersAndHashedRawLogs(t *testing.T) {
	root := fixtureScenarioRoot(t)
	receipt := writePassingFixture(t, root)
	if err := validateFixture(t, root, receipt, receipt.BuildIdentity); err != nil {
		t.Fatalf("valid retained receipt rejected: %v", err)
	}

	for _, tc := range []struct {
		name   string
		mutate func(*Receipt)
	}{
		{name: "stale build", mutate: func(r *Receipt) { r.BuildIdentity = "sha256:old-build" }},
		{name: "missing owner test", mutate: func(r *Receipt) { r.Artifacts[0].Tests = r.Artifacts[0].Tests[:len(r.Artifacts[0].Tests)-1] }},
		{name: "source path outside digest set", mutate: func(r *Receipt) { r.SourceSHA256["extra.go"] = stringsOf('a', 64) }},
		{name: "raw log digest mismatch", mutate: func(r *Receipt) { r.Artifacts[0].SHA256 = stringsOf('a', 64) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := fixtureScenarioRoot(t)
			candidate := writePassingFixture(t, root)
			tc.mutate(&candidate)
			writeReceipt(t, root, candidate)
			if err := validateFixture(t, root, candidate, "sha256:fixture"); err == nil {
				t.Fatal("invalid owner receipt was accepted")
			}
		})
	}
}

func TestValidateRetainedUsesOnlySelectedOpaqueArtifacts(t *testing.T) {
	root := fixtureScenarioRoot(t)
	receipt := writePassingFixture(t, root)
	logBytes := make(map[string][]byte)
	for _, artifact := range receipt.Artifacts {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact.Path)))
		if err != nil {
			t.Fatal(err)
		}
		logBytes[artifact.SHA256] = data
	}
	receiptBytes, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	receiptSum := sha256.Sum256(receiptBytes)
	refs := []*commonv1.EvidenceRef{{Producer: "browser-automation-studio", ArtifactId: "artifact_receipt", Kind: "generic.file", Checksum: hex.EncodeToString(receiptSum[:]), SizeBytes: int64(len(receiptBytes))}}
	byID := map[string][]byte{"artifact_receipt": receiptBytes}
	for i, artifact := range receipt.Artifacts {
		id := "artifact_log_" + string(rune('1'+i))
		data := logBytes[artifact.SHA256]
		refs = append(refs, &commonv1.EvidenceRef{Producer: "browser-automation-studio", ArtifactId: id, Kind: "command.output", Checksum: artifact.SHA256, SizeBytes: int64(len(data))})
		byID[id] = data
	}
	if len(refs) != 3 || refs[0].GetKind() != "generic.file" || refs[1].GetKind() != "command.output" || refs[2].GetKind() != "command.output" {
		t.Fatalf("owner output catalog kinds = %v, want generic.file receipt plus two command.output logs", []string{refs[0].GetKind(), refs[1].GetKind(), refs[2].GetKind()})
	}
	set := &scenariovalidationv1.RetainedEvidenceSet{ProducerReceiptId: "receipt-1", Producer: "evidence-completeness", Target: "browser-automation-studio", RunId: "run-1", CandidateIdentity: "sha256:candidate", CatalogDigest: "sha256:catalog", Artifacts: refs}
	resolver := func(ref *commonv1.EvidenceRef) ([]byte, error) {
		data, ok := byID[ref.GetArtifactId()]
		if !ok {
			return nil, os.ErrNotExist
		}
		return data, nil
	}
	if err := ValidateRetained(root, receipt.BuildIdentity, set, resolver); err != nil {
		t.Fatalf("selected evidence rejected: %v", err)
	}
	if err := validateFixture(t, root, receipt, receipt.BuildIdentity); err != nil {
		t.Fatalf("fixture source-tree check: %v", err)
	}
	changed := *set
	changed.CatalogDigest = "sha256:other"
	changed.Artifacts = append(append([]*commonv1.EvidenceRef(nil), refs...), proto.Clone(refs[1]).(*commonv1.EvidenceRef))
	if err := ValidateRetained(root, receipt.BuildIdentity, &changed, resolver); err == nil {
		t.Fatal("duplicate retained artifact was accepted")
	}
	oversized := proto.Clone(set).(*scenariovalidationv1.RetainedEvidenceSet)
	oversized.Artifacts[1].SizeBytes = maxRetainedByteCount + 1
	if err := ValidateRetained(root, receipt.BuildIdentity, oversized, resolver); err == nil {
		t.Fatal("retained artifact beyond the producer byte bound was accepted")
	}
}

func TestValidateRejectsOwnerLogsWithoutPassingTestEvents(t *testing.T) {
	root := fixtureScenarioRoot(t)
	receipt := writePassingFixture(t, root)
	artifact := &receipt.Artifacts[0]
	content := []byte(`{"Action":"fail","Test":"TestScreenshotAndOutcomeWriteFailuresBothSurvive"}` + "\n")
	path := filepath.Join(root, filepath.FromSlash(artifact.Path))
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	artifact.SHA256 = hex.EncodeToString(sum[:])
	writeReceipt(t, root, receipt)
	if err := validateFixture(t, root, receipt, receipt.BuildIdentity); err == nil {
		t.Fatal("owner log without passing test events was accepted")
	}
}

func validateFixture(t *testing.T, root string, receipt Receipt, liveBuild string) error {
	t.Helper()
	writeReceipt(t, root, receipt)
	receiptPath := filepath.Join(root, EvidenceDir, "evidence-completeness-fixture.json")
	receiptBytes, err := os.ReadFile(receiptPath)
	if err != nil {
		return err
	}
	receiptHash := sha256.Sum256(receiptBytes)
	refs := []*commonv1.EvidenceRef{{Producer: "browser-automation-studio", ArtifactId: "artifact_receipt", Kind: "generic.file", Checksum: hex.EncodeToString(receiptHash[:]), SizeBytes: int64(len(receiptBytes))}}
	dataByID := map[string][]byte{"artifact_receipt": receiptBytes}
	for index, artifact := range receipt.Artifacts {
		data, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact.Path)))
		if readErr != nil {
			return readErr
		}
		id := fmt.Sprintf("artifact_log_%d", index)
		sum := sha256.Sum256(data)
		checksum := hex.EncodeToString(sum[:])
		refs = append(refs, &commonv1.EvidenceRef{Producer: "browser-automation-studio", ArtifactId: id, Kind: "command.output", Checksum: checksum, SizeBytes: int64(len(data))})
		dataByID[id] = data
	}
	set := &scenariovalidationv1.RetainedEvidenceSet{ProducerReceiptId: "producer-receipt", Producer: "evidence-completeness", Target: "browser-automation-studio", RunId: "producer-run", CandidateIdentity: "candidate", CatalogDigest: "catalog-digest", Artifacts: refs}
	return ValidateRetained(root, liveBuild, set, func(ref *commonv1.EvidenceRef) ([]byte, error) {
		data, ok := dataByID[ref.GetArtifactId()]
		if !ok {
			return nil, os.ErrNotExist
		}
		return data, nil
	})
}

func writePassingFixture(t *testing.T, root string) Receipt {
	t.Helper()
	sources := make(map[string][]byte, len(RequiredSources))
	for _, relative := range RequiredSources {
		sources[relative] = []byte("source:" + relative)
	}
	testutil.WriteFiles(t, root, sources)
	if err := os.MkdirAll(filepath.Join(root, EvidenceDir), 0o755); err != nil {
		t.Fatal(err)
	}
	r := Receipt{SchemaVersion: 1, ContractRow: ContractRow, Result: "passed", BuildIdentity: "sha256:fixture", SourceSHA256: map[string]string{}}
	for _, source := range RequiredSources {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(source)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		r.SourceSHA256[source] = hex.EncodeToString(sum[:])
	}
	for index, tests := range [][]string{RequiredTests[:3], RequiredTests[3:]} {
		var content string
		for _, name := range tests {
			line, err := json.Marshal(map[string]string{"Action": "pass", "Test": name})
			if err != nil {
				t.Fatal(err)
			}
			content += string(line) + "\n"
		}
		relative := filepath.ToSlash(filepath.Join(EvidenceDir, "owner-test-"+string(rune('1'+index))+".jsonl"))
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(relative)), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(content))
		r.Artifacts = append(r.Artifacts, Artifact{Path: relative, SHA256: hex.EncodeToString(sum[:]), Tests: tests})
	}
	writeReceipt(t, root, r)
	return r
}

func writeReceipt(t *testing.T, root string, receipt Receipt) {
	t.Helper()
	path := filepath.Join(root, EvidenceDir, "evidence-completeness-fixture.json")
	testutil.WriteJSONFile(t, path, receipt)
}

func fixtureScenarioRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "scenarios", "browser-automation-studio")
}

func stringsOf(value byte, count int) string {
	values := make([]byte, count)
	for index := range values {
		values[index] = value
	}
	return string(values)
}
