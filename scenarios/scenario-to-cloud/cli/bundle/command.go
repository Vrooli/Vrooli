package bundle

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	deploymentcli "scenario-to-cloud/cli/deployment"
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
  vps-list <manifest.json>    List bundles on the VPS
  vps-delete <manifest.json>  Delete bundles from VPS
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
	// Manifest mode: scenario-to-cloud bundle vps-list <manifest.json> [--json]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		manifestPath := args[0]
		jsonOutput := false
		for _, a := range args[1:] {
			switch a {
			case "-h", "--help":
				fmt.Println(`Usage: scenario-to-cloud bundle vps-list <manifest.json> [flags]

Flags:
  --json    Output raw JSON`)
				return nil
			case "--json":
				jsonOutput = true
			}
		}

		manifest, err := internalmanifest.ReadJSONFile(manifestPath)
		if err != nil {
			return err
		}
		req, err := extractVPSBundleListRequestFromManifest(manifest)
		if err != nil {
			return err
		}

		body, resp, err := client.VPSList(req)
		if err != nil {
			return err
		}
		if jsonOutput {
			cliutil.PrintJSON(body)
			return nil
		}

		fmt.Printf("VPS Bundles (host: %s)\n", req.Host)
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

	// Selector mode: scenario-to-cloud bundle vps-list --domain/--host/--target --scenario <id> [--json]
	fs := flag.NewFlagSet("bundle vps-list", flag.ContinueOnError)
	selFlags := registerSelectorFlags(fs)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	sel, err := selFlags.toSelector()
	if err != nil {
		return err
	}
	if strings.TrimSpace(sel.ScenarioID) == "" {
		return fmt.Errorf("--scenario is required")
	}

	resolved, err := deploymentcli.ResolveLatestBySelector(deploymentcli.NewClient(client.APIClient()), sel)
	if err != nil {
		return err
	}
	if resolved == nil {
		return fmt.Errorf("no deployment found for selector")
	}

	body, resp, err := client.DeploymentVPSList(resolved.ID)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("VPS Bundles (deployment: %s)\n", resolved.ID)
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

func runVPSDelete(client *Client, args []string) error {
	var manifestPath string
	var sha256s []string
	var filenames []string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud bundle vps-delete <manifest.json> [flags]

Flags:
  --sha <sha256>      Delete specific bundle (can be repeated)
  --filename <name>   Delete specific bundle filename (can be repeated)
  --all               Delete all bundles on VPS (lists then deletes)
  --json              Output raw JSON`)
			return nil
		case "--sha":
			if i+1 < len(args) {
				i++
				sha256s = append(sha256s, args[i])
			}
		case "--filename":
			if i+1 < len(args) {
				i++
				filenames = append(filenames, args[i])
			}
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

	req, err := extractVPSBundleListRequestFromManifest(manifest)
	if err != nil {
		return err
	}

	// Resolve --sha to filenames by listing once.
	if len(sha256s) > 0 {
		_, listResp, err := client.VPSList(req)
		if err != nil {
			return err
		}
		bySHA := map[string]string{}
		for _, b := range listResp.Bundles {
			if b.Sha256 != "" {
				bySHA[b.Sha256] = b.Filename
			}
		}
		for _, sha := range sha256s {
			if fn, ok := bySHA[sha]; ok {
				filenames = append(filenames, fn)
			} else {
				return fmt.Errorf("sha256 not found on VPS: %s", sha)
			}
		}
	}

	// --all means list and delete everything.
	for _, a := range args {
		if a == "--all" {
			_, listResp, err := client.VPSList(req)
			if err != nil {
				return err
			}
			for _, b := range listResp.Bundles {
				filenames = append(filenames, b.Filename)
			}
			break
		}
	}

	if len(filenames) == 0 {
		return fmt.Errorf("no bundles selected for deletion (use --filename, --sha, or --all)")
	}

	var lastBody []byte
	var totalFreed int64
	deletedCount := 0
	for _, fn := range filenames {
		body, resp, err := client.VPSDelete(VPSBundleDeleteRequest{
			Host:     req.Host,
			Port:     req.Port,
			User:     req.User,
			KeyPath:  req.KeyPath,
			Workdir:  req.Workdir,
			Filename: fn,
		})
		lastBody = body
		if err != nil {
			return err
		}
		if resp.OK {
			deletedCount++
			totalFreed += resp.FreedBytes
		} else {
			return fmt.Errorf("failed to delete %s: %s", fn, resp.Error)
		}
	}

	if jsonOutput {
		// Return last response body for debug; bulk output isn't well-defined.
		cliutil.PrintJSON(lastBody)
		return nil
	}
	fmt.Printf("Deleted %d bundle(s) from VPS (%s)\n", deletedCount, formatSize(totalFreed))
	return nil
}

func runVPSGC(client *Client, args []string) error {
	fs := flag.NewFlagSet("bundle vps-gc", flag.ContinueOnError)
	selFlags := registerSelectorFlags(fs)
	keep := fs.Int("keep", 2, "Keep N newest bundles per scenario (default: 2)")
	dryRun := fs.Bool("dry-run", false, "Report plan only; do not delete")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	sel, err := selFlags.toSelector()
	if err != nil {
		return err
	}
	if strings.TrimSpace(sel.ScenarioID) == "" {
		return fmt.Errorf("--scenario is required")
	}

	depClient := deploymentcli.NewClient(client.APIClient())
	resolved, err := deploymentcli.ResolveLatestBySelector(depClient, sel)
	if err != nil {
		return err
	}
	if resolved == nil {
		return fmt.Errorf("no deployment found for selector")
	}

	body, resp, err := client.DeploymentVPSGC(resolved.ID, VPSBundleGCRequest{
		ScenarioID: sel.ScenarioID,
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

type selectorFlags struct {
	host       *string
	scenarioID *string
	domain     *string
	target     *string
}

func registerSelectorFlags(fs *flag.FlagSet) selectorFlags {
	return selectorFlags{
		host:       fs.String("host", "", "VPS host selector"),
		scenarioID: fs.String("scenario", "", "Scenario ID selector"),
		domain:     fs.String("domain", "", "Domain selector"),
		target:     fs.String("target", "", "Convenience selector (domain or host)"),
	}
}

func (s selectorFlags) toSelector() (deploymentcli.ManifestSelector, error) {
	host := strings.TrimSpace(*s.host)
	scenarioID := strings.TrimSpace(*s.scenarioID)
	domain := strings.TrimSpace(*s.domain)
	target := strings.TrimSpace(*s.target)

	if target != "" && (host != "" || domain != "") {
		return deploymentcli.ManifestSelector{}, fmt.Errorf("--target cannot be combined with --host or --domain")
	}
	if host == "" && domain == "" && target == "" {
		return deploymentcli.ManifestSelector{}, fmt.Errorf("at least one selector is required: --host, --domain, or --target")
	}

	return deploymentcli.ManifestSelector{
		Host:       host,
		ScenarioID: scenarioID,
		Domain:     domain,
		Target:     target,
	}, nil
}

func extractVPSBundleListRequestFromManifest(m map[string]interface{}) (VPSBundleListRequest, error) {
	// Expected manifest shape:
	// { "target": { "vps": { "host": "...", "port": 22, "user": "root", "workdir": "...", "key_path": "..." } } }
	target, _ := m["target"].(map[string]interface{})
	vps, _ := target["vps"].(map[string]interface{})
	host, _ := vps["host"].(string)
	keyPath, _ := vps["key_path"].(string)
	workdir, _ := vps["workdir"].(string)
	user, _ := vps["user"].(string)
	portF, _ := vps["port"].(float64)

	port := 22
	if portF != 0 {
		port = int(portF)
	}
	if user == "" {
		user = "root"
	}
	if host == "" || keyPath == "" || workdir == "" {
		return VPSBundleListRequest{}, fmt.Errorf("manifest missing required VPS fields (host, key_path, workdir)")
	}

	return VPSBundleListRequest{
		Host:    host,
		Port:    port,
		User:    user,
		KeyPath: keyPath,
		Workdir: workdir,
	}, nil
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
