package sessions

import (
	"encoding/json"
	"fmt"
	"strings"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

func runRename(ctx *appctx.Context, args []string) error {
	var identifier string
	var newName string
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
			} else if newName == "" {
				newName = args[i]
			} else {
				return fmt.Errorf("unexpected argument: %s", args[i])
			}
		}
	}

	if identifier == "" || newName == "" {
		return fmt.Errorf("usage: %s session rename <profile-id|name> <new-name> [--json]", ctx.Name)
	}

	// Resolve profile by ID or name
	profile, err := resolveProfileID(ctx, identifier)
	if err != nil {
		return err
	}

	oldName := profile.Name
	if oldName == "" {
		oldName = "(unnamed)"
	}

	// Check for duplicate names and warn (but don't block)
	existingCount, err := countProfilesWithName(ctx, newName)
	if err == nil && existingCount > 0 && !jsonOutput {
		fmt.Printf("WARN: %d existing profile(s) already named %q. Consider using a unique name.\n", existingCount, newName)
	}

	// Rename the profile
	updated, raw, err := renameProfile(ctx, profile.ID, newName)
	if err != nil {
		return err
	}

	if jsonOutput {
		response := map[string]interface{}{
			"status":   "renamed",
			"id":       updated.ID,
			"old_name": profile.Name,
			"new_name": updated.Name,
			"profile":  json.RawMessage(raw),
		}
		data, _ := json.MarshalIndent(response, "", "  ")
		cliutil.PrintJSON(data)
		return nil
	}

	fmt.Printf("OK: Session profile renamed\n")
	fmt.Printf("  ID:       %s\n", updated.ID[:8])
	fmt.Printf("  Old name: %s\n", oldName)
	fmt.Printf("  New name: %s\n", updated.Name)

	return nil
}
