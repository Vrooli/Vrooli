package execution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// fixGateGovProvider is a minimal GovernanceProvider returning a fixed mode.
type fixGateGovProvider struct {
	mode string
}

func (p fixGateGovProvider) LoadGovernance() (GovernanceSettings, error) {
	g := DefaultGovernanceSettings()
	g.FixBeforeFeature = p.mode
	return g, nil
}

// writeSpec writes a backlog item spec.json under dataRoot/<kindDir>/<name>.
func writeSpec(t *testing.T, dataRoot, kindDir, name string, spec map[string]any) {
	t.Helper()
	dir := filepath.Join(dataRoot, kindDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.json"), data, 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
}

func TestApplyFixBeforeFeatureGate_Service(t *testing.T) {
	dataRoot := t.TempDir()
	// Open fix item targeting scenario "demo".
	writeSpec(t, dataRoot, "fix", "demo-bug", map[string]any{
		"name":             "demo-bug",
		"kind":             "fix",
		"status":           "backlog",
		"acceptance_allow": []string{"scenarios/demo/**"},
	})
	// A completed fix item that must NOT count.
	writeSpec(t, dataRoot, "fix", "demo-done", map[string]any{
		"name":             "demo-done",
		"kind":             "fix",
		"status":           "completed",
		"acceptance_allow": []string{"scenarios/demo/**"},
	})

	featureItem := backlogItem{
		Name:            "demo-feature",
		Kind:            "execute",
		AcceptanceAllow: []string{"scenarios/demo/**"},
	}

	t.Run("block mode populates forceable blocking reason", func(t *testing.T) {
		svc := &Service{dataRoot: dataRoot, governanceProvider: fixGateGovProvider{mode: FixBeforeFeatureBlock}}
		pf := ProcessPreflight{}
		svc.applyFixBeforeFeatureGate(featureItem, &pf)
		if len(pf.ForceableBlockingReasons) != 1 {
			t.Fatalf("ForceableBlockingReasons = %v, want 1", pf.ForceableBlockingReasons)
		}
		if len(pf.Advisories) != 0 {
			t.Errorf("Advisories = %v, want none", pf.Advisories)
		}
	})

	t.Run("suggest mode populates advisory only", func(t *testing.T) {
		svc := &Service{dataRoot: dataRoot, governanceProvider: fixGateGovProvider{mode: FixBeforeFeatureSuggest}}
		pf := ProcessPreflight{}
		svc.applyFixBeforeFeatureGate(featureItem, &pf)
		if len(pf.Advisories) != 1 {
			t.Fatalf("Advisories = %v, want 1", pf.Advisories)
		}
		if len(pf.ForceableBlockingReasons) != 0 {
			t.Errorf("ForceableBlockingReasons = %v, want none", pf.ForceableBlockingReasons)
		}
	})

	t.Run("off mode does nothing", func(t *testing.T) {
		svc := &Service{dataRoot: dataRoot, governanceProvider: fixGateGovProvider{mode: FixBeforeFeatureOff}}
		pf := ProcessPreflight{}
		svc.applyFixBeforeFeatureGate(featureItem, &pf)
		if len(pf.Advisories) != 0 || len(pf.ForceableBlockingReasons) != 0 {
			t.Errorf("off mode must not gate, got advisories=%v forceable=%v", pf.Advisories, pf.ForceableBlockingReasons)
		}
	})

	t.Run("non-execute kind does nothing", func(t *testing.T) {
		svc := &Service{dataRoot: dataRoot, governanceProvider: fixGateGovProvider{mode: FixBeforeFeatureBlock}}
		fixItem := backlogItem{Name: "x", Kind: "fix", AcceptanceAllow: []string{"scenarios/demo/**"}}
		pf := ProcessPreflight{}
		svc.applyFixBeforeFeatureGate(fixItem, &pf)
		if len(pf.Advisories) != 0 || len(pf.ForceableBlockingReasons) != 0 {
			t.Errorf("non-execute must not gate, got advisories=%v forceable=%v", pf.Advisories, pf.ForceableBlockingReasons)
		}
	})

	t.Run("feature on unrelated scenario does nothing", func(t *testing.T) {
		svc := &Service{dataRoot: dataRoot, governanceProvider: fixGateGovProvider{mode: FixBeforeFeatureBlock}}
		other := backlogItem{Name: "other-feature", Kind: "execute", AcceptanceAllow: []string{"scenarios/unrelated/**"}}
		pf := ProcessPreflight{}
		svc.applyFixBeforeFeatureGate(other, &pf)
		if len(pf.Advisories) != 0 || len(pf.ForceableBlockingReasons) != 0 {
			t.Errorf("unrelated scenario must not gate, got advisories=%v forceable=%v", pf.Advisories, pf.ForceableBlockingReasons)
		}
	})
}
