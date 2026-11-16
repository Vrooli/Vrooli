package monetization

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestBuildAssessmentEmitsSeverityCountsRequiredByProviderContract(t *testing.T) {
	findings := []*commonv1.AssessmentFinding{
		finding("money.first", "first", "api/first.go", "fix it"),
		finding("money.second", "second", "api/second.go", "fix it"),
	}
	a := buildAssessment("fixture", findings)
	if got := a.GetFindingsBySeverity()["FINDING_SEVERITY_ERROR"]; got != int32(len(findings)) {
		t.Fatalf("error severity count = %d, want %d", got, len(findings))
	}
}

func TestClassBMeterSupportsOutboxAndAtomicReservationAsSeparateModes(t *testing.T) {
	t.Run("complete LPBS reservation", func(t *testing.T) {
		root, decl := writeRemoteReservationFixture(t)
		findings := validateDeclaration(root, filepath.Join(root, ".vrooli", "monetization.json"), decl)
		if got := findCode(findings, "money.remote_reservation_invalid"); got != nil {
			t.Fatalf("valid reservation was rejected: %s", got.Message)
		}
		if got := findCode(findings, "money.no_outbox_for_local_meter"); got != nil {
			t.Fatalf("remote reservation was incorrectly treated as an outbox: %s", got.Message)
		}
	})

	for _, test := range []struct {
		name   string
		mutate func(*testing.T, *declaration, string)
	}{
		{
			name: "missing commit operation",
			mutate: func(t *testing.T, _ *declaration, root string) {
				writeFixtureFile(t, root, "api/commit.go", "package api\nfunc Commit() {}\n")
			},
		},
		{
			name: "commit path not declared as enforcement",
			mutate: func(_ *testing.T, decl *declaration, _ string) {
				decl.Meters[0].EnforcementPaths = []string{"api/begin.go", "api/release.go"}
			},
		},
		{
			name: "wrong authority",
			mutate: func(_ *testing.T, decl *declaration, _ string) {
				decl.Meters[0].Reservation.Authority = "scenario-local"
			},
		},
		{
			name: "reservation source symlink escapes scenario root",
			mutate: func(t *testing.T, _ *declaration, root string) {
				outside := t.TempDir()
				writeFixtureFile(t, outside, "begin.go", "package api\nfunc Begin(ctx context.Context, token string) { quota.Begin(ctx, token, \"workflow_executions\", 1, 0) }\n")
				path := filepath.Join(root, "api", "begin.go")
				if err := os.Remove(path); err != nil {
					t.Fatalf("remove fixture begin path: %v", err)
				}
				if err := os.Symlink(filepath.Join(outside, "begin.go"), path); err != nil {
					t.Fatalf("symlink fixture begin path: %v", err)
				}
			},
		},
		{
			name: "conflicting outbox mode",
			mutate: func(_ *testing.T, decl *declaration, _ string) {
				decl.Meters[0].Outbox = "api/outbox"
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, decl := writeRemoteReservationFixture(t)
			test.mutate(t, &decl, root)
			findings := validateDeclaration(root, filepath.Join(root, ".vrooli", "monetization.json"), decl)
			if got := findCode(findings, "money.remote_reservation_invalid"); got == nil {
				t.Fatal("invalid remote reservation was not rejected")
			}
		})
	}
}

func writeRemoteReservationFixture(t *testing.T) (string, declaration) {
	t.Helper()
	root := t.TempDir()
	decl := declaration{Version: 2, BundleKey: "business_suite", AppKey: "fixture", ProductShape: "meter_only", Meters: []surface{{
		LimitKey: "workflow_executions", Class: "B",
		Reservation:      &reservation{Kind: "lpbs_atomic", Authority: "landing-page-business-suite", BeginPath: "api/begin.go", CommitPath: "api/commit.go", ReleasePath: "api/release.go"},
		EnforcementPaths: []string{"api/begin.go", "api/commit.go", "api/release.go"},
	}}}
	writeFixtureFile(t, root, "api/begin.go", "package api\nfunc Begin(ctx context.Context, token string) { quota.Begin(ctx, token, \"workflow_executions\", 1, 0) }\n")
	writeFixtureFile(t, root, "api/commit.go", "package api\nfunc Commit(ctx context.Context, hold *Hold) { _ = hold.Commit(ctx, 1) }\n")
	writeFixtureFile(t, root, "api/release.go", "package api\nfunc Release(ctx context.Context, hold *Hold) { _ = hold.Release(context.WithoutCancel(ctx)) }\n")
	return root, decl
}

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

func TestProductShapeMustMatchDeclaredEnforcement(t *testing.T) {
	for _, test := range []struct {
		name        string
		shape       string
		entitled    bool
		features    int
		meters      int
		wantFinding bool
	}{
		{name: "feature gated", shape: "feature_gated", entitled: true, features: 1},
		{name: "meter only", shape: "meter_only", meters: 1},
		{name: "meter only cannot require subscription", shape: "meter_only", entitled: true, meters: 1, wantFinding: true},
		{name: "meter only cannot declare gates", shape: "meter_only", features: 1, meters: 1, wantFinding: true},
		{name: "feature gated needs a feature", shape: "feature_gated", entitled: true, wantFinding: true},
		{name: "missing shape", entitled: true, features: 1, wantFinding: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			decl := declaration{Version: 2, BundleKey: "business_suite", AppKey: "fixture", ProductShape: test.shape, RequiresEntitlement: test.entitled}
			for i := 0; i < test.features; i++ {
				decl.Features = append(decl.Features, surface{Key: "feature", Class: "B", EnforcementPaths: []string{"api/gate.go"}})
			}
			for i := 0; i < test.meters; i++ {
				decl.Meters = append(decl.Meters, surface{LimitKey: "meter", Class: "A", EnforcementPaths: []string{"api/gate.go"}})
			}
			writeFixtureFile(t, root, ".vrooli/monetization.json", decl)
			writeFixtureFile(t, root, "api/gate.go", "package gate\nfunc Allow() bool { return true }\n")
			finding := findCode(validateDeclaration(root, filepath.Join(root, ".vrooli", "monetization.json"), decl), "money.product_shape_mismatch")
			if test.wantFinding && finding == nil {
				t.Fatal("expected product shape mismatch")
			}
			if !test.wantFinding && finding != nil {
				t.Fatalf("unexpected product shape mismatch: %s", finding.Message)
			}
		})
	}
}

func TestCostBearingPathClassificationDoesNotTreatServerHTTPClientsAsBrowserCode(t *testing.T) {
	if hasClientPath([]string{"api/handlers/ai/gateway_client.go", "api/services/client_registry.go"}) {
		t.Fatal("server-side HTTP client filenames must not be treated as client-executed paths")
	}
	if !hasClientPath([]string{"ui/src/ai/gateway.ts"}) {
		t.Fatal("browser UI paths must be treated as client-executed")
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
		Version:             2,
		BundleKey:           "business_suite",
		AppKey:              "fixture",
		ProductShape:        "feature_gated",
		RequiresEntitlement: true,
		Features:            []surface{{Key: "feature", Class: "B", EnforcementPaths: []string{"api/gate.go"}}},
		Meters:              []surface{{LimitKey: "workflow_executions", Class: "B", Outbox: "api/outbox", EnforcementPaths: []string{"api/gate.go"}}},
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
