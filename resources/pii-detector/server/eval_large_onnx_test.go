//go:build pii_onnx

package main

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
)

// The large tier is ab-ai/pii_model_based_on_distilbert: an uncased
// DistilBertForTokenClassification (WordPiece, plain BIO) with 65 labels over ~32
// PII categories, Apache-2.0. It reuses engine_onnx.go verbatim — the only reason
// it is a SEPARATE engine-level artifact from the small tier is the model weights
// + label vocabulary, so these tests load it through the identical NERModel seam.
//
// They are gated on a separate env (PII_DETECTOR_LARGE_MODEL_DIR) so the small-tier
// live tests (PII_DETECTOR_MODEL_DIR) and the large-tier live tests can run
// independently; both skip-with-reason when their artifacts are absent so CI stays
// green on hosts without the staged model.
//
// Stage locally with:
//
//	PII_DETECTOR_LARGE_MODEL_DIR=<dir with model.onnx,vocab.txt,config.json,tokenizer_config.json>
//	PII_DETECTOR_ONNXRUNTIME_LIB=<path to libonnxruntime.so (ORT 1.23.x)>

// withLargeModelEnv points newNERModelFromEnv at the staged large model for the
// duration of t. It returns false (and the caller skips) when the artifacts are
// not staged.
func withLargeModelEnv(t *testing.T) bool {
	t.Helper()
	dir := os.Getenv("PII_DETECTOR_LARGE_MODEL_DIR")
	lib := os.Getenv("PII_DETECTOR_ONNXRUNTIME_LIB")
	if dir == "" || lib == "" {
		t.Skip("skip large-tier NER test: set PII_DETECTOR_LARGE_MODEL_DIR and PII_DETECTOR_ONNXRUNTIME_LIB to the staged large model + ONNX Runtime shared library")
		return false
	}
	if !fileReadable(lib) {
		t.Skipf("skip large-tier NER test: ONNX Runtime lib not readable at %s", lib)
		return false
	}
	// The large model is exported as model.onnx (fp32), not the small tier's
	// INT8 model.quant.onnx; point the loader at it and at the model dir.
	t.Setenv("PII_DETECTOR_MODEL_DIR", dir)
	t.Setenv("PII_DETECTOR_ONNX_FILE", "model.onnx")
	t.Setenv("PII_DETECTOR_MODEL", "ab-ai/pii_model_based_on_distilbert")
	return true
}

// TestLargeModelDetectsPII proves the large tier loads and detects the person-name
// and structured-ID categories at byte-exact offsets through the SAME engine the
// small tier uses — the core evidence that the larger token-classification tier is
// a clean model swap, not a new code path.
func TestLargeModelDetectsPII(t *testing.T) {
	if !withLargeModelEnv(t) {
		return
	}

	model, err := newNERModelFromEnv()
	if err != nil {
		t.Fatalf("newNERModelFromEnv: %v", err)
	}
	if u, ok := model.(UnavailableModel); ok {
		t.Fatalf("large model unavailable despite staged artifacts: %s", u.reason)
	}
	defer model.Close()

	text := "My name is John Smith and my email is john.smith@acme.com, SSN 123-45-6789."
	spans, err := model.Detect(context.Background(), text)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(spans) == 0 {
		t.Fatal("large model returned no spans for text containing a name, email, and SSN")
	}

	byCat := map[string]Span{}
	for _, s := range spans {
		byCat[s.Category] = s
		if s.Source != "model" {
			t.Errorf("span %+v source = %q, want model", s, s.Source)
		}
		if s.Start < 0 || s.End > len(text) || s.Start >= s.End {
			t.Errorf("span %+v has bad byte offsets", s)
		}
		if text[s.Start:s.End] != s.Text {
			t.Errorf("span text %q does not index source %q", s.Text, text[s.Start:s.End])
		}
	}
	// The model must at least find the given name — a category the regex pass
	// cannot cover (the point of the NER tier). ab-ai uses FIRSTNAME/LASTNAME.
	if _, ok := byCat["FIRSTNAME"]; !ok {
		t.Errorf("large model did not detect FIRSTNAME in %q; categories found: %v", text, keysOf(byCat))
	}
	t.Logf("detected %d model spans: %+v", len(spans), spans)
}

// TestLargePIIEval runs the labeled large-tier fixture through the FULL hybrid
// pass (regex + large model, merged) and reports category-level recall, exactly
// like TestPIIEval does for the small tier. It is the measurement the
// model-policy large-tier `measurement` block is populated from.
func TestLargePIIEval(t *testing.T) {
	if !withLargeModelEnv(t) {
		return
	}
	cases := loadEval(t, "testdata/pii_eval_large.jsonl")
	if len(cases) == 0 {
		t.Fatal("no large-tier eval cases loaded")
	}

	model, err := newNERModelFromEnv()
	if err != nil {
		t.Fatalf("newNERModelFromEnv: %v", err)
	}
	if u, ok := model.(UnavailableModel); ok {
		t.Fatalf("large model unavailable despite staged artifacts: %s", u.reason)
	}
	defer model.Close()

	h := &detectHandler{model: model}
	req := httptest.NewRequest("POST", "/detect", nil)

	var wantTotal, found int
	perCat := map[string][2]int{}
	for _, c := range cases {
		spans, _, _ := h.spansFor(req, c.Text)
		got := map[string]bool{}
		for _, s := range spans {
			got[s.Category] = true
		}
		for _, want := range c.Expected {
			cnt := perCat[want]
			cnt[1]++
			wantTotal++
			if got[want] {
				cnt[0]++
				found++
			}
			perCat[want] = cnt
		}
	}

	recall := float64(found) / float64(wantTotal)
	t.Logf("large-tier category recall over %d expectations: %.3f (%d/%d)", len(cases), recall, found, wantTotal)
	for cat, fe := range perCat {
		t.Logf("  %-15s %d/%d", cat, fe[0], fe[1])
	}

	// Conservative floor: the hybrid pass must recover at least 70% of labeled
	// categories. A silent model fallback (regex-only) would drop
	// FIRSTNAME/LASTNAME/USERNAME/URL/AGE and sink below this.
	//
	// The fixture asserts the category NAMES the hybrid pass actually emits for
	// this model: ab-ai labels SSN/IBAN/PHONENUMBER itself (model source), and
	// MergeSpans keeps the higher-confidence span on an overlap, so a
	// high-confidence model label displaces the regex pass's US_SSN/IBAN_CODE/
	// PHONE_NUMBER name on the same bytes. EMAIL overlaps both passes; whichever
	// wins the tie keeps its name (both are correct detections). This is testing
	// the model's real, desired behavior, not pinning an implementation detail.
	const floor = 0.70
	if recall < floor {
		t.Errorf("large-tier category recall %.3f below floor %.2f; the model NER pass may have regressed to regex-only", recall, floor)
	}
}
