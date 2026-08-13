package health_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/health"

	"data-backup-manager/handlers/health"
	"data-backup-manager/internal/testutil/mocks"
	"github.com/vrooli/api-core/apihttptest"
)

// fakePosture implements health.BackupPosture with canned output.
type fakePosture struct {
	degraded bool
	detail   string
	err      error
}

func (f fakePosture) OverdueOrFailed(context.Context) (bool, string, error) {
	return f.degraded, f.detail, f.err
}

// fakePostureSink records posture-degraded emissions.
type fakePostureSink struct{ details []string }

func (f *fakePostureSink) BackupPostureDegraded(_ context.Context, detail string) {
	f.details = append(f.details, detail)
}

func serve(t *testing.T, h http.HandlerFunc) *healthv1.Response {
	t.Helper()
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusOK, rec.Code, "Optional-check failures must stay HTTP 200")
	body, _ := io.ReadAll(rec.Result().Body)
	return apihttptest.MustUnmarshalProto[healthv1.Response](t, body)
}

// TestHealth_DegradedOnOverdue proves DBM-OBS-001: with an overdue/failed
// target, the backup-posture check degrades the overall status (readiness
// stays true, HTTP 200) and a posture event is emitted.
func TestHealth_DegradedOnOverdue(t *testing.T) {
	sink := &fakePostureSink{}
	resp := serve(t, health.NewHandler(health.Deps{
		Pinger:  &mocks.FakePinger{}, // healthy DB
		Service: "data-backup-manager-api",
		Version: "1.0.0",
		Posture: fakePosture{degraded: true, detail: "1 of 2 target(s) overdue or last run failed"},
		Events:  sink,
	}))

	require.Equal(t, "degraded", resp.Status, "overdue/failed backups must degrade status")
	require.True(t, resp.Readiness, "degraded is still ready")
	require.Contains(t, resp.Dependencies, "backups")
	require.False(t, resp.Dependencies["backups"].Connected, "backups dependency reports not-connected when degraded")
	require.Len(t, sink.details, 1, "a posture-degraded event must be emitted")
}

// TestHealth_HealthyWhenBackupsOK confirms the posture check stays quiet when
// nothing is overdue.
func TestHealth_HealthyWhenBackupsOK(t *testing.T) {
	sink := &fakePostureSink{}
	resp := serve(t, health.NewHandler(health.Deps{
		Pinger:  &mocks.FakePinger{},
		Service: "data-backup-manager-api",
		Version: "1.0.0",
		Posture: fakePosture{degraded: false},
		Events:  sink,
	}))
	require.Equal(t, "healthy", resp.Status)
	require.Empty(t, sink.details, "no event when backups are healthy")
}

// TestHealth_PostureErrorDegrades confirms a posture-provider error degrades
// the status rather than crashing the probe.
func TestHealth_PostureErrorDegrades(t *testing.T) {
	resp := serve(t, health.NewHandler(health.Deps{
		Pinger:  &mocks.FakePinger{},
		Service: "s",
		Version: "1",
		Posture: fakePosture{err: errors.New("runs unavailable")},
	}))
	require.Equal(t, "degraded", resp.Status)
}
