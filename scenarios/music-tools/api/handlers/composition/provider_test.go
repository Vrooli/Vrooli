package composition

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/vrooli/ai-provider-go/policy"

	"music-tools/internal/models"
	"music-tools/internal/styles"
)

func allowRightsForTest(_ context.Context, _ json.RawMessage, _ string) (models.RightsDecision, error) {
	return models.RightsDecision{Status: "allowed", DecisionSHA256: "test"}, nil
}

func TestExplicitUnsupportedLyricsExplainsCapabilityWithoutDroppingText(t *testing.T) {
	const lyrics = "[Verse]\nOur words remain\n\n[Chorus]\nSing together\n"
	input := styles.Compiled{StyleID: "song", Caption: "original song", Lyrics: lyrics, Params: styles.Params{Variant: "stable-audio-3-medium", Duration: 30}}
	_, err := resolveComposition(context.Background(), input, allowRightsForTest)
	if err == nil || !strings.Contains(err.Error(), "intelligible vocals") || !strings.Contains(err.Error(), "77e0398c26f7d22d801ec1bb036307e8b231f741/docs/guides/prompting.md") {
		t.Fatalf("unsupported lyrics lack actual model reason/source: %v", err)
	}
	if input.Lyrics != lyrics {
		t.Fatal("routing changed authored lyrics")
	}
}

func TestSA3AdmissionRechecksCurrentRightsBeforeResourceReadiness(t *testing.T) {
	s, _ := testStateWithBrokerDB(t, nil)
	defer s.JobManager.Close()

	model, err := sa3RightsModel(nil)
	if err != nil {
		t.Fatal(err)
	}
	var allowed atomic.Bool
	allowed.Store(true)
	var evaluations atomic.Int32
	shared := policy.NewRightsEvaluator(func(context.Context) (policy.RightsContext, error) {
		status, version := "at_or_above_threshold", "crossed"
		if allowed.Load() {
			status, version = "pre_revenue", "below"
		}
		return policy.RightsContext{ID: "operator-business-context/current", Version: version, EntityScope: "operator-group", RevenueStatus: status, AsOf: "2026-11-14", Source: "operator_statement"}, nil
	})
	wrapped := models.AdaptRightsEvaluator(shared)
	s.RightsEvaluator = func(ctx context.Context, raw json.RawMessage, use string) (models.RightsDecision, error) {
		evaluations.Add(1)
		if use != models.RightsUseLocalInference || len(raw) == 0 {
			t.Fatalf("shared rights evaluator input: use=%q raw=%q", use, raw)
		}
		return wrapped(ctx, raw, use)
	}
	var readinessCalls atomic.Int32
	var unexpectedPath atomic.Value
	ready := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		readinessCalls.Add(1)
		if r.URL.Path != "/health" {
			unexpectedPath.Store(r.URL.Path)
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `{"model_loaded":true,"model_id":%q,"model_revision":%q,"device":"cuda","error":null}`, sa3Model, sa3Revision)
	}))
	defer ready.Close()
	s.SA3URL = ready.URL
	s.HTTP = ready.Client()
	payload := composeJobPayload{ModelSnapshot: &model}

	if _, _, err := s.sa3Admission(context.Background(), payload); err != nil {
		t.Fatalf("allowed current-context admission: %v", err)
	}
	if readinessCalls.Load() != 1 || evaluations.Load() != 1 {
		t.Fatalf("first admission: rights evaluations=%d readiness calls=%d", evaluations.Load(), readinessCalls.Load())
	}
	if unexpected := unexpectedPath.Load(); unexpected != nil {
		t.Fatalf("unexpected SA3 readiness path %v", unexpected)
	}

	allowed.Store(false) // Hypothetical trusted-context crossing; not an operator fact.
	if _, _, err := s.sa3Admission(context.Background(), payload); err == nil || !strings.Contains(err.Error(), "revenue_condition_not_satisfied") {
		t.Fatalf("changed context admission=%v, want current-rights refusal", err)
	}
	if readinessCalls.Load() != 1 || evaluations.Load() != 2 {
		t.Fatalf("denied admission reached resource readiness: rights evaluations=%d readiness calls=%d", evaluations.Load(), readinessCalls.Load())
	}

	allowed.Store(true) // Restoring trusted context restores this exact route.
	if _, _, err := s.sa3Admission(context.Background(), payload); err != nil {
		t.Fatalf("restored current-context admission: %v", err)
	}
	if readinessCalls.Load() != 2 || evaluations.Load() != 3 {
		t.Fatalf("restored admission: rights evaluations=%d readiness calls=%d", evaluations.Load(), readinessCalls.Load())
	}
}

func TestCapabilitySelectionAndRecordedProviderValidationPreserveLyrics(t *testing.T) {
	const lyrics = "[Verse]\nExact words\n"
	compiled := styles.Compiled{StyleID: "song", Caption: "original pop song", Lyrics: lyrics, Params: styles.Params{BPM: 150, Duration: 30}}
	selected, err := resolveComposition(context.Background(), compiled, allowRightsForTest)
	if err != nil || selected.Engine != providerACE {
		t.Fatalf("real available lyric-capable route: %+v %v", selected, err)
	}
	compiled.Params.Variant = "stable-audio-3-medium"
	if err := validateProviderInput(providerSA3, compiled, composeRequest{Duration: 30}, false); err == nil || !strings.Contains(err.Error(), "intelligible vocals") {
		t.Fatalf("recorded provider validation skipped declared capability: %v", err)
	}
	compiled.Params.Variant = "acestep-v15-sft"
	if err := validateProviderInput(providerACE, compiled, composeRequest{Duration: 30}, false); err != nil {
		t.Fatalf("explicit real lyric-capable variant rejected: %v", err)
	}
	compiled.Params.BPM = 0
	if err := validateProviderInput(providerACE, compiled, composeRequest{Duration: 30}, false); err == nil || !strings.Contains(err.Error(), "positive structured bpm") {
		t.Fatalf("actual adapter control validation lost: %v", err)
	}
	if compiled.Lyrics != lyrics {
		t.Fatal("capability filtering rewrote lyrics")
	}
}

func TestVocalLanguageUsesModelDeclaredCapabilities(t *testing.T) {
	compiled := styles.Compiled{Caption: "authored song", Lyrics: "[Verse]\nOriginal words", Params: styles.Params{BPM: 120, Duration: 30, Variant: aceDefaultVariant}}
	for _, value := range []string{"en", "ko"} {
		language := value
		if err := validateProviderInput(providerACE, compiled, composeRequest{Duration: 30, VocalLanguage: &language}, false); err != nil {
			t.Fatalf("declared language %q rejected: %v", value, err)
		}
	}
	unsupported := "ja"
	if err := validateProviderInput(providerACE, compiled, composeRequest{Duration: 30, VocalLanguage: &unsupported}, false); err == nil || !strings.Contains(err.Error(), `does not declare vocal language "ja"`) {
		t.Fatalf("undeclared language accepted or unclear: %v", err)
	}
	empty := "  "
	if err := validateProviderInput(providerACE, compiled, composeRequest{Duration: 30, VocalLanguage: &empty}, false); err == nil || !strings.Contains(err.Error(), "must be omitted") {
		t.Fatalf("explicit empty language accepted: %v", err)
	}
	spaced := " ko "
	if err := validateProviderInput(providerACE, compiled, composeRequest{Duration: 30, VocalLanguage: &spaced}, false); err == nil || !strings.Contains(err.Error(), "surrounding whitespace") {
		t.Fatalf("noncanonical language accepted: %v", err)
	}
	compiled.Params.Variant = "stable-audio-3-medium"
	english := "en"
	if err := validateProviderInput(providerSA3, compiled, composeRequest{Duration: 30, VocalLanguage: &english}, false); err == nil || !strings.Contains(err.Error(), "does not declare vocal language") {
		t.Fatalf("unsupported provider language accepted: %v", err)
	}
}

func TestExplicitUnsupportedLyricsRefusedBeforeQueueWithSourcedReason(t *testing.T) {
	s, _ := testStateWithBrokerDB(t, nil)
	const text = "[Verse]\nMy exact words\n"
	_, err := s.Styles.Create(styles.Style{ID: "unsupported-song", Name: "Song", Caption: "a new song", Lyrics: text, Params: styles.Params{Variant: "stable-audio-3-medium", Duration: 30}})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	s.compose(rec, httptest.NewRequest(http.MethodPost, "/api/v1/compose", strings.NewReader(`{"style_id":"unsupported-song","takes":1}`)))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "intelligible vocals") || !strings.Contains(rec.Body.String(), "77e0398c") {
		t.Fatalf("unsupported explicit choice was queued or lacked reason/source: %d %s", rec.Code, rec.Body.String())
	}
	saved, ok := s.Styles.Get("unsupported-song")
	if !ok || saved.Lyrics != text {
		t.Fatal("refused compose discarded or changed lyrics")
	}
}
