package vps

import (
	"context"
	"errors"
	"strings"
	"testing"

	"scenario-to-cloud/reach"
	"scenario-to-cloud/reach/reachtest"
)

// [REQ:STC-P0-024] A TLS renewal is the broker's edge.caddy.reload action
// through the target owner plus a cloud-side verification; no shell string
// and no systemctl argv leaves the cloud.
func TestRunCaddyTLSRenew_ReloadsThroughTheBrokerAndVerifies(t *testing.T) {
	r := &reachtest.Scripted{Prefixes: map[string]reachtest.Answer{
		"vrooli cloud-target host repair --action edge.caddy.reload": {Result: reach.Result{Stdout: `{"result":{"status":"reloaded","changed":true}}`}},
	}}
	verified := ""
	result := RunCaddyTLSRenew(context.Background(), Runtime{Reach: r, Target: liveStateTarget()}, "dep-1", "example.com", func(_ context.Context, domain string) error {
		verified = domain
		return nil
	})
	if !result.OK {
		t.Fatalf("expected OK, got: %s / %s", result.Message, result.Output)
	}
	if verified != "example.com" {
		t.Fatalf("verification must run for the domain, got %q", verified)
	}
	if len(r.Calls) != 1 || !r.Calls[0].Effectful || r.Calls[0].Verb != "cloud-target host repair" {
		t.Fatalf("expected one host repair verb, got %+v", r.Calls)
	}
	for _, a := range r.Calls[0].Args {
		if strings.Contains(a, "systemctl") || strings.Contains(a, "&&") {
			t.Fatalf("argv carries a shell fragment: %q", a)
		}
	}
}

func TestRunCaddyTLSRenew_RefusalAndVerificationFailure(t *testing.T) {
	refused := &reachtest.Scripted{Prefixes: map[string]reachtest.Answer{
		"vrooli cloud-target host repair": {Result: reach.Result{ExitCode: 2, Stdout: `{"error":{"code":"caddy_reload_failed","message":"reload failed"}}`}},
	}}
	result := RunCaddyTLSRenew(context.Background(), Runtime{Reach: refused, Target: liveStateTarget()}, "dep-1", "example.com", nil)
	if result.OK || !strings.Contains(result.Output, "caddy_reload_failed") {
		t.Fatalf("refusal must surface the owner's typed code: %+v", result)
	}

	ok := &reachtest.Scripted{}
	result = RunCaddyTLSRenew(context.Background(), Runtime{Reach: ok, Target: liveStateTarget()}, "dep-1", "example.com", func(context.Context, string) error {
		return errors.New("certificate expired")
	})
	if result.OK || !strings.Contains(result.Output, "certificate expired") {
		t.Fatalf("a failed verification must not report success: %+v", result)
	}
}
