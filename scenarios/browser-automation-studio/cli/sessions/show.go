package sessions

import (
	"browser-automation-studio/cli/internal/appctx"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	fmt.Printf("  Last Used:  %s\n", formatLastUsed(profile.CreatedAt, profile.LastUsedAt))
	fmt.Println()

	// Display browser profile if configured
	bp := parseBrowserProfile(profile.BrowserProfile)
	if bp != nil && (bp.Preset != "" || bp.Behavior != nil || bp.AntiDetection != nil) {
		fmt.Println("Browser Profile")
		fmt.Println("---------------")
		if bp.Preset != "" {
			fmt.Printf("  Preset:     %s\n", bp.Preset)
		}
		if bp.Behavior != nil {
			if bp.Behavior.MouseMovementStyle != "" {
				fmt.Printf("  Mouse:      %s\n", bp.Behavior.MouseMovementStyle)
			}
			if bp.Behavior.ScrollStyle != "" {
				fmt.Printf("  Scroll:     %s\n", bp.Behavior.ScrollStyle)
			}
			if bp.Behavior.TypingDelayMin > 0 || bp.Behavior.TypingDelayMax > 0 {
				fmt.Printf("  Typing:     %d-%dms delay\n", bp.Behavior.TypingDelayMin, bp.Behavior.TypingDelayMax)
			}
			if bp.Behavior.MicroPauseEnabled {
				fmt.Printf("  Pauses:     enabled\n")
			}
		}
		if bp.AntiDetection != nil {
			features := []string{}
			if bp.AntiDetection.DisableAutomationControlled {
				features = append(features, "no-automation-flag")
			}
			if bp.AntiDetection.PatchNavigatorWebdriver {
				features = append(features, "webdriver-patch")
			}
			if bp.AntiDetection.HeadlessDetectionBypass {
				features = append(features, "headless-bypass")
			}
			if bp.AntiDetection.DisableWebRTC {
				features = append(features, "webrtc-disabled")
			}
			if len(features) > 0 {
				fmt.Printf("  Stealth:    %s\n", strings.Join(features, ", "))
			}
			if bp.AntiDetection.AdBlockingMode != "" && bp.AntiDetection.AdBlockingMode != "none" {
				fmt.Printf("  Ad Block:   %s\n", bp.AntiDetection.AdBlockingMode)
			}
		}
		fmt.Println()
	}

	fmt.Println("Storage State")
	fmt.Println("-------------")

	// Count expired cookies
	expiredCount := countExpiredCookies(storage.Cookies)
	if expiredCount > 0 {
		fmt.Printf("  Cookies:      %d (%d expired)\n", storage.Stats.CookieCount, expiredCount)
	} else {
		fmt.Printf("  Cookies:      %d\n", storage.Stats.CookieCount)
	}
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

// formatLastUsed returns "-" if the profile was never used (last_used equals created),
// otherwise returns the formatted timestamp.
func formatLastUsed(createdAt, lastUsedAt string) string {
	if lastUsedAt == "" || strings.HasPrefix(lastUsedAt, "0001-") {
		return "-"
	}
	// If last used is within 1 second of created, consider it "never used"
	// (the API sets last_used_at on creation, so they're nearly identical)
	created, err1 := time.Parse(time.RFC3339, createdAt)
	lastUsed, err2 := time.Parse(time.RFC3339, lastUsedAt)
	if err1 == nil && err2 == nil {
		diff := lastUsed.Sub(created)
		if diff < time.Second && diff > -time.Second {
			return "- (never used)"
		}
	}
	return formatTimestamp(lastUsedAt)
}

// countExpiredCookies returns the number of cookies that have expired.
// Cookie.Expires is a Unix timestamp in seconds (float64). A value of -1 or 0 means session cookie.
func countExpiredCookies(cookies []storageStateCookie) int {
	now := float64(time.Now().Unix())
	count := 0
	for _, c := range cookies {
		// Session cookies (expires <= 0) don't expire
		if c.Expires > 0 && c.Expires < now {
			count++
		}
	}
	return count
}
