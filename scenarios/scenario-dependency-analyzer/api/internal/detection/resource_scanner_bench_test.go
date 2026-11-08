package detection

import (
	"fmt"
	"strings"
	"testing"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

func BenchmarkResourceHeuristics(b *testing.B) {
	resources := make(map[string]struct{}, 32)
	for i := 0; i < 32; i++ {
		resources[fmt.Sprintf("r%02d", i)] = struct{}{}
	}
	scanner := newResourceScanner(&catalogManager{loaded: true, knownResources: resources})
	content := strings.Repeat("// application source without infrastructure references\n", 100) + "r00://service R01_HOST\n"
	visit := func() {
		results := map[string]types.ScenarioDependency{}
		scanner.scanFile(content, "api/main.go", "owner", results)
		if len(results) != 2 {
			b.Fatalf("resource evidence changed: %+v", results)
		}
	}
	visit()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		visit()
	}
}
