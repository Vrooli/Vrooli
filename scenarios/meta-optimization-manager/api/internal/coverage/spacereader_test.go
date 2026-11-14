package coverage

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/api-core/spacedoc"
)

func TestValidateDenominatorIncludesDescriptorCapabilityTier(t *testing.T) {
	def, err := NewSpaceReader().Read(context.Background(), spacedoc.ProjectionValidate)
	if err != nil {
		t.Fatal(err)
	}
	if len(def.Cells) < 121 {
		t.Fatalf("validate denominator has %d cells, want authored 21 plus at least 100 generated capabilities", len(def.Cells))
	}
	generated := 0
	seen := map[string]bool{}
	for _, cell := range def.Cells {
		if !strings.HasPrefix(cell.ID, "cap/") {
			continue
		}
		generated++
		if cell.Owner == "" || !strings.Contains(strings.Join(cell.Notes, " "), ".vrooli/test-genie.json:maturity.capabilities") {
			t.Errorf("generated cell lacks descriptor provenance: %+v", cell)
		}
		if seen[cell.ID] {
			t.Errorf("duplicate generated cell %q", cell.ID)
		}
		seen[cell.ID] = true
	}
	if generated < 100 {
		t.Fatalf("generated capability cells = %d, want at least 100", generated)
	}
}
