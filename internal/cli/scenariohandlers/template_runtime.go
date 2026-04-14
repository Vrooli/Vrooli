package scenariohandlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/scenarioexec"
)

var unresolvedTemplatePattern = regexp.MustCompile(`\{\{[A-Z0-9_]+\}\}`)

func TemplateCommandHandler[C any](deps HandlerDeps[C]) func(C, []string) error {
	return func(ctx C, args []string) error {
		if len(args) == 0 {
			return bindGlobal(deps.Stdout,
				func(ctx C, args []string) (TemplateListRequest, error) { return ParseTemplateListRequest(args) },
				func(ctx C, req TemplateListRequest) (cliout.Format, []TemplateInfo, error) {
					return runTemplateList(deps, ctx, req)
				},
				RenderTemplateListResponse,
			)(ctx, nil)
		}
		switch commandtree.NormalizeName(args[0]) {
		case "list":
			return bindGlobal(deps.Stdout,
				func(ctx C, args []string) (TemplateListRequest, error) { return ParseTemplateListRequest(args) },
				func(ctx C, req TemplateListRequest) (cliout.Format, []TemplateInfo, error) {
					return runTemplateList(deps, ctx, req)
				},
				RenderTemplateListResponse,
			)(ctx, args[1:])
		case "show":
			return bindGlobal(deps.Stdout,
				func(ctx C, args []string) (TemplateShowRequest, error) { return ParseTemplateShowRequest(args) },
				func(ctx C, req TemplateShowRequest) (cliout.Format, TemplateInfo, error) {
					return runTemplateShow(deps, ctx, req)
				},
				RenderTemplateShowResponse,
			)(ctx, args[1:])
		case "--help", "-h":
			RenderTemplateHelp(deps.Stdout(ctx))
			return nil
		default:
			return rootcli.UsageErrorf("scenario template", "unknown scenario template command: %s", args[0])
		}
	}
}

func GenerateHandler[C any](deps HandlerDeps[C]) func(C, []string) error {
	return bindGlobal(deps.Stdout,
		func(ctx C, args []string) (GenerateRequest, error) {
			return ParseGenerateRequest(args, deps.Stderr(ctx), func(name string) (TemplateInfo, error) {
				return loadTemplate(deps.Root(ctx), name)
			}, ParseGenerateArgs)
		},
		func(ctx C, req GenerateRequest) (cliout.Format, GenerateResult, error) {
			return runGenerate(deps, ctx, req)
		},
		RenderGenerateResponse,
	)
}

func runTemplateList[C any](deps HandlerDeps[C], ctx C, _ TemplateListRequest) (cliout.Format, []TemplateInfo, error) {
	templates, err := loadTemplates(deps.Root(ctx))
	if err != nil {
		return "", nil, err
	}
	format, err := deps.OutputFormat(ctx)
	if err != nil {
		return "", nil, err
	}
	return format, templates, nil
}

func runTemplateShow[C any](deps HandlerDeps[C], ctx C, req TemplateShowRequest) (cliout.Format, TemplateInfo, error) {
	info, err := loadTemplate(deps.Root(ctx), req.Name)
	if err != nil {
		return "", TemplateInfo{}, err
	}
	return cliout.FormatHuman, info, nil
}

func runGenerate[C any](deps HandlerDeps[C], ctx C, req GenerateRequest) (cliout.Format, GenerateResult, error) {
	info := req.TemplateInfo
	opts := req.Options
	currentDate := time.Now().UTC().Format("2006-01-02")
	randomToken, err := randomTemplateToken()
	if err != nil {
		return "", GenerateResult{}, err
	}
	values := copyStringMap(opts.Values)
	values["CURRENT_DATE"] = currentDate
	values["RANDOM_TOKEN"] = randomToken
	optionalKeys := make([]string, 0, len(info.Manifest.OptionalVars))
	for key := range info.Manifest.OptionalVars {
		optionalKeys = append(optionalKeys, key)
	}
	sort.Strings(optionalKeys)
	for _, key := range optionalKeys {
		if strings.TrimSpace(values[key]) == "" {
			values[key] = renderTemplateString(info.Manifest.OptionalVars[key].Default, values)
		}
	}
	destination := opts.Destination
	if destination == "" {
		destination = filepath.Join(deps.Root(ctx), "scenarios", values["SCENARIO_ID"])
	}
	if !filepath.IsAbs(destination) {
		destination = filepath.Join(deps.Root(ctx), filepath.FromSlash(destination))
	}
	destination = filepath.Clean(destination)
	if opts.DryRun {
		return cliout.FormatHuman, GenerateResult{
			TemplateName: info.Name,
			DisplayName:  coalesce(values["SCENARIO_DISPLAY_NAME"], values["SCENARIO_ID"]),
			Destination:  destination,
			Values:       values,
			Manifest:     info.Manifest,
			DryRun:       true,
		}, nil
	}
	if stat, err := os.Stat(destination); err == nil && stat != nil {
		if !opts.Force {
			return "", GenerateResult{}, fmt.Errorf("destination already exists: %s (use --force to overwrite)", destination)
		}
		if err := os.RemoveAll(destination); err != nil {
			return "", GenerateResult{}, err
		}
	}
	if err := copyTemplate(info.Path, destination, values); err != nil {
		return "", GenerateResult{}, err
	}
	if err := verifyTemplate(destination); err != nil {
		return "", GenerateResult{}, err
	}
	result := GenerateResult{
		TemplateName: info.Name,
		DisplayName:  coalesce(values["SCENARIO_DISPLAY_NAME"], values["SCENARIO_ID"]),
		Destination:  destination,
		Values:       values,
		Manifest:     info.Manifest,
		RunHooks:     opts.RunHooks,
	}
	if opts.RunHooks {
		if err := runTemplateHooks(deps, ctx, destination, info.Manifest); err != nil {
			return "", GenerateResult{}, err
		}
	}
	return cliout.FormatHuman, result, nil
}

func loadTemplates(root string) ([]TemplateInfo, error) {
	baseDir := config.TemplateBaseDir(root)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	templates := make([]TemplateInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		info, err := loadTemplate(root, name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				templates = append(templates, TemplateInfo{Name: name, Path: filepath.Join(baseDir, name), Missing: true})
				continue
			}
			return nil, err
		}
		templates = append(templates, info)
	}
	sort.Slice(templates, func(i, j int) bool { return templates[i].Name < templates[j].Name })
	return templates, nil
}

func loadTemplate(root, name string) (TemplateInfo, error) {
	templateDir := filepath.Join(config.TemplateBaseDir(root), name)
	info, err := os.Stat(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return TemplateInfo{}, fmt.Errorf("template not found: %s", name)
		}
		return TemplateInfo{}, err
	}
	if !info.IsDir() {
		return TemplateInfo{}, fmt.Errorf("template path is not a directory: %s", templateDir)
	}
	data, err := os.ReadFile(filepath.Join(templateDir, "template.json"))
	if err != nil {
		return TemplateInfo{}, err
	}
	var manifest TemplateManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return TemplateInfo{}, err
	}
	if manifest.Name == "" {
		manifest.Name = name
	}
	if manifest.RequiredVars == nil {
		manifest.RequiredVars = map[string]TemplateVar{}
	}
	if manifest.OptionalVars == nil {
		manifest.OptionalVars = map[string]TemplateVar{}
	}
	if manifest.Docs == nil {
		manifest.Docs = map[string]string{}
	}
	return TemplateInfo{Name: name, Path: templateDir, Manifest: manifest}, nil
}

func copyTemplate(templateDir, destination string, values map[string]string) error {
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
		if filepath.Base(path) == ".DS_Store" || relPath == "template.json" {
			return nil
		}
		targetPath := filepath.Join(destination, renderTemplateString(relPath, values))
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
			data = []byte(renderTemplateString(string(data), values))
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}

func verifyTemplate(destination string) error {
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
		if looksLikeTextFile(data) && unresolvedTemplatePattern.Match(data) {
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
	return len(data) == 0 || (bytes.IndexByte(data, 0) < 0 && utf8.Valid(data))
}

func LooksLikeTextFile(data []byte) bool {
	return looksLikeTextFile(data)
}

func renderTemplateString(value string, values map[string]string) string {
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

func runTemplateHooks[C any](deps HandlerDeps[C], ctx C, destination string, manifest TemplateManifest) error {
	if len(manifest.PostHooks) == 0 {
		_, _ = fmt.Fprintln(deps.Stdout(ctx), "No post hooks defined for this template")
		return nil
	}
	for index, hook := range manifest.PostHooks {
		description := strings.TrimSpace(hook.Description)
		if description == "" {
			description = hook.Cmd
		}
		_, _ = fmt.Fprintf(deps.Stdout(ctx), "[Hook %d] %s\n", index+1, description)
		cwd := destination
		if strings.TrimSpace(hook.Cwd) != "" && hook.Cwd != "." {
			cwd = filepath.Join(destination, filepath.FromSlash(hook.Cwd))
		}
		if err := deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
			Name:   "bash",
			Args:   []string{"-lc", hook.Cmd},
			Dir:    cwd,
			Env:    deps.CommandEnv(ctx),
			Stdout: deps.Stdout(ctx),
			Stderr: deps.Stderr(ctx),
		}); err != nil {
			return err
		}
	}
	return nil
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

func FormatTemplateRequiredFlags(manifest TemplateManifest) string {
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
