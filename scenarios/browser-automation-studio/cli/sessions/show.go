package sessions

import (
	"encoding/json"
	"fmt"
	"strings"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

func runShow(ctx *appctx.Context, args []string) error {
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
		return fmt.Errorf("usage: %s session show <profile-id|name> [--json]", ctx.Name)
	}

	// Resolve profile by ID or name
	profile, err := resolveProfileID(ctx, identifier)
	if err != nil {
		return err
	}

	// Get storage state details
	storage, _, err := getStorageState(ctx, profile.ID)
	if err != nil {
		return fmt.Errorf("failed to get storage state: %w", err)
	}

	if jsonOutput {
		// Build combined response for JSON output
		combined := map[string]interface{}{
			"profile": profile,
			"storage": storage,
		}
		data, err := json.MarshalIndent(combined, "", "  ")
		if err != nil {
			return err
		}
		cliutil.PrintJSON(data)
		return nil
	}

	// Human-readable output
	displayName := profile.Name
	if displayName == "" {
		displayName = "(unnamed)"
	}

	fmt.Println("Session Profile")
	fmt.Println("===============")
	fmt.Printf("  ID:         %s\n", profile.ID)
	fmt.Printf("  Name:       %s\n", displayName)
	fmt.Printf("  Created:    %s\n", formatTimestamp(profile.CreatedAt))
	fmt.Printf("  Updated:    %s\n", formatTimestamp(profile.UpdatedAt))
	if profile.LastUsedAt != "" && !strings.HasPrefix(profile.LastUsedAt, "0001-") {
		fmt.Printf("  Last Used:  %s\n", formatTimestamp(profile.LastUsedAt))
	}
	fmt.Println()

	fmt.Println("Storage State")
	fmt.Println("-------------")
	fmt.Printf("  Cookies:      %d\n", storage.Stats.CookieCount)
	fmt.Printf("  Origins:      %d\n", storage.Stats.OriginCount)
	fmt.Printf("  LocalStorage: %d items\n", storage.Stats.LocalStorageCount)

	if storage.Stats.CookieCount > 0 {
		fmt.Println()
		fmt.Println("  Cookies:")
		for _, cookie := range storage.Cookies {
			valueDisplay := cookie.Value
			if cookie.ValueMasked {
				valueDisplay = "[HIDDEN]"
			} else if len(valueDisplay) > 30 {
				valueDisplay = valueDisplay[:30] + "..."
			}
			fmt.Printf("    %s (%s): %s\n", cookie.Name, cookie.Domain, valueDisplay)
		}
	}

	if storage.Stats.OriginCount > 0 {
		fmt.Println()
		fmt.Println("  LocalStorage:")
		for _, origin := range storage.Origins {
			fmt.Printf("    %s:\n", origin.Origin)
			for _, item := range origin.LocalStorage {
				valueDisplay := item.Value
				if len(valueDisplay) > 40 {
					valueDisplay = valueDisplay[:40] + "..."
				}
				fmt.Printf("      %s: %s\n", item.Name, valueDisplay)
			}
		}
	}

	return nil
}

func formatTimestamp(ts string) string {
	if ts == "" || strings.HasPrefix(ts, "0001-") {
		return "-"
	}
	// Simplify ISO timestamp for display
	parts := strings.Split(ts, "T")
	if len(parts) == 2 {
		timePart := strings.Split(parts[1], ".")[0]
		timePart = strings.TrimSuffix(timePart, "Z")
		return parts[0] + " " + timePart
	}
	return ts
}
