// Package evidencecompletenessqualification validates retained artifact-integrity
// owner tests before the rehabilitation provider credits the evidence row.
package evidencecompletenessqualification

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

const (
	ContractRow          = "evidence-completeness"
	EvidenceDir          = ".vrooli/runtime/rehabilitation-evidence"
	maxRetainedByteCount = 16 << 20
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
	"api/cmd/qualification-support.mjs",
}

// ValidateRetained validates the exact opaque artifact references admitted by
// Test Genie. The resolver is the Test Genie run-artifact byte route; paths
// inside the source tree are never used to discover a replacement receipt.
func ValidateRetained(scenarioRoot, liveBuild string, set *scenariovalidationv1.RetainedEvidenceSet, resolve func(*commonv1.EvidenceRef) ([]byte, error)) error {
	if set == nil || strings.TrimSpace(set.GetProducerReceiptId()) == "" || strings.TrimSpace(set.GetProducer()) == "" || strings.TrimSpace(set.GetTarget()) == "" || strings.TrimSpace(set.GetRunId()) == "" || strings.TrimSpace(set.GetCandidateIdentity()) == "" || strings.TrimSpace(set.GetCatalogDigest()) == "" {
		return fmt.Errorf("retained evidence identity is incomplete")
	}
	if resolve == nil || len(set.GetArtifacts()) != 3 {
		return fmt.Errorf("retained evidence artifact set is incomplete")
	}
	seen := map[string]bool{}
	var receiptBytes []byte
	var retainedByteCount int64
	contents := make(map[string][]byte, len(set.GetArtifacts()))
	for _, ref := range set.GetArtifacts() {
		if ref == nil || strings.TrimSpace(ref.GetArtifactId()) == "" || ref.GetProducer() != set.GetTarget() || !supportedOwnerArtifactKind(ref.GetKind()) || len(ref.GetChecksum()) != sha256.Size*2 || ref.GetSizeBytes() < 0 || ref.GetSizeBytes() > maxRetainedByteCount {
			return fmt.Errorf("retained evidence reference is incomplete")
		}
		if seen[ref.GetArtifactId()] {
			return fmt.Errorf("duplicate retained artifact %q", ref.GetArtifactId())
		}
		seen[ref.GetArtifactId()] = true
		data, err := resolve(ref)
		if err != nil {
			return fmt.Errorf("resolve retained artifact %s: %w", ref.GetArtifactId(), err)
		}
		if int64(len(data)) != ref.GetSizeBytes() {
			return fmt.Errorf("retained artifact size changed: %s", ref.GetArtifactId())
		}
		retainedByteCount += int64(len(data))
		if retainedByteCount > maxRetainedByteCount {
			return fmt.Errorf("retained evidence bundle exceeds %d bytes", maxRetainedByteCount)
		}
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != ref.GetChecksum() {
			return fmt.Errorf("retained artifact checksum changed: %s", ref.GetArtifactId())
		}
		if isOwnerReceipt(data) {
			if ref.GetKind() != "generic.file" {
				return fmt.Errorf("evidence-completeness receipt has catalog kind %q; want generic.file", ref.GetKind())
			}
			if receiptBytes != nil {
				return fmt.Errorf("multiple evidence-completeness receipts were selected")
			}
			receiptBytes = data
		}
		contents[ref.GetChecksum()] = data
	}
	if receiptBytes == nil {
		return fmt.Errorf("selected evidence set lacks its evidence-completeness receipt")
	}
	var candidate Receipt
	if err := json.Unmarshal(receiptBytes, &candidate); err != nil {
		return fmt.Errorf("decode retained owner receipt: %w", err)
	}
	if candidate.BuildIdentity != liveBuild {
		return fmt.Errorf("retained receipt build %q does not match live build %q", candidate.BuildIdentity, liveBuild)
	}
	return validateRetained(scenarioRoot, candidate, contents)
}

func supportedOwnerArtifactKind(kind string) bool {
	return kind == "generic.file" || kind == "command.output"
}

func isOwnerReceipt(data []byte) bool {
	var receipt Receipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return false
	}
	return receipt.SchemaVersion == 1 && receipt.ContractRow == ContractRow
}

func validateRetained(root string, r Receipt, contents map[string][]byte) error {
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
		data, ok := contents[artifact.SHA256]
		if !ok {
			return fmt.Errorf("selected retained set lacks raw owner log digest %s", artifact.SHA256)
		}
		passed, err := validateArtifactBytes(data, Artifact{SHA256: artifact.SHA256, Tests: artifact.Tests})
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

func validateArtifactBytes(data []byte, artifact Artifact) (map[string]bool, error) {
	digest := sha256.Sum256(data)
	if len(artifact.SHA256) != sha256.Size*2 || hex.EncodeToString(digest[:]) != artifact.SHA256 {
		return nil, fmt.Errorf("owner log digest mismatch")
	}
	if len(artifact.Tests) == 0 {
		return nil, fmt.Errorf("owner log has no declared tests")
	}
	expected := make(map[string]bool, len(artifact.Tests))
	for _, name := range artifact.Tests {
		if name == "" || expected[name] {
			return nil, fmt.Errorf("owner log declares an empty or duplicate test name")
		}
		expected[name] = true
	}
	passed := make(map[string]bool, len(expected))
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 2*1024*1024)
	for line := 1; scanner.Scan(); line++ {
		var event struct {
			Action string `json:"Action"`
			Test   string `json:"Test"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("decode owner test log line %d: %w", line, err)
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
