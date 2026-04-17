package workflows

import (
	"browser-automation-studio/cli/internal/appctx"
	"fmt"
	"net/url"
)

func runDelete(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: browser-automation-studio workflow delete <id>")
	}
	workflowID := args[0]
	_, err := ctx.Core.Request("DELETE", "/workflows/"+workflowID, url.Values{}, nil)
	if err != nil {
		return err
	}
	fmt.Println("OK: Workflow deleted")
	return nil
}
