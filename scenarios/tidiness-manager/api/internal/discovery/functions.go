package discovery

import "github.com/vrooli/api-core/factgraph"

// FunctionMetric projects shared parser facts; it does not parse files or run
// another analyzer. Test functions remain available to explicit metric callers.
type FunctionMetric struct {
	Location
	Complexity int
}

func FunctionMetricsFromIndex(index *factgraph.Index) ([]FunctionMetric, []string) {
	symbols, warnings, _ := graphSymbolsWithLanguageFilesFromIndex(index, true)
	result := make([]FunctionMetric, 0, len(symbols))
	for _, symbol := range symbols {
		if symbol.complexity > 0 {
			result = append(result, FunctionMetric{Location: location(symbol), Complexity: symbol.complexity})
		}
	}
	return result, warnings
}
