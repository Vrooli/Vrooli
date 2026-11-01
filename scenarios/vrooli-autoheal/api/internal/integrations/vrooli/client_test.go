package vrooli

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
)

func TestScenarioStatus_CurrentContract(t *testing.T) {
	executor := testutil.NewMockExecutor()
	executor.Responses["vrooli scenario status alpha --json"] = testutil.MockResponse{
		Output: []byte(`{"success":true,"scenario":{"name":"alpha","status":"running","health_status":"healthy"}}`),
	}

	client := NewClient(executor)
	status, _, err := client.ScenarioStatus(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("ScenarioStatus returned error: %v", err)
	}
	if status.Scenario.Status != "running" {
		t.Fatalf("status = %q", status.Scenario.Status)
	}
	health, err := status.Scenario.NormalizedHealthStatus()
	if err != nil {
		t.Fatalf("NormalizedHealthStatus returned error: %v", err)
	}
	if health != "healthy" {
		t.Fatalf("health = %q", health)
	}
}

func TestResourceStatuses_CurrentTypedFleetContract(t *testing.T) {
	executor := testutil.NewMockExecutor()
	executor.Responses["vrooli resource status --json"] = testutil.MockResponse{Output: []byte(`{"success":true,"resources":[
		{"resource":{"name":"postgres"},"installed":true,"running":true,"healthy":true,"message":"healthy"},
		{"resource":{"name":"redis"},"installed":true,"running":false,"healthy":false,"message":"stopped"}]}`)}
	client := NewClient(executor)
	statuses, success, failures, _, err := client.ResourceStatuses(context.Background())
	if err != nil {
		t.Fatalf("ResourceStatuses returned error: %v", err)
	}
	if !success || failures != 0 || len(statuses) != 2 {
		t.Fatalf("fleet result = success:%v failures:%d statuses:%+v", success, failures, statuses)
	}
	if statuses[0].Name != "postgres" || statuses[0].Healthy == nil || !*statuses[0].Healthy {
		t.Fatalf("postgres mapping = %+v", statuses[0])
	}
	if statuses[1].Name != "redis" || statuses[1].Running || statuses[1].Healthy == nil || *statuses[1].Healthy {
		t.Fatalf("redis mapping = %+v", statuses[1])
	}
}

func TestResourceStatuses_PreservesTypedSignals(t *testing.T) {
	executor := testutil.NewMockExecutor()
	executor.Responses["vrooli resource status --json"] = testutil.MockResponse{Output: []byte(`{"success":true,"resources":[
		{"resource":{"name":"llama"},"installed":true,"running":true,"healthy":false,"serving":true,"status_code":"mode_drift","message":"companion down","probe_error":"declared cuda; observed cpu","declared_mode":"cuda","observed_mode":"cpu","mode_drift":true,"mode_reason":"fallback"}]}`)}
	status, success, failures, _, err := NewClient(executor).ResourceStatuses(context.Background())
	if err != nil {
		t.Fatalf("ResourceStatuses returned error: %v", err)
	}
	if !success || failures != 0 || len(status) != 1 {
		t.Fatalf("fleet result = success:%v failures:%d status:%+v", success, failures, status)
	}
	got := status[0]
	if got.Name != "llama" || got.Healthy == nil || *got.Healthy || got.Serving == nil || !*got.Serving ||
		!got.ModeDrift || !got.CompanionDown || got.DeclaredMode != "cuda" || got.ObservedMode != "cpu" ||
		got.StatusCode != "mode_drift" || got.ProbeError == "" {
		t.Fatalf("typed signal mapping = %+v", got)
	}
}

func TestScenarioStatus_MissingRequiredFieldFails(t *testing.T) {
	executor := testutil.NewMockExecutor()
	executor.Responses["vrooli scenario status alpha --json"] = testutil.MockResponse{
		Output: []byte(`{"success":true,"scenario":{"name":"alpha","health_status":"healthy"}}`),
	}

	client := NewClient(executor)
	_, _, err := client.ScenarioStatus(context.Background(), "alpha")
	if err == nil {
		t.Fatal("expected contract error")
	}
}

func TestMaintenanceCommands_CurrentContracts(t *testing.T) {
	executor := testutil.NewMockExecutor()
	executor.Responses["vrooli orphans --json"] = testutil.MockResponse{
		Output: []byte(`{"success":true,"orphans":[{"pid":100,"ppid":1,"command":"alpha"}]}`),
	}
	executor.Responses["vrooli locks --json"] = testutil.MockResponse{
		Output: []byte(`{"success":true,"locks":[{"port":8080,"path":"/tmp/a","stale":true}]}`),
	}
	executor.Responses["vrooli cleanup orphans --json"] = testutil.MockResponse{
		Output: []byte(`{"success":true,"data":{"stopped":[{"name":"100","message":"alpha"}],"failed":[],"message":"Stopped 1 processes (0 failed)"}}`),
	}
	executor.Responses["vrooli cleanup locks --json"] = testutil.MockResponse{
		Output: []byte(`{"success":true,"data":{"stopped":[{"name":"8080","message":"Removed stale lock"}],"failed":[],"message":"Stopped 1 processes (0 failed)"}}`),
	}
	executor.Responses["vrooli diagnose-port 8080 alpha --json"] = testutil.MockResponse{
		Output: []byte(`{"success":true,"diagnostic":{"port":8080,"scenario":"alpha","in_use":true,"listener_inspection":{"available":true},"host_orphan_count":1,"recommendations":["Inspect listener"]}}`),
	}

	client := NewClient(executor)

	orphans, _, err := client.ListOrphans(context.Background())
	if err != nil || len(orphans) != 1 {
		t.Fatalf("ListOrphans = (%v, %v)", orphans, err)
	}

	orphanCleanup, _, err := client.CleanupOrphans(context.Background())
	if err != nil || len(orphanCleanup.Stopped) != 1 {
		t.Fatalf("CleanupOrphans = (%+v, %v)", orphanCleanup, err)
	}

	lockCleanup, _, err := client.CleanupLocks(context.Background())
	if err != nil || len(lockCleanup.Stopped) != 1 {
		t.Fatalf("CleanupLocks = (%+v, %v)", lockCleanup, err)
	}

	diagnostic, _, err := client.DiagnosePort(context.Background(), "8080", "alpha")
	if err != nil || diagnostic.Port != 8080 {
		t.Fatalf("DiagnosePort = (%+v, %v)", diagnostic, err)
	}
}

func TestListRegistryClaims_MapsAndClassifies(t *testing.T) {
	executor := testutil.NewMockExecutor()
	executor.Responses["vrooli locks --json"] = testutil.MockResponse{
		Output: []byte(`{"success":true,"registry_claims":[
			{"claim_id":"c1","instance_id":"i1","scenario":"alpha","port":17701,"claim_status":"bound","instance_status":"running","reconciliation":"verified_running","authoritative":true,"recommendation_code":"port-ok"},
			{"claim_id":"c2","instance_id":"i2","scenario":"beta","port":17702,"reconciliation":"stale_claim","authoritative":false,"recommendation_code":"stale-claim-expire"}
		]}`),
	}

	client := NewClient(executor)
	claims, _, err := client.ListRegistryClaims(context.Background())
	if err != nil {
		t.Fatalf("ListRegistryClaims: %v", err)
	}
	if len(claims) != 2 {
		t.Fatalf("claims = %#v", claims)
	}
	if claims[0].StaleExpireRecommended() {
		t.Fatalf("verified_running claim should not be stale: %#v", claims[0])
	}
	if !claims[1].StaleExpireRecommended() {
		t.Fatalf("stale-claim-expire claim should be stale: %#v", claims[1])
	}
	if claims[1].ClaimID != "c2" || claims[1].Reconciliation != "stale_claim" || claims[1].Port != 17702 {
		t.Fatalf("stale claim mapping: %#v", claims[1])
	}
}
