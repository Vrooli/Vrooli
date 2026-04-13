package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/config"
)

type (
	scenarioTemplateVar      = scenariocli.TemplateVar
	scenarioTemplateHook     = scenariocli.TemplateHook
	scenarioTemplateManifest = scenariocli.TemplateManifest
	scenarioTemplateInfo     = scenariocli.TemplateInfo
	scenarioGenerateOptions  = scenariocli.GenerateOptions
)

var unresolvedTemplatePattern = regexp.MustCompile(`\{\{[A-Z0-9_]+\}\}`)

func runScenarioTemplateCommandWithApp(app *App, ctx *commandContext, args []string) error {
	if len(args) == 0 {
		return bindContextCommand(
			func(ctx *commandContext, args []string) (scenariocli.TemplateListRequest, error) {
				return scenariocli.ParseTemplateListRequest(args)
			},
			runScenarioTemplateListRequest,
			scenariocli.RenderTemplateListResponse,
		)(app, ctx, nil)
	}
	switch commandtree.NormalizeName(args[0]) {
	case "list":
		return bindContextCommand(
			func(ctx *commandContext, args []string) (scenariocli.TemplateListRequest, error) {
				return scenariocli.ParseTemplateListRequest(args)
			},
			runScenarioTemplateListRequest,
			scenariocli.RenderTemplateListResponse,
		)(app, ctx, args[1:])
	case "show":
		return bindContextCommand(
			func(ctx *commandContext, args []string) (scenariocli.TemplateShowRequest, error) {
				return scenariocli.ParseTemplateShowRequest(args)
			},
			runScenarioTemplateShowRequest,
			scenariocli.RenderTemplateShowResponse,
		)(app, ctx, args[1:])
	case "--help", "-h":
		showScenarioTemplateHelp(ctx.Stdout)
		return nil
	default:
		return usageErrorf("scenario template", "unknown scenario template command: %s", args[0])
	}
}

func runScenarioGenerateCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return bindContextCommand(
		func(ctx *commandContext, args []string) (scenariocli.GenerateRequest, error) {
			return scenariocli.ParseGenerateRequest(
				args,
				ctx.Stderr,
				func(name string) (scenariocli.TemplateInfo, error) { return loadScenarioTemplate(ctx.Root, name) },
				scenariocli.ParseGenerateArgs,
			)
		},
		runScenarioGenerateRequest,
		scenariocli.RenderGenerateResponse,
	)(app, ctx, args)
}

func showScenarioTemplateHelp(w io.Writer) {
	scenariocli.RenderTemplateHelp(w)
}

func showScenarioGenerateHelp(w io.Writer) {
	scenariocli.RenderGenerateHelp(w)
}

func loadScenarioTemplates(root string) ([]scenarioTemplateInfo, error) {
	baseDir := config.TemplateBaseDir(root)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	templates := make([]scenarioTemplateInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		info, err := loadScenarioTemplate(root, name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				templates = append(templates, scenarioTemplateInfo{
					Name:    name,
					Path:    filepath.Join(baseDir, name),
					Missing: true,
				})
				continue
			}
			return nil, err
		}
		templates = append(templates, info)
	}
	sort.Slice(templates, func(i, j int) bool { return templates[i].Name < templates[j].Name })
	return templates, nil
}

func loadScenarioTemplate(root, name string) (scenarioTemplateInfo, error) {
	baseDir := config.TemplateBaseDir(root)
	templateDir := filepath.Join(baseDir, name)
	info, err := os.Stat(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return scenarioTemplateInfo{}, fmt.Errorf("template not found: %s", name)
		}
		return scenarioTemplateInfo{}, err
	}
	if !info.IsDir() {
		return scenarioTemplateInfo{}, fmt.Errorf("template path is not a directory: %s", templateDir)
	}

	data, err := os.ReadFile(filepath.Join(templateDir, "template.json"))
	if err != nil {
		return scenarioTemplateInfo{}, err
	}
	var manifest scenarioTemplateManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return scenarioTemplateInfo{}, err
	}
	if manifest.Name == "" {
		manifest.Name = name
	}
	if manifest.RequiredVars == nil {
		manifest.RequiredVars = map[string]scenarioTemplateVar{}
	}
	if manifest.OptionalVars == nil {
		manifest.OptionalVars = map[string]scenarioTemplateVar{}
	}
	if manifest.Docs == nil {
		manifest.Docs = map[string]string{}
	}

	return scenarioTemplateInfo{
		Name:     name,
		Path:     templateDir,
		Manifest: manifest,
	}, nil
}

func copyScenarioTemplate(templateDir, destination string, values map[string]string) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}

	return filepath.WalkDir(templateDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == templateDir {
			return nil
		}

		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		if filepath.Base(path) == ".DS_Store" {
			return nil
		}
		if relPath == "template.json" {
			return nil
		}

		renderedRel := renderScenarioTemplateString(relPath, values)
		targetPath := filepath.Join(destination, renderedRel)

		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if looksLikeTextFile(data) {
			data = []byte(renderScenarioTemplateString(string(data), values))
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}

func verifyScenarioTemplate(destination string) error {
	var unresolved []string
	err := filepath.WalkDir(destination, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if unresolvedTemplatePattern.MatchString(path) {
			unresolved = append(unresolved, path)
			return nil
		}
		if entry.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !looksLikeTextFile(data) {
			return nil
		}
		if unresolvedTemplatePattern.Match(data) {
			unresolved = append(unresolved, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(unresolved) == 0 {
		return nil
	}
	sort.Strings(unresolved)
	return fmt.Errorf("unresolved placeholders remain in: %s", strings.Join(unresolved, ", "))
}

func looksLikeTextFile(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return false
	}
	return utf8.Valid(data)
}

func renderScenarioTemplateString(value string, values map[string]string) string {
	if value == "" {
		return value
	}
	rendered := value
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", values[key])
	}
	return rendered
}

func writeScenarioTemplateValues(w io.Writer, values map[string]string) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	_, _ = fmt.Fprintln(w, "Applied variables:")
	for _, key := range keys {
		_, _ = fmt.Fprintf(w, "  - %s=%s\n", key, values[key])
	}
}

func writeScenarioTemplateNextSteps(w io.Writer, destination string, manifest scenarioTemplateManifest) {
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Next steps:")
	_, _ = fmt.Fprintf(w, "  1. Draft PRD.md -> %s\n", filepath.Join(destination, "PRD.md"))
	_, _ = fmt.Fprintf(w, "  2. Seed requirements -> %s\n", filepath.Join(destination, "requirements", "index.json"))
	_, _ = fmt.Fprintf(w, "  3. Update progress log -> %s\n", filepath.Join(destination, "docs", "PROGRESS.md"))
	_, _ = fmt.Fprintf(w, "  4. Run: vrooli scenario status %s\n", filepath.Base(destination))

	if len(manifest.Docs) == 0 {
		return
	}
	docKeys := make([]string, 0, len(manifest.Docs))
	for key := range manifest.Docs {
		docKeys = append(docKeys, key)
	}
	sort.Strings(docKeys)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Reference docs:")
	for _, key := range docKeys {
		_, _ = fmt.Fprintf(w, "  - %s: %s\n", key, manifest.Docs[key])
	}
}

func writeScenarioTemplateHooks(w io.Writer, manifest scenarioTemplateManifest) {
	if len(manifest.PostHooks) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Post hooks (run manually if needed):")
	for _, hook := range manifest.PostHooks {
		line := hook.Description
		if line == "" {
			line = hook.Cmd
		}
		_, _ = fmt.Fprintf(w, "  - %s\n", line)
	}
}

func runScenarioTemplateHooksWithApp(app *App, root string, globals globalOptions, destination string, manifest scenarioTemplateManifest, stdout, stderr io.Writer) error {
	if len(manifest.PostHooks) == 0 {
		_, _ = fmt.Fprintln(stdout, "No post hooks defined for this template")
		return nil
	}

	for index, hook := range manifest.PostHooks {
		description := strings.TrimSpace(hook.Description)
		if description == "" {
			description = hook.Cmd
		}
		_, _ = fmt.Fprintf(stdout, "[Hook %d] %s\n", index+1, description)
		cwd := destination
		if strings.TrimSpace(hook.Cwd) != "" && hook.Cwd != "." {
			cwd = filepath.Join(destination, filepath.FromSlash(hook.Cwd))
		}
		if err := app.runScenarioSubprocess(scenarioSubprocessSpec{
			name:   "bash",
			args:   []string{"-lc", hook.Cmd},
			dir:    cwd,
			env:    app.commandEnv(root, globals),
			stdout: stdout,
			stderr: stderr,
		}); err != nil {
			return err
		}
	}
	return nil
}

func writeScenarioTemplateVarTable(w io.Writer, title string, vars map[string]scenarioTemplateVar) {
	if len(vars) == 0 {
		return
	}
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "%s:\n", title)
	for _, key := range keys {
		item := vars[key]
		flag := ""
		if item.Flag != "" {
			flag = " (--" + item.Flag + ")"
		}
		line := fmt.Sprintf("  - %s%s", key, flag)
		if item.Description != "" {
			line += ": " + item.Description
		}
		if item.Default != "" {
			line += " [default: " + item.Default + "]"
		}
		_, _ = fmt.Fprintln(w, line)
	}
}

func formatScenarioTemplateRequiredVars(manifest scenarioTemplateManifest) string {
	keys := make([]string, 0, len(manifest.RequiredVars))
	for key := range manifest.RequiredVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "-"
	}

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		label := key
		if flag := manifest.RequiredVars[key].Flag; flag != "" {
			label += " (--" + flag + ")"
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, ", ")
}

func formatScenarioTemplateRequiredFlags(manifest scenarioTemplateManifest) string {
	keys := make([]string, 0, len(manifest.RequiredVars))
	for key := range manifest.RequiredVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return " --id <slug>"
	}

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		flag := manifest.RequiredVars[key].Flag
		if flag == "" {
			flag = strings.ToLower(key)
		}
		parts = append(parts, fmt.Sprintf(" --%s <%s>", flag, strings.ToLower(key)))
	}
	return strings.Join(parts, "")
}

func currentDateUTC() string {
	return time.Now().UTC().Format("2006-01-02")
}

func randomTemplateToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func copyStringMap(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
