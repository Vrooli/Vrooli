package sessions

import (
	"browser-automation-studio/cli/internal/appctx"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func runCreate(ctx *appctx.Context, args []string) error {
	var name string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return fmt.Errorf("unknown option: %s", args[i])
			}
			if name == "" {
				name = args[i]
			} else {
				return fmt.Errorf("unexpected argument: %s", args[i])
			}
		}
	}

	// Check for duplicate names and warn (but don't block)
	if name != "" {
		existingCount, err := countProfilesWithName(ctx, name)
		if err == nil && existingCount > 0 && !jsonOutput {
			fmt.Printf("WARN: %d existing profile(s) already named %q. Consider using a unique name.\n", existingCount, name)
		}
	}

	profile, raw, err := createProfile(ctx, name)
	if err != nil {
		return err
	}
	if jsonOutput {
		cliutil.PrintJSON(raw)
		return nil
	}

	displayName := profile.Name
	if displayName == "" {
		displayName = "(unnamed)"
	}
	fmt.Printf("OK: Session profile created\n")
	fmt.Printf("  ID:   %s\n", profile.ID)
	fmt.Printf("  Name: %s\n", displayName)

	return nil
}
