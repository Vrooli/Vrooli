package hub

// Markdown headings as section_review sections: what a question with
// sections_from follows. The Hub derives them (and derives them again when a
// revision replaces the file), so the review always matches the document the
// operator opens.

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Limits of review sections beyond those in ask_questions.go.
const (
	maxSectionKeyLength = 64
	// anchorSuffixRoom leaves space for the "-N" that makes repeated anchors unique.
	anchorSuffixRoom = 8
)

var (
	atxHeading    = regexp.MustCompile(`^ {0,3}(#{2,3})[ \t]+(.*)$`)
	closingHashes = regexp.MustCompile(`[ \t]+#+[ \t]*$`)
	markdownLink  = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	fenceOpening  = regexp.MustCompile("^ {0,3}(```+|~~~+)")
)

// markdownSections reads the `##` and `###` headings of a Markdown document as
// review sections. Headings inside fenced code blocks and a leading front-matter
// block are ignored; setext (underlined) headings are not recognised.
//
// The Hub's limits apply: at most maxReviewSections sections are returned (the
// second result is how many headings the document has), titles and anchors are
// cut to their limits, and keys to maxSectionKeyLength.
//
// The anchor is the GitHub-style heading slug (the second "Plan" becomes
// "plan-1"); the key is the same text restricted to the characters a Hub key
// allows, and both are unique within the result.
func markdownSections(r io.Reader, attachment string) ([]AskQuestionSection, int, error) {
	var sections []AskQuestionSection
	total := 0
	anchors, keys := map[string]int{}, map[string]bool{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4<<20)

	inFrontMatter, fence, first := false, "", true
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if first {
			first = false
			if strings.TrimSpace(line) == "---" {
				inFrontMatter = true
				continue
			}
		}
		if inFrontMatter {
			if t := strings.TrimSpace(line); t == "---" || t == "..." {
				inFrontMatter = false
			}
			continue
		}
		if fence != "" {
			if strings.HasPrefix(strings.TrimSpace(line), fence) {
				fence = ""
			}
			continue
		}
		if m := fenceOpening.FindStringSubmatch(line); m != nil {
			fence = m[1][:3]
			continue
		}
		m := atxHeading.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		title := plainHeading(closingHashes.ReplaceAllString(m[2], ""))
		if title == "" {
			continue
		}
		total++
		anchor := uniqueAnchor(truncateRunes(githubSlug(title), maxSectionAnchor-anchorSuffixRoom), anchors)
		if len(sections) >= maxReviewSections {
			continue // keep counting headings so the caller can say how many were left out
		}
		sections = append(sections, AskQuestionSection{
			Key:        uniqueKey(anchor, len(sections)+1, keys),
			Title:      truncateRunes(title, maxSectionTitle),
			Attachment: attachment,
			Anchor:     anchor,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}
	if len(sections) == 0 {
		return nil, 0, fmt.Errorf("no ## or ### headings found")
	}
	return sections, total, nil
}

// plainHeading removes inline Markdown from heading text for display.
func plainHeading(text string) string {
	text = markdownLink.ReplaceAllString(text, "$1")
	text = strings.NewReplacer("`", "", "**", "", "__", "", "*", "").Replace(text)
	return strings.TrimSpace(text)
}

// githubSlug mirrors GitHub's heading anchors: lower-case, spaces to hyphens,
// punctuation removed.
func githubSlug(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}

// uniqueAnchor numbers repeated slugs the way GitHub's slugger does, skipping
// numbers already taken: plan, plan-1, plan-2. A heading with no sluggable text
// has no anchor.
func uniqueAnchor(slug string, occurrences map[string]int) string {
	if slug == "" {
		return ""
	}
	original := slug
	for {
		if _, taken := occurrences[slug]; !taken {
			break
		}
		occurrences[original]++
		slug = original + "-" + strconv.Itoa(occurrences[original])
	}
	occurrences[slug] = 0
	return slug
}

// truncateRunes cuts text to at most limit code points.
func truncateRunes(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	return strings.TrimSpace(string([]rune(text)[:limit]))
}

// uniqueKey restricts an anchor to the Hub key alphabet (A-Z a-z 0-9 . _ -),
// falling back to section-N, cuts it to maxSectionKeyLength, and makes it
// unique. A numeric suffix is added after cutting, so it always fits.
func uniqueKey(anchor string, position int, used map[string]bool) string {
	var b strings.Builder
	for _, r := range anchor {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	base := strings.Trim(b.String(), "-_")
	if base == "" {
		base = "section-" + strconv.Itoa(position)
	}
	fit := func(suffix string) string {
		room := maxSectionKeyLength - len(suffix)
		stem := base
		if len(stem) > room {
			stem = strings.Trim(stem[:room], "-_")
		}
		return stem + suffix
	}
	key := fit("")
	for n := 2; used[key]; n++ {
		key = fit("-" + strconv.Itoa(n))
	}
	used[key] = true
	return key
}
