package main

import (
	"io"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
)

type appCommandHandler func(app *App, ctx *commandContext, args []string) error

type resourceCommandHandler func(app *App, ctx *commandContext, controller *resources.Controller, args []string) error

type commandAction[Req any, Resp any] struct {
	parse  func(ctx *commandContext, args []string) (Req, error)
	run    func(app *App, ctx *commandContext, req Req) (cliout.Format, Resp, error)
	render func(w io.Writer, format cliout.Format, resp Resp) error
}

type helpOnlyError struct {
	text string
}

func (e helpOnlyError) Error() string    { return e.text }
func (e helpOnlyError) HelpText() string { return e.text }

func commandHelpOnly(text string) error {
	return helpOnlyError{text: text}
}

func executeCommandAction[Req any, Resp any](app *App, ctx *commandContext, args []string, action commandAction[Req, Resp]) error {
	type output struct {
		format cliout.Format
		resp   Resp
	}
	return commandtree.ExecuteAction(ctx.Stdout, args, commandtree.Action[Req, output]{
		Parse: func(args []string) (Req, error) {
			return action.parse(ctx, args)
		},
		Execute: func(req Req) (output, error) {
			format, resp, err := action.run(app, ctx, req)
			return output{format: format, resp: resp}, err
		},
		Render: func(w io.Writer, item output) error {
			return action.render(w, item.format, item.resp)
		},
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
	type output struct {
		format cliout.Format
		resp   Resp
	}
	return func(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
		return commandtree.ExecuteAction(ctx.Stdout, args, commandtree.Action[Req, output]{
			Parse: func(args []string) (Req, error) {
				return parse(ctx.Globals, args)
			},
			Execute: func(req Req) (output, error) {
				format, resp, err := run(controller, ctx, req)
				return output{format: format, resp: resp}, err
			},
			Render: func(w io.Writer, item output) error {
				return render(w, item.format, item.resp)
			},
		})
	}
}

func runAppSubcommandSet(app *App, ctx *commandContext, args []string, usage func(io.Writer), command string, handlers map[string]appCommandHandler) error {
	if len(args) == 0 || wantsCommandHelp(args) {
		usage(ctx.Stdout)
		return nil
	}
	handler, ok := handlers[commandtree.NormalizeName(args[0])]
	if !ok {
		return usageErrorf(command, "unknown %s command: %s", command, args[0])
	}
	return handler(app, ctx, args[1:])
}

func runResourceSubcommandSet(app *App, ctx *commandContext, controller *resources.Controller, args []string, usage func(io.Writer), command string, handlers map[string]resourceCommandHandler) error {
	if len(args) == 0 || wantsCommandHelp(args) {
		usage(ctx.Stdout)
		return nil
	}
	handler, ok := handlers[commandtree.NormalizeName(args[0])]
	if !ok {
		return usageErrorf(command, "unknown %s command: %s", command, args[0])
	}
	return handler(app, ctx, controller, args[1:])
}
