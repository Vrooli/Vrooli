package templateengine

import (
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/scenarioexec"
)

type HandlerDeps[C any] struct {
	Stdout             func(C) io.Writer
	Stderr             func(C) io.Writer
	Root               func(C) string
	Globals            func(C) rootcli.GlobalOptions
	OutputFormat       func(C) (cliout.Format, error)
	HomeDir            func(C) (string, error)
	RunSubprocess      func(C, scenarioexec.SubprocessSpec) error
	LocateTestGenieCLI func(C) (string, error)
	CommandEnv         func(C) []string
}

func bindGlobal[C any, Req any, Resp any](
	stdout func(C) io.Writer,
	parse func(C, []string) (Req, error),
	run func(C, Req) (cliout.Format, Resp, error),
	render func(io.Writer, cliout.Format, Resp) error,
) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		req, err := parse(ctx, args)
		if err != nil {
			if helpErr, ok := err.(interface{ HelpText() string }); ok {
				_, _ = fmt.Fprint(stdout(ctx), helpErr.HelpText())
				return nil
			}
			return err
		}
		format, resp, err := run(ctx, req)
		if err != nil {
			return err
		}
		return render(stdout(ctx), format, resp)
	}
}
