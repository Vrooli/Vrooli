package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"brand-manager/internal/brandsurface"
)

// Relocating a branding file breaks any service worker that precaches it at
// its old URL: cache.addAll rejects on a single 404, so the worker never
// installs and the offline app shell is gone. 37 scenarios precache their root
// manifest this way (`APP_SHELL_URLS = ["./", "./site.webmanifest"]`), so the
// relocation fixer rewrites those entries with the move, and refuses a
// scenario whose precache list it cannot read statically rather than guessing.

const serviceWorkerSourceRel = brandsurface.RootAssetSourceDir + "/sw.js"

var (
	serviceWorkerRegisterRE = regexp.MustCompile(`serviceWorker\s*\.\s*register\(\s*["'\x60]([^"'\x60]+)["'\x60]`)
	precacheCallRE          = regexp.MustCompile(`\.addAll\(\s*`)
	identifierRE            = regexp.MustCompile(`^[A-Za-z_$][\w$]*`)
	stringListEntryRE       = regexp.MustCompile(`^\s*(?:"[^"\\]*"|'[^'\\]*'|\x60[^\x60$\\]*\x60)\s*$`)
)

// serviceWorkerPlan is the rewrite of every service worker that names a moved
// file, computed before anything changes so a refusal leaves the tree intact.
type serviceWorkerPlan struct {
	content map[string]string // scenario-relative path -> rewritten source
	refs    []publicRefFix    // for the preview, prefixed with the file
	refusal string
}

// planServiceWorkerRepoint rewrites, in each service worker the scenario
// ships or registers, the same-origin string literals that name a relocated
// root file ("./x", "/x" or "x") to the same form under public/.
func planServiceWorkerRepoint(root string, movedRootFiles []string) serviceWorkerPlan {
	plan := serviceWorkerPlan{content: map[string]string{}}
	if len(movedRootFiles) == 0 {
		return plan
	}
	moved := make([]string, 0, len(movedRootFiles))
	for _, rel := range movedRootFiles {
		moved = append(moved, filepath.Base(rel))
	}
	for _, rel := range serviceWorkerSources(root) {
		source, ok := readFile(root, rel)
		if !ok {
			continue
		}
		if reason := unreadablePrecache(source); reason != "" {
			if mentionsAny(source, moved) {
				plan.refusal = fmt.Sprintf("%s: %s, so the relocated files' cache entries cannot be rewritten safely", rel, reason)
				return plan
			}
			continue
		}
		updated := source
		for _, base := range moved {
			literal := regexp.MustCompile("([\"'\\x60])(\\./|/)?" + regexp.QuoteMeta(base) + "([\"'\\x60])")
			rewritten := 0
			updated = literal.ReplaceAllStringFunc(updated, func(match string) string {
				parts := literal.FindStringSubmatch(match)
				if parts[1] != parts[3] {
					return match
				}
				rewritten++
				from := parts[2] + base
				to := parts[2] + brandsurface.PublicRefPrefix + base
				plan.refs = append(plan.refs, publicRefFix{From: rel + " " + from, To: rel + " " + to})
				return parts[1] + to + parts[3]
			})
			if unpublishedMentions(source, base) != rewritten {
				plan.refusal = fmt.Sprintf("%s names %s in a form other than a plain string literal, so the fixer cannot rewrite it safely", rel, base)
				return plan
			}
		}
		if updated != source {
			plan.content[rel] = updated
		}
	}
	return plan
}

// serviceWorkerSources lists ui/public/sw.js and every worker that index.html
// or the UI source registers by literal path, as scenario-relative paths.
func serviceWorkerSources(root string) []string {
	found := map[string]bool{}
	if _, ok := readFile(root, serviceWorkerSourceRel); ok {
		found[serviceWorkerSourceRel] = true
	}
	var registrants []string
	if html, ok := loadIndexHTML(root); ok {
		registrants = append(registrants, html)
	}
	_ = filepath.WalkDir(filepath.Join(root, "ui", "src"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".ts", ".tsx", ".js", ".jsx":
			if data, readErr := os.ReadFile(path); readErr == nil && strings.Contains(string(data), "serviceWorker") {
				registrants = append(registrants, string(data))
			}
		}
		return nil
	})
	for _, text := range registrants {
		for _, match := range serviceWorkerRegisterRE.FindAllStringSubmatch(text, -1) {
			target := strings.SplitN(match[1], "?", 2)[0]
			target = strings.TrimPrefix(strings.TrimPrefix(target, "./"), "/")
			if target == "" || strings.Contains(target, "://") {
				continue
			}
			rel := brandsurface.RootAssetSourceDir + "/" + filepath.ToSlash(target)
			if _, ok := readFile(root, rel); ok {
				found[rel] = true
			}
		}
	}
	out := make([]string, 0, len(found))
	for rel := range found {
		out = append(out, rel)
	}
	sort.Strings(out)
	return out
}

// unreadablePrecache explains why a worker's precache list is not a static
// array of string literals, or returns "" when every addAll call is readable.
func unreadablePrecache(source string) string {
	if strings.Contains(source, "__WB_MANIFEST") || strings.Contains(source, "importScripts(") {
		return "its precache list comes from a build manifest or an imported script"
	}
	for _, call := range precacheCallRE.FindAllStringIndex(source, -1) {
		argument := source[call[1]:]
		var list string
		switch {
		case strings.HasPrefix(argument, "["):
			list, _ = bracketed(argument)
		default:
			name := identifierRE.FindString(argument)
			if name == "" {
				return "a cache.addAll argument is computed"
			}
			binding := regexp.MustCompile(`(?:const|let|var)\s+` + regexp.QuoteMeta(name) + `\s*=\s*\[`).FindStringIndex(source)
			if binding == nil {
				return fmt.Sprintf("the precache list %s is not an array literal", name)
			}
			list, _ = bracketed(source[binding[1]-1:])
		}
		if list == "" {
			return "a precache array is not closed"
		}
		for _, entry := range strings.Split(list, ",") {
			if strings.TrimSpace(entry) == "" {
				continue
			}
			if !stringListEntryRE.MatchString(entry) {
				return fmt.Sprintf("the precache entry %q is not a string literal", strings.TrimSpace(entry))
			}
		}
	}
	return ""
}

// bracketed returns the text between a leading "[" and its matching "]".
func bracketed(text string) (string, bool) {
	depth := 0
	for index, char := range text {
		switch char {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return text[1:index], true
			}
		}
	}
	return "", false
}

func mentionsAny(source string, names []string) bool {
	for _, name := range names {
		if strings.Contains(source, name) {
			return true
		}
	}
	return false
}

// unpublishedMentions counts the places source names base outside the public
// surface; a reference already under public/ needs no rewrite.
func unpublishedMentions(source, base string) int {
	count := 0
	for offset := 0; ; {
		index := strings.Index(source[offset:], base)
		if index < 0 {
			return count
		}
		at := offset + index
		if !strings.HasSuffix(source[:at], brandsurface.PublicRefPrefix) {
			count++
		}
		offset = at + len(base)
	}
}
