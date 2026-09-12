package cleanup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type conformanceRecorder struct {
	t      *testing.T
	mu     sync.Mutex
	errors int
}

func (r *conformanceRecorder) Helper() {}
func (r *conformanceRecorder) Errorf(format string, args ...any) {
	r.mu.Lock()
	r.errors++
	r.mu.Unlock()
	r.t.Logf(format, args...)
}

func TestRunOwnerConformanceAcceptsCorrectOwner(t *testing.T) {
	var mu sync.Mutex
	done := map[string]OwnerApplyResponse{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/cleanup/estimate":
			age := r.URL.Query().Get("min_age_seconds")
			bytes, count := int64(100), 1
			if age == "2592000" {
				bytes, count = 40, 1
			}
			_ = json.NewEncoder(w).Encode(OwnerEstimateResponse{ProviderID: "fake-owner", EstimatedBytes: bytes, ItemCount: count, MinAgeSeconds: parseInt(age)})
		case "/api/v1/cleanup/preview":
			var request OwnerPreviewRequest
			_ = json.NewDecoder(r.Body).Decode(&request)
			_ = json.NewEncoder(w).Encode(OwnerPreviewResponse{ProviderID: "fake-owner", MinAgeSeconds: request.Estimate.MinAgeSeconds, Items: []OwnerPreviewItem{{ID: "old", Bytes: 100, AgeSeconds: request.Estimate.MinAgeSeconds + 1}}})
		case "/api/v1/cleanup/apply":
			var request OwnerApplyRequest
			_ = json.NewDecoder(r.Body).Decode(&request)
			mu.Lock()
			response, ok := done[request.IdempotencyKey]
			if !ok {
				response = OwnerApplyResponse{ReclaimedBytes: int64(len(request.Preview.Items)) * 100}
				if len(request.Preview.Items) > 0 {
					response.RemovedItemIDs = []string{"old"}
				}
				done[request.IdempotencyKey] = response
			}
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(response)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	recorder := &conformanceRecorder{t: t}
	RunOwnerConformance(recorder, server.URL)
	if recorder.errors != 0 {
		t.Fatalf("correct owner had %d conformance errors", recorder.errors)
	}
}

func TestRunOwnerConformanceReportsIncorrectOwner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/cleanup/estimate":
			_ = json.NewEncoder(w).Encode(OwnerEstimateResponse{ProviderID: "bad-owner", EstimatedBytes: 100, ItemCount: 1, MinAgeSeconds: 3600})
		case "/api/v1/cleanup/preview":
			_ = json.NewEncoder(w).Encode(OwnerPreviewResponse{ProviderID: "bad-owner", MinAgeSeconds: 3600, Items: []OwnerPreviewItem{{ID: "young", AgeSeconds: 1, Protected: true}}})
		case "/api/v1/cleanup/apply":
			_ = json.NewEncoder(w).Encode(OwnerApplyResponse{ReclaimedBytes: 1, RemovedItemIDs: []string{"bad"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	recorder := &conformanceRecorder{t: t}
	RunOwnerConformance(recorder, server.URL)
	if recorder.errors < 3 {
		t.Fatalf("incorrect owner produced %d errors, want at least three violated guarantees", recorder.errors)
	}
}

func parseInt(value string) int64 {
	var result int64
	_, _ = fmt.Sscan(value, &result)
	return result
}
