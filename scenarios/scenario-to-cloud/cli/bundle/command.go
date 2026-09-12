package bundle

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	internalmanifest "scenario-to-cloud/cli/internal/manifest"
	"scenario-to-cloud/cli/internal/selector"
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
	case "vps-gc":
		return runVPSGC(client, args[1:])
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
  vps-list <selector>         List bundles retained on the deployment's target
  vps-gc                      Garbage-collect VPS bundle cache by deployment selector

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
	fmt.Printf("%-20s %-12s %-15s %-10s %s\n", "SHA256", "SIZE", "SCENARIO", "FILE", "CREATED")

	for _, b := range resp.Bundles {
		sha := b.Sha256
		if len(sha) > 16 {
			sha = sha[:16] + "..."
		}
		fmt.Printf("%-20s %-12s %-15s %-10s %s\n", sha, formatSize(b.SizeBytes), truncate(b.ScenarioID, 15), truncate(b.Filename, 10), b.CreatedAt)
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
	fmt.Printf("Total Bundles:    %d\n", resp.Stats.TotalCount)
	fmt.Printf("Total Size:       %s\n", formatSize(resp.Stats.TotalSizeBytes))
	if resp.Stats.OldestCreatedAt != "" {
		fmt.Printf("Oldest Created:   %s\n", resp.Stats.OldestCreatedAt)
	}
	if resp.Stats.NewestCreatedAt != "" {
		fmt.Printf("Newest Created:   %s\n", resp.Stats.NewestCreatedAt)
	}
	if len(resp.Stats.ByScenario) > 0 {
		fmt.Printf("Scenarios:        %d\n", len(resp.Stats.ByScenario))
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

	if resp.OK {
		fmt.Printf("Deleted bundle (%s)\n", formatSize(resp.FreedBytes))
	} else if resp.Message != "" {
		fmt.Printf("Delete failed: %s\n", resp.Message)
	}

	return nil
}

func runCleanup(client *Client, args []string) error {
	req := CleanupRequest{KeepLatest: 3}
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud bundle cleanup [flags]

Flags:
  --scenario <id>     Only clean bundles for one scenario
  --keep <n>          Keep N newest bundles (default: 3)
  --json              Output raw JSON`)
			return nil
		case "--scenario":
			if i+1 < len(args) {
				i++
				req.ScenarioID = strings.TrimSpace(args[i])
			}
		case "--keep":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					req.KeepLatest = n
				}
			}
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

	if resp.OK {
		fmt.Printf("Deleted %d local bundle(s) (%s)\n", len(resp.LocalDeleted), formatSize(resp.LocalFreedBytes))
		if len(resp.LocalDeleted) > 0 && len(resp.LocalDeleted) <= 10 {
			fmt.Println("Deleted:")
			for _, b := range resp.LocalDeleted {
				fmt.Printf("  %s\n", b.Filename)
			}
		}
	} else if resp.Message != "" {
		fmt.Printf("Cleanup failed: %s\n", resp.Message)
	}

	return nil
}

func runVPSList(client *Client, args []string) error {
	// Selector mode: scenario-to-cloud bundle vps-list <selector> [--json]
	fs := flag.NewFlagSet("bundle vps-list", flag.ContinueOnError)
	selFlags := selector.Register(fs)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	sel, err := selFlags.Selector(fs.Args())
	if err != nil {
		return err
	}
	deploymentID, err := selector.ResolveID(context.Background(), client.Deployments, sel)
	if err != nil {
		return err
	}

	body, resp, err := client.DeploymentVPSList(deploymentID)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("VPS Bundles (deployment: %s)\n", deploymentID)
	fmt.Println(strings.Repeat("-", 100))
	if len(resp.Bundles) == 0 {
		fmt.Println("No bundles found on VPS.")
		return nil
	}
	fmt.Printf("%-20s %-25s %-12s %s\n", "SHA256", "SCENARIO", "SIZE", "MODIFIED")
	for _, b := range resp.Bundles {
		sha := b.Sha256
		if len(sha) > 16 {
			sha = sha[:16] + "..."
		}
		fmt.Printf("%-20s %-25s %-12s %s\n", sha, truncate(b.ScenarioID, 25), formatSize(b.SizeBytes), b.ModTime)
	}
	fmt.Printf("\nTotal: %s\n", formatSize(resp.TotalSizeBytes))
	return nil
}

func runVPSGC(client *Client, args []string) error {
	fs := flag.NewFlagSet("bundle vps-gc", flag.ContinueOnError)
	selFlags := selector.Register(fs)
	keep := fs.Int("keep", 2, "Keep N newest bundles per scenario (default: 2)")
	dryRun := fs.Bool("dry-run", false, "Report plan only; do not delete")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	sel, err := selFlags.Selector(fs.Args())
	if err != nil {
		return err
	}
	ref, err := selector.Resolve(context.Background(), client.Deployments, sel)
	if err != nil {
		return err
	}

	body, resp, err := client.DeploymentVPSGC(ref.GetId(), VPSBundleGCRequest{
		ScenarioID: ref.GetScenarioId(),
		KeepLatest: *keep,
		DryRun:     *dryRun,
	})
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	action := "Deleted"
	if resp.DryRun {
		action = "Would delete"
	}
	if !resp.OK {
		return fmt.Errorf("vps-gc failed: %s", resp.Error)
	}

	fmt.Printf("%s %d bundle(s) (%s)\n", action, resp.DeletedCount, formatSize(resp.DeletedBytes))
	if resp.DeletedCount > 0 && resp.DeletedCount <= 10 {
		fmt.Println("Deleted:")
		for _, b := range resp.Deleted {
			fmt.Printf("  %s\n", b.Filename)
		}
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
