package validation

import "testing"

func TestDiscoverFromFactsRequiresManagerEvidenceForJavaScript(t *testing.T) {
	registry := DefaultAdapterRegistry()
	unknown := registry.DiscoverFromFacts([]FactTarget{{Root: "/repo/ui", Language: "typescript", Files: []string{"package.json"}}})
	if len(unknown) != 1 || unknown[0].Coverage != CoverageUnknown {
		t.Fatalf("language-only facts = %+v, want one unknown target", unknown)
	}
	pnpm := registry.DiscoverFromFacts([]FactTarget{{Root: "/repo/ui", Language: "typescript", Files: []string{"package.json", "pnpm-lock.yaml"}}})
	if len(pnpm) != 1 || pnpm[0].Ecosystem != EcosystemPnpm {
		t.Fatalf("pnpm facts = %+v, want only pnpm", pnpm)
	}
	npm := registry.DiscoverFromFacts([]FactTarget{{Root: "/repo/ui", Language: "javascript", Manager: "npm", Files: []string{"package.json"}}})
	if len(npm) != 1 || npm[0].Ecosystem != EcosystemNPM {
		t.Fatalf("explicit npm facts = %+v, want only npm", npm)
	}
}

func TestDetectSubstrateFromFactsFeedsApplicableScannerFlags(t *testing.T) {
	substrate := DetectSubstrateFromFacts([]FactTarget{{
		Root:  "scenario/ui",
		Files: []string{"go.mod", "pnpm-lock.yaml"},
	}})
	if !substrate.Go || !substrate.PnpmUI {
		t.Fatalf("substrate flags = %+v, want Go and pnpm", substrate)
	}
	if len(substrate.GoModDirs) != 1 || len(substrate.PnpmLockDirs) != 1 {
		t.Fatalf("manifest directories = %+v, %+v", substrate.GoModDirs, substrate.PnpmLockDirs)
	}
}
