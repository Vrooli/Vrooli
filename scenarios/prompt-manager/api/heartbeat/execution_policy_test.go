package heartbeat

import (
	"fmt"
	"prompt-manager/teamconfig"
	"testing"
)

func TestDefaultProfileKeyForRuntimeMode(t *testing.T) {
	tests := []struct {
		runtimeMode string
		want        string
	}{
		{teamconfig.RuntimeModeSingleProcess, DefaultProfileKeySingleProcess},
		{teamconfig.RuntimeModeMultiProcess, DefaultProfileKeyMultiProcess},
		{"", DefaultProfileKeyMultiProcess},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("runtimeMode=%q", tt.runtimeMode), func(t *testing.T) {
			got := DefaultProfileKeyForRuntimeMode(tt.runtimeMode)
			if got != tt.want {
				t.Errorf("DefaultProfileKeyForRuntimeMode(%q) = %q, want %q", tt.runtimeMode, got, tt.want)
			}
		})
	}
}
