package capabilities

import (
	"errors"
	"testing"
)

func TestReconcileDeclaredUsesLiveStatusAndCountsAssets(t *testing.T) {
	declarations := map[string][]string{
		"asset-a": {"keyboard-operable", "contrast-floor"},
		"asset-b": {"keyboard-operable"},
	}
	statuses := map[string]liveStatus{
		"keyboard-operable": {Title: "Keyboard operable", Status: "no-checker", Blockers: []string{"claim type keyboard-operable has no evaluator"}},
		"contrast-floor":    {Title: "Contrast floor", Status: "provable"},
	}

	report := ReconcileDeclaredWithStatuses(declarations, statuses, nil)
	if report.DeclaredAssetCount != 2 {
		t.Fatalf("declared assets = %d, want 2", report.DeclaredAssetCount)
	}
	if report.DeclarationCount != 3 {
		t.Fatalf("declarations = %d, want 3", report.DeclarationCount)
	}
	if report.UncheckableAssetCount != 2 || report.UnmeasuredAssetCount != 2 {
		t.Fatalf("uncheckable/unmeasured = %d/%d, want 2/2", report.UncheckableAssetCount, report.UnmeasuredAssetCount)
	}
	if len(report.Capabilities) != 2 || report.Capabilities[0].Capability != "contrast-floor" || report.Capabilities[1].Capability != "keyboard-operable" {
		t.Fatalf("capabilities not stably ordered: %#v", report.Capabilities)
	}
	if report.Capabilities[1].Checkable {
		t.Fatal("no-checker capability reported checkable")
	}
	if got := report.Capabilities[1].DeclaredAssetCount; got != 2 {
		t.Fatalf("keyboard declaration count = %d, want 2", got)
	}
}

func TestReconcileDeclaredFailsClosedWhenDerivationUnavailable(t *testing.T) {
	report := ReconcileDeclaredWithStatuses(map[string][]string{"asset-a": {"keyboard-operable"}}, nil, errors.New("experience-manager unavailable"))
	if report.UncheckableAssetCount != 1 || report.UnmeasuredAssetCount != 1 {
		t.Fatalf("unavailable derivation counts = %d/%d, want 1/1", report.UncheckableAssetCount, report.UnmeasuredAssetCount)
	}
	row := report.Capabilities[0]
	if row.Status != "unmeasured" || len(row.Blockers) != 1 || row.Blockers[0] != "capability derivation unavailable" {
		t.Fatalf("unexpected unavailable row: %#v", row)
	}
}
