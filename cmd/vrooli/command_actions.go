package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
)

type commandAction[Req any, Resp any] struct {
	parse  func(ctx *commandContext, args []string) (Req, error)
	run    func(app *App, ctx *commandContext, req Req) (cliout.Format, Resp, error)
	render func(w io.Writer, format cliout.Format, resp Resp) error
}

type boundCommandAction[Deps any, Req any, Resp any] struct {
	parse  func(globals globalOptions, args []string) (Req, error)
	run    func(deps Deps, ctx *commandContext, req Req) (cliout.Format, Resp, error)
	render func(w io.Writer, format cliout.Format, resp Resp) error
}

func executeCommandAction[Req any, Resp any](app *App, ctx *commandContext, args []string, action commandAction[Req, Resp]) error {
	req, err := action.parse(ctx, args)
	if err != nil {
		var helpErr commandHelpError
		if errors.As(err, &helpErr) {
			_, _ = fmt.Fprintln(ctx.Stdout, helpErr.message)
			return nil
		}
		return err
	}
	format, resp, err := action.run(app, ctx, req)
	if err != nil {
		return err
	}
	return action.render(ctx.Stdout, format, resp)
}

func executeBoundCommand[Deps any, Req any, Resp any](app *App, ctx *commandContext, deps Deps, args []string, action boundCommandAction[Deps, Req, Resp]) error {
	return executeCommandAction(app, ctx, args, commandAction[Req, Resp]{
		parse: func(ctx *commandContext, args []string) (Req, error) {
			return action.parse(ctx.Globals, args)
		},
		run: func(_ *App, ctx *commandContext, req Req) (cliout.Format, Resp, error) {
			return action.run(deps, ctx, req)
		},
		render: action.render,
	})
}

func bindGlobalCommand[Req any, Resp any](
	parse func(globals globalOptions, args []string) (Req, error),
	run func(app *App, ctx *commandContext, req Req) (cliout.Format, Resp, error),
	render func(w io.Writer, format cliout.Format, resp Resp) error,
) appCommandHandler {
	return func(app *App, ctx *commandContext, args []string) error {
		return executeCommandAction(app, ctx, args, commandAction[Req, Resp]{
			parse: func(ctx *commandContext, args []string) (Req, error) {
				return parse(ctx.Globals, args)
			},
			run:    run,
			render: render,
		})
	}
}

func bindContextCommand[Req any, Resp any](
	parse func(ctx *commandContext, args []string) (Req, error),
	run func(app *App, ctx *commandContext, req Req) (cliout.Format, Resp, error),
	render func(w io.Writer, format cliout.Format, resp Resp) error,
) appCommandHandler {
	return func(app *App, ctx *commandContext, args []string) error {
		return executeCommandAction(app, ctx, args, commandAction[Req, Resp]{
			parse:  parse,
			run:    run,
			render: render,
		})
	}
}

func bindResourceCommand[Req any, Resp any](
	parse func(globals globalOptions, args []string) (Req, error),
	run func(controller *resources.Controller, ctx *commandContext, req Req) (cliout.Format, Resp, error),
	render func(w io.Writer, format cliout.Format, resp Resp) error,
) resourceCommandHandler {
	return func(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
		return executeResourceCommandWithApp(app, ctx, controller, args, boundCommandAction[*resources.Controller, Req, Resp]{
			parse:  parse,
			run:    run,
			render: render,
		})
	}
}
