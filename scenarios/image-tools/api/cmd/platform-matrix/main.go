// platform-matrix generates an Image Tools per-capability support matrix from
// the current capability list, fleet snapshot and evidence-bearing runner files.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"image-tools/internal/platformmatrix"
)

func main() {
	var capabilitiesPath, targetsPath, fleetPath, receiptsDir, outPath string
	flag.StringVar(&capabilitiesPath, "capabilities", "../../../../.vrooli/research/topics/image-tools-foundation/CAPABILITIES.md", "Image Tools capability list")
	flag.StringVar(&targetsPath, "targets", "../docs/reference/platform-targets.json", "target platform identifiers")
	flag.StringVar(&fleetPath, "fleet", "../docs/reference/fleet-platform-snapshot.json", "normalized current fleet snapshot")
	flag.StringVar(&receiptsDir, "receipts", "../docs/reference/platform-receipts", "runner receipt directory")
	flag.StringVar(&outPath, "out", "../docs/reference/platform-support-matrix.json", "generated JSON matrix output")
	flag.Parse()
	if err := run(capabilitiesPath, targetsPath, fleetPath, receiptsDir, outPath); err != nil {
		fmt.Fprintln(os.Stderr, "platform-matrix:", err)
		os.Exit(1)
	}
}

func run(capabilitiesPath, targetsPath, fleetPath, receiptsDir, outPath string) error {
	capabilityData, err := os.ReadFile(capabilitiesPath)
	if err != nil {
		return fmt.Errorf("read capability list: %w", err)
	}
	capabilities, err := platformmatrix.ParseCapabilities(string(capabilityData))
	if err != nil {
		return err
	}
	var targets []platformmatrix.Target
	if err := readJSON(targetsPath, &targets); err != nil {
		return fmt.Errorf("read platform targets: %w", err)
	}
	var fleet platformmatrix.FleetSnapshot
	if err := readJSON(fleetPath, &fleet); err != nil {
		return fmt.Errorf("read fleet snapshot: %w", err)
	}
	receipts, err := readReceipts(receiptsDir)
	if err != nil {
		return err
	}
	matrix, err := platformmatrix.Build(targets, fleet, receipts, capabilities, filepath.ToSlash(capabilitiesPath), time.Now().UTC())
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(matrix, "", "  ")
	if err != nil {
		return fmt.Errorf("encode matrix: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(outPath, append(encoded, '\n'), 0o644); err != nil {
		return fmt.Errorf("write matrix: %w", err)
	}
	fmt.Printf("wrote %d platform rows × %d capabilities to %s\n", len(matrix.Platforms), len(capabilities), outPath)
	return nil
}

func readJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func readReceipts(dir string) ([]platformmatrix.Receipt, error) {
	var receipts []platformmatrix.Receipt
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			return nil
		}
		var receipt platformmatrix.Receipt
		if err := readJSON(path, &receipt); err != nil {
			return fmt.Errorf("read receipt %s: %w", path, err)
		}
		receipts = append(receipts, receipt)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read receipt directory: %w", err)
	}
	return receipts, nil
}
