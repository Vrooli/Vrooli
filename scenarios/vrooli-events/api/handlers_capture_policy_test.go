package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestReceiptCapturePolicySnapshotUsesTypedContract(t *testing.T) {
	_, ts := newTestServer(t)
	body := `{"policy_id":"plan-manager-create-plan","enabled":true,"selector":{"target_scenario":"plan-manager","operation":"POST /vrooli.plan_manager.v1.plans.PlansService/CreatePlan","protocol":"connect","event_type":"vrooli.events.receipt.v1"},"response_type":"vrooli.plan_manager.v1.plans.CreatePlanResponse","response_projection_paths":["plan.id"],"retention_days":30,"access":{"read_principals":["agent-manager"]}}`
	resp, err := http.Post(ts.URL+"/api/v1/receipt-capture-policies", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", resp.StatusCode)
	}
	resp, err = http.Get(ts.URL + "/api/v1/policies/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var snapshot struct {
		Version  string `json:"version"`
		Policies []struct {
			PolicyID string `json:"policy_id"`
			Selector struct {
				Target    string `json:"target_scenario"`
				Operation string `json:"operation"`
				Protocol  string `json:"protocol"`
				EventType string `json:"event_type"`
			} `json:"selector"`
			ResponseType string   `json:"response_type"`
			Paths        []string `json:"response_projection_paths"`
		} `json:"receipt_capture_policies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Version == "" || len(snapshot.Policies) != 1 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	p := snapshot.Policies[0]
	if p.PolicyID != "plan-manager-create-plan" || p.Selector.Target != "plan-manager" || p.Selector.Operation != "POST /vrooli.plan_manager.v1.plans.PlansService/CreatePlan" || p.Selector.Protocol != "connect" || p.Selector.EventType != receiptEventType || p.ResponseType != "vrooli.plan_manager.v1.plans.CreatePlanResponse" || len(p.Paths) != 1 || p.Paths[0] != "plan.id" {
		t.Fatalf("policy=%+v", p)
	}
}
