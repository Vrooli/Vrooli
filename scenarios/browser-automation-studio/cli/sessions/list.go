package sessions

import (
	"browser-automation-studio/cli/internal/appctx"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func runList(ctx *appctx.Context, args []string) error {
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	profiles, raw, err := listProfiles(ctx)
	if err != nil {
		return err
	}
	if jsonOutput {
		cliutil.PrintJSON(raw)
		return nil
	}

	fmt.Println("Session Profiles")
	fmt.Println("================")

	if len(profiles) == 0 {
		fmt.Println("No session profiles found")
		return nil
	}

	for _, profile := range profiles {
		name := profile.Name
		if name == "" {
			name = "(unnamed)"
		}
		storageStatus := ""
		if profile.HasStorageState {
			storageStatus = " [has storage]"
		}
		lastUsed := formatListLastUsed(profile.CreatedAt, profile.LastUsedAt)
		fmt.Printf("  %s - %s%s%s\n", profile.ID[:8], name, storageStatus, lastUsed)
	}

	return nil
}

// formatListLastUsed returns "(last used: date)" if actually used, empty string otherwise.
func formatListLastUsed(createdAt, lastUsedAt string) string {
	if lastUsedAt == "" || strings.HasPrefix(lastUsedAt, "0001-") {
		return ""
	}
	// If last used is within 1 second of created, consider it "never used"
	created, err1 := time.Parse(time.RFC3339, createdAt)
	lastUsed, err2 := time.Parse(time.RFC3339, lastUsedAt)
	if err1 == nil && err2 == nil {
		diff := lastUsed.Sub(created)
		if diff < time.Second && diff > -time.Second {
			return "" // Never actually used
		}
	}
	// Format date portion only
	parts := strings.Split(lastUsedAt, "T")
	if len(parts) > 0 && parts[0] != "" {
		return fmt.Sprintf(" (last used: %s)", parts[0])
	}
	return ""
}
