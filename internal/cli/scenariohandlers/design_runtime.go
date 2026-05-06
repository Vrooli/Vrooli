package scenariohandlers

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive // scenariohandlers is a thin glue layer over scenariocli; dot-import keeps wiring readable.
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/config"
)

func DesignCommandHandler[C any](deps HandlerDeps[C]) func(C, []string) error {
	return func(ctx C, args []string) error {
		if len(args) == 0 {
			return bindGlobal(deps.Stdout,
				func(ctx C, args []string) (DesignListRequest, error) { return ParseDesignListRequest(args) },
				func(ctx C, req DesignListRequest) (cliout.Format, []DesignKitInfo, error) {
					return runDesignList(deps, ctx, req)
				},
				RenderDesignListResponse,
			)(ctx, nil)
		}
		switch commandtree.NormalizeName(args[0]) {
		case "list":
			return bindGlobal(deps.Stdout,
				func(ctx C, args []string) (DesignListRequest, error) { return ParseDesignListRequest(args) },
				func(ctx C, req DesignListRequest) (cliout.Format, []DesignKitInfo, error) {
					return runDesignList(deps, ctx, req)
				},
				RenderDesignListResponse,
			)(ctx, args[1:])
		case "show":
			return bindGlobal(deps.Stdout,
				func(ctx C, args []string) (DesignShowRequest, error) { return ParseDesignShowRequest(args) },
				func(ctx C, req DesignShowRequest) (cliout.Format, DesignKitInfo, error) {
					return runDesignShow(deps, ctx, req)
				},
				RenderDesignShowResponse,
			)(ctx, args[1:])
		case "validate":
			req, err := ParseDesignValidateRequest(args[1:])
			if err != nil {
				return rootcli.UsageErrorf("scenario design validate", "%s", err.Error())
			}
			format, report, err := runDesignValidate(deps, ctx, req)
			if err != nil {
				return err
			}
			if err := RenderDesignValidateResponse(deps.Stdout(ctx), format, report); err != nil {
				return err
			}
			if len(report.Issues) > 0 {
				return fmt.Errorf("scenario design validation failed")
			}
			return nil
		case "--help", "-h":
			RenderDesignHelp(deps.Stdout(ctx))
			return nil
		default:
			return rootcli.UsageErrorf("scenario design", "unknown scenario design command: %s", args[0])
		}
	}
}

func runDesignList[C any](deps HandlerDeps[C], ctx C, _ DesignListRequest) (cliout.Format, []DesignKitInfo, error) {
	kits, err := loadDesignKits(deps.Root(ctx))
	if err != nil {
		return "", nil, err
	}
	format, err := deps.OutputFormat(ctx)
	if err != nil {
		return "", nil, err
	}
	return format, kits, nil
}

func runDesignShow[C any](deps HandlerDeps[C], ctx C, req DesignShowRequest) (cliout.Format, DesignKitInfo, error) {
	info, err := loadDesignKit(deps.Root(ctx), req.ID)
	if err != nil {
		return "", DesignKitInfo{}, err
	}
	format, err := deps.OutputFormat(ctx)
	if err != nil {
		return "", DesignKitInfo{}, err
	}
	return format, info, nil
}

func runDesignValidate[C any](deps HandlerDeps[C], ctx C, req DesignValidateRequest) (cliout.Format, DesignValidationReport, error) {
	var kits []DesignKitInfo
	var err error
	if strings.TrimSpace(req.ID) != "" {
		info, loadErr := loadDesignKit(deps.Root(ctx), req.ID)
		if loadErr != nil {
			return "", DesignValidationReport{}, loadErr
		}
		kits = []DesignKitInfo{info}
	} else {
		kits, err = loadDesignKits(deps.Root(ctx))
		if err != nil {
			return "", DesignValidationReport{}, err
		}
	}
	format, err := deps.OutputFormat(ctx)
	if err != nil {
		return "", DesignValidationReport{}, err
	}
	report := DesignValidationReport{Count: len(kits)}
	defaults := 0
	for _, kit := range kits {
		if kit.Manifest.Default {
			defaults++
		}
		report.Issues = append(report.Issues, validateDesignKit(kit)...)
	}
	if req.All && defaults > 1 {
		report.Issues = append(report.Issues, DesignValidationIssue{Message: "only one design kit may be marked default"})
	}
	return format, report, nil
}

func loadDesignKits(root string) ([]DesignKitInfo, error) {
	baseDir := config.DesignBaseDir(root)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	kits := make([]DesignKitInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := loadDesignKit(root, entry.Name())
		if err != nil {
			if os.IsNotExist(err) {
				kits = append(kits, DesignKitInfo{ID: entry.Name(), Path: filepath.Join(baseDir, entry.Name()), Missing: true})
				continue
			}
			return nil, err
		}
		kits = append(kits, info)
	}
	sort.Slice(kits, func(i, j int) bool { return kits[i].ID < kits[j].ID })
	return kits, nil
}

func loadDesignKit(root, id string) (DesignKitInfo, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return DesignKitInfo{}, fmt.Errorf("design kit id is required")
	}
	kitDir := filepath.Join(config.DesignBaseDir(root), id)
	stat, err := os.Stat(kitDir)
	if err != nil {
		if os.IsNotExist(err) {
			return DesignKitInfo{}, fmt.Errorf("design kit not found: %s", id)
		}
		return DesignKitInfo{}, err
	}
	if !stat.IsDir() {
		return DesignKitInfo{}, fmt.Errorf("design kit path is not a directory: %s", kitDir)
	}
	data, err := os.ReadFile(filepath.Join(kitDir, "metadata.json"))
	if err != nil {
		return DesignKitInfo{}, err
	}
	var manifest DesignKitManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return DesignKitInfo{}, err
	}
	if manifest.ID == "" {
		manifest.ID = id
	}
	if manifest.Adapters == nil {
		manifest.Adapters = map[string]DesignKitAdapter{}
	}
	return DesignKitInfo{ID: id, Path: kitDir, Manifest: manifest}, nil
}

func validateDesignKit(info DesignKitInfo) []DesignValidationIssue {
	if info.Missing {
		return []DesignValidationIssue{{Kit: info.ID, Message: "metadata.json is missing"}}
	}
	var issues []DesignValidationIssue
	if info.Manifest.ID != info.ID {
		issues = append(issues, DesignValidationIssue{Kit: info.ID, Path: "metadata.json", Message: fmt.Sprintf("metadata id %q must match folder name", info.Manifest.ID)})
	}
	for _, required := range []string{"metadata.json", "DESIGN.md"} {
		if stat, err := os.Stat(filepath.Join(info.Path, required)); err != nil || stat.IsDir() {
			msg := "required file is missing"
			if err == nil && stat.IsDir() {
				msg = "required file is a directory"
			}
			issues = append(issues, DesignValidationIssue{Kit: info.ID, Path: required, Message: msg})
		}
	}
	for id, adapter := range info.Manifest.Adapters {
		cleanPath, ok := cleanRelativePath(adapter.Path)
		if !ok {
			issues = append(issues, DesignValidationIssue{Kit: info.ID, Adapter: id, Path: adapter.Path, Message: "adapter path must be relative and stay inside the kit"})
			continue
		}
		adapterDir := filepath.Join(info.Path, cleanPath)
		if stat, err := os.Stat(adapterDir); err != nil || !stat.IsDir() {
			issues = append(issues, DesignValidationIssue{Kit: info.ID, Adapter: id, Path: adapter.Path, Message: "adapter path is missing or not a directory"})
			continue
		}
		manifest, err := loadDesignAdapterManifest(adapterDir)
		if err != nil {
			issues = append(issues, DesignValidationIssue{Kit: info.ID, Adapter: id, Path: filepath.ToSlash(filepath.Join(cleanPath, "adapter.json")), Message: err.Error()})
			continue
		}
		if manifest.ID != "" && manifest.ID != id {
			issues = append(issues, DesignValidationIssue{Kit: info.ID, Adapter: id, Path: filepath.ToSlash(filepath.Join(cleanPath, "adapter.json")), Message: fmt.Sprintf("adapter id %q must match metadata adapter key", manifest.ID)})
		}
		for index, rule := range manifest.Copy {
			if _, ok := cleanRelativePath(rule.From); !ok {
				issues = append(issues, DesignValidationIssue{Kit: info.ID, Adapter: id, Path: fmt.Sprintf("copy[%d].from", index), Message: "copy source must be relative and stay inside the adapter"})
				continue
			}
			if _, ok := cleanRelativePath(rule.To); !ok {
				issues = append(issues, DesignValidationIssue{Kit: info.ID, Adapter: id, Path: fmt.Sprintf("copy[%d].to", index), Message: "copy destination must be scenario-relative"})
				continue
			}
			if stat, err := os.Stat(filepath.Join(adapterDir, filepath.FromSlash(rule.From))); err != nil || stat.IsDir() {
				issues = append(issues, DesignValidationIssue{Kit: info.ID, Adapter: id, Path: rule.From, Message: "copy source is missing or not a file"})
			}
		}
	}
	return issues
}

func loadDesignAdapterManifest(adapterDir string) (DesignAdapterManifest, error) {
	data, err := os.ReadFile(filepath.Join(adapterDir, "adapter.json"))
	if err != nil {
		return DesignAdapterManifest{}, err
	}
	var manifest DesignAdapterManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return DesignAdapterManifest{}, err
	}
	return manifest, nil
}

func resolveDesign(root string, info TemplateInfo, requested, destination string, values map[string]string) (ResolvedDesign, error) {
	templateDesign := info.Manifest.Design
	selection := strings.TrimSpace(requested)
	if selection == "" {
		selection = strings.TrimSpace(templateDesign.Default)
	}
	if strings.EqualFold(selection, "none") {
		if templateDesign.Required {
			return ResolvedDesign{}, fmt.Errorf("template %s requires a design kit; --design none is not allowed", info.Name)
		}
		return ResolvedDesign{}, nil
	}
	if selection == "" {
		if templateDesign.Required {
			return ResolvedDesign{}, fmt.Errorf("template %s requires a design kit but declares no default", info.Name)
		}
		return ResolvedDesign{}, nil
	}
	kit, err := loadDesignKit(root, selection)
	if err != nil {
		return ResolvedDesign{}, err
	}
	if issues := validateDesignKit(kit); len(issues) > 0 {
		return ResolvedDesign{}, fmt.Errorf("design kit %s is invalid: %s", selection, formatDesignValidationIssues(issues))
	}
	adapterID := strings.TrimSpace(templateDesign.Adapter)
	if adapterID == "" {
		return ResolvedDesign{
			KitID:   kit.ID,
			KitName: kit.Manifest.Name,
			Version: kit.Manifest.Version,
			Copies:  []ResolvedDesignCopy{{From: filepath.Join(kit.Path, "DESIGN.md"), To: filepath.Join(destination, "DESIGN.md")}},
		}, nil
	}
	adapter, ok := kit.Manifest.Adapters[adapterID]
	if !ok {
		return ResolvedDesign{}, fmt.Errorf("design kit %s does not provide required adapter %s for template %s", selection, adapterID, info.Name)
	}
	adapterRel, ok := cleanRelativePath(adapter.Path)
	if !ok {
		return ResolvedDesign{}, fmt.Errorf("design kit %s adapter %s path must be relative", selection, adapterID)
	}
	adapterDir := filepath.Join(kit.Path, adapterRel)
	adapterManifest, err := loadDesignAdapterManifest(adapterDir)
	if err != nil {
		return ResolvedDesign{}, fmt.Errorf("load design adapter %s: %w", adapterID, err)
	}
	resolved := ResolvedDesign{
		KitID:     kit.ID,
		KitName:   kit.Manifest.Name,
		Version:   kit.Manifest.Version,
		AdapterID: adapterID,
		Copies: []ResolvedDesignCopy{
			{From: filepath.Join(kit.Path, "DESIGN.md"), To: filepath.Join(destination, "DESIGN.md")},
		},
	}
	for index, rule := range adapterManifest.Copy {
		from, ok := cleanRelativePath(renderTemplateString(rule.From, values))
		if !ok {
			return ResolvedDesign{}, fmt.Errorf("design adapter %s copy[%d].from must stay inside the adapter", adapterID, index)
		}
		to, ok := cleanRelativePath(renderTemplateString(rule.To, values))
		if !ok {
			return ResolvedDesign{}, fmt.Errorf("design adapter %s copy[%d].to must be scenario-relative", adapterID, index)
		}
		resolved.Copies = append(resolved.Copies, ResolvedDesignCopy{
			From: filepath.Join(adapterDir, from),
			To:   filepath.Join(destination, to),
		})
	}
	return resolved, nil
}

func preflightDesignCopies(resolved ResolvedDesign, force bool) error {
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

func preflightDesignTemplateCollisions(templateDir, destination string, resolved ResolvedDesign) error {
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
			return fmt.Errorf("design copy target %s collides with template file %s", copy.To, templatePath)
		}
	}
	return nil
}

func copyDesignAssets(resolved ResolvedDesign, values map[string]string) error {
	for _, copy := range resolved.Copies {
		if err := copyDesignFile(copy.From, copy.To, values); err != nil {
			return fmt.Errorf("copy design asset %s -> %s: %w", copy.From, copy.To, err)
		}
	}
	return nil
}

func formatDesignValidationIssues(issues []DesignValidationIssue) string {
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

func directoryExists(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}

func fileExists(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && !stat.IsDir()
}

func walkDesignFiles(root string, visit func(path string, entry fs.DirEntry) error) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return visit(path, entry)
	})
}
