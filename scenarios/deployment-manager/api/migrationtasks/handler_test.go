package migrationtasks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

func TestMigrationRequestHelpers(t *testing.T) {
	request := &ReportRequest{Scenario: "Demo", FromDependency: "postgres", ToDependency: "sqlite", ProfileID: "p1", Notes: "local first"}
	built := buildCreateRequest(request)
	if built.Name != "Migrate Demo: postgres → sqlite" || built.Kind != "fix" || len(built.Tags) != 4 || !strings.Contains(built.GetDescription(), "local first") {
		t.Fatalf("create request = %#v", built)
	}
	request.Title = "Custom migration"
	if built := buildCreateRequest(request); built.Name != "Custom migration" {
		t.Fatalf("custom title = %q", built.Name)
	}
	if got := buildDescription(&ReportRequest{Scenario: "demo", FromDependency: "redis", ToDependency: "in-process"}); !strings.Contains(got, "From") || strings.Contains(got, "Deployment profile") {
		t.Fatalf("description = %q", got)
	}
	response := &apipb.BacklogItemResponse{Deduped: true, Item: &domainpb.BacklogItem{Kind: "fix", Name: "task", Status: "pending", Priority: 5, QueuePosition: int32Ptr(3)}}
	feedback := feedbackFromResponse(response)
	if feedback.ItemID != "fix/task" || feedback.QueuePosition == nil || *feedback.QueuePosition != 3 || !feedback.Deduped {
		t.Fatalf("feedback = %#v", feedback)
	}
	if got := deepLink("fix", "task"); got != "/apps/swarm-manager/proxy/backlog/fix/task" {
		t.Fatalf("deep link = %q", got)
	}
}

func int32Ptr(value int32) *int32 { return &value }

func TestMigrationHandlersValidateAndSurfaceDependencyFailures(t *testing.T) {
	h := NewHandler(func(string, map[string]interface{}) {})
	h.resolveURL = func(context.Context) (string, error) { return "", errors.New("not running") }
	for name, request := range map[string]string{
		"bad JSON":         "{",
		"missing scenario": `{"from_dependency":"postgres","to_dependency":"sqlite"}`,
		"missing swap":     `{"scenario":"demo"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.Report(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(request)))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
	rec := httptest.NewRecorder()
	h.Report(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"scenario":"demo","from_dependency":"postgres","to_dependency":"sqlite"}`)))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("dependency report status = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.Status(rec, httptest.NewRequest(http.MethodGet, "/?name=task", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("dependency status = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.Status(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing name status = %d", rec.Code)
	}
}
