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

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/config"
)

type scenarioTemplateVar struct {
	Flag        string `json:"flag,omitempty"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type scenarioTemplateHook struct {
	Description string `json:"description,omitempty"`
	Cmd         string `json:"cmd,omitempty"`
	Cwd         string `json:"cwd,omitempty"`
}

type scenarioTemplateManifest struct {
	Name         string                         `json:"name,omitempty"`
	DisplayName  string                         `json:"displayName,omitempty"`
	Description  string                         `json:"description,omitempty"`
	Stack        []string                       `json:"stack,omitempty"`
	RequiredVars map[string]scenarioTemplateVar `json:"requiredVars,omitempty"`
	OptionalVars map[string]scenarioTemplateVar `json:"optionalVars,omitempty"`
	Docs         map[string]string              `json:"docs,omitempty"`
	PostHooks    []scenarioTemplateHook         `json:"postHooks,omitempty"`
}

type scenarioTemplateInfo struct {
	Name     string
	Path     string
	Manifest scenarioTemplateManifest
	Missing  bool
}

type scenarioGenerateOptions struct {
	Destination string
	Force       bool
	DryRun      bool
	RunHooks    bool
	Values      map[string]string
}

var unresolvedTemplatePattern = regexp.MustCompile(`\{\{[A-Z0-9_]+\}\}`)

func runScenarioTemplateCommandWithApp(app *App, ctx *commandContext, args []string) error {
	action := "list"
	if len(args) > 0 {
		action = args[0]
		args = args[1:]
	}

	switch action {
	case "list":
		return executeScenarioTemplateCommand(app, ctx, args, scenarioTemplateCommandAction[scenarioTemplateListRequest, []scenarioTemplateInfo]{
			parse:  parseScenarioTemplateListRequest,
			run:    runScenarioTemplateListRequest,
			render: renderScenarioTemplateListResponse,
		})
	case "show":
		return executeScenarioTemplateCommand(app, ctx, args, scenarioTemplateCommandAction[scenarioTemplateShowRequest, scenarioTemplateInfo]{
			parse:  parseScenarioTemplateShowRequest,
			run:    runScenarioTemplateShowRequest,
			render: renderScenarioTemplateShowResponse,
		})
	case "help", "--help", "-h":
		showScenarioTemplateHelp(ctx.Stdout)
		return nil
	default:
		return usageErrorf("scenario template", "unknown template command: %s", action)
	}
}

func runScenarioGenerateCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeScenarioTemplateCommand(app, ctx, args, scenarioTemplateCommandAction[scenarioGenerateRequest, scenarioGenerateResult]{
		parse:  parseScenarioGenerateRequest,
		run:    runScenarioGenerateRequest,
		render: renderScenarioGenerateResponse,
	})
}

func runScenarioTemplateListCommand(root string, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) > 0 {
		for _, arg := range args {
			if arg == "--help" || arg == "-h" {
				showScenarioTemplateHelp(stdout)
				return nil
			}
			return unknownOptionError("scenario template", arg)
		}
	}

	templates, err := loadScenarioTemplates(root)
	if err != nil {
		return err
	}

	if globals.json {
		return cliout.WriteJSON(stdout, map[string]any{
			"success":   true,
			"templates": templates,
		})
	}

	rows := make([][]string, 0, len(templates))
	for _, item := range templates {
		required := formatScenarioTemplateRequiredVars(item.Manifest)
		if item.Missing {
			required = "?"
		}
		display := item.Manifest.DisplayName
		if display == "" {
			display = "(template.json missing)"
		}
		rows = append(rows, []string{item.Name, display, required})
	}
	_ = cliout.RenderTable(stdout, []string{"Name", "Display Name", "Required Vars"}, rows)
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Tip: vrooli scenario template show <name>")
	return nil
}

func runScenarioTemplateShowCommand(root string, globals globalOptions, args []string, stdout io.Writer) error {
	_ = globals
	if len(args) == 0 {
		return usageErrorf("scenario template show", "scenario template show requires a template name")
	}
	if len(args) > 1 {
		return usageErrorf("scenario template show", "scenario template show accepts exactly one template name")
	}

	info, err := loadScenarioTemplate(root, args[0])
	if err != nil {
		return err
	}

	manifest := info.Manifest
	title := manifest.DisplayName
	if title == "" {
		title = info.Name
	}

	_, _ = fmt.Fprintf(stdout, "%s (%s)\n", title, info.Name)
	if manifest.Description != "" {
		_, _ = fmt.Fprintln(stdout, manifest.Description)
	}
	if len(manifest.Stack) > 0 {
		_, _ = fmt.Fprintf(stdout, "Stack: %s\n", strings.Join(manifest.Stack, ", "))
	}

	writeScenarioTemplateVarTable(stdout, "Required Variables", manifest.RequiredVars)
	writeScenarioTemplateVarTable(stdout, "Optional Variables", manifest.OptionalVars)

	if len(manifest.PostHooks) > 0 {
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "Post Hooks:")
		for _, hook := range manifest.PostHooks {
			line := hook.Description
			if line == "" {
				line = hook.Cmd
			}
			_, _ = fmt.Fprintf(stdout, "  - %s\n", line)
		}
	}

	if len(manifest.Docs) > 0 {
		docKeys := make([]string, 0, len(manifest.Docs))
		for key := range manifest.Docs {
			docKeys = append(docKeys, key)
		}
		sort.Strings(docKeys)
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "Docs:")
		for _, key := range docKeys {
			_, _ = fmt.Fprintf(stdout, "  - %s: %s\n", key, manifest.Docs[key])
		}
	}

	entries, err := os.ReadDir(info.Path)
	if err == nil {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		sort.Strings(names)
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "Files:")
		for _, name := range names {
			_, _ = fmt.Fprintf(stdout, "  - %s\n", name)
		}
	}

	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintf(stdout, "Tip: vrooli scenario generate %s%s\n", info.Name, formatScenarioTemplateRequiredFlags(manifest))
	return nil
}

func showScenarioTemplateHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Scenario Template Commands:")
	_, _ = fmt.Fprintln(w, "  vrooli scenario template list")
	_, _ = fmt.Fprintln(w, "  vrooli scenario template show <template>")
	_, _ = fmt.Fprintln(w, "  vrooli scenario generate <template> [options]")
}

func showScenarioGenerateHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli scenario generate <template> --id <slug> --display-name <name> --description <text> [options]")
	_, _ = fmt.Fprintln(w, "Options:")
	_, _ = fmt.Fprintln(w, "  --dest <path>         Destination directory (defaults to scenarios/<id>)")
	_, _ = fmt.Fprintln(w, "  --var KEY=VALUE       Additional placeholder override (repeatable)")
	_, _ = fmt.Fprintln(w, "  --force               Overwrite destination if it already exists")
	_, _ = fmt.Fprintln(w, "  --dry-run             Print the planned actions without writing files")
	_, _ = fmt.Fprintln(w, "  --run-hooks           Execute template post hooks after generation")
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

func parseScenarioGenerateArgs(args []string, manifest scenarioTemplateManifest, stderr io.Writer) (scenarioGenerateOptions, error) {
	opts := scenarioGenerateOptions{Values: map[string]string{}}
	flagMap := make(map[string]string, len(manifest.RequiredVars)+len(manifest.OptionalVars))
	for key, variable := range manifest.RequiredVars {
		if variable.Flag != "" {
			flagMap[variable.Flag] = key
		}
	}
	for key, variable := range manifest.OptionalVars {
		if variable.Flag != "" {
			flagMap[variable.Flag] = key
		}
	}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--dest":
			if index+1 >= len(args) {
				return scenarioGenerateOptions{}, usageErrorf("scenario generate", "scenario generate --dest requires a value")
			}
			index++
			opts.Destination = args[index]
		case strings.HasPrefix(arg, "--dest="):
			opts.Destination = strings.TrimPrefix(arg, "--dest=")
		case arg == "--force":
			opts.Force = true
		case arg == "--dry-run":
			opts.DryRun = true
		case arg == "--run-hooks":
			opts.RunHooks = true
		case arg == "--var":
			if index+1 >= len(args) {
				return scenarioGenerateOptions{}, usageErrorf("scenario generate", "scenario generate --var requires KEY=VALUE")
			}
			index++
			key, value, err := parseScenarioTemplateKeyValue(args[index])
			if err != nil {
				return scenarioGenerateOptions{}, err
			}
			opts.Values[key] = value
		case strings.HasPrefix(arg, "--var="):
			key, value, err := parseScenarioTemplateKeyValue(strings.TrimPrefix(arg, "--var="))
			if err != nil {
				return scenarioGenerateOptions{}, err
			}
			opts.Values[key] = value
		case strings.HasPrefix(arg, "--"):
			flagName, flagValue, consumesNext, err := parseScenarioTemplateFlag(arg, args, index)
			if err != nil {
				return scenarioGenerateOptions{}, err
			}
			if consumesNext {
				index++
			}
			key, ok := flagMap[flagName]
			if !ok {
				_, _ = fmt.Fprintf(stderr, "Warning: unknown flag --%s; use --var KEY=VALUE for arbitrary placeholders\n", flagName)
				continue
			}
			opts.Values[key] = flagValue
		default:
			return scenarioGenerateOptions{}, usageErrorf("scenario generate", "unexpected argument: %s", arg)
		}
	}

	return opts, nil
}

func parseScenarioTemplateFlag(arg string, args []string, index int) (string, string, bool, error) {
	if strings.Contains(arg, "=") {
		parts := strings.SplitN(strings.TrimPrefix(arg, "--"), "=", 2)
		if parts[1] == "" {
			return "", "", false, usageErrorf("scenario generate", "--%s requires a value", parts[0])
		}
		return parts[0], parts[1], false, nil
	}
	if index+1 >= len(args) {
		return "", "", false, usageErrorf("scenario generate", "%s requires a value", arg)
	}
	return strings.TrimPrefix(arg, "--"), args[index+1], true, nil
}

func parseScenarioTemplateKeyValue(value string) (string, string, error) {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return "", "", usageErrorf("scenario generate", "invalid KEY=VALUE pair: %s", value)
	}
	return parts[0], parts[1], nil
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
