package pagescheme

import (
	"regexp"
	"strings"
)

// The markup patterns the theme and readability rules read a page with.
var (
	tagPattern         = regexp.MustCompile(`(?is)<([a-z][a-z0-9-]*)\b([^>]*)>`)
	attributePattern   = regexp.MustCompile(`(?is)([a-z][a-z0-9:-]*)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+))`)
	commentPattern     = regexp.MustCompile(`(?s)<!--.*?-->`)
	scriptBodyPattern  = regexp.MustCompile(`(?is)(<script\b[^>]*>).*?(</script\s*>)`)
	styleBodyPattern   = regexp.MustCompile(`(?is)(<style\b[^>]*>).*?(</style\s*>)`)
	scriptBlockPattern = regexp.MustCompile(`(?is)<script\b[^>]*>(.*?)</script\s*>`)
)

// attributes returns the attributes of one tag's source text, names lowered.
func attributes(tagSource string) map[string]string {
	out := map[string]string{}
	for _, match := range attributePattern.FindAllStringSubmatch(tagSource, -1) {
		out[strings.ToLower(match[1])] = match[2] + match[3] + match[4]
	}
	return out
}
