package hub

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/vrooli/api-core/operatordecision"
)

// The explainer rules of the packet check. An explainer is shown in a sealed
// frame with a strict content security policy: nothing it loads from the
// network works, it cannot submit forms, and it can only reach the files
// attached to the ask. Each rule below names a way a page breaks in that
// frame. They are advice; the Hub still shows the page.

// htmlSniffBytes is how much of a page looksLikeHTML reads.
const htmlSniffBytes = 1024

// networkSourceAttributes are the attributes that load a resource by address.
var networkSourceAttributes = []string{"src", "poster", "srcset"}

var (
	tagPattern         = regexp.MustCompile(`(?is)<([a-z][a-z0-9-]*)\b([^>]*)>`)
	attributePattern   = regexp.MustCompile(`(?is)([a-z][a-z0-9:-]*)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+))`)
	commentPattern     = regexp.MustCompile(`(?s)<!--.*?-->`)
	scriptBodyPattern  = regexp.MustCompile(`(?is)(<script\b[^>]*>).*?(</script\s*>)`)
	styleBodyPattern   = regexp.MustCompile(`(?is)(<style\b[^>]*>).*?(</style\s*>)`)
	scriptBlockPattern = regexp.MustCompile(`(?is)<script\b[^>]*>(.*?)</script\s*>`)
	cssURLPattern      = regexp.MustCompile(`(?i)url\(\s*["']?\s*((?:https?:)?//[^\s"')]*)`)
	cssImportPattern   = regexp.MustCompile(`(?i)@import\s+(?:url\(\s*)?["']?\s*((?:https?:)?//[^\s"')]*)`)
	scriptFetchPattern = regexp.MustCompile(`(?i)\bfetch\(\s*["'\x60]\s*((?:https?:)?//[^\s"'\x60)]*)`)
	viewportPattern    = regexp.MustCompile(`(?i)<meta\b[^>]*\bname\s*=\s*["']?viewport\b`)
	formElementPattern = regexp.MustCompile(`(?i)<(form|input|select|textarea)\b`)
	hubOpenCallPattern = regexp.MustCompile(`\bhub\.open\(\s*["'\x60]([^"'\x60]+)["'\x60]`)
	schemePattern      = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
	htmlStartPattern   = regexp.MustCompile(`(?i)^\s*(?:<!doctype\s+html|<html\b)`)
)

func checkExplainer(explainer PacketExplainer, attached map[string]bool) []PacketWarning {
	if explainer.IsFolder || !looksLikeHTML(explainer) {
		return []PacketWarning{advice(WarnExplainerNotHTML, explainer.Field,
			fmt.Sprintf("the explainer %q is not an HTML file, so the Hub cannot show it as a page; attach one .html file with role explainer, or use evidence for other files", explainer.Name))}
	}
	if explainer.Size > MaxExplainerBytes {
		return []PacketWarning{advice(WarnExplainerTooLarge, explainer.Field,
			fmt.Sprintf("the explainer %q is %.1f MiB; the Hub shows pages up to %d MiB, so shrink images or move them to evidence attachments", explainer.Name, float64(explainer.Size)/(1<<20), MaxExplainerBytes>>20))}
	}
	if explainer.HTML == nil {
		return nil
	}
	page := commentPattern.ReplaceAllString(string(explainer.HTML), "")
	markup := scriptBodyPattern.ReplaceAllString(styleBodyPattern.ReplaceAllString(page, "$1$2"), "$1$2")
	var warnings []PacketWarning
	if urls := networkReferences(page, markup); len(urls) > 0 {
		warnings = append(warnings, advice(WarnExplainerNetwork, explainer.Field,
			fmt.Sprintf("the explainer loads resources from the network (%s); the sealed frame blocks them, so attach the file and refer to it by name", operatordecision.QuoteList(urls))))
	}
	if !viewportPattern.MatchString(markup) {
		warnings = append(warnings, advice(WarnExplainerNoViewport, explainer.Field,
			`the explainer has no <meta name="viewport" content="width=device-width, initial-scale=1">; it may not fit a phone`))
	}
	if forms := formElements(markup); len(forms) > 0 {
		warnings = append(warnings, advice(WarnExplainerForm, explainer.Field,
			fmt.Sprintf("the explainer contains form elements (%s); answers belong in the Hub form, and the page can pre-fill it through hub.prefill", strings.Join(forms, ", "))))
	}
	if unknown := unknownAttachmentNames(page, markup, attached); len(unknown) > 0 {
		warnings = append(warnings, advice(WarnExplainerUnknownAttachment, explainer.Field,
			fmt.Sprintf("the explainer refers to files that are not attached (%s); attach them, or fix the names", operatordecision.QuoteList(unknown))))
	}
	return append(warnings, checkReadable(explainer)...)
}

// looksLikeHTML reports whether the explainer is an HTML page, by its file
// name, its media type, or its first bytes.
func looksLikeHTML(explainer PacketExplainer) bool {
	for _, name := range []string{explainer.SourcePath, explainer.Name} {
		switch strings.ToLower(path.Ext(name)) {
		case ".html", ".htm":
			return true
		}
	}
	mediaType := strings.ToLower(explainer.MediaType)
	if strings.HasPrefix(mediaType, "text/html") || strings.HasPrefix(mediaType, "application/xhtml") {
		return true
	}
	head := explainer.HTML
	if len(head) > htmlSniffBytes {
		head = head[:htmlSniffBytes]
	}
	return htmlStartPattern.Match(head)
}

// attributes returns the attributes of one tag's source text, names lowered.
func attributes(tagSource string) map[string]string {
	out := map[string]string{}
	for _, match := range attributePattern.FindAllStringSubmatch(tagSource, -1) {
		value := match[2] + match[3] + match[4]
		out[strings.ToLower(match[1])] = value
	}
	return out
}

func isNetworkURL(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "//")
}

// srcsetURLs returns the URLs of a srcset value.
func srcsetURLs(value string) []string {
	var out []string
	for _, candidate := range strings.Split(value, ",") {
		if fields := strings.Fields(candidate); len(fields) > 0 {
			out = append(out, fields[0])
		}
	}
	return out
}

// networkReferences returns the network resources the page tries to load:
// element sources, stylesheet links, CSS url() and @import, and fetch calls.
// Anchors are navigation, not resources, and are not reported. The xmlns
// declaration of an inline SVG is not a reference.
func networkReferences(page, markup string) []string {
	var found []string
	seen := map[string]bool{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			found = append(found, value)
		}
	}
	for _, tag := range tagPattern.FindAllStringSubmatch(markup, -1) {
		name, attrs := strings.ToLower(tag[1]), attributes(tag[2])
		for _, attribute := range networkSourceAttributes {
			value, ok := attrs[attribute]
			if !ok {
				continue
			}
			candidates := []string{value}
			if attribute == "srcset" {
				candidates = srcsetURLs(value)
			}
			for _, candidate := range candidates {
				if isNetworkURL(candidate) {
					add(candidate)
				}
			}
		}
		if name == "link" && isNetworkURL(attrs["href"]) {
			add(attrs["href"])
		}
	}
	for _, pattern := range []*regexp.Regexp{cssURLPattern, cssImportPattern, scriptFetchPattern} {
		for _, match := range pattern.FindAllStringSubmatch(page, -1) {
			add(match[1])
		}
	}
	return found
}

func formElements(markup string) []string {
	var found []string
	seen := map[string]bool{}
	for _, match := range formElementPattern.FindAllStringSubmatch(markup, -1) {
		name := "<" + strings.ToLower(match[1]) + ">"
		if !seen[name] {
			seen[name] = true
			found = append(found, name)
		}
	}
	return found
}

// unknownAttachmentNames returns the attachment names the page refers to that
// the ask does not hold: element sources and links, data-hub-open attributes,
// and hub.open calls. A reference is a bare name (or name/path inside a
// folder); fragments, data: and other schemes, and network URLs are not.
func unknownAttachmentNames(page, markup string, attached map[string]bool) []string {
	var references []string
	for _, tag := range tagPattern.FindAllStringSubmatch(markup, -1) {
		attrs := attributes(tag[2])
		for _, attribute := range []string{"src", "href", "poster"} {
			if value, ok := attrs[attribute]; ok {
				references = append(references, value)
			}
		}
		if value, ok := attrs["srcset"]; ok {
			references = append(references, srcsetURLs(value)...)
		}
		if value, ok := attrs["data-hub-open"]; ok {
			references = append(references, value)
		}
	}
	for _, block := range scriptBlockPattern.FindAllStringSubmatch(page, -1) {
		for _, match := range hubOpenCallPattern.FindAllStringSubmatch(block[1], -1) {
			references = append(references, match[1])
		}
	}
	var unknown []string
	seen := map[string]bool{}
	for _, reference := range references {
		name, isReference := attachmentReferenceName(reference)
		if isReference && !attached[name] && !seen[name] {
			seen[name] = true
			unknown = append(unknown, name)
		}
	}
	sort.Strings(unknown)
	return unknown
}

// attachmentReferenceName returns the attachment a page reference names, and
// false for references that are not about attachments.
func attachmentReferenceName(reference string) (string, bool) {
	reference = strings.TrimSpace(reference)
	if reference == "" || strings.HasPrefix(reference, "#") || strings.HasPrefix(reference, "//") || schemePattern.MatchString(reference) {
		return "", false
	}
	if cut := strings.IndexAny(reference, "?#"); cut >= 0 {
		reference = reference[:cut]
	}
	if decoded, err := url.PathUnescape(reference); err == nil {
		reference = decoded
	}
	reference = strings.TrimLeft(strings.TrimPrefix(reference, "./"), "/")
	name, _, _ := strings.Cut(reference, "/")
	return name, name != ""
}
