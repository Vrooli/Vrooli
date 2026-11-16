package courier

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vrooli/api-core/operatordecision"

	"research-desk/internal/home"
)

// Every decision goes out with a page the operator can read cold. An item whose
// `Explainer:` line names a page sends that page, with the images it names by a
// local path inlined as data: URIs, because the Hub's page loads nothing from
// elsewhere. Every other item, and one whose page cannot travel, sends the
// summary page the desk builds from the item itself. The page rides inside
// the decision event, so one event bounds it (operatordecision.MaxExplainerBytes,
// measured as JSON encodes it).

// maxPageBytes is the largest page file the desk reads (FILE-CONTRACT §5).
const maxPageBytes = 2 << 20

// Page is the explainer page an item's `Explainer:` line names, ready to send.
type Page struct {
	HTML string
	// SHA256 is the hash of HTML; it is in the technical details, so a revised
	// page (or image) is a changed request and the Hub's page is replaced.
	SHA256 string
	// LeftOut says which images could not travel with the page; empty when
	// every one did.
	LeftOut string
}

// Pages are the item pages of the feed, read once a pass so that every
// decision about an item sees the same page.
type Pages struct {
	byKey    map[string]*Page
	problems map[string]string
	// effects are what executes on each answer, by item key and option key,
	// read from the item's packet for its summary page.
	effects map[string]map[string]string
}

// readPages reads the page of every item that names one. A page that cannot be
// sent never stops its item: the item goes with its summary page, and the
// reason is kept to say so. research is the research home; nothing outside it
// is read, because the desk would otherwise send whatever file an item names.
func readPages(feed home.Feed, research string) Pages {
	pages := Pages{byKey: map[string]*Page{}, problems: map[string]string{}, effects: map[string]map[string]string{}}
	for i := range feed.Items {
		f := &feed.Items[i]
		if effects := readEffects(f, research); len(effects) > 0 {
			pages.effects[f.Key()] = effects
		}
		path := f.Item.ExplainerPath()
		if path == "" {
			continue
		}
		page, err := loadPage(path, research)
		switch {
		case err != nil:
			pages.problems[f.Key()] = "sent with the desk's summary page instead of its own: " + err.Error()
			continue
		case page.LeftOut != "":
			pages.problems[f.Key()] = "sent with its page, without " + page.LeftOut
		}
		pages.byKey[f.Key()] = page
	}
	return pages
}

// of is the item's own page, nil when it names none or it cannot travel.
func (p Pages) of(f *home.FeedItem) *Page { return p.byKey[f.Key()] }

// problem says what of the item's page did not travel; empty when all did.
func (p Pages) problem(f *home.FeedItem) string { return p.problems[f.Key()] }

// explainer is the page sent with the item's decision.
func (p Pages) explainer(f *home.FeedItem) string {
	if page := p.of(f); page != nil {
		return page.HTML
	}
	return summaryPage(f, p.effects[f.Key()])
}

// maxPacketBytes is the largest packet the desk reads for a summary page.
const maxPacketBytes = 256 << 10

// readEffects is what executes on each answer, from the item's packet: the
// bullets of its "## On each answer" section that start with an offered
// option key, the text of an "## On approval" section for the recommended
// key, and any section whose heading names exactly one offered key in
// backticks ("## If `hold` is chosen"). Nil when the packet says none of it
// or cannot be read inside the research home.
func readEffects(f *home.FeedItem, research string) map[string]string {
	path := f.Item.PacketPath()
	root, err := filepath.EvalSymlinks(research)
	if path == "" || err != nil {
		return nil
	}
	_, content, err := readHomeFile(path, root, maxPacketBytes)
	if err != nil {
		return nil
	}
	effects := map[string]string{}
	offered := func(key string) bool { _, ok := f.Item.Choice(key); return ok }
	put := func(key string, lines []string) {
		if text := plainMarkdown(strings.Join(lines, "\n")); key != "" && text != "" && effects[key] == "" {
			effects[key] = operatordecision.Clip(text, 700)
		}
	}
	for _, section := range strings.Split("\n"+string(content), "\n## ")[1:] {
		heading, body, _ := strings.Cut(section, "\n")
		lines := strings.Split(body, "\n")
		if m := answerHeading.FindStringSubmatch(heading); m != nil {
			key, keyed, current := "", false, []string(nil)
			for _, line := range lines {
				if b := keyedBullet.FindStringSubmatch(line); b != nil && offered(b[1]) {
					put(key, current)
					key, keyed, current = b[1], true, []string{b[2]}
					continue
				}
				if key != "" && strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "- ") {
					current = append(current, line)
				} else if key != "" {
					put(key, current)
					key, current = "", nil
				}
			}
			put(key, current)
			if !keyed && strings.EqualFold(m[1], "approval") && f.Item.Recommended != nil {
				put(f.Item.Recommended.Key, lines)
			}
			continue
		}
		if keys := backticked.FindAllStringSubmatch(heading, -1); len(keys) == 1 && offered(keys[0][1]) {
			put(keys[0][1], lines)
		}
	}
	return effects
}

var (
	answerHeading = regexp.MustCompile(`(?i)^on (each answer|approval)\b`)
	keyedBullet   = regexp.MustCompile("^[-*]\\s+(?:\\*\\*)?`?([a-z0-9-]{1,32})`?(?:\\s*\\([^)]*\\))?[.:]?(?:\\*\\*)?[.:]?\\s*(?:— )?(.*)$")
	backticked    = regexp.MustCompile("`([a-z0-9-]{1,32})`")
	markdownLinkT = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
)

// plainMarkdown is packet text as the operator reads it on the page: links
// as their words, without emphasis or code marks.
func plainMarkdown(text string) string {
	text = markdownLinkT.ReplaceAllString(text, "$1")
	text = strings.NewReplacer("**", "", "`", "").Replace(text)
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// loadPage reads one page and inlines the images it names.
func loadPage(path, research string) (*Page, error) {
	root, err := filepath.EvalSymlinks(research)
	if err != nil {
		return nil, fmt.Errorf("the research home %s cannot be read: %w", research, err)
	}
	real, content, err := readHomeFile(path, root, maxPageBytes)
	if err != nil {
		return nil, fmt.Errorf("the explainer %w", err)
	}
	if ext := strings.ToLower(filepath.Ext(real)); ext != ".html" && ext != ".htm" {
		return nil, fmt.Errorf("the explainer %s is not an .html file", path)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("the explainer %s is empty", path)
	}
	if head := bytes.ToLower(content[:min(len(content), 512)]); !bytes.Contains(head, []byte("<!doctype html")) && !bytes.Contains(head, []byte("<html")) {
		return nil, fmt.Errorf("the explainer %s does not start like an HTML page", path)
	}
	if n := operatordecision.EncodedLen(string(content)); n > operatordecision.MaxExplainerBytes {
		return nil, fmt.Errorf("the explainer %s is %d KiB as sent, over the %d KiB one decision carries", path, n>>10, operatordecision.MaxExplainerBytes>>10)
	}
	html, leftOut := inlineImages(string(content), filepath.Dir(real), root)
	sum := sha256.Sum256([]byte(html))
	return &Page{HTML: html, SHA256: hex.EncodeToString(sum[:]), LeftOut: leftOut}, nil
}

// readHomeFile reads a regular file inside root of at most limit bytes and
// returns its resolved path. Errors start with the path, for "the explainer …".
func readHomeFile(path, root string, limit int) (string, []byte, error) {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", nil, fmt.Errorf("%s cannot be read: %w", path, err)
	}
	if rel, err := filepath.Rel(root, real); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", nil, fmt.Errorf("%s is outside the research home, so the desk will not send it", path)
	}
	file, err := os.Open(real)
	if err != nil {
		return "", nil, fmt.Errorf("%s cannot be read: %w", path, err)
	}
	defer file.Close()
	if info, err := file.Stat(); err != nil || !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("%s is not a regular file", path)
	}
	content, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return "", nil, fmt.Errorf("%s cannot be read: %w", path, err)
	}
	if len(content) > limit {
		return "", nil, fmt.Errorf("%s is over %d KiB", path, limit>>10)
	}
	return real, content, nil
}

// imageTypes are the image files a page may name by path.
var imageTypes = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".webp": "image/webp",
	".gif": "image/gif", ".svg": "image/svg+xml", ".avif": "image/avif",
}

// imageRef finds the places a page names a file: src= and poster= attributes
// and CSS url(); group 1 or 2 is the reference, quoted or not.
var imageRef = regexp.MustCompile(`(?i)(?:\b(?:src|poster)\s*=\s*|url\(\s*)(?:"([^"]*)"|'([^']*)'|([^"'()\s>]+))`)

// inlineImages replaces each local image a page names with a data: URI, in
// page order, while the page still fits one decision. A reference to the
// network, a fragment or a data: URI is left as it is; a local image that is
// missing, outside the home, not an image or over the budget is left as it is
// and named in leftOut.
func inlineImages(html, dir, root string) (string, string) {
	size := operatordecision.EncodedLen(html)
	var left []string
	out := imageRef.ReplaceAllStringFunc(html, func(match string) string {
		m := imageRef.FindStringSubmatch(match)
		ref := m[1] + m[2] + m[3]
		local, ok := localRef(ref, dir)
		if !ok {
			return match
		}
		uri, err := dataURI(local, root, operatordecision.MaxExplainerBytes)
		switch {
		case err != nil:
			left = append(left, ref+" ("+err.Error()+")")
			return match
		case size+len(uri)-len(ref) > operatordecision.MaxExplainerBytes:
			left = append(left, fmt.Sprintf("%s (%d KiB more than one decision carries)", ref, (size+len(uri)-len(ref)-operatordecision.MaxExplainerBytes)>>10+1))
			return match
		}
		size += len(uri) - len(ref)
		at := strings.LastIndex(match, ref)
		return match[:at] + uri + match[at+len(ref):]
	})
	if len(left) == 0 {
		return out, ""
	}
	return out, fmt.Sprintf("%d image(s) left as links the Hub will not load: %s", len(left), strings.Join(left, "; "))
}

// localRef is the file a page reference names, or false for a reference that
// is not a local path (a URL, a fragment, a data: URI).
func localRef(ref, dir string) (string, bool) {
	ref, _, _ = strings.Cut(ref, "#")
	ref, _, _ = strings.Cut(ref, "?")
	if ref == "" || strings.HasPrefix(ref, "//") || strings.Contains(ref, ":") {
		return "", false
	}
	if unescaped, err := url.PathUnescape(ref); err == nil {
		ref = unescaped
	}
	if filepath.IsAbs(ref) {
		return ref, true
	}
	return filepath.Join(dir, ref), true
}

// dataURI reads one image inside root as a data: URI.
func dataURI(path, root string, limit int) (string, error) {
	media, ok := imageTypes[strings.ToLower(filepath.Ext(path))]
	if !ok {
		return "", fmt.Errorf("not an image the desk inlines")
	}
	_, content, err := readHomeFile(path, root, limit)
	if err != nil {
		return "", fmt.Errorf("%s", strings.TrimPrefix(err.Error(), path+" "))
	}
	return "data:" + media + ";base64," + base64.StdEncoding.EncodeToString(content), nil
}

// noAnswer is what the operator is told happens without an answer: nothing is
// chosen for them, and the item stays answerable.
const noAnswer = "Nothing is chosen for you. The question stays open in the research files until you answer, here or later in the research desk."

// summaryPage is the page the desk builds from the item itself: the question,
// the item's prose, who asks, the options with the recommendation and its
// reason, and what happens without an answer. It is a function of the item
// alone, so an unchanged item is the same request.
func summaryPage(f *home.FeedItem, effects map[string]string) string {
	facts := operatordecision.Facts{RecommendationReason: f.Item.Reason}
	for _, o := range f.Item.Choices() {
		facts.Options = append(facts.Options, operatordecision.Option{Key: o.Key, Label: o.Label})
	}
	if r := f.Item.Recommended; r != nil {
		facts.Recommended = r.Key
		if facts.RecommendationReason == "" && r.Label != facts.Label(r.Key) {
			facts.RecommendationReason = r.Label
		}
	}
	page := operatordecision.NewPage(f.Item.Title).Lead(f.Item.Prose())
	var names []string
	switch {
	case f.Topic != "":
		names = []string{f.Topic}
	case f.Commission != nil:
		for _, t := range f.Commission.Topics {
			names = append(names, t.Topic)
		}
	}
	asker := "the research supervisor"
	if len(names) > 0 {
		asker = "the research on " + strings.ReplaceAll(strings.Join(names, ", "), "-", " ")
	}
	page.Facts(operatordecision.Row{Label: "Asked by", Value: asker})
	return page.Options(facts, effects).IfNoAnswer(noAnswer).HTML()
}
