package hostapp

import (
	"context"
	"strings"
	"testing"
)

func TestDesktopSessionDispatchRejectsUnsafeIdentity(t *testing.T) {
	_, err := (Service{}).InspectDesktopSession(context.Background(), "../2", 42)
	if err == nil || !strings.Contains(err.Error(), "invalid desktop session identity") {
		t.Fatalf("inspection error = %v", err)
	}
}
