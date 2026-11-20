package review_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	db "github.com/vrooli/api-core/databasetest"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	reviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review/review_v1connect"

	reviewhandler "react-component-library/handlers/review"
	domain "react-component-library/internal/review"
)

type staticProjector struct{ findings []domain.Finding }

func (p staticProjector) Project(context.Context) ([]domain.Finding, error) {
	return append([]domain.Finding(nil), p.findings...), nil
}

func newRouter(t *testing.T) *mux.Router {
	t.Helper()
	return newRouterWithFindings(t, []domain.Finding{{
		ID:           "finding-1",
		SubjectKind:  "asset",
		SubjectID:    "button",
		Gate:         "accessibility",
		Severity:     domain.SeverityBlocker,
		Status:       domain.StatusOpen,
		Message:      "accessibility gate failed",
		Remediation:  "raise contrast",
		EvidenceJSON: `{"target":"react-vite"}`,
		DocsRef:      "docs/concepts/STORY-CONTRACT.md",
		CreatedAt:    time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}})
}

func newRouterWithFindings(t *testing.T, findings []domain.Finding) *mux.Router {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.ExecContext(context.Background(), domain.Schema())
	require.NoError(t, err)

	service := domain.NewService(sqlDB, staticProjector{findings: findings}).WithTTL(0)
	router := mux.NewRouter()
	reviewhandler.ModuleFromService(service).Mount(router)
	return router
}

func call(router *mux.Router, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestListAndGetFinding(t *testing.T) {
	router := newRouter(t)

	response := call(router, reviewconnect.ReviewServiceListFindingsProcedure, `{"severity":"FINDING_SEVERITY_BLOCKER"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var listed reviewv1.ListFindingsResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &listed))
	require.Len(t, listed.Findings, 1)
	require.Equal(t, int32(1), listed.Total)
	require.Equal(t, "finding-1", listed.Findings[0].Id)
	require.Equal(t, reviewv1.FindingSeverity_FINDING_SEVERITY_BLOCKER, listed.Findings[0].Severity)
	require.Equal(t, reviewv1.FindingStatus_FINDING_STATUS_OPEN, listed.Findings[0].Status)

	response = call(router, reviewconnect.ReviewServiceGetFindingProcedure, `{"id":"finding-1"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var got reviewv1.GetFindingResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &got))
	require.Equal(t, "button", got.Finding.SubjectId)
	require.Equal(t, "raise contrast", got.Finding.Remediation)
}

func TestTriageRPCs(t *testing.T) {
	router := newRouter(t)

	// The queue is projected on read; triage targets a finding the caller has
	// already listed.
	response := call(router, reviewconnect.ReviewServiceListFindingsProcedure, `{}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	response = call(router, reviewconnect.ReviewServiceRecordDispositionProcedure, `{"id":"finding-1","decision":"DISPOSITION_DECISION_ACKNOWLEDGE","reason":"known issue","actor":"ada"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var disposed reviewv1.RecordDispositionResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &disposed))
	require.Equal(t, reviewv1.FindingStatus_FINDING_STATUS_ACKNOWLEDGED, disposed.Finding.Status)
	require.Len(t, disposed.Finding.Dispositions, 1)
	require.Equal(t, "ada", disposed.Finding.Dispositions[0].Actor)

	response = call(router, reviewconnect.ReviewServiceAssignFindingProcedure, `{"id":"finding-1","owner":"lin"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var assigned reviewv1.AssignFindingResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &assigned))
	require.Equal(t, "lin", assigned.Finding.Owner)

	response = call(router, reviewconnect.ReviewServiceGetReviewSummaryProcedure, `{}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var summary reviewv1.GetReviewSummaryResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &summary))
	require.Equal(t, int32(1), summary.Summary.Blockers)
	require.Equal(t, int32(1), summary.Summary.Acknowledged)
}

func TestListFindingsPagesAndReturnsTotal(t *testing.T) {
	base := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	finding := func(id string, minute int) domain.Finding {
		return domain.Finding{
			ID:           id,
			SubjectKind:  "asset",
			SubjectID:    "asset-" + id,
			Gate:         "accessibility",
			Severity:     domain.SeverityHigh,
			Status:       domain.StatusOpen,
			Message:      "gate failed",
			EvidenceJSON: "{}",
			CreatedAt:    base.Add(time.Duration(minute) * time.Minute).Format(time.RFC3339),
		}
	}
	router := newRouterWithFindings(t, []domain.Finding{
		finding("oldest", 0),
		finding("middle", 1),
		finding("newest", 2),
	})

	response := call(router, reviewconnect.ReviewServiceListFindingsProcedure, `{"limit":2,"offset":0}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var first reviewv1.ListFindingsResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &first))
	require.Equal(t, int32(3), first.Total)
	require.Len(t, first.Findings, 2)
	require.Equal(t, "newest", first.Findings[0].Id)
	require.Equal(t, "middle", first.Findings[1].Id)

	response = call(router, reviewconnect.ReviewServiceListFindingsProcedure, `{"limit":2,"offset":2}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var second reviewv1.ListFindingsResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &second))
	require.Equal(t, int32(3), second.Total)
	require.Len(t, second.Findings, 1)
	require.Equal(t, "oldest", second.Findings[0].Id)

	response = call(router, reviewconnect.ReviewServiceListFindingsProcedure, `{"limit":2,"offset":50}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var beyond reviewv1.ListFindingsResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &beyond))
	require.Empty(t, beyond.Findings)
	require.Equal(t, int32(3), beyond.Total)

	response = call(router, reviewconnect.ReviewServiceListFindingsProcedure, `{"offset":-1}`)
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), `"code":"invalid_argument"`)
}

func TestErrorsMapToConnectCodes(t *testing.T) {
	router := newRouter(t)

	response := call(router, reviewconnect.ReviewServiceGetFindingProcedure, `{"id":"missing"}`)
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Contains(t, response.Body.String(), `"code":"not_found"`)

	response = call(router, reviewconnect.ReviewServiceRecordDispositionProcedure, `{"id":"finding-1","decision":"DISPOSITION_DECISION_ACKNOWLEDGE","reason":"","actor":"ada"}`)
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), `"code":"invalid_argument"`)
}
