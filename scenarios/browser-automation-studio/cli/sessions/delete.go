package sessions

import (
	"fmt"
	"strings"

	"browser-automation-studio/cli/internal/appctx"
)

func runDelete(ctx *appctx.Context, args []string) error {
	var identifier string

	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			return fmt.Errorf("unknown option: %s", args[i])
		}
		if identifier == "" {
			identifier = args[i]
		} else {
			return fmt.Errorf("unexpected argument: %s", args[i])
		}
	}

	if identifier == "" {
		return fmt.Errorf("usage: %s session delete <profile-id|name>", ctx.Name)
	}

	// Resolve profile by ID or name
	profile, err := resolveProfileID(ctx, identifier)
	if err != nil {
		return err
	}

	if err := deleteProfile(ctx, profile.ID); err != nil {
		return err
	}

	displayName := profile.Name
	if displayName == "" {
		displayName = "(unnamed)"
	}
	fmt.Printf("OK: Session profile deleted (%s - %s)\n", profile.ID[:8], displayName)
	return nil
}
