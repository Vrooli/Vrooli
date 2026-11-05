package generation

import (
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const wailsAPIVersion = "v3.0.0-beta.27"

func (s *DefaultService) runWailsProjectGenerator(buildID string, config *DesktopConfig) {
	fail := func(err error) {
		s.updateBuildStatus(buildID, func(status *BuildStatus) {
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, err.Error())
			status.CompletedAt = ptrTime(time.Now())
		})
	}
	scenarioRoot := s.ScenarioRoot(config.AppName)
	apiDir := filepath.Join(scenarioRoot, "api")
	apiModule, err := readGoModulePath(filepath.Join(apiDir, "go.mod"))
	if err != nil {
		fail(fmt.Errorf("Wails strategy requires an API Go module: %w", err))
		return
	}
	if _, err := os.Stat(filepath.Join(apiDir, "desktop", "embed.go")); err != nil {
		fail(fmt.Errorf("Wails strategy requires api/desktop/embed.go: %w", err))
		return
	}
	originalTemplateHash, err := templateTreeHash(s.templateDir)
	if err != nil {
		fail(fmt.Errorf("hash Wails template tree: %w", err))
		return
	}
	snapshotDir := filepath.Join(os.TempDir(), "desktop-wails-template-"+buildID)
	if err := os.RemoveAll(snapshotDir); err != nil {
		fail(fmt.Errorf("clear Wails template snapshot: %w", err))
		return
	}
	defer os.RemoveAll(snapshotDir)
	if err := copyTemplateTree(s.templateDir, snapshotDir); err != nil {
		fail(fmt.Errorf("snapshot Wails template tree: %w", err))
		return
	}
	snapshotHash, err := templateTreeHash(snapshotDir)
	if err != nil {
		fail(fmt.Errorf("hash Wails template snapshot: %w", err))
		return
	}

	outputPath, err := filepath.Abs(config.OutputPath)
	if err != nil {
		fail(fmt.Errorf("resolve Wails output path: %w", err))
		return
	}
	canonicalOutput, err := filepath.Abs(filepath.Join(scenarioRoot, "platforms", WailsStrategyID))
	if err != nil || filepath.Clean(outputPath) != filepath.Clean(canonicalOutput) {
		fail(fmt.Errorf("Wails generation requires the canonical platforms/%s dependency surface", WailsStrategyID))
		return
	}
	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		fail(fmt.Errorf("create Wails output path: %w", err))
		return
	}
	apiRelative, err := filepath.Rel(outputPath, apiDir)
	if err != nil {
		fail(fmt.Errorf("API module path cannot be represented safely relative to Wails project"))
		return
	}
	if filepath.Clean(filepath.Join(outputPath, apiRelative)) != filepath.Clean(apiDir) {
		fail(fmt.Errorf("resolved Wails API replacement does not point to the selected scenario API"))
		return
	}
	apiRelative = filepath.ToSlash(apiRelative)

	templatePath := filepath.Join(snapshotDir, "strategies", WailsStrategyID, "main.go.tmpl")
	mainTemplate, err := os.ReadFile(templatePath)
	if err != nil {
		fail(fmt.Errorf("read Wails project template: %w", err))
		return
	}
	mainSource := strings.NewReplacer(
		"{{API_MODULE}}", apiModule,
		"{{APP_DISPLAY_NAME}}", strconv.Quote(config.AppDisplayName),
		"{{APP_DESCRIPTION}}", strconv.Quote(config.AppDescription),
	).Replace(string(mainTemplate))
	for _, token := range []string{"{{API_MODULE}}", "{{APP_DISPLAY_NAME}}", "{{APP_DESCRIPTION}}"} {
		if strings.Contains(mainSource, token) {
			fail(fmt.Errorf("unresolved token %s in Wails main template", token))
			return
		}
	}
	formattedMain, err := format.Source([]byte(mainSource))
	if err != nil {
		fail(fmt.Errorf("format generated Wails main.go: %w", err))
		return
	}
	if err := os.WriteFile(filepath.Join(outputPath, "main.go"), formattedMain, 0o644); err != nil {
		fail(fmt.Errorf("write Wails main.go: %w", err))
		return
	}

	modulePath := "vrooli.local/desktop/" + config.AppName
	goMod := fmt.Sprintf("module %s\n\ngo 1.25.0\n\nrequire %s v0.0.0\n\nreplace %s => %s\n", modulePath, apiModule, apiModule, apiRelative)
	if err := os.WriteFile(filepath.Join(outputPath, "go.mod"), []byte(goMod), 0o644); err != nil {
		fail(fmt.Errorf("write Wails go.mod: %w", err))
		return
	}
	marker := fmt.Sprintf("{\"strategy_id\":%q,\"wails_version\":%q,\"api_module\":%q,\"binary_name\":%q}\n", WailsStrategyID, wailsAPIVersion, apiModule, config.AppName)
	if err := os.WriteFile(filepath.Join(outputPath, "wails-strategy.json"), []byte(marker), 0o644); err != nil {
		fail(fmt.Errorf("write Wails strategy marker: %w", err))
		return
	}
	buildDir := filepath.Join(outputPath, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		fail(fmt.Errorf("create Wails build configuration directory: %w", err))
		return
	}
	if err := renderWailsConfig(snapshotDir, outputPath, config); err != nil {
		fail(err)
		return
	}
	assetsDir := filepath.Join(outputPath, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		fail(fmt.Errorf("create Wails assets directory: %w", err))
		return
	}
	assetSource := config.ScenarioPath
	if assetSource != "" && !filepath.IsAbs(assetSource) {
		assetSource = filepath.Join(scenarioRoot, assetSource)
	}
	if err := copyWailsAssets(assetSource, assetsDir); err != nil {
		fail(err)
		return
	}
	if currentHash, hashErr := templateTreeHash(s.templateDir); hashErr != nil {
		fail(fmt.Errorf("verify Wails template tree after generation: %w", hashErr))
		return
	} else if currentHash != originalTemplateHash {
		fail(fmt.Errorf("Wails template tree changed during generation (snapshot %s; current %s)", snapshotHash, currentHash))
		return
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "index.html")); os.IsNotExist(err) {
		if err := os.WriteFile(filepath.Join(assetsDir, "index.html"), []byte("<!doctype html><html><head><meta charset=\"utf-8\"><title>"+config.AppDisplayName+"</title></head><body><main id=\"app\"></main></body></html>\n"), 0o644); err != nil {
			fail(fmt.Errorf("write fallback Wails index: %w", err))
			return
		}
	}
	if err := installWailsLaunchObserver(snapshotDir, assetsDir); err != nil {
		fail(err)
		return
	}

	s.updateBuildStatus(buildID, func(status *BuildStatus) {
		status.Status = "ready"
		status.Artifacts["output_path"] = outputPath
		status.Artifacts["strategy_id"] = WailsStrategyID
		status.Artifacts["api_module"] = apiModule
		status.Artifacts["template_snapshot_hash"] = snapshotHash
		status.BuildLog = append(status.BuildLog, "Generated Wails in-process project; Wails dependency installation is delegated to Scenario Dependency Analyzer.")
		status.CompletedAt = ptrTime(time.Now())
	})
}

func installWailsLaunchObserver(templateRoot, assetsDir string) error {
	contents, err := os.ReadFile(filepath.Join(templateRoot, "strategies", WailsStrategyID, "launch-observer.js.tmpl"))
	if err != nil {
		return fmt.Errorf("read Wails launch observer: %w", err)
	}
	indexPath := filepath.Join(assetsDir, "index.html")
	index, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("read Wails index for launch observer: %w", err)
	}
	const script = `<script src="/__s2d/launch-observer.js"></script>`
	observerDir := filepath.Join(assetsDir, "__s2d")
	observerPath := filepath.Join(observerDir, "launch-observer.js")
	if strings.Contains(string(index), script) {
		if info, statErr := os.Stat(observerPath); statErr == nil && info.Mode().IsRegular() {
			return nil
		}
		return fmt.Errorf("Wails index references a missing or invalid launch observer asset")
	}
	if err := os.MkdirAll(observerDir, 0o755); err != nil {
		return fmt.Errorf("create Wails launch observer asset directory: %w", err)
	}
	if _, err := os.Lstat(observerPath); err == nil {
		return fmt.Errorf("Wails UI assets already use reserved launch observer path %s", observerPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect Wails launch observer path: %w", err)
	}
	if err := os.WriteFile(observerPath, contents, 0o644); err != nil {
		return fmt.Errorf("write Wails launch observer: %w", err)
	}
	updated := string(index)
	if head := strings.Index(strings.ToLower(updated), "</head>"); head >= 0 {
		updated = updated[:head] + script + updated[head:]
	} else if html := strings.Index(strings.ToLower(updated), ">") + 1; html > 0 {
		updated = updated[:html] + script + updated[html:]
	} else {
		updated = script + updated
	}
	if err := os.WriteFile(indexPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("install Wails launch observer in index: %w", err)
	}
	return nil
}

func renderWailsConfig(templateRoot, outputPath string, config *DesktopConfig) error {
	replacements := map[string]string{
		"{{APP_NAME}}":         config.AppName,
		"{{AUTHOR}}":           strconv.Quote(config.Author),
		"{{APP_DISPLAY_NAME}}": strconv.Quote(config.AppDisplayName),
		"{{APP_ID}}":           strconv.Quote(config.AppID),
		"{{APP_DESCRIPTION}}":  strconv.Quote(config.AppDescription),
		"{{COPYRIGHT}}":        strconv.Quote(fmt.Sprintf("(c) %d %s", time.Now().Year(), config.Author)),
		"{{VERSION}}":          strconv.Quote(config.Version),
	}
	for sourceName, destination := range map[string]string{
		"Taskfile.yml.tmpl": filepath.Join(outputPath, "Taskfile.yml"),
		"config.yml.tmpl":   filepath.Join(outputPath, "build", "config.yml"),
	} {
		contents, err := os.ReadFile(filepath.Join(templateRoot, "strategies", WailsStrategyID, sourceName))
		if err != nil {
			return fmt.Errorf("read Wails configuration template %s: %w", sourceName, err)
		}
		for token, value := range replacements {
			contents = []byte(strings.ReplaceAll(string(contents), token, value))
		}
		if err := os.WriteFile(destination, contents, 0o644); err != nil {
			return fmt.Errorf("write Wails configuration %s: %w", destination, err)
		}
	}
	return nil
}

func readGoModulePath(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("no module directive in %s", path)
}

func copyWailsAssets(source, destination string) error {
	if strings.TrimSpace(source) == "" {
		return nil
	}
	info, err := os.Stat(source)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect Wails UI assets: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("Wails UI assets path is not a directory: %s", source)
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("Wails UI assets cannot contain symlinks: %s", path)
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
