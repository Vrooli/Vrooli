package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	evalH "audio-tools/handlers/eval"
	"audio-tools/internal/ai/sttchain"
	inteval "audio-tools/internal/eval"
	exprecipe "audio-tools/internal/experiment/recipe"
	sttpkg "audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"

	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

func TestBuildSpeakerConditions_AblationGrid(t *testing.T) {
	got := buildSpeakerConditions(&experimentv1.SpeakerExperimentRecipe{
		TargetProfileId:     "target-voice",
		ExtractionEnabled:   true,
		VerificationEnabled: true,
		VerificationMode:    sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY,
		Threshold:           0.72,
		AblationEnabled:     true,
	})

	wantIDs := []string{
		"extract_off_verify_off",
		"extract_on_verify_off",
		"extract_off_verify_on",
		"extract_on_verify_on",
	}
	if len(got) != len(wantIDs) {
		t.Fatalf("condition count = %d, want %d", len(got), len(wantIDs))
	}
	for i, want := range wantIDs {
		if got[i].ID != want {
			t.Fatalf("condition[%d].ID = %q, want %q", i, got[i].ID, want)
		}
		if got[i].Config.ProfileIDs[0] != "target-voice" {
			t.Fatalf("condition[%d] profile = %v", i, got[i].Config.ProfileIDs)
		}
		if got[i].Config.Threshold != 0.72 {
			t.Fatalf("condition[%d] threshold = %.2f", i, got[i].Config.Threshold)
		}
	}
	if !got[1].ExtractionEnabled || got[1].VerificationEnabled {
		t.Fatalf("extract_on_verify_off toggles = extraction %t verification %t", got[1].ExtractionEnabled, got[1].VerificationEnabled)
	}
	if got[2].ExtractionEnabled || !got[2].VerificationEnabled {
		t.Fatalf("extract_off_verify_on toggles = extraction %t verification %t", got[2].ExtractionEnabled, got[2].VerificationEnabled)
	}
}

func TestBuildSpeakerConditions_SkipsMissingTargetProfile(t *testing.T) {
	got := buildSpeakerConditions(&experimentv1.SpeakerExperimentRecipe{
		VerificationEnabled: true,
	})
	if len(got) != 1 {
		t.Fatalf("condition count = %d, want 1", len(got))
	}
	if !got[0].Skipped {
		t.Fatalf("condition should be skipped without a target profile")
	}
	if got[0].Note == "" {
		t.Fatalf("skipped condition should explain why")
	}
}

func TestExperimentEvalDeps_CarriesSpeakerResource(t *testing.T) {
	client := &sttpipeline.SpeakerClient{}
	deps := newExperimentEvalDeps(nil, nil, func() sttchain.Provider { return nil }, sttpkg.Defaults(), client)

	if deps.SpeakerResource != client {
		t.Fatalf("SpeakerResource not wired into experiment eval deps")
	}
}

func TestApplySpeakerResourceAvailability_SkipsEnabledConditionsWhenResourceMissing(t *testing.T) {
	conditions := []speakerEvalCondition{
		{ID: "extract_off_verify_off"},
		{ID: "extract_on_verify_off", ExtractionEnabled: true},
		{ID: "extract_off_verify_on", VerificationEnabled: true},
	}

	got := applySpeakerResourceAvailability(context.Background(), conditions, nil)

	if got[0].Skipped {
		t.Fatalf("speaker-off condition should remain runnable")
	}
	for i := 1; i < len(got); i++ {
		if !got[i].Skipped {
			t.Fatalf("condition %s should be skipped when speaker resource is missing", got[i].ID)
		}
		if got[i].Note == "" {
			t.Fatalf("condition %s should explain the skipped resource", got[i].ID)
		}
	}
}

func TestApplySpeakerResourceAvailability_TreatsOKReadyStatusAsReady(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","model_loaded":true,"profile_store_ok":true,"temp_dir_ok":true}`))
	}))
	defer srv.Close()
	client := &sttpipeline.SpeakerClient{BaseURL: srv.URL, Doer: srv.Client()}
	conditions := []speakerEvalCondition{{
		ID:                "extract_on_verify_on",
		ExtractionEnabled: true,
	}}

	got := applySpeakerResourceAvailability(context.Background(), conditions, client)

	if got[0].Skipped {
		t.Fatalf("status=ok with all readiness booleans true should be treated as ready: %#v", got[0])
	}
}

func TestRunSpeakerConditionReportsWithOptions_AllSkippedReturnsWarningReport(t *testing.T) {
	report, err := runSpeakerConditionReportsWithOptions(context.Background(), evalDepsNoop(), nil, nil, 0, 0, inteval.EvalOptions{}, []speakerEvalCondition{{
		ID:      "speaker_recipe",
		Skipped: true,
		Note:    "speaker condition skipped: speaker resource is not configured",
	}}, "", nil)
	if err != nil {
		t.Fatalf("all-skipped speaker conditions should return a warning report, got error: %v", err)
	}
	if len(report.PerStrategy) != 0 {
		t.Fatalf("all-skipped report should not emit clean strategy rows: %#v", report.PerStrategy)
	}
	if len(report.Warnings) != 1 || report.Warnings[0].Code != "speaker_condition_skipped" {
		t.Fatalf("warnings = %#v", report.Warnings)
	}
	if report.Summary.Confidence != "low" {
		t.Fatalf("summary confidence = %q, want low", report.Summary.Confidence)
	}
}

func TestAppendUniqueReportWarnings_DeduplicatesRepeatedConditionWarnings(t *testing.T) {
	warning := inteval.ReportWarning{Severity: "warning", Code: "tiny_corpus", Message: "Only 1 clips were evaluated"}
	got := appendUniqueReportWarnings([]inteval.ReportWarning{warning}, warning, inteval.ReportWarning{
		Severity: "warning",
		Code:     "short_audio",
		Message:  "The evaluated audio totals 2.7 seconds",
	})

	if len(got) != 2 {
		t.Fatalf("warning count = %d, want 2: %#v", len(got), got)
	}
	if got[0].Code != "tiny_corpus" || got[1].Code != "short_audio" {
		t.Fatalf("warnings = %#v", got)
	}
}

func TestExperimentRunsForReportStoresPerCellConditions(t *testing.T) {
	report := inteval.EvalReport{PerStrategy: []inteval.StrategyReport{
		{
			Strategy:            sttchain.StrategyKind("batch/extract_off_verify_off/clean"),
			BaseStrategy:        sttchain.StrategyKind("batch"),
			Label:               "Batch / extract_off_verify_off / clean",
			ConditionGroup:      "clean",
			ExtractionEnabled:   false,
			VerificationEnabled: false,
		},
		{
			Strategy:            sttchain.StrategyKind("batch/extract_on_verify_on/noisy"),
			BaseStrategy:        sttchain.StrategyKind("batch"),
			Label:               "Batch / extract_on_verify_on / noisy",
			ConditionGroup:      "noisy",
			ExtractionEnabled:   true,
			VerificationEnabled: true,
		},
	}}
	realized := map[string]any{"phase": "materialized", "clip_count": 2}
	runs, err := experimentRunsForReport(report, evalH.ReportToProto(report), realized)
	if err != nil {
		t.Fatalf("experimentRunsForReport: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("run count = %d, want 2", len(runs))
	}
	var first, second map[string]any
	if err := json.Unmarshal(runs[0].ConditionJSON, &first); err != nil {
		t.Fatalf("unmarshal first condition: %v", err)
	}
	if err := json.Unmarshal(runs[1].ConditionJSON, &second); err != nil {
		t.Fatalf("unmarshal second condition: %v", err)
	}
	if first["condition_group"] == second["condition_group"] {
		t.Fatalf("condition groups should differ: first=%v second=%v", first, second)
	}
	firstSpeaker := first["speaker"].(map[string]any)
	secondSpeaker := second["speaker"].(map[string]any)
	if firstSpeaker["extraction_enabled"].(bool) || firstSpeaker["verification_enabled"].(bool) {
		t.Fatalf("first speaker condition should be disabled: %v", firstSpeaker)
	}
	if !secondSpeaker["extraction_enabled"].(bool) || !secondSpeaker["verification_enabled"].(bool) {
		t.Fatalf("second speaker condition should be enabled: %v", secondSpeaker)
	}
}

func evalDepsNoop() evalH.Deps {
	return evalH.Deps{}
}

func TestLongFormSourceDurationWarning_WhenTargetExceedsUniqueSourceAudio(t *testing.T) {
	warning, ok := longFormSourceDurationWarning(&experimentv1.LongFormRecipe{
		TargetDurationSeconds: 10,
	}, []exprecipe.Clip{{
		ID:         "clip-1",
		PCM:        make([]byte, exprecipe.CanonicalSampleRate*2*2),
		SampleRate: exprecipe.CanonicalSampleRate,
	}})

	if !ok {
		t.Fatalf("expected under-target source-duration warning")
	}
	if warning.Code != "source_audio_under_target" {
		t.Fatalf("warning code = %q", warning.Code)
	}
	if warning.Severity != "warning" {
		t.Fatalf("warning severity = %q", warning.Severity)
	}
}

func TestLongFormSourceDurationWarning_SuppressedWhenSourceCoversTarget(t *testing.T) {
	_, ok := longFormSourceDurationWarning(&experimentv1.LongFormRecipe{
		SweepDurationsSeconds: []int32{1, 2},
	}, []exprecipe.Clip{{
		ID:         "clip-1",
		PCM:        make([]byte, exprecipe.CanonicalSampleRate*2*3),
		SampleRate: exprecipe.CanonicalSampleRate,
	}})

	if ok {
		t.Fatalf("did not expect warning when unique source audio covers the largest target")
	}
}
