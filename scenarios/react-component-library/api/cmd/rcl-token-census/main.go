package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"react-component-library/internal/gates"
)

func main() {
	jsonOutput := flag.Bool("json", false, "emit the complete census as JSON")
	outputPath := flag.String("output", "", "write the complete JSON census to this path")
	rootFlag := flag.String("root", "", "repository root (auto-detected when omitted)")
	flag.Parse()

	root := *rootFlag
	if root == "" {
		var err error
		root, err = findRepositoryRoot()
		if err != nil {
			fail(err)
		}
	}
	census, err := gates.TokenCensus(root)
	if err != nil {
		fail(err)
	}
	if *jsonOutput || *outputPath != "" {
		writer := os.Stdout
		if *outputPath != "" {
			path := *outputPath
			if !filepath.IsAbs(path) {
				path = filepath.Join(root, path)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				fail(err)
			}
			writer, err = os.Create(path)
			if err != nil {
				fail(err)
			}
			defer writer.Close()
		}
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(census); err != nil {
			fail(err)
		}
		return
	}

	fmt.Printf("Components scanned: %d\n", census.ComponentsScanned)
	fmt.Printf("Distinct external references: %d\n", len(census.ReferencedProperties))
	fmt.Printf("Required after BaseStyles/--rcl-* exclusions: %d\n", len(census.RequiredProperties))
	kits := make([]string, 0, len(census.KitPublishedProperties))
	for kit := range census.KitPublishedProperties {
		kits = append(kits, kit)
	}
	sort.Strings(kits)
	for _, kit := range kits {
		properties := census.KitPublishedProperties[kit]
		fmt.Printf("Kit %-28s %d properties\n", kit+":", len(properties))
	}
	fmt.Printf("Undefined properties: %d\n", len(census.UndefinedProperties))
	for _, verdict := range []string{"universal", "restricted", "unsatisfiable", "undefined-vocabulary"} {
		fmt.Printf("Verdict %-22s %d\n", verdict+":", census.VerdictCounts[verdict])
	}
	fmt.Printf("Affinity overclaims: %d\n", len(census.AffinityOverclaims))
	fmt.Printf("Scenario token files: %d\n", len(census.ScenarioRamps))
}

func findRepositoryRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		marker := filepath.Join(current, "templates", "design", "vrooli-default", "adapters", "react-vite-tailwind", "tokens.css")
		if _, err := os.Stat(marker); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("repository root not found from %s", current)
		}
		current = parent
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "rcl-token-census:", err)
	os.Exit(1)
}
