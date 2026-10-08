package tunnel

import (
	"tunnel-manager/internal/tunnel"

	tunnelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/tunnel"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// statusToProto converts an internal tunnel.TunnelStatus into the wire shape
// the tunnel proto declares. Lives in the handler package by intent — the
// conversion is mechanical and only used at the transport edge.
func statusToProto(s tunnel.TunnelStatus) *tunnelv1.TunnelStatus {
	return &tunnelv1.TunnelStatus{
		Status:         string(s.Status),
		Systemd:        s.Systemd,
		Ready:          s.Ready,
		ReadyLatencyMs: int32(s.ReadyLatencyMS),
		Score:          int32(s.Score),
		Message:        s.Message,
		CheckedAt:      timestamppb.New(s.CheckedAt.UTC()),
	}
}

// sampleToProto converts an internal tunnel.MetricsSample into its wire shape.
// Returns nil for a nil pointer so GetStatus can pass through "no sample yet".
func sampleToProto(m *tunnel.MetricsSample) *tunnelv1.MetricsSample {
	if m == nil {
		return nil
	}
	return &tunnelv1.MetricsSample{
		Id:            m.ID,
		HaConnections: int32(m.HAConnections),
		RequestErrors: m.RequestErrors,
		ActiveStreams: int32(m.ActiveStreams),
		SmoothedRttMs: m.SmoothedRTTMS,
		ScrapedAt:     timestamppb.New(m.ScrapedAt.UTC()),
	}
}
