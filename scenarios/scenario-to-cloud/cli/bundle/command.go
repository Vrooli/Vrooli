package bundle

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	internalmanifest "scenario-to-cloud/cli/internal/manifest"
)

// Run executes bundle subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "build":
		return runBuild(client, args[1:])
	case "list":
		return runList(client, args[1:])
	case "stats":
		return runStats(client, args[1:])
	case "delete":
		return runDelete(client, args[1:])
	case "cleanup":
		return runCleanup(client, args[1:])
	case "vps-list":
		return runVPSList(client, args[1:])
	case "vps-delete":
		return runVPSDelete(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud bundle help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud bundle <command> [arguments]

Commands:
  build <manifest.json>       Build a mini-Vrooli tarball for deployment
  list                        List all stored bundles
  stats                       Show bundle storage statistics
  delete <sha256>             Delete a bundle by SHA256
  cleanup                     Remove old or orphaned bundles
  vps-list <manifest.json>    List bundles on the VPS
  vps-delete <manifest.json>  Delete bundles from VPS

Run 'scenario-to-cloud bundle <command> -h' for command-specific options.`)
	return nil
}

func runBuild(client *Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: scenario-to-cloud bundle build <manifest.json>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	body, _, err := client.Build(manifest)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func runList(client *Client, args []string) error {
	jsonOutput := false

	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud bundle list [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		}
	}

	body, resp, err := client.List()
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	if len(resp.Bundles) == 0 {
		fmt.Println("No bundles found.")
		return nil
	}

	fmt.Printf("Bundles: %d\n", len(resp.Bundles))
	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("%-20s %-12s %-15s %-10s %s\n", "SHA256", "SIZE", "SCENARIO", "USES", "CREATED")

	for _, b := range resp.Bundles {
		sha := b.SHA256
		if len(sha) > 16 {
			sha = sha[:16] + "..."
		}
		scenario := b.ScenarioID
		if scenario == "" {
			scenario = "-"
		}
		fmt.Printf("%-20s %-12s %-15s %-10d %s\n", sha, formatSize(b.SizeBytes), truncate(scenario, 15), b.UseCount, b.CreatedAt)
	}

	return nil
}

func runStats(client *Client, args []string) error {
	jsonOutput := false

	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud bundle stats [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		}
	}

	body, resp, err := client.Stats()
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Println("Bundle Storage Statistics")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Total Bundles:    %d\n", resp.TotalCount)
	fmt.Printf("Total Size:       %s\n", formatSize(resp.TotalSizeBytes))
	fmt.Printf("Orphaned Bundles: %d (%s)\n", resp.OrphanedCount, formatSize(resp.OrphanedBytes))
	if resp.OldestBundle != "" {
		fmt.Printf("Oldest Bundle:    %s\n", resp.OldestBundle)
	}
	if resp.NewestBundle != "" {
		fmt.Printf("Newest Bundle:    %s\n", resp.NewestBundle)
	}

	return nil
}

func runDelete(client *Client, args []string) error {
	var sha256 string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud bundle delete <sha256> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && sha256 == "" {
				sha256 = args[i]
			}
		}
	}

	if sha256 == "" {
		return fmt.Errorf("usage: scenario-to-cloud bundle delete <sha256>")
	}

	body, resp, err := client.Delete(sha256)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Printf("Deleted bundle: %s\n", resp.SHA256)
	} else {
		fmt.Printf("Failed to delete bundle: %s\n", resp.Message)
	}

	return nil
}

func runCleanup(client *Client, args []string) error {
	req := CleanupRequest{}
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud bundle cleanup [flags]

Flags:
  --max-age <days>    Remove bundles older than this many days
  --max-count <n>     Keep only this many bundles
  --orphaned-only     Only remove orphaned bundles (not used by any deployment)
  --dry-run           Just report what would be cleaned, don't actually delete
  --json              Output raw JSON`)
			return nil
		case "--max-age":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					req.MaxAgeDays = n
				}
			}
		case "--max-count":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					req.MaxCount = n
				}
			}
		case "--orphaned-only":
			req.OrphanedOnly = true
		case "--dry-run":
			req.DryRun = true
		case "--json":
			jsonOutput = true
		}
	}

	body, resp, err := client.Cleanup(req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	action := "Cleaned up"
	if resp.DryRun {
		action = "Would clean up"
	}

	if resp.Success {
		fmt.Printf("%s %d bundle(s) (%s)\n", action, resp.RemovedCount, formatSize(resp.RemovedBytes))
		if len(resp.RemovedBundles) > 0 && len(resp.RemovedBundles) <= 10 {
			fmt.Println("Removed:")
			for _, sha := range resp.RemovedBundles {
				fmt.Printf("  %s\n", sha)
			}
		}
	} else {
		fmt.Printf("Cleanup failed: %s\n", resp.Message)
	}

	return nil
}

func runVPSList(client *Client, args []string) error {
	var manifestPath string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud bundle vps-list <manifest.json> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && manifestPath == "" {
				manifestPath = args[i]
			}
		}
	}

	if manifestPath == "" {
		return fmt.Errorf("usage: scenario-to-cloud bundle vps-list <manifest.json>")
	}

	manifest, err := internalmanifest.ReadJSONFile(manifestPath)
	if err != nil {
		return err
	}

	body, resp, err := client.VPSList(manifest)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	fmt.Printf("Bundles on VPS: %s\n", resp.Host)
	fmt.Println(strings.Repeat("-", 80))

	if len(resp.Bundles) == 0 {
		fmt.Println("No bundles found on VPS.")
		return nil
	}

	fmt.Printf("%-20s %-12s %s\n", "SHA256", "SIZE", "CREATED")
	for _, b := range resp.Bundles {
		sha := b.SHA256
		if len(sha) > 16 {
			sha = sha[:16] + "..."
		}
		fmt.Printf("%-20s %-12s %s\n", sha, formatSize(b.SizeBytes), b.CreatedAt)
	}

	fmt.Printf("\nTotal: %s\n", formatSize(resp.TotalBytes))
	return nil
}

func runVPSDelete(client *Client, args []string) error {
	var manifestPath string
	var sha256s []string
	req := VPSDeleteRequest{}
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud bundle vps-delete <manifest.json> [flags]

Flags:
  --sha <sha256>      Delete specific bundle (can be repeated)
  --all               Delete all bundles on VPS
  --orphaned-only     Only delete orphaned bundles
  --json              Output raw JSON`)
			return nil
		case "--sha":
			if i+1 < len(args) {
				i++
				sha256s = append(sha256s, args[i])
			}
		case "--all":
			req.All = true
		case "--orphaned-only":
			req.OrphanedOnly = true
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && manifestPath == "" {
				manifestPath = args[i]
			}
		}
	}

	if manifestPath == "" {
		return fmt.Errorf("usage: scenario-to-cloud bundle vps-delete <manifest.json>")
	}

	manifest, err := internalmanifest.ReadJSONFile(manifestPath)
	if err != nil {
		return err
	}

	req.SHA256s = sha256s

	body, resp, err := client.VPSDelete(manifest, req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Printf("Deleted %d bundle(s) from VPS (%s)\n", resp.RemovedCount, formatSize(resp.RemovedBytes))
	} else {
		fmt.Printf("VPS delete failed: %s\n", resp.Message)
	}

	return nil
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// formatSize formats bytes into a human-readable string.
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
