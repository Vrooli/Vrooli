package executions

import (
	"fmt"

	"browser-automation-studio/cli/internal/appctx"
)

func runStop(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("execution ID is required")
	}
	executionID := args[0]

	_, err := ctx.Core.APIClient.Request("POST", ctx.APIPath("/executions/"+executionID+"/stop"), nil, nil)
	if err != nil {
		return err
	}
	fmt.Printf("OK: Execution stopped\n")
	return nil
}
