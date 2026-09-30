package cliutil

import (
	"fmt"
	"os"
)

const (
	installedManifestSuffix      = ".manifest.json"
	installedBuildMetadataSuffix = ".build.meta"
)

// InstalledManifestPath returns the canonical sibling manifest path for an installed CLI binary.
func InstalledManifestPath(binaryPath string) string {
	return binaryPath + installedManifestSuffix
}

// InstalledBuildMetadataPath returns the canonical sibling build metadata path for an installed CLI binary.
func InstalledBuildMetadataPath(binaryPath string) string {
	return binaryPath + installedBuildMetadataSuffix
}

// LoadRuntimeManifest reads the installed sibling manifest first and only falls
// back to an explicit source manifest path when one is configured.
func LoadRuntimeManifest(installedPath, sourcePath string) ([]byte, error) {
	if installedPath != "" {
		data, err := os.ReadFile(installedPath)
		if err == nil {
			return data, nil
		}
	}
	if sourcePath != "" {
		data, err := os.ReadFile(sourcePath)
		if err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("read runtime manifest: installed=%q source=%q", installedPath, sourcePath)
}
