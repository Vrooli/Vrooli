//go:build pii_onnx

package main

import (
	"bufio"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// evalCase is one labeled row in testdata/pii_eval.jsonl: the text and the set
// of PII categories a correct detector should find somewhere in it.
type evalCase struct {
	Text     string   `json:"text"`
	Expected []string `json:"expected"`
}

// TestPIIEval runs the labeled fixture through the FULL hybrid pass (regex +
// model, merged by the HTTP handler's spansFor) and reports category-level
// recall. It is the measurement the model-policy small-tier `measurement` block
// is populated from. It skips without the staged model/runtime so CI stays green.
//
// This is a measurement, not a hard gate: it asserts a conservative recall floor
// so an accidental regression (e.g. the model path silently falling back to
// regex-only) fails loudly, without pinning exact model behavior.
func TestPIIEval(t *testing.T) {
	if os.Getenv("PII_DETECTOR_MODEL_DIR") == "" || os.Getenv("PII_DETECTOR_ONNXRUNTIME_LIB") == "" {
		t.Skip("skip PII eval: set PII_DETECTOR_MODEL_DIR and PII_DETECTOR_ONNXRUNTIME_LIB to the staged model + ONNX Runtime shared library")
	}
	cases := loadEval(t, "testdata/pii_eval.jsonl")
	if len(cases) == 0 {
		t.Fatal("no eval cases loaded")
	}

	model, err := newNERModelFromEnv()
	if err != nil {
		t.Fatalf("newNERModelFromEnv: %v", err)
	}
	if u, ok := model.(UnavailableModel); ok {
		t.Fatalf("model unavailable despite staged artifacts: %s", u.reason)
	}
	defer model.Close()

	h := &detectHandler{model: model}
	req := httptest.NewRequest("POST", "/detect", nil)

	var wantTotal, found int
	perCat := map[string][2]int{} // category -> [found, expected]
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
	t.Logf("category recall over %d expectations: %.3f (%d/%d)", len(cases), recall, found, wantTotal)
	for cat, fe := range perCat {
		t.Logf("  %-15s %d/%d", cat, fe[0], fe[1])
	}

	// Conservative floor: the hybrid pass must recover at least 70% of labeled
	// categories. A silent model fallback (regex-only) would drop PERSON/LOCATION
	// and sink below this.
	const floor = 0.70
	if recall < floor {
		t.Errorf("category recall %.3f below floor %.2f; the model NER pass may have regressed to regex-only", recall, floor)
	}
}

func loadEval(t *testing.T, path string) []evalCase {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open eval fixture: %v", err)
	}
	defer f.Close()
	var cases []evalCase
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var c evalCase
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			t.Fatalf("parse eval line %q: %v", line, err)
		}
		cases = append(cases, c)
	}
	return cases
}
