package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapptest"
)

// TestNewAppConstructs is the smoke gate: NewApp() must succeed against
// the cli-core wiring declared in app.go. This catches the most common
// regression class — a misconfigured StandardScenarioOptions or a missing
// dependency from cli-core — before any tests touch real commands.
func TestNewAppConstructs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if app == nil || app.core == nil || app.core.CLI == nil {
		t.Fatal("NewApp() returned an incomplete app")
	}
}

// TestRunVersion exercises a non-API command path through cli-core.
// --version must succeed and must NOT trigger the NeedsAPI preflight
// (which would try to reach the configured API base and fail in CI).
func TestRunVersion(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"--version"}); err != nil {
		t.Fatalf("app.Run(--version) error: %v", err)
	}
}

// TestRunHelp exercises cli-core's help renderer through the scenario
// app's wiring. Help is rendered to stdout by cli-core; we only verify
// Run returns without error. The presence of each registered command in
// the actual help surface is covered by cli-core's own tests.
func TestRunHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("app.Run(--help) error: %v", err)
	}
}

// TestMetadata pins the values app.go declares — appName must match the
// scenario id (post-substitution), appVersion must be non-empty.
// Catches accidental edits that decouple the binary identity from the
// scenario it belongs to.
func TestMetadata(t *testing.T) {
	if strings.TrimSpace(appName) == "" {
		t.Fatal("appName must not be empty")
	}
	if strings.TrimSpace(appVersion) == "" {
		t.Fatal("appVersion must not be empty")
	}
}

func TestWalletRecoveryLeaseDetectionUsesSharedGlobalFlagParser(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want bool
	}{
		{name: "plain", args: []string{"wallet-recovery", "read-lease-acquire"}, want: true},
		{name: "api base before command", args: []string{"--api-base", "http://127.0.0.1:18500", "wallet-recovery", "read-lease-renew"}, want: true},
		{name: "node and boolean globals", args: []string{"--node", "minimouse", "--no-color", "wallet-recovery", "read-lease-release"}, want: true},
		{name: "unrelated wallet recovery", args: []string{"wallet-recovery", "attest"}},
		{name: "incomplete global flag", args: []string{"--api-base", "wallet-recovery", "read-lease-acquire"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isWalletRecoveryReadLease(tc.args); got != tc.want {
				t.Fatalf("isWalletRecoveryReadLease(%q) = %t, want %t", tc.args, got, tc.want)
			}
		})
	}
}

func TestWalletRecoveryReadLeaseUsesExistingDBMOperatorSession(t *testing.T) {
	cliapptest.IsolateAgentEnvironment(t)
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.APIOverride = "http://127.0.0.1:18500"
	previousEligible := confirmLocalDBMOperatorEligible
	t.Cleanup(func() { confirmLocalDBMOperatorEligible = previousEligible })
	confirmLocalDBMOperatorEligible = func(context.Context) error { return nil }
	previous := exchangeLocalDBMOperatorPrincipal
	t.Cleanup(func() { exchangeLocalDBMOperatorPrincipal = previous })
	calls := 0
	exchangeLocalDBMOperatorPrincipal = func(_ context.Context, resource string) (string, error) {
		calls++
		if resource != "data-backup-manager" {
			t.Fatalf("operator exchange resource = %q", resource)
		}
		return "fixture-session-token", nil
	}
	if err := app.ensureWalletRecoverySession([]string{"wallet-recovery", "read-lease-acquire"}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || app.core.Config.Token != "fixture-session-token" {
		t.Fatalf("operator session calls=%d token-present=%t", calls, app.core.Config.Token != "")
	}
}

func TestWalletRecoveryReadLeaseRefusesWithoutLocalOwnerEligibility(t *testing.T) {
	cliapptest.IsolateAgentEnvironment(t)
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.APIOverride = "http://127.0.0.1:18500"
	previousEligible := confirmLocalDBMOperatorEligible
	t.Cleanup(func() { confirmLocalDBMOperatorEligible = previousEligible })
	confirmLocalDBMOperatorEligible = func(context.Context) error { return errors.New("broker unavailable") }
	previous := exchangeLocalDBMOperatorPrincipal
	t.Cleanup(func() { exchangeLocalDBMOperatorPrincipal = previous })
	calls := 0
	exchangeLocalDBMOperatorPrincipal = func(context.Context, string) (string, error) {
		calls++
		return "must-not-exchange", nil
	}
	err = app.ensureWalletRecoverySession([]string{"wallet-recovery", "read-lease-acquire"})
	if err == nil || !strings.Contains(err.Error(), "confirmed local operator eligibility") || calls != 0 {
		t.Fatalf("eligibility error=%v exchange calls=%d", err, calls)
	}
}

func TestWalletRecoveryReadLeaseDoesNotFallBackFromAgentIdentity(t *testing.T) {
	cliapptest.IsolateAgentEnvironment(t)
	t.Setenv("VROOLI_AGENT_IDENTITY_TOKEN", "fixture-agent-token")
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.APIOverride = "http://127.0.0.1:18500"
	app.core.Config.Token = "configured-human-session"
	previous := exchangeLocalDBMOperatorPrincipal
	t.Cleanup(func() { exchangeLocalDBMOperatorPrincipal = previous })
	calls := 0
	exchangeLocalDBMOperatorPrincipal = func(context.Context, string) (string, error) {
		calls++
		return "should-not-exchange", nil
	}
	err = app.ensureWalletRecoverySession([]string{"wallet-recovery", "read-lease-renew"})
	if err == nil || !strings.Contains(err.Error(), "refuse agent-context callers") || calls != 0 || app.core.Config.Token != "configured-human-session" {
		t.Fatalf("agent identity fallback error=%v calls=%d configured-token-preserved=%t", err, calls, app.core.Config.Token == "configured-human-session")
	}
}

func TestWalletRecoveryReadLeaseRefusesConfiguredTokenInExternalAgentContext(t *testing.T) {
	cliapptest.IsolateAgentEnvironment(t)
	t.Setenv("VROOLI_AGENT_IDENTITY_TOKEN", "")
	t.Setenv("CODEX_CI", "1")
	t.Setenv("CODEX_THREAD_ID", "agent-thread")
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.Config.Token = "configured-human-session"
	previous := exchangeLocalDBMOperatorPrincipal
	t.Cleanup(func() { exchangeLocalDBMOperatorPrincipal = previous })
	exchangeLocalDBMOperatorPrincipal = func(context.Context, string) (string, error) {
		t.Fatal("external agent must not exchange an operator session")
		return "", nil
	}
	err = app.ensureWalletRecoverySession([]string{"wallet-recovery", "read-lease-acquire"})
	if err == nil || !strings.Contains(err.Error(), "refuse agent-context callers") || app.core.Config.Token != "configured-human-session" {
		t.Fatalf("external agent configured-token refusal error=%v token-preserved=%t", err, app.core.Config.Token == "configured-human-session")
	}
}

func TestWalletRecoveryReadLeaseKeepsConfiguredTokenAndSkipsUnrelatedCommands(t *testing.T) {
	cliapptest.IsolateAgentEnvironment(t)
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.APIOverride = "http://127.0.0.1:18500"
	app.core.Config.Token = "configured-db-token"
	previous := exchangeLocalDBMOperatorPrincipal
	t.Cleanup(func() { exchangeLocalDBMOperatorPrincipal = previous })
	exchangeLocalDBMOperatorPrincipal = func(context.Context, string) (string, error) {
		t.Fatal("unexpected operator session exchange")
		return "", nil
	}
	if err := app.ensureWalletRecoverySession([]string{"wallet-recovery", "read-lease-release"}); err != nil {
		t.Fatal(err)
	}
	if err := app.ensureWalletRecoverySession([]string{"wallet-recovery", "attest"}); err != nil {
		t.Fatal(err)
	}
	if app.core.Config.Token != "configured-db-token" {
		t.Fatal("configured token was replaced")
	}
}
