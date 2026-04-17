package sessions

import (
	"browser-automation-studio/cli/internal/appctx"
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func runDelete(ctx *appctx.Context, args []string) error {
	var identifier string
	jsonOutput := false
	forceDelete := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--force", "-f":
			forceDelete = true
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
		return fmt.Errorf("usage: %s session delete <profile-id|name> [--force] [--json]", ctx.Name)
	}

	// Resolve profile by ID or name
	profile, err := resolveProfileID(ctx, identifier)
	if err != nil {
		return err
	}

	displayName := profile.Name
	if displayName == "" {
		displayName = "(unnamed)"
	}

	// Prompt for confirmation unless --force or --json is used
	if !forceDelete && !jsonOutput {
		fmt.Printf("Delete session profile %q (%s)? [y/N] ", displayName, profile.ID[:8])
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := deleteProfile(ctx, profile.ID); err != nil {
		return err
	}

	if jsonOutput {
		response := map[string]string{
			"status": "deleted",
			"id":     profile.ID,
			"name":   profile.Name,
		}
		data, _ := json.MarshalIndent(response, "", "  ")
		cliutil.PrintJSON(data)
		return nil
	}

	fmt.Printf("OK: Session profile deleted (%s - %s)\n", profile.ID[:8], displayName)
	return nil
}
