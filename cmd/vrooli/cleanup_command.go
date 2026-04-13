package main

import (
	"errors"
	"fmt"
	"io"
)

func runCleanupCommand(root string, parsed parsedArgs, stdout, stderr io.Writer) error {
	app, ctx := newConfiguredCommandContext(root, parsed.globals, stdout, stderr)
	return runCleanupCommandWithApp(app, ctx, parsed)
}

func runCleanupCommandWithApp(app *App, ctx *commandContext, parsed parsedArgs) error {
	if len(parsed.args) == 0 {
		showCleanupHelp(ctx.Stdout)
		return nil
	}

	target := parsed.args[0]
	rest := parsed.args[1:]
	switch target {
	case "orphans":
		return bindGlobalCommand(parseTopLevelOrphansRequest, runTopLevelOrphansRequest, renderTopLevelOrphansResponse)(app, ctx, append([]string{"kill"}, rest...))
	case "locks":
		return bindGlobalCommand(parseTopLevelLocksRequest, runTopLevelLocksRequest, renderTopLevelLocksResponse)(app, ctx, append([]string{"clean"}, rest...))
	case "help", "--help", "-h":
		showCleanupHelp(ctx.Stdout)
		return nil
	default:
		return newErrorWithCategory(
			errors.New(fmt.Sprintf("unknown cleanup target: %s", target)),
			errorCategoryUsage,
			usageHint("cleanup"),
			[]string{"orphans", "locks"},
		)
	}
}
