package adoptions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"react-component-library/internal/librarywalk"

	"github.com/vrooli/api-core/storage"
)

// FindImportSites performs a narrow, deterministic import-specifier scan. It
// intentionally reports only direct static/dynamic/require imports; callers
// get actionable replacement evidence without pretending to build a full TS
// dependency graph here.
func (r *FSScenarioFileReader) FindImportSites(ctx context.Context, scenario, adoptedPath string) ([]string, error) {
	target, err := r.resolve(scenario, adoptedPath)
	if err != nil {
		return nil, err
	}
	base := filepath.Join(r.root, scenario)
	sites := make([]string, 0)
	err = librarywalk.WalkContext(ctx, base, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || path == target {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			return nil
		}
		relTarget, err := filepath.Rel(filepath.Dir(path), target)
		if err != nil {
			return err
		}
		module := strings.TrimSuffix(filepath.ToSlash(relTarget), filepath.Ext(relTarget))
		if !strings.HasPrefix(module, ".") {
			module = "./" + module
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(content)
		if strings.Contains(text, `"`+module+`"`) || strings.Contains(text, `'`+module+`'`) || strings.Contains(text, `"`+filepath.ToSlash(relTarget)+`"`) || strings.Contains(text, `'`+filepath.ToSlash(relTarget)+`'`) {
			rel, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			sites = append(sites, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("find import sites for %q: %w", adoptedPath, err)
	}
	sort.Strings(sites)
	return sites, nil
}

func (r *FSScenarioFileReader) RewriteImportSites(ctx context.Context, scenario, adoptedPath, replacement string) ([]string, error) {
	base := filepath.Join(r.root, scenario)
	updated := make([]string, 0)
	if strings.HasPrefix(replacement, linkedPackageName+"/") {
		modulePrefix := replacement
		if slash := strings.LastIndex(modulePrefix, "/"); slash >= 0 {
			modulePrefix = modulePrefix[:slash+1]
		}
		// A row records the adopter-owned provenance path, while source files
		// import the package subpath. Rewrite the whole component family, not
		// only the row's current exact version, so a repeated governed Link
		// repairs stale package imports as well.
		oldModulePattern := regexp.MustCompile(`(["'])` + regexp.QuoteMeta(modulePrefix) + `[^"']+(["'])`)
		err := librarywalk.WalkInfo(ctx, base, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() || strings.Contains(filepath.ToSlash(path), "/node_modules/") || !isSourceFile(path) {
				return nil
			}
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			text := string(body)
			rewritten := oldModulePattern.ReplaceAllString(text, `$1`+replacement+`$2`)
			if rewritten == text {
				return nil
			}
			relative, relErr := filepath.Rel(base, path)
			if relErr != nil {
				return relErr
			}
			root, openErr := openScenarioRoot(base)
			if openErr != nil {
				return openErr
			}
			writeErr := storage.WriteFileAtomicInRoot(root, relative, []byte(rewritten), 0o644)
			closeErr := root.Close()
			if writeErr != nil {
				return writeErr
			}
			if closeErr != nil {
				return closeErr
			}
			updated = append(updated, filepath.ToSlash(relative))
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("rewrite package import sites for %q: %w", adoptedPath, err)
		}
	}
	if strings.HasPrefix(filepath.ToSlash(adoptedPath), "./") {
		sort.Strings(updated)
		return updated, nil
	}
	sites, err := r.FindImportSites(ctx, scenario, adoptedPath)
	if err != nil {
		return nil, err
	}
	target, err := r.resolve(scenario, adoptedPath)
	if err != nil {
		return nil, err
	}
	for _, site := range sites {
		absolute := filepath.Join(base, filepath.FromSlash(site))
		content, readErr := os.ReadFile(absolute)
		if readErr != nil {
			return nil, fmt.Errorf("read import site %q: %w", site, readErr)
		}
		relTarget, relErr := filepath.Rel(filepath.Dir(absolute), target)
		if relErr != nil {
			return nil, relErr
		}
		module := strings.TrimSuffix(filepath.ToSlash(relTarget), filepath.Ext(relTarget))
		if !strings.HasPrefix(module, ".") {
			module = "./" + module
		}
		body := string(content)
		for _, quote := range []string{"\"", "'"} {
			body = strings.ReplaceAll(body, quote+module+quote, quote+replacement+quote)
		}
		if body == string(content) {
			continue
		}
		root, openErr := openScenarioRoot(base)
		if openErr != nil {
			return nil, openErr
		}
		writeErr := storage.WriteFileAtomicInRoot(root, filepath.FromSlash(site), []byte(body), 0o644)
		closeErr := root.Close()
		if writeErr != nil {
			return nil, fmt.Errorf("rewrite import site %q: %w", site, writeErr)
		}
		if closeErr != nil {
			return nil, closeErr
		}
		updated = append(updated, site)
	}
	return updated, nil
}
