package phases

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

var inventoryRowRE = regexp.MustCompile("(?m)^\\| `([a-z0-9-]+)` \\|")

// TestCapabilityContractInventoryMatchesCatalog is the anti-drift guard for the
// Phase Capability Contract inventory: the committed inventory table must list
// exactly the live catalog phases, so adding or removing a phase forces the
// inventory to be updated (no phase silently escapes the contract).
func TestCapabilityContractInventoryMatchesCatalog(t *testing.T) {
	path := filepath.Join(scenarioRoot(t), "docs", "phases", "capability-contract-inventory.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read inventory: %v", err)
	}
	listed := map[string]bool{}
	for _, m := range inventoryRowRE.FindAllStringSubmatch(string(raw), -1) {
		listed[m[1]] = true
	}
	if len(listed) == 0 {
		t.Fatal("inventory table parsed zero phase rows; the row format changed")
	}

	catalog := map[string]bool{}
	for _, name := range ValidPhaseNames() {
		catalog[name] = true
	}

	var missing, extra []string
	for name := range catalog {
		if !listed[name] {
			missing = append(missing, name)
		}
	}
	for name := range listed {
		if !catalog[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 {
		t.Errorf("inventory missing catalog phase(s): %v — add them to docs/phases/capability-contract-inventory.md", missing)
	}
	if len(extra) > 0 {
		t.Errorf("inventory lists non-catalog phase(s): %v — remove them from the inventory", extra)
	}
}
