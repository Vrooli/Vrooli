package main

import (
	"log"

	aisearchpkg "github.com/vrooli/aisearch-go"
)

// defaultFindingsTuning is the dense search recipe for the findings corpus:
// nomic embeddings with the asymmetric task prefix, cross-encoder rerank with
// RRF blend (junk rejection without burying a strongly-retrieved claim), and
// the zero floor band. It mirrors cli-health's measured-best production recipe.
func defaultFindingsTuning() aisearchpkg.TuningConfig {
	return aisearchpkg.TuningConfig{
		Engine:          "dense",
		EmbedModel:      "nomic-embed-text",
		EmbedTaskPrefix: true,
		RerankEnabled:   true,
		RerankBlend:     true,
		RerankShortlist: 50,
	}.WithDefaults()
}

// loadFindingsTuning reads the findings provider's tuning from the scenario's
// .vrooli/search.json (the SSOT) when it exists, falling back to the code
// default so the API boots before the federation provider block is authored.
func loadFindingsTuning(searchJSONPath, providerID string) aisearchpkg.TuningConfig {
	file, err := aisearchpkg.LoadSearchFile(searchJSONPath)
	if err != nil {
		log.Printf("[web-search] search.json not loaded (%v); using default findings tuning", err)
		return defaultFindingsTuning()
	}
	provider, ok := file.Provider(providerID)
	if !ok {
		log.Printf("[web-search] provider %q absent from search.json; using default findings tuning", providerID)
		return defaultFindingsTuning()
	}
	return provider.ResolvedTuning()
}
