package evidence

import (
	"testing"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	"scenario-to-desktop-api/smoketest"
)

func TestManifestGatesUseIndependentSignals(t *testing.T) {
	now := time.Now().UTC()
	journey := &deliveryramp.JourneyResult{Disposition: deliveryramp.DispositionPass}
	cases := []struct {
		name      string
		input     smoketest.EvidenceManifestInput
		recording bool
		want      map[deliveryramp.GateName]deliveryramp.GateDisposition
	}{
		{"protocol failure", smoketest.EvidenceManifestInput{ProtocolPassed: false, VisualReadiness: "usable", Journey: journey, JourneyCaptureID: "j", RecordingCaptureID: "r"}, true, map[deliveryramp.GateName]deliveryramp.GateDisposition{deliveryramp.GateProtocol: deliveryramp.GateFailed, deliveryramp.GateVisual: deliveryramp.GatePassed}},
		{"journey failure does not fail visual", smoketest.EvidenceManifestInput{ProtocolPassed: true, VisualReadiness: "usable", Journey: &deliveryramp.JourneyResult{Disposition: deliveryramp.DispositionFailed}, JourneyCaptureID: "j", RecordingCaptureID: "r"}, true, map[deliveryramp.GateName]deliveryramp.GateDisposition{deliveryramp.GateVisual: deliveryramp.GatePassed, deliveryramp.GateJourney: deliveryramp.GateFailed}},
		{"readiness unavailable", smoketest.EvidenceManifestInput{ProtocolPassed: true, VisualReadiness: "unavailable", Journey: journey, JourneyCaptureID: "j", RecordingCaptureID: "r"}, true, map[deliveryramp.GateName]deliveryramp.GateDisposition{deliveryramp.GateVisual: deliveryramp.GateUnavailable}},
		{"recording failure", smoketest.EvidenceManifestInput{ProtocolPassed: true, VisualReadiness: "usable", Journey: journey, JourneyCaptureID: "j", RecordingCaptureID: "r"}, false, map[deliveryramp.GateName]deliveryramp.GateDisposition{deliveryramp.GateCapture: deliveryramp.GateFailed}},
		{"persistence failure", smoketest.EvidenceManifestInput{ProtocolPassed: true, VisualReadiness: "usable", Journey: journey, JourneyCaptureID: "", RecordingCaptureID: "r"}, true, map[deliveryramp.GateName]deliveryramp.GateDisposition{deliveryramp.GatePersistence: deliveryramp.GateFailed}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, gate := range manifestGates(tc.input, deliveryramp.ProfileVisual, tc.recording, now, now.Add(time.Second)) {
				if want, ok := tc.want[gate.Name]; ok && gate.Disposition != want {
					t.Fatalf("%s=%s want %s", gate.Name, gate.Disposition, want)
				}
			}
		})
	}
}
