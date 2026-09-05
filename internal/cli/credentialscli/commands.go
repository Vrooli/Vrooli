package credentialscli

import credentialsapp "github.com/vrooli/vrooli/internal/app/credentials"

// Context is the narrow context passed from the root CLI to credential commands.
type Context = credentialsapp.CommandContext

// Run dispatches `vrooli credentials` through the credentials application service.
func Run(ctx *Context, args []string) error {
	return (&credentialsapp.App{}).Run(ctx, args)
}

// RunBreakGlass dispatches `vrooli break-glass` through the credentials service.
func RunBreakGlass(ctx *Context, args []string) error {
	return (&credentialsapp.App{}).RunBreakGlass(ctx, args)
}
