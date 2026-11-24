package deployments

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectManagedDNSStandingUsesExactStepReceipt(t *testing.T) {
	tests := []struct {
		name     string
		standing cloudOperationStanding
		want     string
	}{
		{name: "pending before receipt", standing: cloudOperationStanding{State: "running"}, want: "pending"},
		{name: "succeeded", standing: cloudOperationStanding{State: "succeeded", StepReceipts: []cloudOperationStepReceipt{{Step: "edge.dns.ensure", Outcome: "succeeded", Detail: "record id must not be retained"}}}, want: "succeeded"},
		{name: "failed", standing: cloudOperationStanding{State: "failed", StepReceipts: []cloudOperationStepReceipt{{Step: "edge.dns.ensure", Outcome: "failed", Error: "refused"}}}, want: "failed"},
		{name: "unknown effect", standing: cloudOperationStanding{State: "reconciling", UnknownEffects: []cloudOperationUnknownEffect{{Step: "edge.dns.ensure", Reason: "reply lost"}}}, want: "ambiguous"},
		{name: "terminal before DNS", standing: cloudOperationStanding{State: "failed"}, want: "not_reached"},
		{name: "terminal success missing configured DNS", standing: cloudOperationStanding{State: "succeeded"}, want: "ambiguous"},
		{name: "contradictory evidence", standing: cloudOperationStanding{State: "reconciling", StepReceipts: []cloudOperationStepReceipt{{Step: "edge.dns.ensure", Outcome: "succeeded"}}, UnknownEffects: []cloudOperationUnknownEffect{{Step: "edge.dns.ensure"}}}, want: "ambiguous"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := projectManagedDNSStanding(tt.standing)
			if got.State != tt.want {
				t.Fatalf("state=%q reason=%q, want %q", got.State, got.Reason, tt.want)
			}
			if tt.want == "succeeded" && got.EvidenceSHA256 == "" {
				t.Fatal("succeeded receipt has no source evidence digest")
			}
		})
	}
}

func TestMergeManagedDNSReadbackDoesNotRegressOrRewriteTerminalEvidence(t *testing.T) {
	state, reason, digest := mergeManagedDNSReadback("pending", "", "", "succeeded", "committed", "sha256:one")
	if state != "succeeded" || reason != "committed" || digest != "sha256:one" {
		t.Fatalf("pending → success = %q %q %q", state, reason, digest)
	}
	state, reason, digest = mergeManagedDNSReadback(state, reason, digest, "pending", "stale", "")
	if state != "ambiguous" || reason == "" || digest != "" {
		t.Fatalf("stale regression = %q %q %q", state, reason, digest)
	}
	state, _, _ = mergeManagedDNSReadback("succeeded", "committed", "sha256:one", "succeeded", "different source receipt", "sha256:two")
	if state != "ambiguous" {
		t.Fatalf("different terminal receipt state=%q", state)
	}
}

func TestManagedDNSConfiguredRequiresManifestOptIn(t *testing.T) {
	if managedDNSConfigured([]byte(`{"edge":{"managed_dns_profile":"staging"}}`)) != true {
		t.Fatal("explicit managed DNS profile was not detected")
	}
	if managedDNSConfigured([]byte(`{"edge":{"managed_dns_profile":" "}}`)) {
		t.Fatal("blank managed DNS profile was treated as configured")
	}
	if managedDNSConfigured([]byte(`not json`)) {
		t.Fatal("invalid manifest was treated as configured")
	}
}

func TestCloudOperationStandingNormalizesProtoJSONFieldNames(t *testing.T) {
	standing := cloudOperationStanding{
		OperationIDCamel: "op-exact", DeploymentIDCamel: "dep-exact", RequestKeyCamel: "dm-cloud:op-exact",
		ActiveStepCamel:   "edge.dns.ensure",
		StepReceiptsCamel: []cloudOperationStepReceipt{{Step: "edge.dns.ensure", Outcome: "succeeded"}},
	}
	standing.normalize()
	if standing.OperationID != "op-exact" || standing.DeploymentID != "dep-exact" || standing.RequestKey != "dm-cloud:op-exact" || standing.ActiveStep != "edge.dns.ensure" || len(standing.StepReceipts) != 1 {
		t.Fatalf("normalized standing = %+v", standing)
	}
}

func TestHTTPCloudOperationStandingReadsDNSReceiptOverExistingReadRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vrooli.scenario_to_cloud.v1.operations.OperationsService/GetOperation" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"operationId":"s2c-op","deploymentId":"dep","requestKey":"dm-cloud:dm-op","state":"running","stepReceipts":[{"step":"edge.dns.ensure","outcome":"succeeded","detail":"provider record id"}]}`))
	}))
	defer server.Close()
	client := &HTTPCloudHealthClient{baseURL: server.URL, httpClient: server.Client()}
	standing, err := client.cloudOperationStanding(context.Background(), "s2c-op")
	if err != nil {
		t.Fatal(err)
	}
	if standing.OperationID != "s2c-op" || standing.DeploymentID != "dep" || standing.RequestKey != "dm-cloud:dm-op" || len(standing.StepReceipts) != 1 {
		t.Fatalf("standing=%+v", standing)
	}
	projection := projectManagedDNSStanding(standing)
	if projection.State != "succeeded" || projection.EvidenceSHA256 == "" {
		t.Fatalf("projection=%+v", projection)
	}
}
