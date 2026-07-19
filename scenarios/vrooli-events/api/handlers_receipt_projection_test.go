package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestReceiptProjectionAPIAppearsInSnapshot(t *testing.T) {
	_, ts := newTestServer(t)
	body := `{"source_scenario":"agent-manager","target_scenario":"plan-manager","operation_pattern":"plans.create","response_fields":["plan_id"],"max_bytes":1024,"sample_per_ten_k":10000,"retention_days":30,"enabled":true}`
	resp, err := http.Post(ts.URL+"/api/v1/receipt-projections", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", resp.StatusCode)
	}
	resp, err = http.Get(ts.URL + "/api/v1/policies/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var snapshot struct {
		ReceiptProjections []struct {
			OperationPattern string `json:"operation_pattern"`
		} `json:"receipt_projections"`
	}
	snapshot = decodeJSON[struct {
		ReceiptProjections []struct {
			OperationPattern string `json:"operation_pattern"`
		} `json:"receipt_projections"`
	}](t, resp)
	if len(snapshot.ReceiptProjections) != 1 || snapshot.ReceiptProjections[0].OperationPattern != "plans.create" {
		t.Fatalf("receipt projections = %#v", snapshot.ReceiptProjections)
	}
}
