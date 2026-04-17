package executions

import (
	"browser-automation-studio/cli/internal/appctx"
	"fmt"
)

func runStop(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("execution ID is required")
	}
	executionID := args[0]

	_, err := ctx.Core.Request("POST", "/executions/"+executionID+"/stop", nil, nil)
	if err != nil {
		return err
	}
	fmt.Printf("OK: Execution stopped\n")
	return nil
}
