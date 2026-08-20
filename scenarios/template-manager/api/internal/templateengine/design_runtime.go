package templateengine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func runDesignList[C any](deps HandlerDeps[C], ctx C, _ templatecontracts.DesignListRequest) ([]templatecontracts.DesignKitInfo, error) {
	return loadDesignKits(deps.Root(ctx))
}

func runDesignShow[C any](deps HandlerDeps[C], ctx C, req templatecontracts.DesignShowRequest) (templatecontracts.DesignKitInfo, error) {
	return loadDesignKit(deps.Root(ctx), req.ID)
}

func runDesignValidate[C any](deps HandlerDeps[C], ctx C, req templatecontracts.DesignValidateRequest) (templatecontracts.DesignValidationReport, error) {
	var kits []templatecontracts.DesignKitInfo
	var err error
	if strings.TrimSpace(req.ID) != "" {
		info, loadErr := loadDesignKit(deps.Root(ctx), req.ID)
		if loadErr != nil {
			return templatecontracts.DesignValidationReport{}, loadErr
		}
		kits = []templatecontracts.DesignKitInfo{info}
	} else {
		kits, err = loadDesignKits(deps.Root(ctx))
		if err != nil {
			return templatecontracts.DesignValidationReport{}, err
		}
	}
	report := templatecontracts.DesignValidationReport{Count: len(kits)}
	defaults := 0
	for _, kit := range kits {
		if kit.Manifest.Default {
			defaults++
		}
		kitIssues := validateDesignKit(kit)
		report.Issues = append(report.Issues, kitIssues...)
		report.Results = append(report.Results, templatecontracts.DesignKitValidationResult{
			Kit:    kit.ID,
			Status: validationStatus(len(kitIssues) == 0),
			Issues: kitIssues,
		})
	}
	if req.All && defaults > 1 {
		report.Issues = append(report.Issues, templatecontracts.DesignValidationIssue{Message: "only one design kit may be marked default"})
	}
	return report, nil
}

func loadDesignKits(root string) ([]templatecontracts.DesignKitInfo, error) {
	baseDir := config.DesignBaseDir(root)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	kits := make([]templatecontracts.DesignKitInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := loadDesignKit(root, entry.Name())
		if err != nil {
			if os.IsNotExist(err) {
				kits = append(kits, templatecontracts.DesignKitInfo{ID: entry.Name(), Path: filepath.Join(baseDir, entry.Name()), Missing: true})
				continue
			}
			return nil, err
		}
		kits = append(kits, info)
	}
	sort.Slice(kits, func(i, j int) bool { return kits[i].ID < kits[j].ID })
	return kits, nil
}

func loadDesignKit(root, id string) (templatecontracts.DesignKitInfo, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return templatecontracts.DesignKitInfo{}, fmt.Errorf("design kit id is required")
	}
	kitDir := filepath.Join(config.DesignBaseDir(root), id)
	stat, err := os.Stat(kitDir)
	if err != nil {
		if os.IsNotExist(err) {
			return templatecontracts.DesignKitInfo{}, fmt.Errorf("design kit not found: %s", id)
		}
		return templatecontracts.DesignKitInfo{}, err
	}
	if !stat.IsDir() {
		return templatecontracts.DesignKitInfo{}, fmt.Errorf("design kit path is not a directory: %s", kitDir)
	}
	data, err := os.ReadFile(filepath.Join(kitDir, "metadata.json"))
	if err != nil {
		return templatecontracts.DesignKitInfo{}, err
	}
	var manifest templatecontracts.DesignKitManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return templatecontracts.DesignKitInfo{}, err
	}
	if manifest.ID == "" {
		manifest.ID = id
	}
	if manifest.Adapters == nil {
		manifest.Adapters = map[string]templatecontracts.DesignKitAdapter{}
	}
	return templatecontracts.DesignKitInfo{ID: id, Path: kitDir, Manifest: manifest}, nil
}

func validateDesignKit(info templatecontracts.DesignKitInfo) []templatecontracts.DesignValidationIssue {
	if info.Missing {
		return []templatecontracts.DesignValidationIssue{{Kit: info.ID, Message: "metadata.json is missing"}}
	}
	var issues []templatecontracts.DesignValidationIssue
	if info.Manifest.ID != info.ID {
		issues = append(issues, templatecontracts.DesignValidationIssue{Kit: info.ID, Path: "metadata.json", Message: fmt.Sprintf("metadata id %q must match folder name", info.Manifest.ID)})
	}
	for _, required := range []string{"metadata.json", "DESIGN.md"} {
		if stat, err := os.Stat(filepath.Join(info.Path, required)); err != nil || stat.IsDir() {
			msg := "required file is missing"
			if err == nil && stat.IsDir() {
				msg = "required file is a directory"
			}
			issues = append(issues, templatecontracts.DesignValidationIssue{Kit: info.ID, Path: required, Message: msg})
		}
	}
	issues = append(issues, validateDesignDocument(info)...)
	for id, adapter := range info.Manifest.Adapters {
		cleanPath, ok := cleanRelativePath(adapter.Path)
		if !ok {
			issues = append(issues, templatecontracts.DesignValidationIssue{Kit: info.ID, Adapter: id, Path: adapter.Path, Message: "adapter path must be relative and stay inside the kit"})
			continue
		}
		adapterDir := filepath.Join(info.Path, cleanPath)
		if stat, err := os.Stat(adapterDir); err != nil || !stat.IsDir() {
			issues = append(issues, templatecontracts.DesignValidationIssue{Kit: info.ID, Adapter: id, Path: adapter.Path, Message: "adapter path is missing or not a directory"})
			continue
		}
		manifest, err := loadDesignAdapterManifest(adapterDir)
		if err != nil {
			issues = append(issues, templatecontracts.DesignValidationIssue{Kit: info.ID, Adapter: id, Path: filepath.ToSlash(filepath.Join(cleanPath, "adapter.json")), Message: err.Error()})
			continue
		}
		if manifest.ID != "" && manifest.ID != id {
			issues = append(issues, templatecontracts.DesignValidationIssue{Kit: info.ID, Adapter: id, Path: filepath.ToSlash(filepath.Join(cleanPath, "adapter.json")), Message: fmt.Sprintf("adapter id %q must match metadata adapter key", manifest.ID)})
		}
		for index, rule := range manifest.Copy {
			if _, ok := cleanRelativePath(rule.From); !ok {
				issues = append(issues, templatecontracts.DesignValidationIssue{Kit: info.ID, Adapter: id, Path: fmt.Sprintf("copy[%d].from", index), Message: "copy source must be relative and stay inside the adapter"})
				continue
			}
			if _, ok := cleanRelativePath(rule.To); !ok {
				issues = append(issues, templatecontracts.DesignValidationIssue{Kit: info.ID, Adapter: id, Path: fmt.Sprintf("copy[%d].to", index), Message: "copy destination must be scenario-relative"})
				continue
			}
			if stat, err := os.Stat(filepath.Join(adapterDir, filepath.FromSlash(rule.From))); err != nil || stat.IsDir() {
				issues = append(issues, templatecontracts.DesignValidationIssue{Kit: info.ID, Adapter: id, Path: rule.From, Message: "copy source is missing or not a file"})
			}
		}
	}
	return issues
}

func validateDesignDocument(info templatecontracts.DesignKitInfo) []templatecontracts.DesignValidationIssue {
	path := filepath.Join(info.Path, "DESIGN.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	content := string(data)
	frontMatter, ok := extractDesignFrontMatter(content)
	if !ok {
		return []templatecontracts.DesignValidationIssue{{Kit: info.ID, Path: "DESIGN.md", Message: "DESIGN.md must start with YAML front matter delimited by ---"}}
	}
	topLevel := designFrontMatterTopLevelKeys(frontMatter)
	var issues []templatecontracts.DesignValidationIssue
	for _, key := range []string{"name", "colors", "typography", "rounded", "spacing", "components"} {
		if !topLevel[key] {
			issues = append(issues, templatecontracts.DesignValidationIssue{Kit: info.ID, Path: "DESIGN.md", Message: fmt.Sprintf("missing official-style top-level %q token group", key)})
		}
	}
	for _, key := range []string{
		"button-primary-loading",
		"button-disabled",
		"input-error",
		"alert-error",
		"toast-success",
		"empty-state",
		"skeleton",
		"inline-progress",
		"retry-action",
	} {
		if !strings.Contains(frontMatter, "\n  "+key+":") {
			issues = append(issues, templatecontracts.DesignValidationIssue{Kit: info.ID, Path: "DESIGN.md", Message: fmt.Sprintf("components must include %q state guidance token", key)})
		}
	}
	for _, section := range []string{"## Feedback & State", "## Request Lifecycle"} {
		if !strings.Contains(content, section) {
			issues = append(issues, templatecontracts.DesignValidationIssue{Kit: info.ID, Path: "DESIGN.md", Message: fmt.Sprintf("missing %s section for asynchronous and error-state UX", section)})
		}
	}
	for _, term := range []string{"loading", "success", "validation-error", "request-error", "retry"} {
		if !strings.Contains(strings.ToLower(content), term) {
			issues = append(issues, templatecontracts.DesignValidationIssue{Kit: info.ID, Path: "DESIGN.md", Message: fmt.Sprintf("missing required UX state term %q", term)})
		}
	}
	return issues
}

func extractDesignFrontMatter(content string) (string, bool) {
	content = strings.TrimPrefix(content, "\ufeff")
	if !strings.HasPrefix(content, "---\n") {
		return "", false
	}
	rest := content[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

func designFrontMatterTopLevelKeys(frontMatter string) map[string]bool {
	keys := map[string]bool{}
	for _, line := range strings.Split(frontMatter, "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		key, _, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		keys[strings.TrimSpace(key)] = true
	}
	return keys
}

func loadDesignAdapterManifest(adapterDir string) (templatecontracts.DesignAdapterManifest, error) {
	data, err := os.ReadFile(filepath.Join(adapterDir, "adapter.json"))
	if err != nil {
		return templatecontracts.DesignAdapterManifest{}, err
	}
	var manifest templatecontracts.DesignAdapterManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return templatecontracts.DesignAdapterManifest{}, err
	}
	return manifest, nil
}

func resolveDesign(root string, info templatecontracts.TemplateInfo, requested, destination string, values map[string]string) (templatecontracts.ResolvedDesign, error) {
	templateDesign := info.Manifest.Design
	selection := strings.TrimSpace(requested)
	if selection == "" {
		selection = strings.TrimSpace(templateDesign.Default)
	}
	if strings.EqualFold(selection, "none") {
		if templateDesign.Required {
			return templatecontracts.ResolvedDesign{}, fmt.Errorf("template %s requires a design kit; --design none is not allowed", info.Name)
		}
		return templatecontracts.ResolvedDesign{}, nil
	}
	if selection == "" {
		if templateDesign.Required {
			return templatecontracts.ResolvedDesign{}, fmt.Errorf("template %s requires a design kit but declares no default", info.Name)
		}
		return templatecontracts.ResolvedDesign{}, nil
	}
	kit, err := loadDesignKit(root, selection)
	if err != nil {
		return templatecontracts.ResolvedDesign{}, err
	}
	if issues := validateDesignKit(kit); len(issues) > 0 {
		return templatecontracts.ResolvedDesign{}, fmt.Errorf("design kit %s is invalid: %s", selection, formatDesignValidationIssues(issues))
	}
	adapterID := strings.TrimSpace(templateDesign.Adapter)
	if adapterID == "" {
		return templatecontracts.ResolvedDesign{
			KitID:   kit.ID,
			KitName: kit.Manifest.Name,
			Version: kit.Manifest.Version,
			Copies:  []templatecontracts.ResolvedDesignCopy{{From: filepath.Join(kit.Path, "DESIGN.md"), To: filepath.Join(destination, "DESIGN.md")}},
		}, nil
	}
	adapter, ok := kit.Manifest.Adapters[adapterID]
	if !ok {
		return templatecontracts.ResolvedDesign{}, fmt.Errorf("design kit %s does not provide required adapter %s for template %s", selection, adapterID, info.Name)
	}
	adapterRel, ok := cleanRelativePath(adapter.Path)
	if !ok {
		return templatecontracts.ResolvedDesign{}, fmt.Errorf("design kit %s adapter %s path must be relative", selection, adapterID)
	}
	adapterDir := filepath.Join(kit.Path, adapterRel)
	adapterManifest, err := loadDesignAdapterManifest(adapterDir)
	if err != nil {
		return templatecontracts.ResolvedDesign{}, fmt.Errorf("load design adapter %s: %w", adapterID, err)
	}
	resolved := templatecontracts.ResolvedDesign{
		KitID:     kit.ID,
		KitName:   kit.Manifest.Name,
		Version:   kit.Manifest.Version,
		AdapterID: adapterID,
		Copies: []templatecontracts.ResolvedDesignCopy{
			{From: filepath.Join(kit.Path, "DESIGN.md"), To: filepath.Join(destination, "DESIGN.md")},
		},
	}
	for index, rule := range adapterManifest.Copy {
		from, ok := cleanRelativePath(renderTemplateString(rule.From, values))
		if !ok {
			return templatecontracts.ResolvedDesign{}, fmt.Errorf("design adapter %s copy[%d].from must stay inside the adapter", adapterID, index)
		}
		to, ok := cleanRelativePath(renderTemplateString(rule.To, values))
		if !ok {
			return templatecontracts.ResolvedDesign{}, fmt.Errorf("design adapter %s copy[%d].to must be scenario-relative", adapterID, index)
		}
		resolved.Copies = append(resolved.Copies, templatecontracts.ResolvedDesignCopy{
			From: filepath.Join(adapterDir, from),
			To:   filepath.Join(destination, to),
		})
	}
	return resolved, nil
}

func preflightDesignCopies(resolved templatecontracts.ResolvedDesign, force bool) error {
	for _, copy := range resolved.Copies {
		if !fileExists(copy.From) {
			return fmt.Errorf("design copy source is missing: %s", copy.From)
		}
		if stat, err := os.Stat(copy.To); err == nil && stat != nil && !force {
			return fmt.Errorf("design copy target already exists: %s (use --force to overwrite)", copy.To)
		}
	}
	return nil
}

func preflightDesignTemplateCollisions(templateDir, destination string, resolved templatecontracts.ResolvedDesign) error {
	if strings.TrimSpace(resolved.KitID) == "" {
		return nil
	}
	for _, copy := range resolved.Copies {
		rel, err := filepath.Rel(destination, copy.To)
		if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			return fmt.Errorf("design copy target escapes scenario destination: %s", copy.To)
		}
		templatePath := filepath.Join(templateDir, rel)
		if fileExists(templatePath) {
			// The canonical scenario token ramp is deliberately present in
			// React templates so static template consumers can inspect it. The
			// selected design adapter remains the generator-time owner of that
			// path, including its palette-specific values, so this overlap is
			// intentional rather than a template/design collision.
			if filepath.ToSlash(rel) == "ui/src/design-tokens.css" {
				continue
			}
			return fmt.Errorf("design copy target %s collides with template file %s", copy.To, templatePath)
		}
	}
	return nil
}

func copyDesignAssets(resolved templatecontracts.ResolvedDesign, values map[string]string) error {
	for _, copy := range resolved.Copies {
		if err := copyDesignFile(copy.From, copy.To, values); err != nil {
			return fmt.Errorf("copy design asset %s -> %s: %w", copy.From, copy.To, err)
		}
	}
	return nil
}

func formatDesignValidationIssues(issues []templatecontracts.DesignValidationIssue) string {
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		scope := issue.Kit
		if issue.Adapter != "" {
			scope += "/" + issue.Adapter
		}
		if issue.Path != "" {
			scope += " " + issue.Path
		}
		if scope == "" {
			parts = append(parts, issue.Message)
		} else {
			parts = append(parts, scope+": "+issue.Message)
		}
	}
	return strings.Join(parts, "; ")
}

func cleanRelativePath(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false
	}
	return clean, true
}

func copyDesignFile(src, dest string, values map[string]string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if looksLikeTextFile(data) {
		data = []byte(renderTemplateString(string(data), values))
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, info.Mode())
}

func fileExists(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && !stat.IsDir()
}
