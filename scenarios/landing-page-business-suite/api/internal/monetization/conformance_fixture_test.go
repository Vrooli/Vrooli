package monetization

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestAdditionalConformanceChecksHavePositiveAndNegativeFixtures(t *testing.T) {
	tests := []struct {
		name string
		code string
		bad  fixtureOptions
	}{
		{name: "local limit", code: "money.limits_from_local_config", bad: fixtureOptions{localLimit: true}},
		{name: "offline gate", code: "money.gate_blocks_offline", bad: fixtureOptions{offlineGate: true}},
		{name: "durable outbox", code: "money.no_outbox_for_local_meter", bad: fixtureOptions{missingOutbox: true}},
		{name: "service token", code: "money.service_token_in_client_bundle", bad: fixtureOptions{serviceToken: true}},
		{name: "symmetric token", code: "money.symmetric_token_verification", bad: fixtureOptions{symmetricToken: true}},
		{name: "request identity", code: "money.identity_from_request_body", bad: fixtureOptions{requestIdentity: true}},
		{name: "cost-bearing client", code: "money.cost_bearing_meter_client_executed", bad: fixtureOptions{costBearingClient: true}},
	}
	for _, test := range tests {
		t.Run(test.name+" conforming", func(t *testing.T) {
			root := writeFixture(t, fixtureOptions{})
			if finding := findCode(scan(root), test.code); finding != nil {
				t.Fatalf("conforming fixture emitted %s: %s", test.code, finding.Message)
			}
		})
		t.Run(test.name+" non-conforming", func(t *testing.T) {
			root := writeFixture(t, test.bad)
			if finding := findCode(scan(root), test.code); finding == nil {
				t.Fatalf("non-conforming fixture did not emit %s", test.code)
			}
		})
	}
}

type fixtureOptions struct {
	localLimit        bool
	offlineGate       bool
	missingOutbox     bool
	serviceToken      bool
	symmetricToken    bool
	requestIdentity   bool
	costBearingClient bool
}

func findCode(findings []*commonv1.AssessmentFinding, code string) *commonv1.AssessmentFinding {
	for _, finding := range findings {
		if finding.Code == code {
			return finding
		}
	}
	return nil
}

func writeFixture(t *testing.T, options fixtureOptions) string {
	t.Helper()
	root := t.TempDir()
	manifest := declaration{
		Version:   2,
		BundleKey: "business_suite",
		AppKey:    "fixture",
		Features:  []surface{{Key: "feature", Class: "B", EnforcementPaths: []string{"api/gate.go"}}},
		Meters:    []surface{{LimitKey: "workflow_executions", Class: "B", Outbox: "api/outbox", EnforcementPaths: []string{"api/gate.go"}}},
	}
	if options.costBearingClient {
		manifest.Meters = append(manifest.Meters, surface{LimitKey: "ai_credits", Class: "A", EnforcementPaths: []string{"ui/meter.tsx"}})
	}
	if options.missingOutbox {
		manifest.Meters[0].Outbox = ""
	}
	writeFixtureFile(t, root, ".vrooli/monetization.json", manifest)
	gate := "package gate\n// cached lease is consulted before a transient error\nfunc Allow() bool { return true }\n"
	if options.offlineGate {
		gate = "package gate\nfunc Allow() bool { /* offline */ return false }\n"
	}
	writeFixtureFile(t, root, "api/gate.go", gate)
	if !options.missingOutbox {
		writeFixtureFile(t, root, "api/outbox/store.go", "package outbox\nconst insert = `INSERT INTO usage_outbox (operation_id) VALUES (?)`\n")
	}
	if options.localLimit {
		writeFixtureFile(t, root, ".vrooli/config.json", `{"workflow_executions": 10}`)
	}
	if options.costBearingClient {
		writeFixtureFile(t, root, "ui/meter.tsx", "export const meter = 'ai_credits';")
	}
	if options.serviceToken {
		writeFixtureFile(t, root, "api/client.go", "package client\nvar token = ValidateServiceToken\n")
	}
	if options.symmetricToken {
		writeFixtureFile(t, root, "api/token.go", "package token\nconst algorithm = HS256\n")
	}
	if options.requestIdentity {
		writeFixtureFile(t, root, "api/usage.go", "package usage\nvar user_identity string\nfunc HandleReportUsage() { _ = json.Valid(nil) }\n")
	}
	return root
}

func writeFixtureFile(t *testing.T, root, relative string, value any) {
	t.Helper()
	path := filepath.Join(root, filepath.Clean(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	var data []byte
	switch typed := value.(type) {
	case string:
		data = []byte(typed)
	default:
		var err error
		data, err = json.Marshal(typed)
		if err != nil {
			t.Fatalf("encode fixture %s: %v", relative, err)
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", relative, err)
	}
}
