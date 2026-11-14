package backup

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scenario-to-cloud/certification"
)

// WriteEvidenceReceipt writes one certification receipt as
// <dir>/<case-id>.json (lowercase). It is used by the package-lane drill
// to publish DATA-* cells; the receipt is validated against the embedded
// matrix first so a malformed receipt never lands in the evidence index.
func WriteEvidenceReceipt(dir string, receipt certification.Receipt) (string, error) {
	matrix, err := certification.LoadEmbedded()
	if err != nil {
		return "", err
	}
	receipt.SchemaVersion = 1
	if receipt.Limitations == nil {
		receipt.Limitations = []string{}
	}
	if receipt.ArtifactRefs == nil {
		receipt.ArtifactRefs = []string{}
	}
	if receipt.OperationRefs == nil {
		receipt.OperationRefs = []string{}
	}
	if receipt.Ref == "" {
		receipt.Ref = receipt.CaseID
	}
	if err := receipt.Validate(matrix); err != nil {
		return "", err
	}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, receipt.Ref+".json")
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil { //nolint:gosec // evidence receipts are shared review artifacts
		return "", err
	}
	return path, nil
}
