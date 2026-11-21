package measures

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	measurelib "github.com/vrooli/measures-go"
)

func TestHandlerRequiresEveryComputeProvider(t *testing.T) {
	_, err := Handler(map[string]Provider{})
	if err == nil {
		t.Fatal("expected missing-provider error")
	}
}

func TestHandlerExecutesEveryDeclaredMeasure(t *testing.T) {
	providers := make(map[string]Provider)
	for _, declaration := range declarations() {
		name := declaration.Name
		providers[name] = func(context.Context, measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			return measurelib.MeasureResult{
				Value:      "1",
				Fields:     []map[string]string{{"id": name}},
				Provenance: measurelib.Provenance{ExecutedQuery: "fixture " + name},
			}, nil
		}
	}
	handler, err := Handler(providers)
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}

	for _, declaration := range declarations() {
		body, err := json.Marshal(measurelib.MeasureRequest{Measure: declaration.Name})
		if err != nil {
			t.Fatalf("marshal %s request: %v", declaration.Name, err)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body)))
		if recorder.Code != http.StatusOK {
			t.Errorf("execute %s: status %d, body %s", declaration.Name, recorder.Code, recorder.Body.String())
		}
	}
}
