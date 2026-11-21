package decisionpage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/vrooli/api-core/operatordecision"
)

// pageCodes are the Hub's check codes about the page itself that are not
// explainer_* (explainer_restates among those): text the operator cannot
// read in a scheme the Hub shows it in, and a command the operator must run
// that the page does not show. The Hub's other codes, such as
// missing_recommendation_reason and recurring_subject, are about the
// sender's cover and go back to it as advice.
var pageCodes = map[string]bool{
	"command_hidden":  true,
	"page_unreadable": true,
}

// coverOnly are explainer_* codes about something other than this page.
var coverOnly = map[string]bool{
	"explainer_missing": true, // the writer always attaches one
}

// coverFields are the ask fields the sender writes. A finding on one of them
// is the sender's to fix, even when the Hub blocks on it
// (missing_recommendation_reason): no page can fix it.
var coverFields = map[string]bool{
	"question": true, "big_picture": true, "options": true, "recommended": true, "recommendation_reason": true,
	"technical_details": true, "subject_key": true, "sender": true, "packet": true, "deadline": true, "if_no_answer": true,
}

// PageFinding reports whether a Hub finding is about the page, so the
// writer must fix it, rather than about the cover the sender wrote. A
// finding the Hub blocks on that names no cover field is the page's too: a
// blocked ask is not sent.
func PageFinding(f Finding) bool {
	switch {
	case coverOnly[f.Code]:
		return false
	case strings.HasPrefix(f.Code, "explainer_"), pageCodes[f.Code]:
		return true
	case strings.EqualFold(f.Severity, "blocking"):
		return !coverFields[f.Field]
	}
	return false
}

var (
	prefillCall = regexp.MustCompile(`hub\.prefill\s*\(`)
	htmlStart   = regexp.MustCompile(`(?i)^\s*(<!--.*?-->\s*)*<(!doctype html|html)`)
)

// pageProblem is why a file is not a decision page at all, before the Hub's
// check: not HTML, too large to travel, or an answer it offers no button for.
func pageProblem(r Request, html []byte) string {
	switch {
	case len(bytes.TrimSpace(html)) == 0:
		return "the page is empty"
	case !htmlStart.Match(html):
		return "the page does not start with <!doctype html>"
	case operatordecision.EncodedLen(string(html)) > operatordecision.MaxExplainerBytes:
		return fmt.Sprintf("the page is %d KiB as sent, over the %d KiB one decision carries; move pictures to smaller files", operatordecision.EncodedLen(string(html))>>10, operatordecision.MaxExplainerBytes>>10)
	case !prefillCall.Match(html):
		return "no button calls hub.prefill(\"answer\", KEY), so the page cannot fill in an answer"
	}
	var missing []string
	for _, key := range r.Keys() {
		if !bytes.Contains(html, []byte(`"`+key+`"`)) && !bytes.Contains(html, []byte(`'`+key+`'`)) {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return "no Choose-this button for " + strings.Join(missing, ", ") + ` (use <button type="button" data-answer="KEY">)`
	}
	return ""
}

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
