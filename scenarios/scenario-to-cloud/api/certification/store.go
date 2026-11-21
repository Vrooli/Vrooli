package certification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteReceipt appends one validated, target-bound receipt to the evidence
// directory. A caller cannot replace an earlier verdict: each receipt gets a
// stable content-derived filename and creation uses O_EXCL. Pass receipts are
// accepted only when schema-v2 validation binds them to a real target, an
// exact matrix requirement row, an operation reference, and a complete
// release/configuration/closure candidate.
func WriteReceipt(dir string, receipt Receipt) (string, error) {
	matrix, err := LoadEmbedded()
	if err != nil {
		return "", err
	}
	prepared, prefix, err := prepareReceiptForWrite(matrix, receipt)
	if err != nil {
		return "", err
	}
	raw, err := marshalReceipt(prepared)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	filename := prefix + "." + hex.EncodeToString(sum[:]) + ".json"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create evidence directory: %w", err)
	}
	path := filepath.Join(dir, filename)
	if err := appendReceiptFile(path, filename, raw); err != nil {
		return "", err
	}
	return path, nil
}

func prepareReceiptForWrite(matrix *Matrix, receipt Receipt) (Receipt, string, error) {
	receipt.SchemaVersion = 2
	receipt.Limitations = nonNilStrings(receipt.Limitations)
	receipt.ArtifactRefs = nonNilStrings(receipt.ArtifactRefs)
	receipt.OperationRefs = nonNilStrings(receipt.OperationRefs)
	if err := receipt.Validate(matrix); err != nil {
		return Receipt{}, "", err
	}
	if receipt.Verdict == VerdictPassed && !receipt.ownerEvidenceVerified {
		return Receipt{}, "", fmt.Errorf("passed receipt requires authenticated owner evidence; structural receipt fields are not proof")
	}
	prefix, err := receiptFilenamePrefix(receipt)
	if err != nil {
		return Receipt{}, "", err
	}
	receipt.Ref = ""
	return receipt, prefix, nil
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func receiptFilenamePrefix(receipt Receipt) (string, error) {
	// Ref is excluded from the receipt JSON. The content hash is the unique
	// append-only suffix for repeated observations of the same cell.
	prefix := strings.TrimSpace(receipt.Ref)
	if prefix == "" {
		prefix = fmt.Sprintf("%s.%s.%s", receipt.CaseID, receipt.Lane, receipt.Target.Ground)
	}
	if filepath.Base(prefix) != prefix || strings.ContainsAny(prefix, "\\/\x00") || prefix == "." || prefix == ".." {
		return "", fmt.Errorf("receipt ref %q is not a safe filename", prefix)
	}
	return prefix, nil
}

func marshalReceipt(receipt Receipt) ([]byte, error) {
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode receipt: %w", err)
	}
	return raw, nil
}

func appendReceiptFile(path, filename string, raw []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("append evidence receipt %s: %w", filename, err)
	}
	if _, err = file.Write(append(raw, '\n')); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("write evidence receipt %s: %w", filename, err)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close evidence receipt %s: %w", filename, closeErr)
	}
	return nil
}
