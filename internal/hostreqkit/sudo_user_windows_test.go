//go:build windows

package hostreqkit

import (
	"errors"
	"testing"
)

func TestRunAsInvokingUserWithSessionIsTypedUnsupported(t *testing.T) {
	err := RunAsInvokingUserWithSession("secret-tool", nil, EnsureOptions{})
	if !errors.Is(err, ErrSessionExecutionUnsupported) {
		t.Fatalf("error = %v, want ErrSessionExecutionUnsupported", err)
	}
}
