package fitness

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScoreFitnessValidationAndDefaults(t *testing.T) {
	h := NewHandler(func(string, map[string]interface{}) {})
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "invalid json", body: "{", want: http.StatusBadRequest},
		{name: "missing scenario", body: `{ "tiers": [2] }`, want: http.StatusBadRequest},
		{name: "all tiers default", body: `{ "scenario": "demo" }`, want: http.StatusOK},
		{name: "explicit invalid tier blocker", body: `{ "scenario": "demo", "tiers": [0] }`, want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/fitness/score", bytes.NewBufferString(tt.body))
			rr := httptest.NewRecorder()
			h.ScoreFitness(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("status=%d body=%s want=%d", rr.Code, rr.Body.String(), tt.want)
			}
			if tt.want == http.StatusOK {
				var response struct {
					Scores   map[string]interface{} `json:"scores"`
					Blockers []string               `json:"blockers"`
				}
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if len(response.Scores) == 0 {
					t.Fatal("expected fitness scores")
				}
				if tt.name == "explicit invalid tier blocker" && len(response.Blockers) != 1 {
					t.Fatalf("blockers=%v", response.Blockers)
				}
			}
		})
	}
}
