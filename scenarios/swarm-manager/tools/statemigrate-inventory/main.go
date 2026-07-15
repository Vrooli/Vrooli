// Command statemigrate-inventory produces a deterministic, read-only inventory
// of all swarm-manager persisted state ahead of the Phase 8 declarative-operations
// state migration. It never mutates runtime state.
//
// Output: a byte-stable JSON payload (inventory.json) plus a human-readable
// Markdown summary. Two back-to-back runs over unchanged state produce
// byte-identical output — the reconciliation anchor for pre/post-migration diffs.
//
// This tool is TEMPORARY. It is scheduled for deletion in Phase 9 per the
// storage-steer one-shot migration policy. See README.md.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	var (
		dataRoot   = flag.String("data-root", "", "override data class root (default: resolved live root)")
		stateRoot  = flag.String("state-root", "", "override state class root")
		cacheRoot  = flag.String("cache-root", "", "override cache class root")
		configFile = flag.String("config-file", "", "path to scenarios/swarm-manager/config/settings.json (optional)")
		outDir     = flag.String("out-dir", "", "directory to write inventory.json + summary (default: stdout JSON only)")
		jsonName   = flag.String("json-name", "inventory-phase1.json", "JSON output filename")
		mdName     = flag.String("summary-name", "inventory-phase1-summary.md", "Markdown summary filename")
	)
	flag.Parse()

	cfg := resolveRoots(*dataRoot, *stateRoot, *cacheRoot, *configFile)
	inv := Scan(cfg)

	payload, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "encode inventory:", err)
		os.Exit(1)
	}
	payload = append(payload, '\n')
	summary := renderSummary(inv)

	if *outDir == "" {
		os.Stdout.Write(payload)
		return
	}
	if err := os.MkdirAll(*outDir, 0o750); err != nil {
		fmt.Fprintln(os.Stderr, "create out dir:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*outDir, *jsonName), payload, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write json:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*outDir, *mdName), []byte(summary), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write summary:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %s and %s (files=%d anomalies=%d findings=%d content_hash=%s)\n",
		*jsonName, *mdName, inv.Totals.FilesScanned, inv.Totals.AnomalyCount, inv.Totals.FindingCount, inv.Totals.ContentHash)
}

// resolveRoots mirrors internal/runtimepaths resolution: VROOLI_STORAGE_ROOT (or
// ~/.vrooli) → <root>/<class>/vrooli/swarm-manager. Explicit flags win.
func resolveRoots(data, state, cache, configFile string) Config {
	const app, scenario = "vrooli", "swarm-manager"
	from := "default(~/.vrooli)"
	classRoot := func(class string) string {
		if root := os.Getenv("VROOLI_STORAGE_ROOT"); root != "" {
			from = "env(VROOLI_STORAGE_ROOT)"
			return filepath.Join(root, class, app, scenario)
		}
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return ""
		}
		return filepath.Join(home, ".vrooli", class, app, scenario)
	}
	if data == "" {
		data = classRoot("data")
	}
	if state == "" {
		state = classRoot("state")
	}
	if cache == "" {
		cache = classRoot("cache")
	}
	if configFile == "" {
		if sr := os.Getenv("SCENARIO_ROOT"); sr != "" {
			configFile = filepath.Join(sr, "config", "settings.json")
		}
	}
	return Config{DataRoot: data, StateRoot: state, CacheRoot: cache, ConfigFile: configFile, ResolvedFrom: from}
}
