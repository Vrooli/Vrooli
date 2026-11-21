//go:build windows

package platform

import (
	"os"
	"testing"
)

func TestCurrentProcessElevatedMatchesNativeProcessSecurity(t *testing.T) {
	security, err := ProcessSecurity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	got, err := CurrentProcessElevated()
	if err != nil {
		t.Fatal(err)
	}
	if got != security.Elevated {
		t.Fatalf("CurrentProcessElevated() = %v, ProcessSecurity().Elevated = %v", got, security.Elevated)
	}
}
