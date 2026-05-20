package shareddrifthandlers

import (
	"io"

	shareddrift "github.com/vrooli/vrooli/internal/app/shareddrift"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/shareddriftcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

type HandlerDeps[C any] struct {
	Stdout       func(C) io.Writer
	Root         func(C) string
	OutputFormat func(C) (cliout.Format, error)
}

func Handler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		req, err := shareddriftcli.ParseRequest(args)
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
		report, err := shareddrift.Service{Root: deps.Root(ctx)}.Check(shareddrift.CheckRequest{
			Fix:         req.Fix,
			OnlyTouched: req.OnlyTouched,
			Build:       req.Build,
			Concurrency: req.Concurrency,
		})
		if err != nil {
			return err
		}
		if err := shareddriftcli.Render(deps.Stdout(ctx), format, report); err != nil {
			return err
		}
		if !report.Clean {
			return rootcli.ExitCodeError{Code: 1, Silent_: true}
		}
		return nil
	}
}
