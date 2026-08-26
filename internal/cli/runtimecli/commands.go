package runtimecli

import runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"

// Context is the narrow context passed from the root CLI to runtime commands.
type Context = runtimeapp.CommandContext

// App is the runtime command application exposed to handlers.
type App = runtimeapp.App

// Run dispatches `vrooli runtime`.
func Run(app *App, ctx *Context, args []string) error { return app.Run(ctx, args) }
