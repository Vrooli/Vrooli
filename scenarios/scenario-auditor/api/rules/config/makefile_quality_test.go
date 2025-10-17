//go:build ruletests
// +build ruletests

package config

import "testing"

func TestMakefileQualityDocCases(t *testing.T) {
	runDocTestsCustom(t, "makefile_quality.go", "Makefile", func(input string, path string) ([]MakefileQualityViolation, error) {
		return CheckMakefileQuality(input, path)
	}, func(v MakefileQualityViolation) string {
		return v.Message
	})
}
