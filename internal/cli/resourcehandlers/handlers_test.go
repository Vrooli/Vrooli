package resourcehandlers

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostreqrun"
	"github.com/vrooli/vrooli/internal/resources"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

type testContext struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func TestRootHandlerRendersHelpWithNoArgs(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	handler := RootHandler(HandlerDeps[testContext]{
		Stdout:  func(ctx testContext) io.Writer { return ctx.stdout },
		Stderr:  func(ctx testContext) io.Writer { return ctx.stderr },
		Globals: func(testContext) rootcli.GlobalOptions { return rootcli.GlobalOptions{} },
		OutputFormat: func(testContext) (cliout.Format, error) {
			return cliout.FormatHuman, nil
		},
	})

	if err := handler(ctx, nil); err != nil {
		t.Fatalf("RootHandler() error = %v", err)
	}
	if got := ctx.stdout.String(); !strings.Contains(got, "vrooli resource") {
		t.Fatalf("RootHandler() help missing resource usage: %q", got)
	}
}

func TestEnforceResourceHostRequirementsSkipsNonMutatingActions(t *testing.T) {
	called := 0
	prev := enforceHostRequirementsFn
	enforceHostRequirementsFn = func(_ hostreqrun.Options) (vrooliruntime.Report, error) {
		called++
		return vrooliruntime.Report{}, nil
	}
	t.Cleanup(func() { enforceHostRequirementsFn = prev })

	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := HandlerDeps[testContext]{
		Stdout: func(c testContext) io.Writer { return c.stdout },
		Stderr: func(c testContext) io.Writer { return c.stderr },
	}
	controller := &resources.Controller{Root: "/root", Home: "/home"}

	for _, action := range []string{"status", "logs", "stop", "uninstall"} {
		if err := enforceResourceHostRequirements(ctx, deps, controller, "postgres", action); err != nil {
			t.Fatalf("enforceResourceHostRequirements(%s): %v", action, err)
		}
	}
	if called != 0 {
		t.Fatalf("enforce must not fire for non-mutating actions; calls=%d", called)
	}
}

func TestEnforceResourceHostRequirementsFiresForMutatingActions(t *testing.T) {
	prev := enforceHostRequirementsFn
	var captured []hostreqrun.Options
	enforceHostRequirementsFn = func(opts hostreqrun.Options) (vrooliruntime.Report, error) {
		captured = append(captured, opts)
		return vrooliruntime.Report{}, nil
	}
	t.Cleanup(func() { enforceHostRequirementsFn = prev })

	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := HandlerDeps[testContext]{
		Stdout: func(c testContext) io.Writer { return c.stdout },
		Stderr: func(c testContext) io.Writer { return c.stderr },
	}
	controller := &resources.Controller{Root: "/root", Home: "/home"}

	for _, action := range []string{"install", "start", "restart"} {
		if err := enforceResourceHostRequirements(ctx, deps, controller, "postgres", action); err != nil {
			t.Fatalf("enforceResourceHostRequirements(%s): %v", action, err)
		}
	}
	if len(captured) != 3 {
		t.Fatalf("expected 3 enforce calls, got %d", len(captured))
	}
	for _, opts := range captured {
		if opts.Resources != "postgres" {
			t.Fatalf("Resources = %q, want postgres", opts.Resources)
		}
		if opts.Environment != "development" {
			t.Fatalf("Environment = %q, want development", opts.Environment)
		}
		if opts.Scenarios != "none" {
			t.Fatalf("Scenarios = %q, want none", opts.Scenarios)
		}
		if opts.When != "develop" {
			t.Fatalf("When = %q, want develop", opts.When)
		}
		if opts.Label != "resource:postgres" {
			t.Fatalf("Label = %q, want resource:postgres", opts.Label)
		}
		if !opts.AutoInstall {
			t.Fatal("AutoInstall must be true")
		}
	}
}

func TestEnforceResourceHostRequirementsPropagatesErrors(t *testing.T) {
	prev := enforceHostRequirementsFn
	enforceHostRequirementsFn = func(_ hostreqrun.Options) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{}, errors.New("docker missing")
	}
	t.Cleanup(func() { enforceHostRequirementsFn = prev })

	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := HandlerDeps[testContext]{
		Stdout: func(c testContext) io.Writer { return c.stdout },
		Stderr: func(c testContext) io.Writer { return c.stderr },
	}
	controller := &resources.Controller{Root: "/root", Home: "/home"}

	err := enforceResourceHostRequirements(ctx, deps, controller, "postgres", "install")
	if err == nil || !strings.Contains(err.Error(), "docker missing") {
		t.Fatalf("expected docker missing error, got %v", err)
	}
}

func TestEnforceResourceHostRequirementsUsesControllerEnvironment(t *testing.T) {
	prev := enforceHostRequirementsFn
	var captured hostreqrun.Options
	enforceHostRequirementsFn = func(opts hostreqrun.Options) (vrooliruntime.Report, error) {
		captured = opts
		return vrooliruntime.Report{}, nil
	}
	t.Cleanup(func() { enforceHostRequirementsFn = prev })

	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := HandlerDeps[testContext]{
		Stdout: func(c testContext) io.Writer { return c.stdout },
		Stderr: func(c testContext) io.Writer { return c.stderr },
	}
	controller := &resources.Controller{Root: "/root", Home: "/home", Environment: "production"}

	if err := enforceResourceHostRequirements(ctx, deps, controller, "postgres", "install"); err != nil {
		t.Fatalf("enforceResourceHostRequirements: %v", err)
	}
	if captured.Environment != "production" {
		t.Fatalf("Environment = %q, want production", captured.Environment)
	}
}
