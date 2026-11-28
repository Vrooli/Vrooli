//go:build audio_lab

package main

import "testing"

// TestLabCoverageFloors keeps lab coverage checks in the lab-tagged test
// profile, so product API tests do not execute experiment/eval/soak suites.
func TestLabCoverageFloors(t *testing.T) {
	floors := map[string]int{
		"audio-tools/internal/experiment":          64,
		"audio-tools/internal/experiment/evaldeps": 70,
		"audio-tools/internal/experiment/recipe":   68,
		"audio-tools/internal/experiment/report":   73,
		"audio-tools/internal/eval":                77,
		"audio-tools/internal/soak":                62,
		"audio-tools/handlers/experiment":          38,
		"audio-tools/handlers/eval":                87,
		"audio-tools/handlers/soak":                32,
		"audio-tools/handlers/validation":          56,
	}
	assertCoverageFloors(t, floors, []string{"-tags", "audio_lab"})
}
