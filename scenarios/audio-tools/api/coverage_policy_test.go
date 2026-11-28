package main

import (
	"testing"
)

// TestCoverageFloors keeps the historical per-package coverage contract in the
// normal Go test suite. Keeping this policy in Go, rather than an unmanaged
// shell script, makes it visible to Test Genie and portable across platforms.
func TestCoverageFloors(t *testing.T) {
	floors := map[string]int{
		"audio-tools/internal/byok/envelope":      90,
		"audio-tools/internal/protomap":           80,
		"audio-tools/internal/database":           80,
		"audio-tools/internal/middleware":         80,
		"audio-tools/internal/modules":            80,
		"audio-tools/internal/ai/chains":          80,
		"audio-tools/internal/server":             63,
		"audio-tools/internal/session":            80,
		"audio-tools/internal/stt/strategy":       80,
		"audio-tools/internal/capabilities":       70,
		"audio-tools/handlers/health_status":      80,
		"audio-tools/handlers/provider_lifecycle": 65,
		"audio-tools/internal/store":              70,
		"audio-tools/internal/byokstore":          70,
		"audio-tools/internal/byok":               80,
		"audio-tools/internal/summarize":          80,
		"audio-tools/internal/stt/segmenter":      75,
		// Chain floors restored after the catalog/model work: the duplicated
		// provider plumbing was consolidated and covered with behaviour tests
		// for fallback ordering, caching, error mapping, and credential
		// redaction.
		"audio-tools/internal/ai/sttchain":       80,
		"audio-tools/internal/ai/summarizechain": 85,
		"audio-tools/internal/ai/ttschain":       80,
		"audio-tools/internal/diagnostics":       90,
		// Catalogs own the scenario's extensibility contract. Keep coverage
		// floors on their loaders, validators, policy resolution, and drift
		// checks so new declarations cannot quietly weaken runtime guarantees.
		"audio-tools/internal/providercatalog":   85,
		"audio-tools/internal/operationcatalog":  80,
		"audio-tools/internal/modelcatalog":      80,
		"audio-tools/internal/modelpromotion":    75,
		"audio-tools/internal/sttengine":         75,
		"audio-tools/handlers/stt":               80,
		"audio-tools/internal/audio":             75,
		"audio-tools/internal/stt/pipeline":      55,
		"audio-tools/internal/tts":               70,
		"audio-tools/internal/usagereport":       65,
		"audio-tools/internal/testutil/vendorws": 80,
		"audio-tools/internal/bootstrap":         50,
	}
	assertCoverageFloors(t, floors, nil)
}
