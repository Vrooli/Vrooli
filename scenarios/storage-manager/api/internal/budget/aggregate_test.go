package budget

import (
	"testing"

	coreStorage "github.com/vrooli/api-core/storage"
)

func TestAggregateFlagsImpossibleReservation(t *testing.T) {
	inventory := coreStorage.OwnerInventory{Owners: []coreStorage.OwnerManifest{
		{Kind: coreStorage.OwnerScenario, ID: "alpha", StorageEntries: []coreStorage.StorageEntry{{Name: "cache", Budget: &coreStorage.BudgetDeclaration{MaxBytes: "1GiB"}}}},
		{Kind: coreStorage.OwnerScenario, ID: "beta", StorageEntries: []coreStorage.StorageEntry{{Name: "cache", Budget: &coreStorage.BudgetDeclaration{MaxBytes: "2GiB"}}}},
	}}
	report := Aggregate(inventory, 2<<30)
	if report.Status != StatusUnreasonable || !report.OverCapacity {
		t.Fatalf("report = %#v, want unreasonable over-capacity", report)
	}
	if report.DeclaredBytes != 3<<30 || report.EntryCount != 2 || report.OwnerCount != 2 {
		t.Fatalf("report = %#v, want 3GiB across two owners", report)
	}
}

func TestAggregateWarnsBeforeCapacityAndIgnoresAgeOnlyBudgets(t *testing.T) {
	inventory := coreStorage.OwnerInventory{Owners: []coreStorage.OwnerManifest{{
		Kind: coreStorage.OwnerResource,
		ID:   "resource",
		StorageEntries: []coreStorage.StorageEntry{
			{Name: "data", Budget: &coreStorage.BudgetDeclaration{MaxBytes: "9GiB", MaxAge: "7d"}},
			{Name: "logs", Budget: &coreStorage.BudgetDeclaration{MaxAge: "7d"}},
		},
	}}}
	report := Aggregate(inventory, 10<<30)
	if report.Status != StatusWarning || report.EntryCount != 1 || report.OwnerCount != 1 {
		t.Fatalf("report = %#v, want warning for one byte budget", report)
	}
	if report.Utilization < 0.89 || report.Utilization > 0.91 {
		t.Fatalf("utilization = %v, want about 0.9", report.Utilization)
	}
}

func TestAggregateReportsUnknownCapacityWithoutSuppressingDeclarations(t *testing.T) {
	inventory := coreStorage.OwnerInventory{Owners: []coreStorage.OwnerManifest{{
		Kind:           coreStorage.OwnerTool,
		ID:             "tool",
		StorageEntries: []coreStorage.StorageEntry{{Name: "cache", Budget: &coreStorage.BudgetDeclaration{MaxBytes: "4GiB"}}},
	}}}
	report := Aggregate(inventory, 0)
	if report.Status != StatusCapacityUnknown || report.DeclaredBytes != 4<<30 {
		t.Fatalf("report = %#v, want declared bytes with unknown capacity", report)
	}
}
