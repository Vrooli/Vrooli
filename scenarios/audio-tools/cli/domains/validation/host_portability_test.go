package validation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	modelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

func TestCollectModelResolutionReceiptsKeepsProfileAndEvidenceBoundaries(t *testing.T) {
	calls := 0
	got, err := collectModelResolutionReceipts(context.Background(), func(_ context.Context, capability, engine string, platform *modelv1.Platform) (*modelv1.ModelRecord, string, error) {
		calls++
		if capability == "speaker" && platform.GetRamGb() == 4 {
			return nil, "", errors.New("no eligible model")
		}
		return &modelv1.ModelRecord{Id: capability + "-model", Readiness: modelv1.Readiness_READINESS_CANDIDATE, CommercialUse: modelv1.CommercialUse_COMMERCIAL_USE_ALLOWED}, capability + " explanation", nil
	})
	require.NoError(t, err)
	require.Equal(t, 8, calls)
	require.Contains(t, got, "D1-linux-amd64-cpu")
	require.Contains(t, got, "D2-linux-amd64-constrained")
	require.Equal(t, "not established by resolution; join the matching corpus-quality receipt separately", got["quality_evidence"])
	d2 := got["D2-linux-amd64-constrained"].(map[string]any)
	require.Equal(t, 2, d2["cpu_limit"])
	require.Equal(t, int32(2), d2["platform_constraints"].(*modelv1.Platform).GetCpuLimit())
	require.Equal(t, int32(4), d2["platform_constraints"].(*modelv1.Platform).GetRamGb())
	rows := d2["observations"].([]map[string]any)
	stt := rows[0]
	require.Equal(t, "sherpa-streaming", stt["engine_id"])
	require.Equal(t, "READINESS_CANDIDATE", stt["readiness"])
	require.Equal(t, "unavailable", rows[3]["state"])
	require.Equal(t, "no eligible model", rows[3]["reason"])
	require.Equal(t, map[string]any{"limit_cores": int32(2), "limit_state": "recorded", "model_minimum_cores": "unknown", "qualification": "not_proven"}, rows[0]["cpu_capacity"])
}

func TestD2SherpaSmokeSetsBothTranscriptExpectations(t *testing.T) {
	args := d2SherpaSmokeExpectationArgs()
	require.Equal(t, []string{
		"--env", "SHERPA_ONNX_STREAMING_EXPECT_TEXT=" + d2ExpectedText,
		"--env", "SHERPA_ONNX_STREAMING_EXPECT_FINAL_SUFFIX=" + d2ExpectedFinalSuffix,
	}, args)
	require.NotEmpty(t, d2ExpectedFinalSuffix)
	require.Contains(t, d2ExpectedText, d2ExpectedFinalSuffix)
}

func TestD1AcceleratorInventoryReportsSupportWithoutQualification(t *testing.T) {
	for _, tt := range []struct {
		name      string
		engine    *sttv1.EngineInfo
		fallback  *sttv1.EngineInfo
		available bool
		reason    string
	}{
		{name: "supported route", engine: &sttv1.EngineInfo{Id: "kyutai", Available: true, Verdict: "selected"}, available: true},
		{name: "unsupported route", engine: &sttv1.EngineInfo{Id: "kyutai", Available: false, Verdict: "rejected", Reason: "host unsupported"}, reason: "host unsupported"},
		{name: "missing route", reason: "Kyutai engine is absent from the live engine inventory"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			status := d1AcceleratorStatus(tt.engine, tt.fallback)
			require.Equal(t, "kyutai", status["engine_id"])
			require.Equal(t, "unqualified", status["state"], "inventory is not inference evidence")
			require.Equal(t, "not_run", status["inference"])
			require.Equal(t, tt.reason, status["reason"])
			if tt.engine != nil {
				require.Equal(t, tt.available, status["available"])
				require.Equal(t, tt.engine.GetVerdict(), status["verdict"])
			}
		})
	}
}

func TestHostPortabilityDetailsDefaultsMissingInventoryAndRunnerResults(t *testing.T) {
	details := hostPortabilityDetails(map[string]*sttv1.EngineInfo{"nil-engine": nil}, nil, nil)
	require.Empty(t, details["engine_inventory"])
	require.Equal(t, "D1 CPU runner did not return a result", details["d1_cpu"].(map[string]any)["reason"])
	require.Equal(t, "D2 runner did not return a result", details["d2"].(map[string]any)["reason"])
	require.Equal(t, "not_run", details["d1_cpu"].(map[string]any)["inference"])
	require.Equal(t, "not_run", details["d2"].(map[string]any)["inference"])
}

func TestHostPortabilityProductPathDiagnosticNamesAuthorizedGapWithoutClaimingCapture(t *testing.T) {
	diagnostic := hostPortabilityProductPathDiagnostic()
	require.Equal(t, "not_attempted_by_host_inventory_producer", diagnostic["state"])
	require.Equal(t, "test-genie execute audio-tools --phases soak", diagnostic["registered_authorized_route"])
	require.Contains(t, diagnostic["registered_phase_scope"], "does not prove the local D1 CPU")
	require.Contains(t, diagnostic["remaining_owner_work"], "managed D1 product-path cell")
	require.Contains(t, diagnostic["required_prerequisites"], "active Test Genie demand-lease receipt verified for target audio-tools")
}

func TestHostInferenceEvidenceDoesNotPersistModelOutput(t *testing.T) {
	transcript := "private fixture transcript: the quick brown fox jumps over the lazy dog"
	d2Output := []byte("model output: private sherpa transcript")
	receipt := map[string]any{
		"d1_cpu": map[string]any{"transcript_match": d1TranscriptEvidence(transcript)},
		"d2":     map[string]any{"smoke_test": d2SmokeEvidence(d2Output)},
	}
	encoded, err := json.Marshal(receipt)
	require.NoError(t, err)
	serialized := string(encoded)
	require.NotContains(t, serialized, transcript)
	require.NotContains(t, serialized, string(d2Output))
	require.True(t, d1TranscriptEvidence(transcript), "the actual transcript phrase check must remain in force")
	require.Contains(t, serialized, `"transcript_match":true`)
	require.Contains(t, serialized, `"passed":true`)
	require.Contains(t, serialized, outputSHA256(d2Output))
}
