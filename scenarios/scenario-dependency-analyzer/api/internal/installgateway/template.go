package installgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Templates use paths relative to the generated scenario. Resolve their lock in
// a temporary scenario-shaped surface, then persist the declaration and
// lockfile. Exact Electron tooling pins also update the generated app manifest.
func installTemplate(ctx context.Context, r Resolution, installer PackageInstaller) (string, error) {
	parent := filepath.Join(r.RepositoryRoot, "scenarios")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(parent, ".template-deps-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	ui := filepath.Join(stage, "ui")
	if err = os.MkdirAll(ui, 0o755); err != nil {
		return "", err
	}
	raw, err := os.ReadFile(r.ManifestPath)
	if err != nil {
		return "", err
	}
	var original map[string]any
	if err = json.Unmarshal(raw, &original); err != nil {
		return "", err
	}
	originalName := original["name"]
	original["name"] = "template-dependency-staging"
	raw, err = json.MarshalIndent(original, "", "  ")
	if err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(ui, "package.json"), raw, 0o644); err != nil {
		return "", err
	}
	for _, name := range []string{"pnpm-lock.yaml", "package-lock.json", ".npmrc", "pnpm-workspace.yaml"} {
		data, e := os.ReadFile(filepath.Join(r.SurfaceRoot, name))
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return "", e
		}
		if e = os.WriteFile(filepath.Join(ui, name), data, 0o644); e != nil {
			return "", e
		}
	}
	lockfile := ""
	switch r.PackageManager {
	case "pnpm":
		lockfile = "pnpm-lock.yaml"
	case "npm":
		lockfile = "package-lock.json"
	default:
		return "", fmt.Errorf("template lock generation does not support package manager %q", r.PackageManager)
	}
	staged := r
	staged.TemplateVersion = ""
	staged.SurfaceRoot = ui
	staged.ManifestPath = filepath.Join(ui, "package.json")
	staged.Argv = append(append([]string{}, r.Argv...), "--lockfile-only")
	out, err := installer.Install(ctx, staged)
	if err != nil {
		return out, err
	}
	raw, err = os.ReadFile(staged.ManifestPath)
	if err != nil {
		return out, err
	}
	var updated map[string]any
	if err = json.Unmarshal(raw, &updated); err != nil {
		return out, err
	}
	if originalName == nil {
		delete(updated, "name")
	} else {
		updated["name"] = originalName
	}
	raw, err = json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return out, err
	}
	lock, err := os.ReadFile(filepath.Join(ui, lockfile))
	if err != nil {
		return out, err
	}
	// The lock must refer to the generated scenario's relative package location.
	if strings.Contains(string(lock), stage) {
		return out, fmt.Errorf("template lock contains staging path")
	}
	if err = os.WriteFile(filepath.Join(r.SurfaceRoot, lockfile), lock, 0o644); err != nil {
		return out, err
	}
	if err = os.WriteFile(r.ManifestPath, append(raw, '\n'), 0o644); err != nil {
		return out, err
	}
	if r.GeneratedManifestPath != "" && r.ExactVersion {
		if err = setGeneratedTemplateVersion(r.GeneratedManifestPath, r.PackageName, r.TemplateVersion); err != nil {
			return out, err
		}
	}
	return out + "\nGoverned template dependency and lock updated; scenario setup installs it.", nil
}

// setGeneratedTemplateVersion keeps the Electron app emitted from the canonical
// desktop template on the same exact package version as the governed template
// dependency. It is deliberately limited to the two matched build packages.
func setGeneratedTemplateVersion(path, packageName, version string) error {
	if packageName != "electron" && packageName != "electron-mksnapshot" {
		return fmt.Errorf("generated desktop manifest pin is unsupported for %q", packageName)
	}
	if !exactNpmVersion.MatchString(version) {
		return fmt.Errorf("generated desktop manifest requires a concrete semantic version")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read generated desktop package template: %w", err)
	}
	section := regexp.MustCompile(`(?s)("devDependencies"\s*:\s*)\{([^{}]*)\}`)
	loc := section.FindSubmatchIndex(data)
	if loc == nil {
		return fmt.Errorf("generated desktop package template has no devDependencies object")
	}
	var dependencies map[string]string
	if err := json.Unmarshal(data[loc[3]:loc[1]], &dependencies); err != nil {
		return fmt.Errorf("parse generated desktop devDependencies: %w", err)
	}
	dependencies[packageName] = version
	lineStart := bytes.LastIndex(data[:loc[0]], []byte("\n")) + 1
	indent := string(data[lineStart:loc[0]])
	if strings.TrimSpace(indent) != "" {
		indent = ""
	}
	encoded, err := json.MarshalIndent(dependencies, indent, "  ")
	if err != nil {
		return fmt.Errorf("encode generated desktop devDependencies: %w", err)
	}
	updated := append([]byte(nil), data[:loc[3]]...)
	updated = append(updated, encoded...)
	updated = append(updated, data[loc[1]:]...)
	if err := os.WriteFile(path, updated, 0o644); err != nil {
		return fmt.Errorf("write generated desktop package template: %w", err)
	}
	return nil
}
