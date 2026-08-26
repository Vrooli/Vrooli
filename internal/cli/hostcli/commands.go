package hostcli

import hostapp "github.com/vrooli/vrooli/internal/app/host"

// Context is the narrow context passed from root wiring to host commands.
type Context = hostapp.CommandContext

// App is the host command application exposed to handlers.
type App = hostapp.App

// Run dispatches `vrooli host`.
func Run(app *App, ctx *Context, args []string) error { return app.Run(ctx, args) }

// RunWorkload dispatches `vrooli workload`.
func RunWorkload(app *App, ctx *Context, args []string) error { return app.RunWorkload(ctx, args) }
