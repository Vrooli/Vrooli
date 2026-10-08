package visualhealth

import (
	"regexp"
	"strings"
)

var (
	scriptLikeRE = regexp.MustCompile(`(?is)<(script|style|template|noscript)[^>]*>.*?</(script|style|template|noscript)>`)
	attrTextRE   = regexp.MustCompile(`(?is)\s(?:aria-label|alt|title|placeholder)=["']([^"']+)["']`)
	tagRE        = regexp.MustCompile(`(?is)<[^>]+>`)
	spaceRE      = regexp.MustCompile(`\s+`)
	loadingRE    = regexp.MustCompile(`(?i)\b(loading|please wait|spinner|progressbar|skeleton|aria-busy=["']true["'])\b`)
)

func domTextBlank(html string) bool {
	cleaned := scriptLikeRE.ReplaceAllString(html, " ")
	var meaningful []string
	for _, match := range attrTextRE.FindAllStringSubmatch(cleaned, -1) {
		if len(match) > 1 {
			meaningful = append(meaningful, match[1])
		}
	}
	text := tagRE.ReplaceAllString(cleaned, " ")
	meaningful = append(meaningful, text)
	joined := htmlEntityTrim(strings.Join(meaningful, " "))
	return joined == ""
}

func htmlEntityTrim(s string) string {
	replacer := strings.NewReplacer(
		"&nbsp;", " ",
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
	)
	s = replacer.Replace(s)
	return strings.TrimSpace(spaceRE.ReplaceAllString(s, " "))
}

func domLoadingFindings(step stepArtifact) []*visualFinding {
	html := strings.TrimSpace(step.GetDomHtml())
	if html == "" || !loadingRE.MatchString(html) {
		return nil
	}
	text := strings.ToLower(htmlEntityTrim(tagRE.ReplaceAllString(scriptLikeRE.ReplaceAllString(html, " "), " ")))
	text = strings.ReplaceAll(text, "please wait", "")
	text = strings.ReplaceAll(text, "loading", "")
	text = strings.ReplaceAll(text, "spinner", "")
	text = strings.ReplaceAll(text, "skeleton", "")
	text = strings.TrimSpace(spaceRE.ReplaceAllString(text, " "))
	if len(text) > 8 {
		return nil
	}
	return []*visualFinding{{
		Code:        "visual_stuck_loading",
		Severity:    severityError,
		Category:    categoryDOM,
		Message:     "DOM snapshot still appears to be in a loading state with no meaningful final content",
		Location:    locationFor(step),
		Remediation: "Ensure the page leaves loading state and renders meaningful final content before capture.",
		StepId:      step.GetStepId(),
	}}
}
