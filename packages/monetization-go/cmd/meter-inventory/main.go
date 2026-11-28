package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/packages/monetization-go"
)

func main() {
	root, err := repositoryRoot()
	if err != nil {
		fatal(err)
	}
	inventory, err := monetization.BuildMeterInventory(root)
	if err != nil {
		fatal(err)
	}
	data, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		fatal(fmt.Errorf("encode meter inventory: %w", err))
	}
	data = append(data, '\n')
	path := filepath.Join(root, "packages", "monetization-go", "meter-inventory.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fatal(fmt.Errorf("write meter inventory: %w", err))
	}
}

func repositoryRoot() (string, error) {
	path, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(path, "scenarios")); err == nil {
			if _, err := os.Stat(filepath.Join(path, "packages", "monetization-go")); err == nil {
				return path, nil
			}
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf("repository root not found above working directory")
		}
		path = parent
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
