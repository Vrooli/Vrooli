package secrets

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"resource-vault/cli/internal/content"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"gopkg.in/yaml.v3"
)

const defaultField = "value"

type Handlers struct {
	Runner Runner
	Stdout io.Writer
	Stderr io.Writer
	Root   string
}

type Runner interface {
	Run(ctx context.Context, vaultArgs []string, stdin []byte) ([]byte, []byte, error)
}

func Default() *Handlers {
	return &Handlers{
		Runner: content.NewDockerRunner(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "secrets",
		Description: "Inventory, check, validate, export, and provision resource config/secrets.yaml declarations",
		Subcommands: []cliapp.Command{
			{Name: "scan", Description: "List resources with config/secrets.yaml declarations", Usage: "resource-vault secrets scan [--json]", Run: h.Scan},
			{Name: "check", Description: "Check one resource's declared secrets without printing values", Usage: "resource-vault secrets check <resource> [--json]", Run: h.Check},
			{Name: "validate", Description: "Validate declared required secrets for one resource or all resources", Usage: "resource-vault secrets validate [resource] [--json]", Run: h.Validate},
			{Name: "export", Description: "Print shell-safe export lines for a resource's present secrets", Usage: "resource-vault secrets export <resource>", Run: h.Export},
			{Name: "provision", Aliases: []string{"init"}, Description: "Generate supported auto-generated secrets without prompting", Usage: "resource-vault secrets provision <resource>", Run: h.Provision},
			{Name: "init", Description: "Alias for provision", Usage: "resource-vault secrets init <resource>", Run: h.Provision},
		},
	}
}

type fileDoc struct {
	Version        string         `yaml:"version" json:"version,omitempty"`
	Resource       string         `yaml:"resource" json:"resource"`
	Description    string         `yaml:"description" json:"description,omitempty"`
	Secrets        yaml.Node      `yaml:"secrets" json:"-"`
	Initialization initialization `yaml:"initialization" json:"initialization,omitempty"`
	HealthCheck    struct {
		Required []string `yaml:"required_secrets" json:"required_secrets,omitempty"`
	} `yaml:"health_check" json:"health_check,omitempty"`
}

type initialization struct {
	AutoGenerate []autoGenerate `yaml:"auto_generate" json:"auto_generate,omitempty"`
}

type autoGenerate struct {
	Name   string `yaml:"name" json:"name"`
	Type   string `yaml:"type" json:"type"`
	Path   string `yaml:"path" json:"path"`
	Length int    `yaml:"length" json:"length,omitempty"`
	Bits   int    `yaml:"bits" json:"bits,omitempty"`
}

type secretDecl struct {
	Name        string              `yaml:"name" json:"name"`
	Path        string              `yaml:"path" json:"path"`
	Key         string              `yaml:"key" json:"key,omitempty"`
	Description string              `yaml:"description" json:"description,omitempty"`
	Required    *bool               `yaml:"required" json:"required,omitempty"`
	Format      string              `yaml:"format" json:"format,omitempty"`
	DefaultEnv  string              `yaml:"default_env" json:"default_env,omitempty"`
	Fallback    string              `yaml:"fallback" json:"fallback,omitempty"`
	Fields      []map[string]string `yaml:"fields" json:"fields,omitempty"`
	Validation  struct {
		Pattern string `yaml:"pattern" json:"pattern,omitempty"`
	} `yaml:"validation" json:"validation,omitempty"`
}

type resourceInventory struct {
	Resource string       `json:"resource"`
	Path     string       `json:"path"`
	Secrets  []secretItem `json:"secrets"`
}

type secretItem struct {
	Category    string `json:"category"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Key         string `json:"key"`
	Required    bool   `json:"required"`
	DefaultEnv  string `json:"default_env,omitempty"`
	Description string `json:"description,omitempty"`
	Dynamic     bool   `json:"dynamic"`
}

type checkResult struct {
	Resource string         `json:"resource"`
	Summary  map[string]int `json:"summary"`
	Items    []checkItem    `json:"items"`
}

type checkItem struct {
	secretItem
	Present bool   `json:"present"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

func (h *Handlers) Scan(args []string) error {
	fs, jsonOut := h.flagSet("secrets scan")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	inventory, err := h.loadAll()
	if err != nil {
		return err
	}
	if *jsonOut {
		return printJSON(h.Stdout, inventory)
	}
	for _, inv := range inventory {
		fmt.Fprintf(h.Stdout, "%s: %d secrets (%s)\n", inv.Resource, len(inv.Secrets), inv.Path)
	}
	fmt.Fprintf(h.Stdout, "summary: %d resources, %d secrets\n", len(inventory), countSecrets(inventory))
	return nil
}

func (h *Handlers) Check(args []string) error {
	fs, jsonOut := h.flagSet("secrets check")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: resource-vault secrets check <resource>")
	}
	result, err := h.checkResource(fs.Arg(0), false)
	if err != nil {
		return err
	}
	if *jsonOut {
		return printJSON(h.Stdout, result)
	}
	renderCheck(h.Stdout, result)
	if result.Summary["missing_required"] > 0 || result.Summary["errors"] > 0 {
		return fmt.Errorf("%s has missing or unreadable required secrets", result.Resource)
	}
	return nil
}

func (h *Handlers) Validate(args []string) error {
	fs, jsonOut := h.flagSet("secrets validate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("usage: resource-vault secrets validate [resource]")
	}
	var results []checkResult
	if fs.NArg() == 1 {
		result, err := h.checkResource(fs.Arg(0), true)
		if err != nil {
			return err
		}
		results = append(results, result)
	} else {
		inventory, err := h.loadAll()
		if err != nil {
			return err
		}
		for _, inv := range inventory {
			result, err := h.checkInventory(inv, true)
			if err != nil {
				return err
			}
			results = append(results, result)
		}
	}
	if *jsonOut {
		return printJSON(h.Stdout, results)
	}
	var failed int
	for _, result := range results {
		renderCheck(h.Stdout, result)
		if result.Summary["missing_required"] > 0 || result.Summary["errors"] > 0 || result.Summary["invalid"] > 0 {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d resource secret declaration(s) failed validation", failed)
	}
	return nil
}

func (h *Handlers) Export(args []string) error {
	fs := flag.NewFlagSet("secrets export", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: resource-vault secrets export <resource>")
	}
	inv, err := h.loadResource(fs.Arg(0))
	if err != nil {
		return err
	}
	for _, item := range inv.Secrets {
		if item.DefaultEnv == "" || item.Dynamic {
			continue
		}
		value, found, err := h.readSecret(item)
		if err != nil {
			return fmt.Errorf("read %s: %w", item.Name, err)
		}
		if !found {
			continue
		}
		fmt.Fprintf(h.Stdout, "export %s=%s\n", item.DefaultEnv, shellQuote(value))
	}
	return nil
}

func (h *Handlers) Provision(args []string) error {
	fs := flag.NewFlagSet("secrets provision", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: resource-vault secrets provision <resource>")
	}
	inv, doc, err := h.loadResourceWithDoc(fs.Arg(0))
	if err != nil {
		return err
	}
	byName := map[string]secretItem{}
	for _, item := range inv.Secrets {
		byName[item.Name] = item
	}
	var created, skipped int
	for _, gen := range doc.Initialization.AutoGenerate {
		name := strings.TrimSpace(gen.Name)
		item, ok := byName[name]
		if !ok {
			item = secretItem{Name: name, Path: expandPath(gen.Path, inv.Resource), Key: defaultField, Required: false, Dynamic: hasUnresolvedTemplate(expandPath(gen.Path, inv.Resource))}
		}
		if item.Dynamic {
			skipped++
			fmt.Fprintf(h.Stdout, "skipped %s: dynamic path %s\n", name, item.Path)
			continue
		}
		if _, found, err := h.readSecret(item); err != nil {
			return err
		} else if found {
			skipped++
			fmt.Fprintf(h.Stdout, "kept %s at %s\n", name, item.Path)
			continue
		}
		value, err := generateValue(gen.Type, gen.Length)
		if err != nil {
			return fmt.Errorf("generate %s: %w", name, err)
		}
		if err := h.writeSecret(item, value); err != nil {
			return fmt.Errorf("store %s: %w", name, err)
		}
		created++
		fmt.Fprintf(h.Stdout, "created %s at %s\n", name, item.Path)
	}
	fmt.Fprintf(h.Stdout, "summary: created=%d skipped=%d\n", created, skipped)
	return nil
}

func (h *Handlers) flagSet(name string) (*flag.FlagSet, *bool) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	jsonOut := fs.Bool("json", false, "Print JSON")
	return fs, jsonOut
}

func (h *Handlers) loadAll() ([]resourceInventory, error) {
	root, err := h.repoRoot()
	if err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(filepath.Join(root, "resources", "*", "config", "secrets.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	var out []resourceInventory
	for _, path := range matches {
		inv, _, err := loadFile(path)
		if err != nil {
			fmt.Fprintf(h.Stderr, "warning: skipping %s: %v\n", path, err)
			continue
		}
		out = append(out, inv)
	}
	return out, nil
}

func (h *Handlers) loadResource(resource string) (resourceInventory, error) {
	inv, _, err := h.loadResourceWithDoc(resource)
	return inv, err
}

func (h *Handlers) loadResourceWithDoc(resource string) (resourceInventory, fileDoc, error) {
	root, err := h.repoRoot()
	if err != nil {
		return resourceInventory{}, fileDoc{}, err
	}
	path := filepath.Join(root, "resources", resource, "config", "secrets.yaml")
	inv, doc, err := loadFile(path)
	return inv, doc, err
}

func (h *Handlers) repoRoot() (string, error) {
	if strings.TrimSpace(h.Root) != "" {
		return h.Root, nil
	}
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return root, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "resources", "vault", "resource.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("could not locate Vrooli repo root; set VROOLI_ROOT")
}

func loadFile(path string) (resourceInventory, fileDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return resourceInventory{}, fileDoc{}, err
	}
	var doc fileDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return resourceInventory{}, fileDoc{}, fmt.Errorf("parse %s: %w", path, err)
	}
	resource := strings.TrimSpace(doc.Resource)
	if resource == "" {
		resource = filepath.Base(filepath.Dir(filepath.Dir(path)))
	}
	inv := resourceInventory{Resource: resource, Path: path}
	byCategory, err := parseSecretsNode(&doc.Secrets)
	if err != nil {
		return resourceInventory{}, fileDoc{}, fmt.Errorf("parse %s secrets: %w", path, err)
	}
	categories := make([]string, 0, len(byCategory))
	for category := range byCategory {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	for _, category := range categories {
		for _, decl := range byCategory[category] {
			required := true
			if decl.Required != nil {
				required = *decl.Required
			}
			item := secretItem{
				Category:    category,
				Name:        decl.Name,
				Path:        expandPath(decl.Path, resource),
				Key:         defaultField,
				Required:    required,
				DefaultEnv:  decl.DefaultEnv,
				Description: decl.Description,
			}
			if strings.TrimSpace(decl.Key) != "" {
				item.Key = strings.TrimSpace(decl.Key)
			}
			item.Dynamic = hasUnresolvedTemplate(item.Path)
			inv.Secrets = append(inv.Secrets, item)
		}
	}
	return inv, doc, nil
}

func parseSecretsNode(node *yaml.Node) (map[string][]secretDecl, error) {
	out := map[string][]secretDecl{}
	if node == nil || node.Kind == 0 {
		return out, nil
	}
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("secrets must be a mapping")
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		category := node.Content[i].Value
		value := node.Content[i+1]
		switch value.Kind {
		case yaml.SequenceNode:
			for _, entry := range value.Content {
				var decl secretDecl
				if err := entry.Decode(&decl); err != nil {
					return nil, err
				}
				out[category] = append(out[category], decl)
			}
		case yaml.MappingNode:
			decls, err := parseLegacyCategory(category, value)
			if err != nil {
				return nil, err
			}
			out[category] = append(out[category], decls...)
		}
	}
	return out, nil
}

func parseLegacyCategory(category string, node *yaml.Node) ([]secretDecl, error) {
	var raw struct {
		Path        string   `yaml:"path"`
		Keys        []string `yaml:"keys"`
		Description string   `yaml:"description"`
		Required    *bool    `yaml:"required"`
		DefaultEnv  string   `yaml:"default_env"`
	}
	if err := node.Decode(&raw); err != nil {
		return nil, err
	}
	if raw.Path == "" {
		return nil, nil
	}
	if len(raw.Keys) == 0 {
		return []secretDecl{{
			Name:        category,
			Path:        raw.Path,
			Description: raw.Description,
			Required:    raw.Required,
			DefaultEnv:  raw.DefaultEnv,
		}}, nil
	}
	decls := make([]secretDecl, 0, len(raw.Keys))
	for _, key := range raw.Keys {
		name := category + "_" + key
		decls = append(decls, secretDecl{
			Name:        name,
			Path:        raw.Path,
			Key:         key,
			Description: raw.Description,
			Required:    raw.Required,
			DefaultEnv:  raw.DefaultEnv,
		})
	}
	return decls, nil
}

func (h *Handlers) checkResource(resource string, validate bool) (checkResult, error) {
	inv, err := h.loadResource(resource)
	if err != nil {
		return checkResult{}, err
	}
	return h.checkInventory(inv, validate)
}

func (h *Handlers) checkInventory(inv resourceInventory, validate bool) (checkResult, error) {
	result := checkResult{Resource: inv.Resource, Summary: map[string]int{}}
	for _, item := range inv.Secrets {
		check := checkItem{secretItem: item}
		switch {
		case item.Dynamic:
			check.Status = "dynamic"
			result.Summary["dynamic"]++
		default:
			value, found, err := h.readSecret(item)
			if err != nil {
				check.Status = "error"
				check.Error = err.Error()
				result.Summary["errors"]++
				break
			}
			check.Present = found
			if !found && item.Required {
				check.Status = "missing_required"
				result.Summary["missing_required"]++
			} else if !found {
				check.Status = "missing_optional"
				result.Summary["missing_optional"]++
			} else if validate && strings.TrimSpace(value) == "" {
				check.Status = "invalid"
				result.Summary["invalid"]++
			} else {
				check.Status = "present"
				result.Summary["present"]++
			}
		}
		result.Items = append(result.Items, check)
	}
	return result, nil
}

func (h *Handlers) readSecret(item secretItem) (string, bool, error) {
	stdout, stderr, err := h.Runner.Run(context.Background(), []string{"kv", "get", "-field=" + item.Key, item.Path}, nil)
	if err != nil {
		msg := strings.TrimSpace(string(stderr) + "\n" + string(stdout))
		if isMissing(msg) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("vault kv get %s/%s: %w%s", item.Path, item.Key, err, formatStderr(stderr))
	}
	return strings.TrimSpace(string(stdout)), true, nil
}

func (h *Handlers) writeSecret(item secretItem, value string) error {
	kv := item.Key + "=" + value
	if _, _, err := h.Runner.Run(context.Background(), []string{"kv", "patch", item.Path, kv}, nil); err == nil {
		return nil
	}
	_, stderr, err := h.Runner.Run(context.Background(), []string{"kv", "put", item.Path, kv}, nil)
	if err != nil {
		return fmt.Errorf("vault kv put %s: %w%s", item.Path, err, formatStderr(stderr))
	}
	return nil
}

func expandPath(path, resource string) string {
	path = strings.ReplaceAll(path, "{resource}", resource)
	path = strings.ReplaceAll(path, "{resource-name}", resource)
	return path
}

var unresolvedTemplate = regexp.MustCompile(`\{[^}]+\}`)

func hasUnresolvedTemplate(path string) bool {
	return unresolvedTemplate.MatchString(path)
}

func isMissing(stderr string) bool {
	msg := strings.ToLower(stderr)
	return strings.Contains(msg, "no value found") ||
		strings.Contains(msg, "no secret exists") ||
		strings.Contains(msg, "not found")
}

func formatStderr(stderr []byte) string {
	msg := strings.TrimSpace(string(stderr))
	if msg == "" {
		return ""
	}
	return ": " + msg
}

func renderCheck(w io.Writer, result checkResult) {
	fmt.Fprintf(w, "%s:\n", result.Resource)
	for _, item := range result.Items {
		fmt.Fprintf(w, "  %s %s (%s)\n", item.Status, item.Name, item.Path)
	}
	fmt.Fprintf(w, "  summary: present=%d missing_required=%d missing_optional=%d dynamic=%d errors=%d invalid=%d\n",
		result.Summary["present"], result.Summary["missing_required"], result.Summary["missing_optional"], result.Summary["dynamic"], result.Summary["errors"], result.Summary["invalid"])
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func generateValue(kind string, length int) (string, error) {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "uuid":
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			return "", err
		}
		b[6] = (b[6] & 0x0f) | 0x40
		b[8] = (b[8] & 0x3f) | 0x80
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
	case "token", "password", "random-32", "":
		size := 32
		if length > 0 {
			size = length
		}
		buf := make([]byte, size)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		return base64.RawURLEncoding.EncodeToString(buf), nil
	case "rsa", "jwt":
		return "", fmt.Errorf("auto-generate type %q is not supported non-interactively", kind)
	default:
		if n, err := strconv.Atoi(strings.TrimPrefix(kind, "random-")); err == nil && n > 0 {
			buf := make([]byte, n)
			if _, err := rand.Read(buf); err != nil {
				return "", err
			}
			return base64.RawURLEncoding.EncodeToString(buf), nil
		}
		return "", fmt.Errorf("unsupported auto-generate type %q", kind)
	}
}

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func countSecrets(inventory []resourceInventory) int {
	var n int
	for _, inv := range inventory {
		n += len(inv.Secrets)
	}
	return n
}
