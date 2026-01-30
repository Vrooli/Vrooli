package sessions

import (
	"encoding/json"
	"fmt"
	"strings"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

func runClearStorage(ctx *appctx.Context, args []string) error {
	var identifier string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return fmt.Errorf("unknown option: %s", args[i])
			}
			if identifier == "" {
				identifier = args[i]
			} else {
				return fmt.Errorf("unexpected argument: %s", args[i])
			}
		}
	}

	if identifier == "" {
		return fmt.Errorf("usage: %s session clear-storage <profile-id|name> [--json]", ctx.Name)
	}

	// Resolve profile by ID or name
	profile, err := resolveProfileID(ctx, identifier)
	if err != nil {
		return err
	}

	if err := clearStorage(ctx, profile.ID); err != nil {
		return err
	}

	if jsonOutput {
		response := map[string]string{
			"status": "cleared",
			"id":     profile.ID,
			"name":   profile.Name,
		}
		data, _ := json.MarshalIndent(response, "", "  ")
		cliutil.PrintJSON(data)
		return nil
	}

	displayName := profile.Name
	if displayName == "" {
		displayName = "(unnamed)"
	}
	fmt.Printf("OK: Storage cleared for profile %s (%s)\n", displayName, profile.ID[:8])
	return nil
}
