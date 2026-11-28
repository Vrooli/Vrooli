package scan

import (
	"encoding/json"
	"strings"
	"testing"

	"tidiness-manager/cli/internal/support"
)

func TestLanguageMetricRowsShowCodeFactsMetrics(t *testing.T) {
	rows := languageMetricRows(map[string]support.LightLanguageMetrics{
		"go": {
			FileCount:   10,
			TotalLines:  100,
			CodeMetrics: json.RawMessage(`{"todo_count":1,"fixme_count":2,"hack_count":3,"avg_imports_per_file":2.5,"max_imports_in_file":8,"avg_functions_per_file":1.5,"max_functions_in_file":4}`),
			Complexity:  json.RawMessage(`{"tool":"code-facts","total_functions":20,"average_complexity":3.5,"max_complexity":12,"high_complexity_count":2}`),
			Duplicates:  json.RawMessage(`{"total_duplicates":2,"total_lines":17,"tool":"code-facts"}`),
		},
	})
	joined := strings.Join(rows, "\n")
	for _, evidence := range []string{
		"TODO/FIXME/HACK 1/2/3",
		"imports avg/max 2.5/8",
		"functions avg/max 1.5/4",
		"complexity functions/avg/max/high 20/3.5/12/2 via code-facts",
		"2 duplicate groups / 17 lines via code-facts",
	} {
		if !strings.Contains(joined, evidence) {
			t.Errorf("language metric rows omitted %q: %s", evidence, joined)
		}
	}
}
