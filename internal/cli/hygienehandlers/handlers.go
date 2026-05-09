package hygienehandlers

import (
	"io"

	hygieneapp "github.com/vrooli/vrooli/internal/app/hygiene"
	"github.com/vrooli/vrooli/internal/cli/hygienecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

type HandlerDeps[C any] struct {
	Stdout       func(C) io.Writer
	Root         func(C) string
	Home         func(C) (string, error)
	OutputFormat func(C) (cliout.Format, error)
}

func Handler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		req, err := hygienecli.ParseRequest(args)
		if err != nil {
			if rootcli.HandleHelp(deps.Stdout(ctx), err) {
				return nil
			}
			return rootcli.UsageErrorf("", "%s", err.Error())
		}
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		home, err := deps.Home(ctx)
		if err != nil {
			return err
		}
		report, err := hygieneapp.Service{Root: deps.Root(ctx), Home: home}.Run(hygieneapp.Request{
			FixSafe:         req.FixSafe,
			Plans:           req.Plans,
			FailOn:          req.FailOn,
			IncludePlans:    !req.ContractOnly,
			IncludeContract: !req.PlansOnly,
		})
		if err != nil {
			return err
		}
		if err := hygienecli.Render(deps.Stdout(ctx), format, report, req.OutputMode); err != nil {
			return err
		}
		if !report.Success {
			return rootcli.ExitCodeError{Code: 1, Silent_: true}
		}
		return nil
	}
}
