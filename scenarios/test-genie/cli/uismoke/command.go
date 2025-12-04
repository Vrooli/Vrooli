package uismoke

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Run executes the ui-smoke command.
func Run(client *Client, args []string) error {
	parsed, err := ParseArgs(args)
	if err != nil {
		return err
	}

	req := Request{
		URL:            parsed.URL,
		BrowserlessURL: parsed.BrowserlessURL,
		TimeoutMs:      parsed.TimeoutMs,
		NoRecovery:     parsed.NoRecovery,
		SharedMode:     parsed.SharedMode,
		AutoStart:      parsed.AutoStart,
	}

	resp, raw, err := client.Run(parsed.Scenario, req)
	if err != nil {
		return err
	}

	if parsed.JSON {
		cliutil.PrintJSON(raw)
		exitWithCode(resp)
		return nil
	}

	printResult(resp)
	return nil
}

// exitWithCode exits with the appropriate code based on status and blocked reason.
// This is extracted so it can be called for both JSON and non-JSON modes.
func exitWithCode(resp Response) {
	switch resp.Status {
	case "passed", "skipped":
		// Success - exit 0 (default)
	case "blocked":
		os.Exit(ExitCodeForBlockedReason(resp.BlockedReason))
	case "failed":
		os.Exit(ExitFailure)
	default:
		os.Exit(ExitFailure)
	}
}

// ParseArgs parses command line arguments for the ui-smoke command.
func ParseArgs(args []string) (Args, error) {
	if len(args) == 0 {
		return Args{}, usageError("usage: ui-smoke <scenario> [--url <ui-url>] [--browserless <url>] [--timeout <ms>] [--no-recovery] [--shared-mode] [--auto-start] [--json]")
	}
	out := Args{Scenario: args[0]}
	fs := flag.NewFlagSet("ui-smoke", flag.ContinueOnError)
	fs.StringVar(&out.URL, "url", "", "Custom UI URL to test (overrides auto-detection)")
	fs.StringVar(&out.BrowserlessURL, "browserless", "", "Custom Browserless URL (default: http://localhost:4110)")
	fs.Int64Var(&out.TimeoutMs, "timeout", 0, "Overall timeout in milliseconds (default: 90000)")
	fs.BoolVar(&out.NoRecovery, "no-recovery", false, "Disable automatic browserless recovery on health failures")
	fs.BoolVar(&out.SharedMode, "shared-mode", false, "Shared mode: don't restart browserless if other sessions are active")
	fs.BoolVar(&out.AutoStart, "auto-start", false, "Auto-start the scenario if UI port is not detected")
	jsonOutput := cliutil.JSONFlag(fs)
	fs.SetOutput(flag.CommandLine.Output())
	if err := fs.Parse(args[1:]); err != nil {
		return Args{}, err
	}
	out.JSON = *jsonOutput
	return out, nil
}

func printResult(resp Response) {
	statusIcon := statusEmoji(resp.Status)

	fmt.Printf("\n%s UI Smoke Test: %s\n", statusIcon, resp.Scenario)
	fmt.Println(strings.Repeat("─", 50))

	fmt.Printf("  Status:    %s %s\n", statusIcon, resp.Status)
	if resp.Message != "" {
		fmt.Printf("  Message:   %s\n", resp.Message)
	}
	if resp.UIURL != "" {
		fmt.Printf("  URL:       %s\n", resp.UIURL)
	}
	if resp.DurationMs > 0 {
		fmt.Printf("  Duration:  %dms\n", resp.DurationMs)
	}

	fmt.Println()

	// Print handshake info if available
	if len(resp.Handshake) > 0 && string(resp.Handshake) != "null" {
		var handshake struct {
			Signaled   bool   `json:"signaled"`
			TimedOut   bool   `json:"timed_out"`
			DurationMs int64  `json:"duration_ms"`
			Error      string `json:"error"`
		}
		if err := json.Unmarshal(resp.Handshake, &handshake); err == nil {
			if handshake.Signaled {
				fmt.Printf("  ✅ @vrooli/iframe-bridge handshake in %dms\n", handshake.DurationMs)
			} else if handshake.TimedOut {
				fmt.Printf("  ⏱️  @vrooli/iframe-bridge timed out after %dms\n", handshake.DurationMs)
			} else if handshake.Error != "" {
				fmt.Printf("  ❌ @vrooli/iframe-bridge error: %s\n", handshake.Error)
			}
		}
	}

	// Print network failure status
	if resp.NetworkFailureCount == 0 {
		fmt.Println("  ✅ No network failures")
	} else {
		fmt.Printf("  ❌ %d network failure(s) detected (see network.json)\n", resp.NetworkFailureCount)
	}

	// Print page error status
	if resp.PageErrorCount == 0 {
		fmt.Println("  ✅ No JavaScript exceptions")
	} else {
		fmt.Printf("  ❌ %d JavaScript exception(s) detected (see console.json)\n", resp.PageErrorCount)
	}

	// Print console error warning (informational, doesn't fail the test)
	if resp.ConsoleErrorCount > 0 {
		fmt.Printf("  ⚠️  %d console error(s) logged (see console.json)\n", resp.ConsoleErrorCount)
	}

	// Print artifacts if available
	if len(resp.Artifacts) > 0 && string(resp.Artifacts) != "null" {
		var artifacts struct {
			Screenshot string `json:"screenshot"`
			Console    string `json:"console"`
			Network    string `json:"network"`
			HTML       string `json:"html"`
			Raw        string `json:"raw"`
			Readme     string `json:"readme"`
		}
		if err := json.Unmarshal(resp.Artifacts, &artifacts); err == nil {
			hasArtifacts := artifacts.Screenshot != "" || artifacts.Console != "" ||
				artifacts.Network != "" || artifacts.HTML != "" || artifacts.Raw != "" ||
				artifacts.Readme != ""
			if hasArtifacts {
				fmt.Println("\n  Artifacts:")
				if artifacts.Readme != "" {
					fmt.Printf("    📖 %s\n", artifacts.Readme)
				}
				if artifacts.Screenshot != "" {
					fmt.Printf("    📷 %s\n", artifacts.Screenshot)
				}
				if artifacts.Console != "" {
					fmt.Printf("    📋 %s\n", artifacts.Console)
				}
				if artifacts.Network != "" {
					fmt.Printf("    🌐 %s\n", artifacts.Network)
				}
				if artifacts.HTML != "" {
					fmt.Printf("    📄 %s\n", artifacts.HTML)
				}
				if artifacts.Raw != "" {
					fmt.Printf("    🔧 %s\n", artifacts.Raw)
				}
			}
		}
	}

	fmt.Println()

	exitWithCode(resp)
}

func statusEmoji(status string) string {
	switch status {
	case "passed":
		return "✅"
	case "failed":
		return "❌"
	case "skipped":
		return "⏭️"
	case "blocked":
		return "🚫"
	default:
		return "❓"
	}
}

func usageError(msg string) error {
	return errors.New(msg)
}
