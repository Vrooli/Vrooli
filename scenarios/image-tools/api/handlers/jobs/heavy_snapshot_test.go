package jobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	internaljobs "image-tools/internal/jobs"
)

type heavySnapshotFixture struct{ snapshot internaljobs.HeavyJobSnapshot }

func (f heavySnapshotFixture) HeavyJobSnapshot(context.Context) internaljobs.HeavyJobSnapshot {
	return f.snapshot
}

func TestHeavyJobSnapshotHandlerReturnsOwnerTelemetryNoStore(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	handler := newHeavyJobSnapshotHandler(heavySnapshotFixture{snapshot: internaljobs.HeavyJobSnapshot{
		ObservedAt: now, Complete: true, Jobs: []internaljobs.HeavyJobObservation{{
			OwnerID: "image-tools", JobID: "job-7", Operation: "text_to_image", State: "running", StartedAt: now.Add(-time.Hour), Heavy: true,
		}},
	}})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/telemetry/heavy", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" || !strings.Contains(response.Body.String(), `"owner_id":"image-tools"`) || !strings.Contains(response.Body.String(), `"job_id":"job-7"`) {
		t.Fatalf("response = %d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
}

func TestHeavyJobSnapshotHandlerFailsClosedWhenReaderUnavailable(t *testing.T) {
	response := httptest.NewRecorder()
	newHeavyJobSnapshotHandler(struct{}{}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/jobs/telemetry/heavy", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.Code)
	}
}
