package validation

import "strings"

// This file holds the social / link-preview metadata rules (Open Graph and
// Twitter cards). Both are derivable from the scenario's own identity and so are
// self-contained autofixable. og:url / og:image need a canonical URL / a real
// 1200x630 asset and are therefore recommended, not required, by these rules.

// requiredOGTags are the Open Graph properties derivable from identity alone.
var requiredOGTags = []string{"og:type", "og:title", "og:description", "og:site_name"}

func ruleOpenGraph(c *scanContext) (Finding, bool) {
	id := c.identity()
	if !id.HasIdentity() {
		return Finding{}, false // nothing to derive a preview from yet
	}
	h := c.head()
	var missing []string
	for _, key := range requiredOGTags {
		if v, ok := h.metaByProperty(key); !ok || strings.TrimSpace(v) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "No Open Graph link-preview metadata",
		Description:            "ui/index.html is missing Open Graph tags, so shared links render without a branded title/description card.",
		FilePath:               indexHTMLRel,
		WhyItMatters:           "Open Graph tags control how the app looks when its link is shared in chat, social, and previews.",
		RecommendedRemediation: "Add og:type/title/description/site_name (and an og:image) to ui/index.html.",
		Evidence:               map[string]any{"missing": missing},
	}, true
}

// requiredTwitterTags are the Twitter card tags derivable from identity.
var requiredTwitterTags = []string{"twitter:card", "twitter:title", "twitter:description"}

func ruleTwitterCard(c *scanContext) (Finding, bool) {
	id := c.identity()
	if !id.HasIdentity() {
		return Finding{}, false
	}
	h := c.head()
	var missing []string
	for _, key := range requiredTwitterTags {
		if v, ok := h.metaByName(key); !ok || strings.TrimSpace(v) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityInfo,
		Title:                  "No Twitter card metadata",
		Description:            "ui/index.html is missing Twitter (X) card tags, so links shared there fall back to a plain preview.",
		FilePath:               indexHTMLRel,
		WhyItMatters:           "Twitter card tags give shared links a branded summary image and text on X.",
		RecommendedRemediation: "Add twitter:card/title/description (and twitter:image) to ui/index.html.",
		Evidence:               map[string]any{"missing": missing},
	}, true
}

func ruleSocialPreviewImage(c *scanContext) (Finding, bool) {
	id := c.identity()
	if !id.HasIdentity() {
		return Finding{}, false
	}
	h := c.head()
	if v, ok := h.metaByProperty("og:image"); ok && strings.TrimSpace(v) != "" {
		return Finding{}, false
	}
	if v, ok := h.metaByName("twitter:image"); ok && strings.TrimSpace(v) != "" {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "No social preview image is declared",
		Description:            "ui/index.html has no og:image or twitter:image metadata for branded link previews.",
		FilePath:               indexHTMLRel,
		WhyItMatters:           "A missing social preview image makes shared links look generic even when title and description metadata exist.",
		RecommendedRemediation: "Add a real preview image under /public/ and reference it with og:image and twitter:image.",
	}, true
}
