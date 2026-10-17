package composition

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"music-tools/internal/styles"
)

func TestExplicitUnsupportedLyricsExplainsCapabilityWithoutDroppingText(t *testing.T) {
	const lyrics = "[Verse]\nOur words remain\n\n[Chorus]\nSing together\n"
	input := styles.Compiled{StyleID: "song", Caption: "original song", Lyrics: lyrics, Params: styles.Params{Variant: "stable-audio-3-medium", Duration: 30}}
	_, err := resolveComposition(input)
	if err == nil || !strings.Contains(err.Error(), "intelligible vocals") || !strings.Contains(err.Error(), "77e0398c26f7d22d801ec1bb036307e8b231f741/docs/guides/prompting.md") {
		t.Fatalf("unsupported lyrics lack actual model reason/source: %v", err)
	}
	if input.Lyrics != lyrics {
		t.Fatal("routing changed authored lyrics")
	}
}

func TestCapabilitySelectionAndRecordedProviderValidationPreserveLyrics(t *testing.T) {
	const lyrics = "[Verse]\nExact words\n"
	compiled := styles.Compiled{StyleID: "song", Caption: "original pop song", Lyrics: lyrics, Params: styles.Params{BPM: 150, Duration: 30}}
	selected, err := resolveComposition(compiled)
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
