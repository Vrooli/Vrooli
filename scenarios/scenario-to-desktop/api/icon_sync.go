package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// syncScenarioIcons copies the best-available scenario icon into the Electron assets directory.
// It prefers UI build artifacts (manifest icons) so desktop installers match the app brand.
func syncScenarioIcons(scenario, outputDir string, logf func(string, map[string]interface{})) error {
	root := detectVrooliRoot()
	type candidate struct {
		Path string
		Size int64
	}
	paths := []string{
		// Prefer the generated renderer assets (already copied into the Electron project)
		filepath.Join(outputDir, "renderer", "manifest-icon-512.maskable.png"),
		filepath.Join(outputDir, "renderer", "apple-icon-180.png"),
		filepath.Join(outputDir, "renderer", "favicon-196.png"),
		// Fall back to UI dist artifacts if renderer copies are missing
		filepath.Join(root, "scenarios", scenario, "ui", "dist", "manifest-icon-512.maskable.png"),
		filepath.Join(root, "scenarios", scenario, "ui", "dist", "apple-icon-180.png"),
		filepath.Join(root, "scenarios", scenario, "ui", "dist", "favicon-196.png"),
	}

	var best *candidate
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			if best == nil || info.Size() > best.Size {
				best = &candidate{Path: path, Size: info.Size()}
			}
		}
	}
	if best == nil {
		return nil // nothing to do
	}

	data, err := os.ReadFile(best.Path)
	if err != nil {
		return fmt.Errorf("read icon: %w", err)
	}

	assetsDir := filepath.Join(outputDir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return fmt.Errorf("create assets dir: %w", err)
	}

	targets := []string{
		"icon.png",
		"icon-1024x1024.png",
		"icon-512x512.png",
		"icon-256x256.png",
		"icon-128x128.png",
		"icon-64x64.png",
		"icon-48x48.png",
		"icon-32x32.png",
		"icon-16x16.png",
	}

	for _, name := range targets {
		if err := os.WriteFile(filepath.Join(assetsDir, name), data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	if logf != nil {
		logf("info", map[string]interface{}{
			"msg":       "synced scenario icon into desktop assets",
			"scenario":  scenario,
			"source":    best.Path,
			"size":      best.Size,
			"assetsDir": assetsDir,
		})
	}
	return nil
}
