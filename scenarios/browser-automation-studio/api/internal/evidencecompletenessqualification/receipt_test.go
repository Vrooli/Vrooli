package evidencecompletenessqualification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/browser-automation-studio/internal/testutil"
)

func TestValidateRequiresCurrentSourcesExactOwnersAndHashedRawLogs(t *testing.T) {
	root := fixtureScenarioRoot(t)
	receipt := writePassingFixture(t, root)
	if err := Validate(root, receipt.BuildIdentity); err != nil {
		t.Fatalf("valid current receipt rejected: %v", err)
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
			if err := Validate(root, "sha256:fixture"); err == nil {
				t.Fatal("invalid owner receipt was accepted")
			}
		})
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
	if err := Validate(root, receipt.BuildIdentity); err == nil {
		t.Fatal("owner log without passing test events was accepted")
	}
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
