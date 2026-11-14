package hub

import (
	"fmt"
	"path"
	"strings"
	"unicode/utf8"
)

// The shape advice of the packet check. Most decisions read best as text, so
// the check is narrow: it speaks only when the cover text has the outline of a
// comparison that an HTML explainer shows better than words, or when images
// were attached with nothing to compare them. It is advice and never blocks.
// The signals are structural (how many options, how long, how many images), not
// topical, so an ordinary yes or no, or a two-option ask with a short
// background, never triggers it.

// WarnExplainerMayHelp advises an explainer for a comparison-shaped decision.
const WarnExplainerMayHelp = "explainer_may_help"

const (
	// shapeManyOptions is the number of real options from which each option's
	// description is probably a comparison.
	shapeManyOptions = 3
	// shapeLongLabelRunes is the average option label length past which the
	// options are descriptions with consequences, not short commands.
	shapeLongLabelRunes = 40
	// shapeLongBackgroundRunes is the big picture length at which a page of text
	// is more than a phone reader takes in at one go.
	shapeLongBackgroundRunes = 1000
	// shapeManyImages is the number of attached images from which the operator
	// is probably choosing between them.
	shapeManyImages = 2
)

var imageExtensions = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".svg": true}

// checkShape returns the advice to add an explainer. It says nothing when the
// ask already has one.
func checkShape(spec AskSpec, attachments []PacketAttachment, hasExplainer bool) []PacketWarning {
	if hasExplainer {
		return nil
	}
	labelRunes, options := 0, 0
	for _, option := range spec.Options {
		if option.Key == "other" {
			continue
		}
		options++
		labelRunes += utf8.RuneCountInString(strings.TrimSpace(option.Label))
	}
	var reason string
	switch {
	case options >= shapeManyOptions && labelRunes/options >= shapeLongLabelRunes:
		reason = fmt.Sprintf("the decision has %d options with long descriptions", options)
	case options >= 2 && utf8.RuneCountInString(spec.BigPicture) >= shapeLongBackgroundRunes:
		reason = "the background is long"
	case options >= 2 && countImages(attachments) >= shapeManyImages && !hasVariantQuestion(spec.Questions):
		reason = "the decision attaches several images but nothing puts them side by side"
	default:
		return nil
	}
	return []PacketWarning{advice(WarnExplainerMayHelp, "attachments", reason+
		"; if the choice rests on a comparison (trade-offs, side by side, before and after, a timeline), a one-page explainer is quicker to read than text. "+
		"Copy a starter page: cp \"$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/plan-tradeoffs.html\" explainer.html, "+
		"replace its words, then send again with --explainer explainer.html (see skills/notification-hub/references/html-explainers.md). "+
		"Ignore this when the options are not a comparison")}
}

func countImages(attachments []PacketAttachment) int {
	count := 0
	for _, attachment := range attachments {
		if imageExtensions[strings.ToLower(path.Ext(attachment.Name))] {
			count++
		}
	}
	return count
}

func hasVariantQuestion(questions []AskQuestion) bool {
	for _, question := range questions {
		if question.Type == "variant" {
			return true
		}
	}
	return false
}
