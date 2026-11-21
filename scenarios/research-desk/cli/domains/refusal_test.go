package domains

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"research-desk/cli/domains/launch"
	"research-desk/cli/domains/operator"

	launchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/launch/launch_v1connect"
	operatorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/operator/operator_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"
)

// refusal is what the desk answers a caller verified as an agent (API
// internal/wire.Caller).
const refusal = "this is the operator's action; write a needs-you item instead (caller: agent run run-7)"

// Run under an agent token, every command that acts for the operator
// says the desk's refusal and what to do instead.
func TestOperatorActionsUnderAnAgentTokenSayTheRefusal(t *testing.T) {
	refuse := connect.WithInterceptors(connect.UnaryInterceptorFunc(func(connect.UnaryFunc) connect.UnaryFunc {
		return func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New(refusal))
		}
	}))
	mux := http.NewServeMux()
	mux.Handle(launchconnect.NewLaunchServiceHandler(launchconnect.UnimplementedLaunchServiceHandler{}, refuse))
	mux.Handle(operatorconnect.NewOperatorServiceHandler(operatorconnect.UnimplementedOperatorServiceHandler{}, refuse))
	core := cliapptest.NewTestApp(t, mux)
	manifest := readManifestForTest(t)
	commands := map[string]cliapp.Command{}
	for _, register := range []func(*cliapp.ScenarioApp, []byte) (cliapp.CommandGroup, error){launch.Register, operator.Register} {
		group, err := register(core, manifest)
		require.NoError(t, err)
		for _, c := range group.Commands {
			commands[c.Name] = c
		}
	}
	for name, args := range map[string][]string{
		"launch":          {"programme-five-20261112"},
		"control":         {"programme-five-20261112", "pause"},
		"launch-test":     nil,
		"answer":          {"programme-five-20261112", "AV4", "keep"},
		"say":             {"programme-five-20261112", "direction", "Put AV6 first."},
		"enable-steering": {"programme-five-20261112"},
		"verdict":         {"ai-video-20261112", "ai-video-production-and-funnels", "Films first."},
		"courier":         {"on", "programme-five-20261112"},
		"courier-test":    nil,
		"mark-seen":       {"commission", "programme-five-20261112"},
	} {
		cmd, ok := commands[name]
		require.True(t, ok, name)
		_, err := cliapptest.RunCommand(t, cmd, core, args...)
		require.ErrorContains(t, err, refusal, name)
	}
}

// A refusal's message names no command; the CLI prints the CommandHint
// the desk attaches beside it.
func TestARefusalsCommandIsPrintedAfterItsMessage(t *testing.T) {
	notFound := connect.WithInterceptors(connect.UnaryInterceptorFunc(func(connect.UnaryFunc) connect.UnaryFunc {
		return func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
			cerr := connect.NewError(connect.CodeNotFound, errors.New(`no commission folder "x" under /r/cohorts`))
			detail, err := connect.NewErrorDetail(&sharedv1.CommandHint{Command: "research-desk commissions"})
			if err != nil {
				return nil, err
			}
			cerr.AddDetail(detail)
			return nil, cerr
		}
	}))
	mux := http.NewServeMux()
	mux.Handle(launchconnect.NewLaunchServiceHandler(launchconnect.UnimplementedLaunchServiceHandler{}, notFound))
	core := cliapptest.NewTestApp(t, mux)
	group, err := launch.Register(core, readManifestForTest(t))
	require.NoError(t, err)
	for _, c := range group.Commands {
		if c.Name == "launch-status" {
			_, err = cliapptest.RunCommand(t, c, core, "x")
			require.ErrorContains(t, err, `no commission folder "x" under /r/cohorts (next: research-desk commissions)`)
			return
		}
	}
	t.Fatal("no launch-status command")
}
