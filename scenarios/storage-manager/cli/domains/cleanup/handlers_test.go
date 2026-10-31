package cleanup

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup"
	cleanupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup/cleanup_v1connect"
	"google.golang.org/protobuf/proto"
)

func TestApplyPlanForwardsEverySelectedItemFromManifestArgs(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(nil, manifest)
	if err != nil {
		t.Fatal(err)
	}
	var schema cliapp.ArgSchema
	for _, command := range group.Subcommands {
		if command.Name == "apply" {
			schema = command.Args
		}
	}
	ctx, err := cliapp.NewTestRunContextFromArgs(schema, []string{"--plan-id", "plan-test", "--policy-version", "policy-test", "--idempotency-key", "selection-test", "--item-id", "first", "--item-id", "second"}, nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	var got cleanupv1.ApplyPlanRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if err := proto.Unmarshal(body, &got); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/proto")
		response, _ := proto.Marshal(&cleanupv1.ApplyPlanResponse{PlanId: "plan-test"})
		_, _ = w.Write(response)
	}))
	defer server.Close()
	h := &handlers{client: cleanupconnect.NewCleanupServiceClient(server.Client(), server.URL)}
	if _, err = h.applyPlanCall(ctx); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.GetItemIds(), []string{"first", "second"}) {
		t.Fatalf("exact selection changed at CLI boundary: %v", got.GetItemIds())
	}
}

func TestCreatePlanReportSurfacesProviderWarnings(t *testing.T) {
	msg := &cleanupv1.CreatePlanResponse{Plan: &cleanupv1.Plan{
		Id: "plan-1",
		Providers: []*cleanupv1.ProviderPlan{{
			ProviderId:     "tmp",
			EstimatedBytes: 12,
			ItemCount:      1,
			ApprovalMode:   "operator",
			Warnings:       []string{"measurement budget exhausted: 3 entries were not measured"},
		}},
	}}

	rendered, err := cliapp.RenderOperationalReportString((&handlers{}).createPlanReport(nil, msg))
	if err != nil {
		t.Fatalf("render report: %v", err)
	}
	if !strings.Contains(rendered, "tmp 12 bytes 1 item(s)") {
		t.Fatalf("provider row missing from report: %q", rendered)
	}
	if !strings.Contains(rendered, "warning: measurement budget exhausted: 3 entries were not measured") {
		t.Fatalf("provider warning missing from human report: %q", rendered)
	}
}

func TestCreatePlanReportLeavesWarningFreeProviderRowUnchanged(t *testing.T) {
	msg := &cleanupv1.CreatePlanResponse{Plan: &cleanupv1.Plan{
		Id: "plan-2",
		Providers: []*cleanupv1.ProviderPlan{{
			ProviderId:     "tmp",
			EstimatedBytes: 12,
			ItemCount:      1,
			ApprovalMode:   "operator",
		}},
	}}

	rendered, err := cliapp.RenderOperationalReportString((&handlers{}).createPlanReport(nil, msg))
	if err != nil {
		t.Fatalf("render report: %v", err)
	}
	if !strings.Contains(rendered, "tmp 12 bytes 1 item(s)") {
		t.Fatalf("provider row missing from report: %q", rendered)
	}
	if strings.Contains(rendered, "warning:") {
		t.Fatalf("warning-free provider row changed: %q", rendered)
	}
}
