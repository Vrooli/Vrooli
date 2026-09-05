package validationbroker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
)

func TestReceiptSchemaHasOneFirstPartyAuthority(t *testing.T) {
	root, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	for _, owner := range []string{"plan-manager", "git-control-tower", "agent-manager"} {
		err := filepath.WalkDir(filepath.Join(root, "scenarios", owner), func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || filepath.Ext(path) != ".sql" {
				return err
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if strings.Contains(strings.ToLower(string(content)), "create table") && strings.Contains(strings.ToLower(string(content)), "validation_receipts") {
				t.Errorf("%s declares Test Genie's canonical receipt table", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestLegacyMigrationCreatesOneCanonicalReceipt(t *testing.T) {
	repo := NewRepository(testsqllite(t))
	adapter := NewLegacyMigrationAdapter(repo)
	record := LegacyValidationRecord{
		SourceKind: "plan-manager-operation", SourceID: "legacy-1", CallerScenario: "plan-manager", TargetScenario: "demo", State: "passed",
		Evidence: []*validationv1.EvidenceReference{{EvidenceId: "run-1", Kind: "test-genie-run", Owner: "test-genie", SubjectId: "demo"}},
	}
	first, err := adapter.Migrate(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Migrated || first.ReadOnly || first.Receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || len(first.Receipt.GetEvidence()) != 1 {
		t.Fatalf("migration = %+v", first)
	}
	second, err := adapter.Migrate(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if second.Receipt.GetReceiptId() != first.Receipt.GetReceiptId() || second.Reason != "already_migrated" {
		t.Fatalf("idempotent migration = %+v", second)
	}
	history, err := repo.History(context.Background(), first.Receipt.GetReceiptId())
	if err != nil || len(history) != 4 {
		t.Fatalf("history = %+v err=%v", history, err)
	}
}

func TestLegacyMigrationPreservesUnmappableStateAsReadOnlyTerminal(t *testing.T) {
	repo := NewRepository(testsqllite(t))
	result, err := NewLegacyMigrationAdapter(repo).Migrate(context.Background(), LegacyValidationRecord{
		SourceKind: "gct-collection", SourceID: "old-1", CallerScenario: "git-control-tower", TargetScenario: "demo", State: "historical-unknown",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReadOnly || result.Migrated || result.Reason != "unmappable_legacy_state" || result.Receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_DEGRADED {
		t.Fatalf("projection = %+v", result)
	}
	if _, err := repo.Get(context.Background(), result.Receipt.GetReceiptId()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("read-only projection was persisted: %v", err)
	}
}

func TestLegacyMigrationDoesNotReexecuteHistoricalActiveWork(t *testing.T) {
	repo := NewRepository(testsqllite(t))
	result, err := NewLegacyMigrationAdapter(repo).Migrate(context.Background(), LegacyValidationRecord{
		SourceKind: "gct-collection", SourceID: "active-1", CallerScenario: "git-control-tower", TargetScenario: "demo", State: "running",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReadOnly || result.Migrated || result.Reason != "legacy_active_state_read_only" || result.Receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_DEGRADED {
		t.Fatalf("projection = %+v", result)
	}
	if _, err := repo.Get(context.Background(), result.Receipt.GetReceiptId()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("active historical work was made recoverable: %v", err)
	}
}

func TestLegacyMigrationCoversEveryFirstPartySourceShape(t *testing.T) {
	tests := []struct {
		kind, id, caller, state string
		want                    validationv1.ReceiptState
		readOnly                bool
	}{
		{"plan-manager-baseline-checkpoint", "baseline-1", "plan-manager", "partial", validationv1.ReceiptState_RECEIPT_STATE_DEGRADED, false},
		{"plan-manager-validation-operation", "operation-1", "plan-manager", "passed", validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, false},
		{"gct-collection-member", "before:alpha", "git-control-tower", "succeeded", validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, false},
		{"test-genie-run", "run-a", "test-genie", "failed", validationv1.ReceiptState_RECEIPT_STATE_FAILED, false},
	}
	for _, test := range tests {
		t.Run(test.kind, func(t *testing.T) {
			repo := NewRepository(testsqllite(t))
			result, err := NewLegacyMigrationAdapter(repo).Migrate(context.Background(), LegacyValidationRecord{SourceKind: test.kind, SourceID: test.id, CallerScenario: test.caller, TargetScenario: "alpha", State: test.state})
			if err != nil {
				t.Fatal(err)
			}
			if result.ReadOnly != test.readOnly || result.Receipt.GetState() != test.want {
				t.Fatalf("migration = %+v", result)
			}
		})
	}
}

func TestCapturedMigrationReplayExplainsEveryComparison(t *testing.T) {
	repo := NewRepository(testsqllite(t))
	fixtures := []LegacyValidationRecord{
		{SourceKind: "plan-manager-baseline-checkpoint", SourceID: "baseline-partial", CallerScenario: "plan-manager", TargetScenario: "alpha", State: "partial"},
		{SourceKind: "plan-manager-validation-operation", SourceID: "validation-pass", CallerScenario: "plan-manager", TargetScenario: "alpha", State: "passed"},
		{SourceKind: "gct-collection-member", SourceID: "before:alpha", CallerScenario: "git-control-tower", TargetScenario: "alpha", State: "ready"},
		{SourceKind: "test-genie-run", SourceID: "run-a", CallerScenario: "test-genie", TargetScenario: "alpha", State: "passed"},
	}
	for _, fixture := range fixtures {
		result, err := NewLegacyMigrationAdapter(repo).Migrate(context.Background(), fixture)
		if err != nil {
			t.Fatal(err)
		}
		comparison, err := repo.RecordShadowComparison(context.Background(), fixture, result.Receipt)
		if err != nil {
			t.Fatal(err)
		}
		if comparison.ReasonCode == "" {
			t.Fatalf("fixture %s has no explained comparison", fixture.SourceKind)
		}
		if comparison.Matched != (comparison.ReasonCode == "state_equivalent") {
			t.Fatalf("fixture %s comparison = %+v", fixture.SourceKind, comparison)
		}
	}
}

func TestShadowComparisonPersistsExplainedMismatchWithoutChangingReceipt(t *testing.T) {
	db := testsqllite(t)
	repo := NewRepository(db)
	result, err := NewLegacyMigrationAdapter(repo).Migrate(context.Background(), LegacyValidationRecord{
		SourceKind: "test-genie-run", SourceID: "run-1", CallerScenario: "test-genie", TargetScenario: "demo", State: "failed",
	})
	if err != nil {
		t.Fatal(err)
	}
	legacy := LegacyValidationRecord{SourceKind: "test-genie-run", SourceID: "run-1", State: "passed"}
	comparison, err := repo.RecordShadowComparison(context.Background(), legacy, result.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	if comparison.Matched || comparison.ReasonCode != "verdict_mismatch" {
		t.Fatalf("comparison = %+v", comparison)
	}
	var matched bool
	var reason string
	if err := db.QueryRow(`SELECT matched, reason_code FROM validation_shadow_comparisons WHERE comparison_id = ?`, comparison.ComparisonID).Scan(&matched, &reason); err != nil {
		t.Fatal(err)
	}
	if matched || reason != "verdict_mismatch" {
		t.Fatalf("stored mismatch = matched:%v reason:%s", matched, reason)
	}
	comparisons, err := repo.ListShadowComparisons(context.Background(), 10)
	if err != nil || len(comparisons) != 1 || comparisons[0].ReasonCode != "verdict_mismatch" {
		t.Fatalf("listed comparisons = %+v err=%v", comparisons, err)
	}
	unchanged, err := repo.Get(context.Background(), result.Receipt.GetReceiptId())
	if err != nil || unchanged.GetState() != validationv1.ReceiptState_RECEIPT_STATE_FAILED {
		t.Fatalf("shadow comparison changed production receipt: %+v err=%v", unchanged, err)
	}
}
