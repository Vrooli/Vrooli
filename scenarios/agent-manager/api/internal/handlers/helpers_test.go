package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/pricing"
	"agent-manager/internal/orchestration/testutil/mocks"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/eventbus"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestRequestParsingHelpersAcceptCanonicalAndProtoForms(t *testing.T) {
	req := httptest.NewRequest("GET", "/?limit=12&offset=bad&runner=RUNNER_TYPE_CLAUDE_CODE&task=TASK_STATUS_RUNNING&run=RUN_STATUS_PARKED&event=log,RUN_EVENT_TYPE_MESSAGE,unknown", nil)
	if got, ok := parseQueryInt(req, "limit"); !ok || got != 12 {
		t.Fatalf("parse query int = %d, %t", got, ok)
	}
	if _, ok := parseQueryInt(req, "offset"); ok {
		t.Fatal("invalid query int accepted")
	}
	if _, present, err := parseQueryIntStrict(req, "offset"); !present || err == nil {
		t.Fatalf("strict int present=%t err=%v", present, err)
	}
	if _, present, err := parseQueryInt64Strict(req, "missing"); present || err != nil {
		t.Fatalf("missing int64 present=%t err=%v", present, err)
	}
	if got, ok := parseRunnerType(req.URL.Query().Get("runner")); !ok || got != domain.RunnerTypeClaudeCode {
		t.Fatalf("runner=%q ok=%t", got, ok)
	}
	if _, ok := parseRunnerType("not-a-runner"); ok {
		t.Fatal("invalid runner accepted")
	}
	if got, ok := parseTaskStatus(req.URL.Query().Get("task")); !ok || got != domain.TaskStatusRunning {
		t.Fatalf("task=%q ok=%t", got, ok)
	}
	if got, ok := parseRunStatus(req.URL.Query().Get("run")); !ok || got != domain.RunStatusParked {
		t.Fatalf("run=%q ok=%t", got, ok)
	}
	types, invalid := parseEventTypes(req.URL.Query()["event"])
	if len(types) != 2 || len(invalid) != 1 || invalid[0] != "unknown" {
		t.Fatalf("types=%v invalid=%v", types, invalid)
	}
}

func TestToJSONValueHandlesPrimitiveCollectionsAndFallbacks(t *testing.T) {
	cases := []struct {
		name  string
		value interface{}
	}{
		{"nil", nil},
		{"bool", true},
		{"string", "text"},
		{"integer", int64(7)},
		{"float", 1.5},
		{"number integer", json.Number("9")},
		{"number float", json.Number("1.25")},
		{"number text", json.Number("invalid")},
		{"strings", []string{"a", "b"}},
		{"interfaces", []interface{}{true, "x"}},
		{"object", map[string]interface{}{"enabled": true}},
		{"string object", map[string]string{"name": "agent"}},
		{"fallback", struct{ Name string }{"agent"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := toJsonValue(tc.value); got == nil || got.Kind == nil {
				t.Fatalf("toJsonValue(%T) = %+v", tc.value, got)
			}
		})
	}
}

func TestStatusParsingRejectsUnknownAndSupportsNumericForms(t *testing.T) {
	if _, ok := parseTaskStatus("unknown"); ok {
		t.Fatal("unknown task status accepted")
	}
	if _, ok := parseRunStatus("unknown"); ok {
		t.Fatal("unknown run status accepted")
	}
	if got, ok := parseRunnerType("1"); !ok || got == "" {
		t.Fatalf("numeric runner=%q ok=%t", got, ok)
	}
}

func TestHandlerStatusAndDependencyConversions(t *testing.T) {
	for input, want := range map[string]commonpb.HealthStatus{
		"healthy": commonpb.HealthStatus_HEALTH_STATUS_HEALTHY, "degraded": commonpb.HealthStatus_HEALTH_STATUS_DEGRADED,
		"unhealthy": commonpb.HealthStatus_HEALTH_STATUS_UNHEALTHY, "unknown": commonpb.HealthStatus_HEALTH_STATUS_UNSPECIFIED,
	} {
		if got := healthStatusToProto(input); got != want {
			t.Fatalf("healthStatusToProto(%q)=%v want=%v", input, got, want)
		}
	}
	if dependencyToJsonValue(nil) != nil {
		t.Fatal("nil dependency was serialized")
	}
	latency := int64(8)
	message := "offline"
	value := dependencyToJsonValue(&orchestration.DependencyStatus{Connected: false, LatencyMs: &latency, Error: &message, Storage: "sqlite"})
	if value.GetObjectValue().Fields["status"].GetStringValue() != "unhealthy" || value.GetObjectValue().Fields["latency_ms"].GetIntValue() != 8 {
		t.Fatalf("dependency=%+v", value)
	}
}

func TestErrorCodeStatusMappingCoversTransportCategories(t *testing.T) {
	cases := map[domain.ErrorCode]int{
		domain.ErrCodeNotFoundRun:        http.StatusNotFound,
		domain.ErrCodeValidationField:    http.StatusBadRequest,
		domain.ErrCodeStateTransition:    http.StatusConflict,
		domain.ErrCodePolicyScope:        http.StatusForbidden,
		domain.ErrCodeCapacityRuns:       http.StatusServiceUnavailable,
		domain.ErrCodeRunnerTimeout:      http.StatusGatewayTimeout,
		domain.ErrCodeRunnerUnavailable:  http.StatusServiceUnavailable,
		domain.ErrCodeRunnerExecution:    http.StatusBadGateway,
		domain.ErrCodeSandboxCreate:      http.StatusServiceUnavailable,
		domain.ErrCodeSandboxOperation:   http.StatusBadGateway,
		domain.ErrCodeDatabaseConnection: http.StatusServiceUnavailable,
		domain.ErrCodeDatabaseQuery:      http.StatusInternalServerError,
		domain.ErrCodeConfigInvalid:      http.StatusInternalServerError,
		domain.ErrCodeInternalPanic:      http.StatusInternalServerError,
		"UNRECOGNIZED_FAILURE":           http.StatusInternalServerError,
	}
	for code, want := range cases {
		if got := mapErrorCodeToStatus(code); got != want {
			t.Fatalf("mapErrorCodeToStatus(%s)=%d want=%d", code, got, want)
		}
	}
}

func TestWriteSimpleErrorProducesStructuredValidationResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks", nil)
	writeSimpleError(recorder, req, "title", "is required")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response commonpb.ErrorResponse
	decodeProtoJSON(t, recorder.Body.Bytes(), &response)
	if response.Code != string(domain.ErrCodeValidationField) || response.Details.GetFields()["request_id"].GetStringValue() == "" {
		t.Fatalf("response=%+v", &response)
	}
}

func TestProfileReconcileProjectionPreservesCountsAndStatuses(t *testing.T) {
	statuses := map[orchestration.ProfileReconcileStatus]apipb.ProfileReconcileStatus{
		orchestration.ProfileReconcileStatusCreated:                 apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_CREATED,
		orchestration.ProfileReconcileStatusUpdated:                 apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_UPDATED,
		orchestration.ProfileReconcileStatusUnchanged:               apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_UNCHANGED,
		orchestration.ProfileReconcileStatusSkipped:                 apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_SKIPPED,
		orchestration.ProfileReconcileStatusConflictedLocalOverride: apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_CONFLICTED_LOCAL_OVERRIDE,
		orchestration.ProfileReconcileStatusFailedValidation:        apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_FAILED_VALIDATION,
	}
	for status, want := range statuses {
		if got := profileReconcileStatusToProto(status); got != want {
			t.Fatalf("status %q = %v want %v", status, got, want)
		}
	}
	if got := profileReconcileStatusToProto("unknown"); got != apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_UNSPECIFIED {
		t.Fatalf("unknown=%v", got)
	}
	response := reconcileScenarioProfilesToProto(&orchestration.ReconcileScenarioProfilesResult{
		Scenario: "agent-manager", Created: 1, Updated: 2, Unchanged: 3, Skipped: 4, Conflicted: 5, Failed: 6, DryRun: true,
		Results: []orchestration.ProfileReconcileResult{{ProfileKey: "reviewer", Status: orchestration.ProfileReconcileStatusCreated, Message: "created"}},
	})
	if response.GetScenario() != "agent-manager" || response.GetCreated() != 1 || len(response.GetResults()) != 1 || response.GetResults()[0].GetStatus() != apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_CREATED {
		t.Fatalf("response=%+v", response)
	}
	if got := reconcileScenarioProfilesToProto(nil); got == nil || got.GetScenario() != "" {
		t.Fatalf("nil result=%+v", got)
	}
}

func TestOperationalHandlersRejectMalformedOrIncompleteRequestsBeforeServiceCalls(t *testing.T) {
	handler := &Handler{}
	for _, tc := range []struct {
		name string
		call func(*httptest.ResponseRecorder)
	}{
		{"validate path missing", func(rr *httptest.ResponseRecorder) {
			handler.ValidatePath(rr, httptest.NewRequest(http.MethodGet, "/api/v1/validate-path", nil))
		}},
		{"verify identity invalid json", func(rr *httptest.ResponseRecorder) {
			handler.VerifyIdentityToken(rr, httptest.NewRequest(http.MethodPost, "/api/v1/identity/verify", bytes.NewBufferString("{")))
		}},
		{"verify identity missing token", func(rr *httptest.ResponseRecorder) {
			handler.VerifyIdentityToken(rr, httptest.NewRequest(http.MethodPost, "/api/v1/identity/verify", bytes.NewBufferString("{}")))
		}},
		{"investigation invalid json", func(rr *httptest.ResponseRecorder) {
			handler.UpdateInvestigationSettings(rr, httptest.NewRequest(http.MethodPut, "/api/v1/investigation-settings", bytes.NewBufferString("{")))
		}},
		{"orchestration invalid json", func(rr *httptest.ResponseRecorder) {
			handler.UpdateOrchestrationSettings(rr, httptest.NewRequest(http.MethodPut, "/api/v1/orchestration-settings", bytes.NewBufferString("{")))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tc.call(rr)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestOperationalSettingsEndpointsExposeDefaultsAndRejectUnconfiguredMutations(t *testing.T) {
	service, router := setupTestHandler(t)
	_ = service
	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/orchestration-settings", nil))
	if get.Code != http.StatusOK || get.Body.Len() == 0 {
		t.Fatalf("get status=%d body=%s", get.Code, get.Body.String())
	}
	investigation := httptest.NewRecorder()
	router.ServeHTTP(investigation, httptest.NewRequest(http.MethodGet, "/api/v1/investigation-settings", nil))
	if investigation.Code != http.StatusOK || investigation.Body.Len() == 0 {
		t.Fatalf("investigation status=%d body=%s", investigation.Code, investigation.Body.String())
	}
	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"update orchestration", http.MethodPut, "/api/v1/orchestration-settings", `{}`},
		{"reset orchestration", http.MethodPost, "/api/v1/orchestration-settings/reset", ``},
		{"reset investigation", http.MethodPost, "/api/v1/investigation-settings/reset", ``},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body)))
			if rr.Code != http.StatusInternalServerError {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestRunControlEndpointsRejectInvalidIdentifiersBeforeOrchestration(t *testing.T) {
	handler := &Handler{}
	withRunID := func(call func(http.ResponseWriter, *http.Request)) func(*httptest.ResponseRecorder) {
		return func(rr *httptest.ResponseRecorder) {
			req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/api/v1/runs/not-a-uuid", bytes.NewBufferString("{}")), map[string]string{"id": "not-a-uuid"})
			call(rr, req)
		}
	}
	cases := []struct {
		name string
		call func(*httptest.ResponseRecorder)
	}{
		{"get", withRunID(handler.GetRun)},
		{"audit", withRunID(handler.GetAuditTranscript)},
		{"delete", withRunID(handler.DeleteRun)},
		{"stop", withRunID(handler.StopRun)},
		{"continue", withRunID(handler.ContinueRun)},
		{"park", withRunID(handler.ParkRun)},
		{"await result", withRunID(handler.GetAwaitResult)},
		{"wake", withRunID(handler.WakeRun)},
		{"recover", withRunID(handler.RecoverRun)},
		{"delete message", func(rr *httptest.ResponseRecorder) {
			req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/api/v1/runs/not-a-uuid/messages/not-a-uuid/delete", bytes.NewBufferString("{}")), map[string]string{"id": "not-a-uuid", "event_id": "not-a-uuid"})
			handler.DeleteRunMessage(rr, req)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tc.call(rr)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestListRunsRejectsInvalidFilterValuesBeforeOrchestration(t *testing.T) {
	handler := &Handler{}
	for _, query := range []string{
		"status=unknown", "taskId=not-a-uuid", "profileId=not-a-uuid", "investigatesRunId=not-a-uuid",
		"appliesInvestigationRunId=not-a-uuid", "limit=not-a-number", "offset=not-a-number",
	} {
		t.Run(query, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler.ListRuns(rr, httptest.NewRequest(http.MethodGet, "/api/v1/runs?"+query, nil))
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestAuxiliaryHandlersRegisterExpectedRoutesAndWiring(t *testing.T) {
	router := mux.NewRouter()
	pricing := NewPricingHandler(&pricingHandlerFake{settings: &pricing.PricingSettings{}}, mocks.NewFakeStatsRepository())
	pricing.RegisterRoutes(router)
	NewEventsHandler(nil).RegisterRoutes(router)
	NewHealthAuditHandler(nil).RegisterRoutes(router)
	paths := []string{"/api/v1/pricing/models", "/api/v1/pricing/aliases", "/api/v1/pricing/settings", "/api/v1/events", "/api/v1/health/models"}
	for _, path := range paths {
		match := &mux.RouteMatch{}
		if !router.Match(httptest.NewRequest(http.MethodGet, path, nil), match) {
			t.Fatalf("route %s was not registered", path)
		}
	}
	h := New(orchestration.EmptyHandlerServices(), WithObservedReceipts(eventbus.Client{}))
	if h.receipts.Enabled() {
		t.Fatal("empty observed receipts client unexpectedly enabled")
	}
	h.SetWebSocketHub(NewWebSocketHub())
	if h.GetWebSocketHub() == nil {
		t.Fatal("websocket hub was not retained")
	}
}

func TestSmallHandlerNormalizationAndCountProjections(t *testing.T) {
	if got := optionalTrimmedString("  value  "); got == nil || *got != "value" {
		t.Fatalf("trimmed=%v", got)
	}
	if got := optionalTrimmedString(" \t "); got != nil {
		t.Fatalf("empty optional=%v", got)
	}
	counts := purgeCountsToProto(orchestration.PurgeCounts{Profiles: 1, Tasks: 2, Runs: 3})
	if counts.GetProfiles() != 1 || counts.GetTasks() != 2 || counts.GetRuns() != 3 {
		t.Fatalf("counts=%+v", counts)
	}
	if normalizeActor("  operator ") != "operator" || normalizeActor(" ") != "unknown" {
		t.Fatal("actor normalization mismatch")
	}
}
