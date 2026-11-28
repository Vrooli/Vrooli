package visualhealth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/databasetest"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
)

func TestBrowserObservationsRetainRealClientGeometryWithBoundedHistory(t *testing.T) {
	db := databasetest.NewSQLite(t)
	_, err := db.Exec(ObservationSchema())
	require.NoError(t, err)
	store := NewSQLObservations(db)
	ctx := context.Background()
	row := &pb.BrowserObservation{Scenario: "git-control-tower", ClientId: "phone-tab", Standalone: true, BuildId: "entry-hash.js", LayoutJson: `{"viewport":{"height":828},"screen":{"height":852},"safeArea":{"bottom":34}}`, CaptureKind: "physical_screenshot", ReceivedAt: "fabricated"}
	saved, err := store.Record(ctx, row)
	require.NoError(t, err)
	require.Equal(t, "browser_geometry", saved.CaptureKind)
	require.NotEqual(t, "fabricated", saved.ReceivedAt)
	require.NotEmpty(t, saved.Id)
	require.Equal(t, "physical_screenshot", row.CaptureKind, "input is not mutated")
	// Read through a fresh repository to prove persistence, not a process-local cache.
	records, err := NewSQLObservations(db).List(ctx, "git-control-tower", "phone-tab", 20)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, saved.LayoutJson, records[0].LayoutJson)
	other, err := store.List(ctx, "another-scenario", "", 100)
	require.NoError(t, err)
	require.Empty(t, other)
	_, err = db.Exec(`UPDATE browser_layout_observations SET received_at = ?`, time.Now().Add(-25*time.Hour).UTC().Format(time.RFC3339Nano))
	require.NoError(t, err)
	records, err = store.List(ctx, "git-control-tower", "", 20)
	require.NoError(t, err)
	require.Empty(t, records)
	for i := 0; i < 260; i++ {
		_, err = store.Record(ctx, row)
		require.NoError(t, err)
	}
	var count int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM browser_layout_observations`).Scan(&count))
	require.Equal(t, 256, count)
}

func TestBrowserObservationsRejectUnboundedOrInvalidInput(t *testing.T) {
	store := NewSQLObservations(databasetest.NewSQLite(t))
	for _, row := range []*pb.BrowserObservation{nil, {}, {Scenario: "gct", ClientId: "x", LayoutJson: "oops"}, {Scenario: "gct", ClientId: "x", LayoutJson: strings.Repeat("x", 16385)}} {
		_, err := store.Record(context.Background(), row)
		require.Error(t, err)
	}
}

func TestBrowserObservationFindingsCatchThePhoneRegressionAndDrawerCanvas(t *testing.T) {
	row := &pb.BrowserObservation{Scenario: "git-control-tower", Standalone: true, LayoutJson: `{"expectations":{"fullScreenCanvas":true,"headerMaxHeight":48},"viewport":{"width":393,"height":793},"screen":{"width":393,"height":852},"cssViewport":{"largeHeight":852},"shell":{"bottom":793},"header":{"height":61},"body":{"background":"rgb(2, 6, 23)"},"overlays":[]}`}
	findings := browserObservationFindings(row)
	require.Len(t, findings, 2)
	require.Equal(t, "visual_drawable_canvas_unused", findings[0].Code)
	require.Contains(t, findings[0].Evidence, "unused 59.0px")
	require.Equal(t, "visual_mobile_toolbar_oversized", findings[1].Code)
	row.LayoutJson = `{"expectations":{"fullScreenCanvas":true,"headerMaxHeight":48},"viewport":{"width":393,"height":793},"screen":{"width":393,"height":852},"cssViewport":{"largeHeight":852},"shell":{"bottom":852},"header":{"height":45},"body":{"background":"rgb(2, 6, 23)"},"overlays":[{"panel":{"background":"rgb(23, 32, 51)"}}]}`
	findings = browserObservationFindings(row)
	require.Len(t, findings, 1)
	require.Equal(t, "visual_overlay_canvas_color_mismatch", findings[0].Code)
	row.LayoutJson = strings.Replace(row.LayoutJson, "rgb(2, 6, 23)", "rgb(23, 32, 51)", 1)
	require.Empty(t, browserObservationFindings(row))
}

func TestBrowserObservationFindingsRespectKeyboardAndModalHitOwnership(t *testing.T) {
	row := &pb.BrowserObservation{Standalone: true, LayoutJson: `{"expectations":{"fullScreenCanvas":true},"keyboardVisible":true,"viewport":{"width":393},"screen":{"width":393,"height":852},"cssViewport":{"largeHeight":852},"shell":{"bottom":500},"controls":[{"centerHit":false}],"overlays":[{"panel":{"background":"rgb(23, 32, 51)"}}],"body":{"background":"rgb(23, 32, 51)"}}`}
	require.Empty(t, browserObservationFindings(row), "modal owns hit targets, keyboard owns reduced height")
	row.LayoutJson = `{"controls":[{"centerHit":false}],"overlays":[]}`
	findings := browserObservationFindings(row)
	require.Len(t, findings, 1)
	require.Equal(t, "visual_nav_control_unreachable", findings[0].Code)
}
