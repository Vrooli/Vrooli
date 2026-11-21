package hub

import (
	"fmt"
	"regexp"
	"strings"
)

// The restating rule of the packet check. The page exists to add what the
// cover cannot hold: the picture, the numbers, the list, the command, what
// each answer leads to. A page made from the cover's own words (a template
// that pours the question, the big picture and the option labels into cards)
// gives the operator nothing new, and an agent's ask with such a page is
// refused (blocking.go).
//
// It counts the page's own words that the cover does not already say:
// ordinary words and the page furniture every page has (Choose this,
// Recommended, If you do not answer) do not count, and a picture, video or
// slider counts as much as a short paragraph.

// WarnExplainerRestates reports a page that mostly repeats the cover.
const WarnExplainerRestates = "explainer_restates"

// Thresholds of the restating rule: a page restates the cover when what it
// adds is under minNewWords, or under restateShare of its words and under
// twice minNewWords.
const (
	minNewWords   = 40
	restateShare  = 0.25
	wordsPerMedia = 30
)

var pageWordPattern = regexp.MustCompile(`[\p{L}\p{N}][\p{L}\p{N}'’.,%$:/-]*[\p{L}\p{N}%]|[\p{L}\p{N}]`)

// commonWords never count as news: the most frequent English words, and the
// page furniture of every decision page.
var commonWords = func() map[string]bool {
	words := map[string]bool{}
	for _, word := range strings.Fields(`a an the and or but if then so of to in on at by for with from as is are was were be been being it its this that these those
		there here not no yes do does did done can could will would should may might must i you we they he she your our their my me us them
		what which who when where why how all any each every some more most less other than too very just also only into over under about up down out
		has have had get got go goes going one two three
		choose chose chosen recommended recommend recommendation your choices choice tap send answer answers happens happen approve approved
		option options nothing work asked asks ask decision decide page`) {
		words[word] = true
	}
	return words
}()

// restateReading is what the restating rule measured.
type restateReading struct {
	pageWords, newWords, media int
}

func (r restateReading) restates() bool {
	added := r.newWords + wordsPerMedia*r.media
	if added < minNewWords {
		return true
	}
	return r.pageWords > 0 && float64(added) < restateShare*float64(r.pageWords+wordsPerMedia*r.media) && added < 2*minNewWords
}

func contentWords(text string) []string {
	var out []string
	for _, word := range pageWordPattern.FindAllString(strings.ToLower(text), -1) {
		word = strings.Trim(word, ".,:/-'’")
		if word == "" || commonWords[word] {
			continue
		}
		out = append(out, word)
	}
	return out
}

// coverText is what the operator reads without the page: the question, the
// big picture, the option labels and the recommendation's reason. Technical
// details are not part of it; a page that brings them forward adds news.
func coverText(spec AskSpec) string {
	parts := []string{spec.Question, spec.BigPicture, spec.RecommendationReason}
	for _, option := range spec.Options {
		parts = append(parts, option.Key, option.Label)
	}
	return strings.Join(parts, "\n")
}

// readRestating measures a page against the cover.
func readRestating(root *element, cover string) restateReading {
	known := map[string]bool{}
	for _, word := range contentWords(cover) {
		known[word] = true
	}
	var reading restateReading
	var walk func(e *element)
	walk = func(e *element) {
		switch e.Tag {
		case "head", "script", "style", "template", "noscript", "title":
			return
		case "img", "video", "picture", "canvas":
			reading.media++
		case "svg":
			reading.media++
			return
		}
		if _, ok := e.Attrs["data-hub-compare"]; ok {
			reading.media++
		}
		if _, hidden := e.Attrs["hidden"]; hidden {
			return
		}
		for _, word := range contentWords(e.Text) {
			reading.pageWords++
			if !known[word] {
				reading.newWords++
			}
		}
		for _, child := range e.Children {
			walk(child)
		}
	}
	walk(root)
	return reading
}

// readablePage reports an explainer whose markup the content rules read: an
// HTML file the Hub holds, within the size it shows.
func readablePage(explainer PacketExplainer) bool {
	return explainer.HTML != nil && !explainer.IsFolder && explainer.Size <= MaxExplainerBytes && looksLikeHTML(explainer)
}

// checkRestates reports a page that mostly repeats the cover. The Hub's own
// page (CoverPage) is never checked: it is shown only for an ask with no page.
func checkRestates(spec AskSpec, explainer PacketExplainer, page *element) []PacketWarning {
	reading := readRestating(page, coverText(spec))
	if !reading.restates() {
		return nil
	}
	return []PacketWarning{advice(WarnExplainerRestates, explainer.Field, fmt.Sprintf(
		"the page mostly repeats the cover: of its %d words, %d say something the question, big picture and option labels do not, and it has %d pictures. "+
			"The operator already read the cover; the page must add what it cannot hold. Show the situation: the numbers or the list as a picture "+
			"(kit blocks hub-proportion, hub-groups, hub-matrix), real screenshots, the exact command with where/what/undo (hub-command), what each answer leads to, "+
			"and the cause if this keeps coming back. Start from a starter in the skill's examples/ folder.",
		reading.pageWords, reading.newWords, reading.media))}
}
