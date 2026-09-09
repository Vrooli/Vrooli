package installgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Templates use paths relative to the generated scenario. Resolve their lock in
// a temporary scenario-shaped surface, then persist only the declaration and
// lockfile. No template source or runtime dependencies execute during this step.
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
	if r.PackageManager != "pnpm" {
		return "", fmt.Errorf("template lock generation currently requires pnpm")
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
	lock, err := os.ReadFile(filepath.Join(ui, "pnpm-lock.yaml"))
	if err != nil {
		return out, err
	}
	// The lock must refer to the generated scenario's relative package location.
	if strings.Contains(string(lock), stage) {
		return out, fmt.Errorf("template lock contains staging path")
	}
	if err = os.WriteFile(filepath.Join(r.SurfaceRoot, "pnpm-lock.yaml"), lock, 0o644); err != nil {
		return out, err
	}
	if err = os.WriteFile(r.ManifestPath, append(raw, '\n'), 0o644); err != nil {
		return out, err
	}
	return out + "\nGoverned template dependency and lock updated; scenario setup installs it.", nil
}
