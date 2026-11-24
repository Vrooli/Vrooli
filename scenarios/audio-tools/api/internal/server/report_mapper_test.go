//go:build audio_lab

package server_test

import (
	inteval "audio-tools/internal/eval"
	"audio-tools/internal/server"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReportToProto_MapsTransportBoundary(t *testing.T) {
	report := server.ReportToProto(inteval.EvalReport{})
	require.NotNil(t, report)
	require.NotNil(t, report.Summary)
}
