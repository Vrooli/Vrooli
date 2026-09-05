// Package authhandlers wires the `vrooli auth` command tree from
// internal/cli/authcli to runnable handlers backed by internal/app/auth.
package authhandlers

import (
	"context"
	"io"

	authapp "github.com/vrooli/vrooli/internal/app/auth"
	"github.com/vrooli/vrooli/internal/cli/authcli"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

// HandlerDeps are the per-context dependencies the auth handlers need.
type HandlerDeps[C any] struct {
	Stdout       func(C) io.Writer
	OutputFormat func(C) (cliout.Format, error)
	// Probes returns the probe set to evaluate. Defaults to
	// authapp.DefaultProbes when nil. Tests inject stubs here.
	Probes func(C) []authapp.SignInProbe
}

type statusService struct {
	probes []authapp.SignInProbe
}

func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	commandHandlers := commandtree.BuildHandlerMap(buildCommandTable(deps))
	return func(ctx C, args []string) error {
		return rootcli.RunSubcommandSet(ctx, args, authcli.RenderCommandHelp, "auth", commandHandlers, deps.Stdout)
	}
}

func buildCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.Handler[C]] {
	handlerMap := map[authcli.CommandID]rootcli.Handler[C]{
		authcli.CommandStatus: statusHandler(deps),
	}
	return commandtree.BindSpecs(authcli.CommandSpecs(), handlerMap)
}

func statusHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return rootcli.BindService(deps.Stdout,
		deps.OutputFormat,
		func(ctx C, _ cliout.Format) (statusService, error) {
			probes := authapp.DefaultProbes()
			if deps.Probes != nil {
				probes = deps.Probes(ctx)
			}
			return statusService{probes: probes}, nil
		},
		func(ctx C, args []string) (authcli.StatusRequest, error) {
			return authcli.ParseStatusRequest(args)
		},
		func(service statusService, req authcli.StatusRequest) (authapp.Report, error) {
			return authapp.Run(context.Background(), service.probes, authapp.ProbeOptions{CheckExpiry: req.CheckExpiry}), nil
		},
		authcli.RenderStatus,
	)
}
