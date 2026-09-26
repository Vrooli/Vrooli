// Package evidencecompletenessqualification validates retained artifact-integrity
// owner tests before the rehabilitation provider credits the evidence row.
package evidencecompletenessqualification

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ContractRow  = "evidence-completeness"
	EvidenceDir  = ".vrooli/runtime/rehabilitation-evidence"
	EvidenceGlob = EvidenceDir + "/evidence-completeness-*.json"
)

var RequiredSources = []string{
	"docs/internal/REFRACTOR_CONTRACT.json",
	"api/automation/execution-writer/file_writer.go",
	"api/automation/execution-writer/file_writer_test.go",
	"api/automation/execution-writer/external_artifacts.go",
	"api/automation/execution-writer/external_artifacts_test.go",
	"api/automation/execution-writer/evidence_manifest.go",
	"api/automation/execution-writer/evidence_manifest_test.go",
	"api/services/retention/retention.go",
	"api/services/retention/retention_test.go",
	"api/internal/evidencecompletenessqualification/receipt.go",
	"api/internal/evidencecompletenessqualification/receipt_test.go",
	"api/internal/testutil/testutil.go",
	"api/handlers/profilevalidation/provider.go",
	"api/handlers/profilevalidation/provider_test.go",
	".vrooli/test-genie.json",
	".vrooli/program-runtime/setpoint-read.py",
	"api/cmd/evidence-completeness-cohort/qualification.mjs",
}

var RequiredTests = []string{
	"TestScreenshotAndOutcomeWriteFailuresBothSurvive",
	"TestInlineTelemetryRemainsAttributableWhenSnapshotStorageFails",
	"TestExternalArtifactsRejectMissingOrUncommittedEvidence",
	"TestActiveEvidenceRefusesDeletionUntilExportFinishes",
}

type Receipt struct {
	SchemaVersion int               `json:"schemaVersion"`
	ContractRow   string            `json:"contractRow"`
	Result        string            `json:"result"`
	BuildIdentity string            `json:"managedBuildIdentity"`
	SourceSHA256  map[string]string `json:"sourceSha256"`
	Artifacts     []Artifact        `json:"artifacts"`
}

type Artifact struct {
	Path   string   `json:"path"`
	SHA256 string   `json:"sha256"`
	Tests  []string `json:"tests"`
}

// Validate accepts only the newest passing receipt bound to the live build,
// exact contract/source versions, four focused owners and their retained logs.
func Validate(scenarioRoot, liveBuild string) error {
	if strings.TrimSpace(liveBuild) == "" {
		return fmt.Errorf("live managed build identity is empty")
	}
	paths, err := filepath.Glob(filepath.Join(scenarioRoot, filepath.FromSlash(EvidenceGlob)))
	if err != nil {
		return err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var candidate Receipt
		if json.Unmarshal(data, &candidate) != nil || candidate.BuildIdentity != liveBuild {
			continue
		}
		if err := validate(scenarioRoot, candidate); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		return nil
	}
	return fmt.Errorf("no retained evidence-completeness receipt matches live build %q", liveBuild)
}

func validate(root string, r Receipt) error {
	if r.SchemaVersion != 1 || r.ContractRow != ContractRow || r.Result != "passed" || strings.TrimSpace(r.BuildIdentity) == "" {
		return fmt.Errorf("receipt schema, outcome, row or build identity is invalid")
	}
	for _, source := range RequiredSources {
		want, ok := r.SourceSHA256[source]
		if !ok || len(want) != sha256.Size*2 {
			return fmt.Errorf("receipt lacks source digest: %s", source)
		}
		got, err := fileSHA(filepath.Join(root, filepath.FromSlash(source)))
		if err != nil {
			return err
		}
		if want != got {
			return fmt.Errorf("source digest mismatch: %s", source)
		}
	}
	if len(r.SourceSHA256) != len(RequiredSources) {
		return fmt.Errorf("receipt has unexpected source digests")
	}
	if len(r.Artifacts) != 2 {
		return fmt.Errorf("receipt has %d raw owner logs; want 2", len(r.Artifacts))
	}
	seen := make(map[string]bool, len(RequiredTests))
	for _, artifact := range r.Artifacts {
		passed, err := validateArtifact(root, artifact)
		if err != nil {
			return err
		}
		for _, name := range artifact.Tests {
			if !contains(RequiredTests, name) || seen[name] || !passed[name] {
				return fmt.Errorf("owner test is missing, failed, unexpected or duplicated: %s", name)
			}
			seen[name] = true
		}
	}
	for _, name := range RequiredTests {
		if !seen[name] {
			return fmt.Errorf("receipt is missing owner test %s", name)
		}
	}
	return nil
}

func validateArtifact(root string, artifact Artifact) (map[string]bool, error) {
	rel := filepath.Clean(filepath.FromSlash(artifact.Path))
	if rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || !strings.HasPrefix(rel, filepath.FromSlash(EvidenceDir)+string(filepath.Separator)) {
		return nil, fmt.Errorf("owner log path escapes retained evidence: %q", artifact.Path)
	}
	path := filepath.Join(root, rel)
	got, err := fileSHA(path)
	if err != nil {
		return nil, err
	}
	if len(artifact.SHA256) != sha256.Size*2 || got != artifact.SHA256 {
		return nil, fmt.Errorf("owner log digest mismatch: %s", artifact.Path)
	}
	if len(artifact.Tests) == 0 {
		return nil, fmt.Errorf("owner log has no declared tests: %s", artifact.Path)
	}
	expected := make(map[string]bool, len(artifact.Tests))
	for _, name := range artifact.Tests {
		if name == "" || expected[name] {
			return nil, fmt.Errorf("owner log declares an empty or duplicate test name: %s", name)
		}
		expected[name] = true
	}
	passed := make(map[string]bool, len(expected))
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 2*1024*1024)
	for line := 1; scanner.Scan(); line++ {
		var event struct {
			Action string `json:"Action"`
			Test   string `json:"Test"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("decode owner test log %s line %d: %w", artifact.Path, line, err)
		}
		if expected[event.Test] {
			if event.Action == "fail" {
				return nil, fmt.Errorf("owner test failed: %s", event.Test)
			}
			if event.Action == "pass" {
				passed[event.Test] = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return passed, nil
}

func fileSHA(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
