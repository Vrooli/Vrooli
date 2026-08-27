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

type hygieneService struct {
	root string
	home string
}

type hygieneResponse struct {
	report hygieneapp.Report
	mode   hygienecli.OutputMode
}

//nolint:gocyclo // scope flags are intentionally explicit so mutually exclusive hygiene lanes remain unambiguous.
func Handler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return rootcli.BindService(deps.Stdout,
		func(ctx C) (cliout.Format, hygieneService, error) {
			format, err := deps.OutputFormat(ctx)
			if err != nil {
				return "", hygieneService{}, err
			}
			home, err := deps.Home(ctx)
			if err != nil {
				return "", hygieneService{}, err
			}
			return format, hygieneService{root: deps.Root(ctx), home: home}, nil
		},
		func(ctx C, args []string) (hygienecli.Request, error) {
			req, err := hygienecli.ParseRequest(args)
			if err != nil {
				return hygienecli.Request{}, rootcli.UsageErrorf("", "%s", err.Error())
			}
			return req, nil
		},
		func(service hygieneService, req hygienecli.Request) (hygieneResponse, error) {
			includeContract := !req.PlansOnly && !req.DriftOnly && !req.TidinessOnly
			includePlans := !req.ContractOnly && !req.DriftOnly && !req.TidinessOnly
			includeDrift := !req.PlansOnly && !req.ContractOnly && !req.NoDrift && !req.TidinessOnly
			includeFreshness := !req.PlansOnly && !req.ContractOnly && !req.DriftOnly && !req.NoFreshness && !req.TidinessOnly
			report, err := hygieneapp.Service{Root: service.root, Home: service.home}.Run(hygieneapp.Request{
				FixSafe:                 req.FixSafe,
				Plans:                   req.Plans,
				FailOn:                  req.FailOn,
				IncludePlans:            includePlans,
				IncludeContract:         includeContract,
				IncludeDrift:            includeDrift,
				IncludeFreshness:        includeFreshness,
				IncludeTidiness:         req.TidinessOnly || (!req.PlansOnly && !req.ContractOnly && !req.DriftOnly),
				RequireTidinessProvider: req.TidinessOnly,
			})
			if err != nil {
				return hygieneResponse{}, err
			}
			return hygieneResponse{report: report, mode: req.OutputMode}, nil
		},
		func(w io.Writer, format cliout.Format, response hygieneResponse) error {
			if err := hygienecli.Render(w, format, response.report, response.mode); err != nil {
				return err
			}
			if !response.report.Success {
				return rootcli.ExitCodeError{Code: 1, Silent_: true}
			}
			return nil
		},
	)
}
