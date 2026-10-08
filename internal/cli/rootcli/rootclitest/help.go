// Package rootclitest provides shared assertions for root CLI handler tests.
package rootclitest

import (
	"fmt"
	"strings"
	"testing"
)

// AssertHelpWithNoArgs proves that a root handler succeeds without arguments
// and renders its command-specific usage text.
func AssertHelpWithNoArgs(t *testing.T, run func() error, output fmt.Stringer, usage string) {
	t.Helper()
	if err := validateHelpWithNoArgs(run, output.String, usage); err != nil {
		t.Fatal(err)
	}
}

func validateHelpWithNoArgs(run func() error, output func() string, usage string) error {
	if err := run(); err != nil {
		return fmt.Errorf("root handler: %w", err)
	}
	got := output()
	if !strings.Contains(got, usage) {
		return fmt.Errorf("root handler help missing %q usage: %q", usage, got)
	}
	return nil
}
