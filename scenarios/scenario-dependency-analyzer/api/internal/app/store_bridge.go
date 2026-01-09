package app

import (
	"scenario-dependency-analyzer/internal/store"
	types "scenario-dependency-analyzer/internal/types"
)

// currentStore returns the store from the active runtime, or nil if unavailable.
func currentStore() *store.Store {
	if analyzer := analyzerInstance(); analyzer != nil {
		return analyzer.Store()
	}
	return nil
}

func loadScenarioMetadataMap() (map[string]types.ScenarioSummary, error) {
	if st := currentStore(); st != nil {
		return st.LoadScenarioMetadataMap()
	}
	return map[string]types.ScenarioSummary{}, nil
}
