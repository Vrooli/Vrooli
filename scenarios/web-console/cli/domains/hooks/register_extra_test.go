package hooks

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRegistrarBranches(t *testing.T) {
	r := &registrar{lookup: func(string) (string, error) { return "", errors.New("missing") }, getenv: func(string) string { return "" }}
	if _, err := r.resourceCLI(); err == nil {
		t.Fatal("missing resource CLI unexpectedly resolved")
	}
	r = newRegistrar()
	if err := r.register([]string{"unexpected"}); err == nil {
		t.Fatal("register args unexpectedly accepted")
	}
	if err := r.remove([]string{"unexpected"}); err == nil {
		t.Fatal("remove args unexpectedly accepted")
	}
	if got := shellQuote("a'b"); got != "'a'\\''b'" {
		t.Fatalf("shell quote = %q", got)
	}

	r = &registrar{run: func(context.Context, string, ...string) ([]byte, error) { return nil, nil }, stdout: &strings.Builder{}}
	if err := r.reconcile("resource", "Stop", "id", "cmd", 1, "Stop", 123); err == nil {
		t.Fatal("empty reconcile output unexpectedly succeeded")
	}
	r.run = func(context.Context, string, ...string) ([]byte, error) { return []byte(`{"status":"applied"}`), nil }
	if err := r.reconcile("resource", "Stop", "id", "cmd", 1, "Stop", 123); err != nil {
		t.Fatal(err)
	}
	r.run = func(context.Context, string, ...string) ([]byte, error) { return []byte(`{"status":"unchanged"}`), nil }
	if err := r.reconcile("resource", "Stop", "id", "cmd", 1, "Stop", 123); err != nil {
		t.Fatal(err)
	}
	if err := r.reconcile("resource", "Stop", "id", "cmd", 1, "Stop", 123); err != nil {
		t.Fatal(err)
	}
}

func TestRegistrarEnvironmentResolution(t *testing.T) {
	r := &registrar{getenv: func(k string) string {
		if k == "API_PORT" {
			return "16382"
		}
		return ""
	}, userHomeDir: func() (string, error) { return "/missing", nil }, readFile: func(string) ([]byte, error) { return nil, errors.New("missing") }}
	if port, err := r.apiPort(); err != nil || port != 16382 {
		t.Fatalf("apiPort = %d, %v", port, err)
	}
	r.getenv = func(string) string { return "bad" }
	if _, err := r.apiPort(); err == nil {
		t.Fatal("bad API_PORT unexpectedly accepted")
	}
	r.maxAttempts = 1
	if _, err := r.awaitHookToken(); err == nil {
		t.Fatal("missing hook token unexpectedly succeeded")
	}
}

func TestRegisterExposesValidationCommands(t *testing.T) {
	group := Register()
	if len(group.Subcommands) != 2 {
		t.Fatalf("commands = %#v", group.Subcommands)
	}
	for _, command := range group.Subcommands {
		if err := command.Run([]string{"unexpected"}); err == nil {
			t.Fatalf("%s accepted unexpected args", command.Name)
		}
	}
}

func TestRegistrarRemoveAndCommandRunner(t *testing.T) {
	r := &registrar{
		lookup: func(string) (string, error) { return "/resource-claude-code", nil },
		run:    func(context.Context, string, ...string) ([]byte, error) { return nil, nil },
		stdout: &strings.Builder{},
	}
	if err := r.remove(nil); err != nil {
		t.Fatal(err)
	}
	if output, err := runCommand(context.Background(), "true"); err != nil || len(output) != 0 {
		t.Fatalf("runCommand = %q, %v", output, err)
	}
}
