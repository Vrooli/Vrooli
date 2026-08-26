package capabilitycli

import capabilityapp "github.com/vrooli/vrooli/internal/app/capability"

// Context is the narrow context passed from root wiring to capability
// commands.
type Context = capabilityapp.CommandContext

// App is the capability command application exposed to handlers.
type App = capabilityapp.App

// Run dispatches `vrooli capability`.
func Run(app *App, ctx *Context, args []string) error { return app.Run(ctx, args) }
