package libraryhealth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	libraryhealthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/libraryhealth"
	libraryhealthconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/libraryhealth/libraryhealth_v1connect"

	libraryhealthhandler "react-component-library/handlers/libraryhealth"
	domain "react-component-library/internal/libraryhealth"
)

func call(router *mux.Router, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, libraryhealthconnect.LibraryHealthServiceGetLibraryHealthProcedure, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestGetLibraryHealthRPC(t *testing.T) {
	var activityDays int
	providers := domain.Providers{
		Overview: func(context.Context) (domain.OverviewCounts, error) {
			return domain.OverviewCounts{Assets: 12, Published: 9, Workspaces: 3, FindingsOpen: 4, CoverageRows: 40}, nil
		},
		Quality: func(context.Context) ([]domain.Metric, error) {
			return []domain.Metric{
				{Key: "coverage_rows", Label: "Coverage rows", Value: 40, Total: 40},
				{Key: "pass", Label: "Passing", Value: 34, Total: 40},
			}, nil
		},
		Activity: func(_ context.Context, days int) ([]domain.Metric, error) {
			activityDays = days
			return []domain.Metric{{Key: "events", Label: "Events", Value: 7}}, nil
		},
	}
	router := mux.NewRouter()
	libraryhealthhandler.ModuleFromProviders(providers).Mount(router)

	response := call(router, `{"activity_days":45}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var got libraryhealthv1.GetLibraryHealthResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &got))
	require.NotNil(t, got.Health)
	require.Equal(t, int32(12), got.Health.Counts.Assets)
	require.Equal(t, int32(9), got.Health.Counts.Published)
	require.Equal(t, int32(3), got.Health.Counts.Workspaces)
	require.Equal(t, int32(4), got.Health.Counts.FindingsOpen)
	require.Equal(t, int32(40), got.Health.Counts.CoverageRows)
	require.Equal(t, libraryhealthv1.SectionState_SECTION_STATE_AVAILABLE, got.Health.Overview.State)
	require.Equal(t, libraryhealthv1.SectionState_SECTION_STATE_AVAILABLE, got.Health.Quality.State)
	require.Equal(t, libraryhealthv1.SectionState_SECTION_STATE_AVAILABLE, got.Health.Activity.State)
	require.Equal(t, libraryhealthv1.SectionState_SECTION_STATE_UNAVAILABLE, got.Health.Services.State)
	require.Contains(t, got.Health.Services.Reason, "not configured")
	require.Equal(t, 45, activityDays)
	require.NotEmpty(t, got.Health.GeneratedAt)
}

func TestDegradedSectionsCarryReasonsAndNoZeroMetrics(t *testing.T) {
	providers := domain.Providers{
		Overview: func(context.Context) (domain.OverviewCounts, error) {
			return domain.OverviewCounts{}, errors.New("workspaces table is missing")
		},
		Quality: func(context.Context) ([]domain.Metric, error) {
			return nil, errors.New("catalog_gate_evidence table is missing")
		},
		Activity: func(context.Context, int) ([]domain.Metric, error) {
			return nil, nil
		},
	}
	router := mux.NewRouter()
	libraryhealthhandler.ModuleFromProviders(providers).Mount(router)

	response := call(router, `{}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var got libraryhealthv1.GetLibraryHealthResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &got))
	require.Equal(t, libraryhealthv1.SectionState_SECTION_STATE_UNAVAILABLE, got.Health.Overview.State)
	require.Contains(t, got.Health.Overview.Reason, "workspaces")
	require.Empty(t, got.Health.Overview.Metrics)
	require.Equal(t, libraryhealthv1.SectionState_SECTION_STATE_UNAVAILABLE, got.Health.Quality.State)
	require.Empty(t, got.Health.Quality.Metrics)
	// A provider returning an empty metric set without an error is a measured
	// empty section, not an unavailable one.
	require.Equal(t, libraryhealthv1.SectionState_SECTION_STATE_EMPTY, got.Health.Activity.State)
}
