//go:build ruletests
// +build ruletests

package structure

import "testing"

func TestRequiredLayoutDocCases(t *testing.T) {
	runDocTests(t, "required_layout.go", "scenario", Check)
}
