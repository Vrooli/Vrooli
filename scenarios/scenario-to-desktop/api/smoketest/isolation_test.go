package smoketest

import (
	"strings"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestIsolationObservationDerivation(t *testing.T) {
	tests := []struct {
		name, want   string
		observations []deliveryramp.IsolationObservation
	}{
		{"unknown", "unknown", nil},
		{"bundled private", "bundled_private", []deliveryramp.IsolationObservation{{StateRoot: "/tmp/runtime", Source: "bundled_runtime"}}},
		{"isolated instance", "isolated_instance", []deliveryramp.IsolationObservation{{StateRoot: "/tmp/isolated", Source: "isolated_instance"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveScreenContentSource(tt.observations); got != tt.want {
				t.Fatalf("source = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsolationObservationRefusesOperatorHome(t *testing.T) {
	if err := validateIsolationObservations([]deliveryramp.IsolationObservation{{StateRoot: "/home/matthalloran8/.local/state/vrooli/live-state"}}); err == nil || !strings.Contains(err.Error(), "live_state_resolved") {
		t.Fatalf("error = %v, want live_state_resolved", err)
	}
}

func TestIsolationObservationAllowsPrivateAppDataUnderUserHome(t *testing.T) {
	if err := validateIsolationObservations([]deliveryramp.IsolationObservation{{StateRoot: "/home/matthalloran8/.config/hello-desktop/runtime", DatabasePath: "/home/matthalloran8/.config/hello-desktop/runtime/storage"}}); err != nil {
		t.Fatalf("private app data should be allowed: %v", err)
	}
}
