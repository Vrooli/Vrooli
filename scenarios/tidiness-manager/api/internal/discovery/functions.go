package discovery

import facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"

// FunctionMetric projects shared parser facts; it does not parse files or run
// another analyzer. Test functions remain available to explicit metric callers.
type FunctionMetric struct {
	Location
	Complexity int
}

func FunctionMetrics(report *facts.CodeFactsReport) ([]FunctionMetric, []string) {
	symbols, warnings := graphSymbols(report, true)
	result := make([]FunctionMetric, 0, len(symbols))
	for _, symbol := range symbols {
		if symbol.complexity > 0 {
			result = append(result, FunctionMetric{Location: location(symbol), Complexity: symbol.complexity})
		}
	}
	return result, warnings
}
