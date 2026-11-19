//go:build !windows

package platform

import (
	"os"
	"testing"
)

func TestCurrentProcessElevatedMatchesEffectiveUID(t *testing.T) {
	got, err := CurrentProcessElevated()
	if err != nil {
		t.Fatal(err)
	}
	if want := os.Geteuid() == 0; got != want {
		t.Fatalf("CurrentProcessElevated() = %v, want %v", got, want)
	}
}
