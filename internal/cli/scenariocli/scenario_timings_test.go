package scenariocli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestParseTimingsRequest(t *testing.T) {
	req, err := ParseTimingsRequest(false, []string{"--scenario", "structure-health", "--json"})
	if err != nil {
		t.Fatalf("ParseTimingsRequest() error = %v", err)
	}
	if req.Scenario != "structure-health" || !req.JSON {
		t.Fatalf("request = %+v, want scenario and json", req)
	}
}

func TestRenderTimingsResponseIncludesRetainedTailCaveat(t *testing.T) {
	var out bytes.Buffer
	err := RenderTimingsResponse(&out, cliout.FormatHuman, TimingsResponse{Rows: []scenarioruntime.StartTimingSummary{{
		Scenario: "fleet", Operation: "restart", Step: "health", Count: 2,
		MeanMS: 1500, P50MS: 1400, P90MS: 1800, TotalMS: 3000, Share: 0.25,
	}}})
	if err != nil {
		t.Fatalf("RenderTimingsResponse() error = %v", err)
	}
	for _, want := range []string{"fleet", "restart", "health", "1.5s", "25.0%", "retained terminal-operation tail"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want %q", out.String(), want)
		}
	}
}
