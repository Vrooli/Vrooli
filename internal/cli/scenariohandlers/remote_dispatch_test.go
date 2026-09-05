package scenariohandlers

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive
	"github.com/vrooli/vrooli/internal/cliout"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

type remoteDispatchCtx struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

type remoteDispatchCall struct {
	node, scenario, command string
	args                    []string
	json                    bool
}

func remoteDispatchDeps(calls *[]remoteDispatchCall) HandlerDeps[*remoteDispatchCtx] {
	return HandlerDeps[*remoteDispatchCtx]{
		Stdout:       func(ctx *remoteDispatchCtx) io.Writer { return ctx.stdout },
		Stderr:       func(ctx *remoteDispatchCtx) io.Writer { return ctx.stderr },
		Globals:      func(*remoteDispatchCtx) rootcli.GlobalOptions { return rootcli.GlobalOptions{} },
		OutputFormat: func(*remoteDispatchCtx) (cliout.Format, error) { return cliout.FormatHuman, nil },
		HomeDir:      func(*remoteDispatchCtx) (string, error) { return "/tmp", nil },
		CommandEnv:   func(*remoteDispatchCtx) []string { return nil },
		RemoteScenarioCall: func(_ *remoteDispatchCtx, node, scenario, command string, args []string, jsonOutput bool) ([]byte, error) {
			*calls = append(*calls, remoteDispatchCall{node: node, scenario: scenario, command: command, args: append([]string(nil), args...), json: jsonOutput})
			switch command {
			case "scenario status":
				return []byte("remote status\n"), nil
			case "scenario wait":
				return protojson.Marshal(&cliv1.ScenarioWaitResponse{Success: true, Scenario: scenario, Verdict: "healthy", Source: "registry"})
			case "scenario port":
				return protojson.Marshal(&cliv1.ScenarioPortSingle{Success: true, Scenario: scenario, PortName: "HTTP_PORT", Port: 4321})
			default:
				return protojson.Marshal(&cliv1.ScenarioLifecycleResponse{Success: true, Scenarios: []*cliv1.ScenarioLifecycleItem{{Name: scenario, Status: "started", Health: "healthy"}}})
			}
		},
	}
}

func TestExplicitNodeDispatchesAllLifecycleAndRuntimeVerbs(t *testing.T) {
	var calls []remoteDispatchCall
	deps := remoteDispatchDeps(&calls)
	ctx := &remoteDispatchCtx{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	handlers := BuildHandlers(deps)

	commands := []struct {
		id   CommandID
		args []string
		want string
	}{
		{CommandStatus, []string{"node-a/example"}, "scenario status"},
		{CommandStart, []string{"node-a/example"}, "scenario start"},
		{CommandRestart, []string{"node-a/example"}, "scenario restart"},
		{CommandStop, []string{"node-a/example"}, "scenario stop"},
		{CommandWait, []string{"node-a/example"}, "scenario wait"},
		{CommandPort, []string{"node-a/example", "HTTP_PORT"}, "scenario port"},
	}
	for _, command := range commands {
		if err := handlers[command.id](ctx, command.args); err != nil {
			t.Fatalf("%s: handler returned error: %v", command.id, err)
		}
	}

	if len(calls) != len(commands) {
		t.Fatalf("remote calls = %d, want %d (%#v)", len(calls), len(commands), calls)
	}
	for i, call := range calls {
		if call.node != "node-a" || call.scenario != "example" || call.command != commands[i].want {
			t.Errorf("call %d = %#v, want node-a/example/%s", i, call, commands[i].want)
		}
		if i > 0 && !call.json {
			t.Errorf("call %d did not request machine-readable remote output", i)
		}
	}
}

func TestRemoteDispatchForwardsArgumentsAndPreservesVariant(t *testing.T) {
	var calls []remoteDispatchCall
	deps := remoteDispatchDeps(&calls)
	ctx := &remoteDispatchCtx{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}

	if err := BuildHandlers(deps)[CommandStart](ctx, []string{"node-a/example@shadow", "--best-effort", "--timeout", "17"}); err != nil {
		t.Fatalf("start handler returned error: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %#v", calls)
	}
	want := remoteDispatchCall{node: "node-a", scenario: "example@shadow", command: "scenario start", args: []string{"--best-effort", "--timeout", "17"}, json: true}
	if !reflect.DeepEqual(calls[0], want) {
		t.Fatalf("call = %#v, want %#v", calls[0], want)
	}
}

func TestManifestLifecycleDispatchPreservesLegacyArguments(t *testing.T) {
	ctx := &remoteDispatchCtx{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	var invoked string
	var received []string
	handlers := make(map[string]rootcli.Handler[*remoteDispatchCtx])
	for _, spec := range CommandSpecs() {
		command := spec.Name
		handlers[command] = func(_ *remoteDispatchCtx, args []string) error {
			invoked = command
			received = append([]string(nil), args...)
			return nil
		}
	}
	lookup := func(name string) (rootcli.Handler[*remoteDispatchCtx], bool) {
		handler, ok := handlers[name]
		return handler, ok
	}
	root := RootHandler(
		func(ctx *remoteDispatchCtx) io.Writer { return ctx.stdout },
		func(ctx *remoteDispatchCtx) io.Writer { return ctx.stderr },
		func(*remoteDispatchCtx) rootcli.GlobalOptions { return rootcli.GlobalOptions{JSON: true} },
		lookup,
		func(string) []string { return nil },
	)

	tests := []struct {
		args []string
		want []string
	}{
		{[]string{"status", "alpha", "--instance", "blue"}, []string{"alpha", "--instance", "blue", "--json"}},
		{[]string{"logs", "alpha", "-f", "--tail", "20"}, []string{"alpha", "--follow", "--tail", "20", "--json"}},
		{[]string{"start", "alpha", "beta", "--best-effort", "--timeout", "17"}, []string{"alpha", "beta", "--best-effort", "--timeout", "17", "--json"}},
		{[]string{"stop", "alpha", "--node", "node-a"}, []string{"alpha", "--node", "node-a", "--json"}},
		{[]string{"restart", "alpha", "--force", "--path", "/tmp/alpha"}, []string{"alpha", "--path", "/tmp/alpha", "--force", "--json"}},
		{[]string{"setup", "alpha", "--path", "/tmp/alpha"}, []string{"alpha", "--path", "/tmp/alpha", "--json"}},
	}
	for _, test := range tests {
		invoked = ""
		received = nil
		if err := root(ctx, test.args); err != nil {
			t.Fatalf("%v: dispatch failed: %v", test.args, err)
		}
		if invoked != test.args[0] {
			t.Errorf("%v invoked %q", test.args, invoked)
		}
		if !reflect.DeepEqual(received, test.want) {
			t.Errorf("%v forwarded %#v, want %#v", test.args, received, test.want)
		}
	}
}

func TestManifestLifecycleDispatchPreservesLeafHelp(t *testing.T) {
	ctx := &remoteDispatchCtx{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	var received []string
	lookup := func(name string) (rootcli.Handler[*remoteDispatchCtx], bool) {
		if name != "start" {
			return nil, false
		}
		return func(_ *remoteDispatchCtx, args []string) error {
			received = append([]string(nil), args...)
			return nil
		}, true
	}
	root := RootHandler(
		func(ctx *remoteDispatchCtx) io.Writer { return ctx.stdout },
		func(ctx *remoteDispatchCtx) io.Writer { return ctx.stderr },
		func(*remoteDispatchCtx) rootcli.GlobalOptions { return rootcli.GlobalOptions{} },
		lookup,
		func(string) []string { return nil },
	)
	if err := root(ctx, []string{"start", "--help"}); err != nil {
		t.Fatalf("help dispatch failed: %v", err)
	}
	if !reflect.DeepEqual(received, []string{"--help"}) {
		t.Fatalf("help args = %#v", received)
	}
}

func TestRemoteDispatchFailuresHaveTriageForEveryExplicitNodeVerb(t *testing.T) {
	var calls []remoteDispatchCall
	deps := remoteDispatchDeps(&calls)
	deps.RemoteScenarioCall = func(_ *remoteDispatchCtx, _, _, _ string, _ []string, _ bool) ([]byte, error) {
		return nil, errors.New("node_offline")
	}
	ctx := &remoteDispatchCtx{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	handlers := BuildHandlers(deps)
	commands := []struct {
		id   CommandID
		args []string
		verb string
	}{
		{CommandStatus, []string{"node-a/example"}, "status"},
		{CommandStart, []string{"node-a/example"}, "start"},
		{CommandStop, []string{"node-a/example"}, "stop"},
		{CommandWait, []string{"node-a/example"}, "wait"},
		{CommandRestart, []string{"node-a/example"}, "restart"},
		{CommandPort, []string{"node-a/example", "HTTP_PORT"}, "query port"},
	}
	for _, command := range commands {
		ctx.stderr.Reset()
		if err := handlers[command.id](ctx, command.args); err == nil {
			t.Fatalf("%s: expected node error", command.id)
		}
		if !strings.Contains(ctx.stderr.String(), "Failed to "+command.verb) {
			t.Errorf("%s: stderr lacks triage block: %q", command.id, ctx.stderr.String())
		}
	}
}
