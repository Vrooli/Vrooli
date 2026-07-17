package executionevidence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// CutoverConfirmation is deliberately awkward to require an operator to opt
// in before moving historical evidence out of the live runtime tree.
const CutoverConfirmation = "ARCHIVE_TEST_GENIE_EVIDENCE"

var ErrCutoverNotConfirmed = errors.New("execution-evidence cutover is not confirmed")

// CutoverPlan is the inspectable, immutable inventory an operator reviews
// before an archive-and-cutover. Digest covers the inventory, not file bytes.
type CutoverPlan struct {
	CoverageRoot string
	ArchiveRoot  string
	Files        int
	Bytes        int64
	Digest       string
}

// CutoverReceipt is the durable operator record written into the archived tree
// only after a reviewed inventory has been atomically moved out of the live
// runtime path. It is deliberately sufficient for a rollback decision without
// reading the live replacement directory.
type CutoverReceipt struct {
	ConfirmedAt time.Time   `json:"confirmedAt"`
	Plan        CutoverPlan `json:"plan"`
}

const CutoverReceiptFile = "cutover-receipt.json"

// PlanCutover inventories the existing coverage tree without changing it.
func PlanCutover(scenarioDir, archiveRoot string) (CutoverPlan, error) {
	coverage := filepath.Join(scenarioDir, sharedartifacts.CoverageRoot)
	plan := CutoverPlan{CoverageRoot: coverage, ArchiveRoot: archiveRoot}
	hash := sha256.New()
	err := filepath.WalkDir(coverage, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(coverage, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(hash, "%s:%d\n", filepath.ToSlash(rel), info.Size())
		plan.Files++
		plan.Bytes += info.Size()
		return nil
	})
	if os.IsNotExist(err) {
		return plan, nil
	}
	if err != nil {
		return CutoverPlan{}, fmt.Errorf("inventory coverage: %w", err)
	}
	plan.Digest = hex.EncodeToString(hash.Sum(nil))
	return plan, nil
}

// ApplyCutover archives the complete historical coverage tree by rename, then
// creates an empty replacement root. It never merges trees or attempts legacy
// reads; callers must review PlanCutover and supply the exact confirmation.
func ApplyCutover(plan CutoverPlan, confirmation string) error {
	if confirmation != CutoverConfirmation {
		return ErrCutoverNotConfirmed
	}
	if plan.CoverageRoot == "" || plan.ArchiveRoot == "" {
		return fmt.Errorf("coverage and archive roots are required")
	}
	current, err := PlanCutover(filepath.Dir(plan.CoverageRoot), plan.ArchiveRoot)
	if err != nil {
		return err
	}
	if current.Files != plan.Files || current.Bytes != plan.Bytes || current.Digest != plan.Digest {
		return fmt.Errorf("cutover inventory changed since review")
	}
	if _, err := os.Stat(plan.ArchiveRoot); !os.IsNotExist(err) {
		return fmt.Errorf("archive destination already exists")
	}
	if err := os.MkdirAll(filepath.Dir(plan.ArchiveRoot), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(plan.CoverageRoot); err == nil {
		if err := os.Rename(plan.CoverageRoot, plan.ArchiveRoot); err != nil {
			return fmt.Errorf("archive coverage: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(plan.CoverageRoot, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(plan.ArchiveRoot, 0o755); err != nil {
		return err
	}
	receipt := CutoverReceipt{ConfirmedAt: time.Now().UTC(), Plan: plan}
	if err := writeReceipt(filepath.Join(plan.ArchiveRoot, CutoverReceiptFile), receipt); err != nil {
		return fmt.Errorf("write cutover receipt: %w", err)
	}
	return nil
}

func writeReceipt(path string, receipt CutoverReceipt) error {
	payload := fmt.Sprintf("{\n  \"confirmedAt\": %q,\n  \"coverageRoot\": %q,\n  \"archiveRoot\": %q,\n  \"files\": %d,\n  \"bytes\": %d,\n  \"digest\": %q\n}\n", receipt.ConfirmedAt.Format(time.RFC3339Nano), receipt.Plan.CoverageRoot, receipt.Plan.ArchiveRoot, receipt.Plan.Files, receipt.Plan.Bytes, receipt.Plan.Digest)
	tmp, err := os.CreateTemp(filepath.Dir(path), ".cutover-receipt-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(payload); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
