package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"brand-manager/internal/brandsurface"
)

// surface names a scenario sub-surface a rule may require. Rules that read UI
// files only fire when the UI surface is present, so a CLI/API-only scenario
// never collects false-positive PWA/visual findings.
type surface string

const (
	surfaceUI  surface = "ui"
	surfaceCLI surface = "cli"
)

// indexHTMLRel is the canonical Vite entry document where head branding lives.
const indexHTMLRel = "ui/index.html"

// webManifestCandidates are the manifest paths a scenario may ship, in priority
// order. The first that exists is the active manifest. The /public/-convention
// locations (ui/public/public/...) rank first so a scenario that has adopted the
// convention resolves its relocated manifest, not a stale root copy.
var webManifestCandidates = []string{
	"ui/public/public/site.webmanifest",
	"ui/public/public/manifest.json",
	"ui/public/public/manifest.webmanifest",
	"ui/public/site.webmanifest",
	"ui/public/manifest.json",
	"ui/public/manifest.webmanifest",
	"ui/manifest.json",
}

// scanContext caches a scenario's on-disk branding artifacts so the rule set
// reads each file at most once and shares parsed views (head document, manifest,
// design tokens) across rules.
type scanContext struct {
	scenario string
	root     string

	files map[string]*fileEntry

	surfaceOnce map[surface]bool
	surfaceDone map[surface]bool

	headOnce *headDoc
	headRead bool

	service     brandsurface.Surface
	serviceRead bool
}

type fileEntry struct {
	content string
	present bool
}

func newScanContext(scenario, root string) *scanContext {
	return &scanContext{
		scenario:    scenario,
		root:        root,
		files:       map[string]*fileEntry{},
		surfaceOnce: map[surface]bool{},
		surfaceDone: map[surface]bool{},
	}
}

// read returns the (cached) content of a scenario-relative file and whether it
// exists/was readable.
func (c *scanContext) read(rel string) (string, bool) {
	if e, ok := c.files[rel]; ok {
		return e.content, e.present
	}
	content, present := readFile(c.root, rel)
	c.files[rel] = &fileEntry{content: content, present: present}
	return content, present
}

// dirExists reports whether a scenario-relative directory exists.
func (c *scanContext) dirExists(rel string) bool {
	info, err := os.Stat(filepath.Join(c.root, filepath.FromSlash(rel)))
	return err == nil && info.IsDir()
}

// has reports whether the named surface is present in the scenario tree.
func (c *scanContext) has(s surface) bool {
	if c.surfaceDone[s] {
		return c.surfaceOnce[s]
	}
	var present bool
	switch s {
	case surfaceUI:
		present = c.dirExists("ui")
	case surfaceCLI:
		present = c.dirExists("cli")
	}
	c.surfaceOnce[s] = present
	c.surfaceDone[s] = true
	return present
}

// surfaces reports whether every required surface is present.
func (c *scanContext) hasAll(required []surface) bool {
	for _, s := range required {
		if !c.has(s) {
			return false
		}
	}
	return true
}

// identity returns the parsed branding identity (service.json), cached.
func (c *scanContext) identity() brandsurface.Surface {
	if !c.serviceRead {
		content, _ := c.read(".vrooli/service.json")
		c.service = brandsurface.ParseService(content)
		c.serviceRead = true
	}
	return c.service
}

// head returns the parsed ui/index.html head document (empty/absent when there
// is no index.html).
func (c *scanContext) head() *headDoc {
	if !c.headRead {
		content, ok := c.read(indexHTMLRel)
		c.headOnce = parseHead(content, ok)
		c.headRead = true
	}
	return c.headOnce
}

// manifest returns the active web-app manifest (path, parsed object, raw,
// present). present is false when no manifest file exists.
func (c *scanContext) manifest() (rel string, obj map[string]any, raw string, present bool) {
	for _, cand := range webManifestCandidates {
		if content, ok := c.read(cand); ok {
			m := map[string]any{}
			_ = json.Unmarshal([]byte(content), &m)
			return cand, m, content, true
		}
	}
	return webManifestCandidates[0], map[string]any{}, "", false
}

// tokens returns the parsed light-scheme design tokens and whether the token
// file is present.
func (c *scanContext) tokens() (map[string]string, bool) {
	content, ok := c.read(designSystemCSSRel)
	if !ok {
		return map[string]string{}, false
	}
	return cssVarsForScheme(content, schemeLight), true
}

// tokenContent returns the raw design-token CSS and whether it is present.
func (c *scanContext) tokenContent() (string, bool) { return c.read(designSystemCSSRel) }

// uiCSSContains reports whether any UI stylesheet (.css/.scss) under ui/ contains
// needle (case-sensitive). Used by the safe-area rule to confirm real
// env(safe-area-inset-*) usage exists, not just a viewport flag.
func (c *scanContext) uiCSSContains(needle string) bool {
	found := false
	_ = filepath.WalkDir(filepath.Join(c.root, "ui"), func(path string, d os.DirEntry, err error) error {
		if err != nil || found || d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".css", ".scss":
			if b, err := os.ReadFile(path); err == nil && strings.Contains(string(b), needle) {
				found = true
			}
		}
		return nil
	})
	return found
}

// appCSSDirs are the scenario's OWN stylesheet roots. Unlike a bare ui/ walk
// this excludes built dependencies (ui/node_modules), so a library's animations
// or @font-face never get attributed to the scenario.
var appCSSDirs = []string{"ui/src", "ui/public"}

// appCSSMatches scans the scenario's own .css/.scss files once and reports which
// of needles appears in any of them (case-insensitive substring). It prunes
// node_modules/dist/build so only authored styles are considered.
func (c *scanContext) appCSSMatches(needles []string) map[string]bool {
	lowered := make([]string, len(needles))
	for i, n := range needles {
		lowered[i] = strings.ToLower(n)
	}
	matched := map[string]bool{}
	for _, dir := range appCSSDirs {
		base := filepath.Join(c.root, filepath.FromSlash(dir))
		_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if skipScanDir(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".css", ".scss":
				b, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				s := strings.ToLower(string(b))
				for i, n := range lowered {
					if !matched[needles[i]] && strings.Contains(s, n) {
						matched[needles[i]] = true
					}
				}
			}
			return nil
		})
	}
	return matched
}

// --- head document ---------------------------------------------------------

// headDoc is a lightweight structured view of an HTML <head>. The scenarios are
// generated Vite templates with a predictable, single-line-per-tag head, so a
// regex/string view is sufficient (and matches the rest of this package's
// style); we never need a full HTML tree.
type headDoc struct {
	present bool
	raw     string
	title   string
	metas   []metaTag
	links   []linkTag
}

type metaTag struct {
	kind    brandsurface.TagKind // name | property
	key     string
	content string
	media   string
}

type linkTag struct {
	rel   string
	href  string
	sizes string
}

var (
	titleRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	metaRe  = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	linkRe  = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	attrRe  = regexp.MustCompile(`(?is)([a-zA-Z-]+)\s*=\s*"([^"]*)"`)
)

func parseHead(content string, present bool) *headDoc {
	h := &headDoc{present: present, raw: content}
	if !present {
		return h
	}
	if m := titleRe.FindStringSubmatch(content); m != nil {
		h.title = strings.TrimSpace(m[1])
	}
	for _, tag := range metaRe.FindAllString(content, -1) {
		attrs := parseAttrs(tag)
		mt := metaTag{content: attrs["content"], media: attrs["media"]}
		if v, ok := attrs["name"]; ok {
			mt.kind, mt.key = brandsurface.KindName, v
		} else if v, ok := attrs["property"]; ok {
			mt.kind, mt.key = brandsurface.KindProperty, v
		} else {
			continue // charset/viewport-only or http-equiv handled separately
		}
		h.metas = append(h.metas, mt)
	}
	for _, tag := range linkRe.FindAllString(content, -1) {
		attrs := parseAttrs(tag)
		h.links = append(h.links, linkTag{rel: attrs["rel"], href: attrs["href"], sizes: attrs["sizes"]})
	}
	return h
}

func parseAttrs(tag string) map[string]string {
	out := map[string]string{}
	for _, m := range attrRe.FindAllStringSubmatch(tag, -1) {
		out[strings.ToLower(m[1])] = m[2]
	}
	return out
}

// metaByName returns the content of the first `<meta name=key>` (case-insensitive
// key) and whether it exists.
func (h *headDoc) metaByName(key string) (string, bool) {
	return h.metaBy(brandsurface.KindName, key)
}

// metaByProperty returns the content of the first `<meta property=key>`.
func (h *headDoc) metaByProperty(key string) (string, bool) {
	return h.metaBy(brandsurface.KindProperty, key)
}

func (h *headDoc) metaBy(kind brandsurface.TagKind, key string) (string, bool) {
	for _, m := range h.metas {
		if m.kind == kind && strings.EqualFold(m.key, key) {
			return m.content, true
		}
	}
	return "", false
}

// metasByName returns every `<meta name=key>` (theme-color ships light+dark
// variants distinguished by media).
func (h *headDoc) metasByName(key string) []metaTag {
	var out []metaTag
	for _, m := range h.metas {
		if m.kind == brandsurface.KindName && strings.EqualFold(m.key, key) {
			out = append(out, m)
		}
	}
	return out
}

// rawMeta returns the literal source of the first `<meta name|property=key>`
// occurrence (used by viewport, which carries no name/property and so is matched
// against raw).
func (h *headDoc) viewportContent() (string, bool) {
	re := regexp.MustCompile(`(?is)<meta\b[^>]*\bname\s*=\s*"viewport"[^>]*>`)
	tag := re.FindString(h.raw)
	if tag == "" {
		return "", false
	}
	return parseAttrs(tag)["content"], true
}

// linkByRel returns the first `<link rel=rel>` and whether it exists.
func (h *headDoc) linkByRel(rel string) (linkTag, bool) {
	for _, l := range h.links {
		if strings.EqualFold(l.rel, rel) {
			return l, true
		}
	}
	return linkTag{}, false
}
