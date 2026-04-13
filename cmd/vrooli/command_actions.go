package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/cliout"
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
