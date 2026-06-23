package validation

import (
	"context"
	"errors"
	"log"
	"testing"

	"ui-health/internal/services/manifestvalidation"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// fakeFreshness is an injectable freshnessClient for the static freshness group.
type fakeFreshness struct {
	resp  *cliv1.ScenarioFreshnessResponse
	err   error
	calls int
}

func (f *fakeFreshness) ScenarioFreshness(_ context.Context, _ string, _ ...vroolicli.ScenarioFreshnessOption) (*cliv1.ScenarioFreshnessResponse, error) {
	f.calls++
	return f.resp, f.err
}

func freshnessHandler(client freshnessClient) *connectHandler {
	return NewConnectHandler(Deps{
		Logger:    log.New(log.Writer(), "", 0),
		Validator: &stubValidator{},
		Freshness: client,
	})
}

// TestFreshnessStaleUIBundleIsGatingError pins parity with the retired smoke
// phase: a stale ui-bundle check is reported as a gating ERROR with the restart
// remediation.
func TestFreshnessStaleUIBundleIsGatingError(t *testing.T) {
	client := &fakeFreshness{resp: &cliv1.ScenarioFreshnessResponse{
		Checks: []*cliv1.ScenarioFreshnessCheck{{
			CheckType: "ui-bundle",
			Stale:     true,
			Target:    "ui/dist/index.html",
			Cause:     "source newer",
			File:      "ui/src/App.tsx",
		}},
	}}
	findings := freshnessHandler(client).freshnessFindings(context.Background(), "demo", "/tmp/demo")
	if client.calls != 1 {
		t.Fatalf("expected one freshness call, got %d", client.calls)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one stale finding, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Code != "freshness_ui_bundle_stale" {
		t.Errorf("code = %q, want freshness_ui_bundle_stale", f.Code)
	}
	if f.Severity != manifestvalidation.SeverityError {
		t.Errorf("severity = %v, want SeverityError (gating, parity with smoke)", f.Severity)
	}
	if f.Message != "ui/dist/index.html stale (source newer): ui/src/App.tsx" {
		t.Errorf("message = %q", f.Message)
	}
	if f.Suggestion == "" {
		t.Error("stale finding must carry a restart remediation")
	}
}

// TestFreshnessIgnoresNonUIBundleAndFreshChecks: only stale ui-bundle checks
// produce a finding — API-binary freshness and fresh bundles are silent.
func TestFreshnessIgnoresNonUIBundleAndFreshChecks(t *testing.T) {
	client := &fakeFreshness{resp: &cliv1.ScenarioFreshnessResponse{
		Checks: []*cliv1.ScenarioFreshnessCheck{
			{CheckType: "ui-bundle", Stale: false}, // fresh ui-bundle
			{CheckType: "binary", Stale: true},     // stale, but not a UI concern
			{CheckType: "binary", Stale: false},    // fresh binary
		},
	}}
	findings := freshnessHandler(client).freshnessFindings(context.Background(), "demo", "/tmp/demo")
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %d: %+v", len(findings), findings)
	}
}

// TestFreshnessDegradesGracefullyOnError: a freshness-engine error never blocks
// validation — it yields no finding (logged), mirroring smoke's graceful path.
func TestFreshnessDegradesGracefullyOnError(t *testing.T) {
	client := &fakeFreshness{err: errors.New("freshness engine unreachable")}
	findings := freshnessHandler(client).freshnessFindings(context.Background(), "demo", "/tmp/demo")
	if findings != nil {
		t.Fatalf("freshness error must degrade to no finding, got %+v", findings)
	}
}

// TestFreshnessStaleReasonDegrades pins the human-reason rendering across the
// present/absent permutations of cause and file.
func TestFreshnessStaleReasonDegrades(t *testing.T) {
	cases := []struct {
		target, cause, file, want string
	}{
		{"ui/dist", "source newer", "a.ts", "ui/dist stale (source newer): a.ts"},
		{"ui/dist", "source newer", "", "ui/dist stale (source newer)"},
		{"ui/dist", "", "", "ui/dist stale"},
		{"", "", "", "UI bundle stale"},
	}
	for _, c := range cases {
		if got := uiBundleStaleReason(c.target, c.cause, c.file); got != c.want {
			t.Errorf("uiBundleStaleReason(%q,%q,%q) = %q, want %q", c.target, c.cause, c.file, got, c.want)
		}
	}
}
