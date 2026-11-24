package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"command-center/upstream"
	"connectrpc.com/connect"
	capreg "github.com/vrooli/vrooli/packages/capability-registry-go"
	integrationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/integrations/integrations_v1connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type recoveringHealthClient struct{ calls int }

func (c *recoveringHealthClient) Name() string { return "provider" }
func (c *recoveringHealthClient) Fetch(context.Context, string) (json.RawMessage, error) {
	c.calls++
	if c.calls == 1 {
		return nil, errors.New("GET https://provider.invalid/health?token=secret: response body private")
	}
	return json.RawMessage(`{"status":"healthy"}`), nil
}

func TestIntegrationHealthLossAndRecoveryThroughConnect(t *testing.T) {
	provider := &recoveringHealthClient{}
	def := capreg.Def{
		ID: "provider", Origin: "scenario", Name: "Provider", Description: "Health dependency.",
		DependencyKind: capreg.DependencyScenario, DependencySlug: "provider", Features: []string{"reports"}, Enabled: true,
		ActionKind: capreg.ActionKindOwnerGuidance, ActionLabel: "Review Provider", OperatorCommand: "vrooli scenario status provider --json",
	}
	registry := capreg.New([]capreg.Def{def}, map[string]capreg.Checker{
		"provider": integrationChecker{client: provider, features: def.Features, slug: "provider", label: "Provider", failureActionKind: capreg.ActionKindOwnerGuidance},
	}, 5*time.Second)
	s := NewServer(testRegistry())
	s.integrationRegistry = registry
	httpServer := httptest.NewServer(s.Handler())
	defer httpServer.Close()
	client := integrationconnect.NewIntegrationsServiceClient(http.DefaultClient, httpServer.URL)

	refresh := func() *commonv1.Integration {
		t.Helper()
		resp, err := client.Refresh(context.Background(), connect.NewRequest(&commonv1.RefreshIntegrationsRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		for _, integration := range resp.Msg.GetIntegrations() {
			if integration.GetId() == "provider" {
				return integration
			}
		}
		t.Fatal("provider state missing from refreshed integration list")
		return nil
	}

	failed := refresh()
	if got := failed.GetLifecycle().GetStatus(); got != commonv1.LifecycleStatus_LIFECYCLE_STATUS_UNAVAILABLE {
		t.Fatalf("outage status = %v", got)
	}
	if failed.GetLifecycle().GetReasonCode() != "upstream_unavailable" || failed.GetLifecycle().GetMessage() != "health endpoint unavailable" {
		t.Fatalf("outage lifecycle = %+v", failed.GetLifecycle())
	}
	if strings.Contains(failed.GetLifecycle().GetMessage(), "secret") || strings.Contains(failed.GetLifecycle().GetMessage(), "provider.invalid") || strings.Contains(failed.GetLifecycle().GetMessage(), "private") {
		t.Fatalf("outward lifecycle leaked connector error: %q", failed.GetLifecycle().GetMessage())
	}
	if failed.GetAction().GetKind() != commonv1.ActionKind_ACTION_KIND_OWNER_GUIDANCE || failed.GetAction().GetEligible() {
		t.Fatalf("outage action policy = %+v", failed.GetAction())
	}

	recovered := refresh()
	if got := recovered.GetLifecycle().GetStatus(); got != commonv1.LifecycleStatus_LIFECYCLE_STATUS_AVAILABLE {
		t.Fatalf("recovery status = %v", got)
	}
	if recovered.GetLifecycle().GetMessage() != "health endpoint reachable; feature compatibility qualified on read" {
		t.Fatalf("recovery message = %q", recovered.GetLifecycle().GetMessage())
	}
}

var _ upstream.Client = (*recoveringHealthClient)(nil)
