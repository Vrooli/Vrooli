package templateengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

var unresolvedTemplatePattern = regexp.MustCompile(`\{\{[A-Z0-9_]+\}\}`)

func copyTemplate(templateDir, destination string, values map[string]string, manifest templatecontracts.TemplateManifest) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	if base := strings.TrimSpace(manifest.BaseTemplate); base != "" {
		baseDir := filepath.Join(filepath.Dir(templateDir), base)
		baseData, err := os.ReadFile(filepath.Join(baseDir, "template.json"))
		if err != nil {
			return fmt.Errorf("read base template %q: %w", base, err)
		}
		var baseManifest templatecontracts.TemplateManifest
		if err := json.Unmarshal(baseData, &baseManifest); err != nil {
			return fmt.Errorf("decode base template %q: %w", base, err)
		}
		baseManifest.CopyExcludes = append(baseManifest.CopyExcludes, manifest.BaseCopyExcludes...)
		if err := copyTemplate(baseDir, destination, values, baseManifest); err != nil {
			return fmt.Errorf("copy base template %q: %w", base, err)
		}
	}
	return walkTemplateEmissions(templateDir, manifest, func(relPath, absPath string, entry fs.DirEntry) error {
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
		data, err := os.ReadFile(absPath)
		if err != nil {
			return err
		}
		if looksLikeTextFile(data) {
			data = []byte(renderTemplateString(string(data), values))
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}

// walkTemplateEmissions iterates the template tree, invoking fn for every
// entry (file or directory) that would be emitted into a generated scenario
// destination. The skip rules — relocation sources, manifest.CopyExcludes,
// hard-coded dirs (node_modules/dist/build/coverage/.turbo/.vite), and the
// always-skipped files (.DS_Store, template.json) — are applied once here so
// copyTemplate and the content-hash walker stay in lockstep.
//
// relPath is the template-relative path (filepath-style separators). For
// directories, fn is called before its children are visited.
func walkTemplateEmissions(templateDir string, manifest templatecontracts.TemplateManifest, fn func(relPath, absPath string, entry fs.DirEntry) error) error {
	relocSources := make(map[string]struct{}, len(manifest.Relocations))
	for _, reloc := range manifest.Relocations {
		from := strings.TrimSpace(reloc.From)
		if from == "" {
			continue
		}
		relocSources[filepath.Clean(filepath.FromSlash(from))] = struct{}{}
	}
	copyExcludes := make(map[string]struct{}, len(manifest.CopyExcludes))
	for _, exclude := range manifest.CopyExcludes {
		exclude = strings.TrimSpace(exclude)
		if exclude == "" {
			continue
		}
		copyExcludes[filepath.Clean(filepath.FromSlash(exclude))] = struct{}{}
	}
	return filepath.WalkDir(templateDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == templateDir {
			return nil
		}
		if entry.IsDir() && shouldSkipTemplateCopyDir(entry.Name()) {
			return filepath.SkipDir
		}
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		cleanRel := filepath.Clean(relPath)
		if entry.IsDir() {
			if _, skip := relocSources[cleanRel]; skip {
				return filepath.SkipDir
			}
			if _, skip := copyExcludes[cleanRel]; skip {
				return filepath.SkipDir
			}
		}
		if filepath.Base(path) == ".DS_Store" || relPath == "template.json" {
			return nil
		}
		if _, skip := copyExcludes[cleanRel]; skip {
			return nil
		}
		return fn(relPath, path, entry)
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

func shouldSkipTemplateCopyDir(name string) bool {
	switch strings.TrimSpace(name) {
	case "node_modules", "dist", "build", "coverage", ".turbo", ".vite":
		return true
	default:
		return false
	}
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
