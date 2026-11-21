package hub

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/vrooli/api-core/pagescheme"
)

// The picture rules of the packet check. A page earns its place by showing
// what the cover can only tell: the numbers as a bar, the list as groups, the
// options against criteria, the repeats, the command, the screen. A page that
// pours the decision into prose cards, however many facts it adds, makes the
// operator read again what they just read. And a page that shows a full phone
// screenshot per option is long before it says anything.
//
// Measured on the decision-quality eval
// (~/.vrooli/research/notification-hub-decision-eval, the pages in
// testdata/page-quality): every baseline page the operator would call a
// restatement had no picture, and every page written to skill revision 11 had
// one. Twelve pages are too few to block on, so both rules are advice.
const (
	// WarnPageNoPicture: the page shows nothing; it is prose, option cards
	// and callouts.
	WarnPageNoPicture = "page_no_picture"
	// WarnPagePictureUncropped: two or more pictures are shown whole, at full
	// height, where the decision is about one part of each.
	WarnPagePictureUncropped = "page_picture_uncropped"
)

// minUncropped is how many whole pictures make a page long.
const minUncropped = 2

// pictureBlocks are the kit's decision blocks that show the situation. The
// layout blocks (.hub-tradeoffs, .hub-pair, .hub-callout, .hub-chip) do not:
// they arrange words.
var pictureBlocks = []string{
	"hub-proportion", "hub-groups", "hub-matrix", "hub-timeline", "hub-compare", "hub-recurrence", "hub-command",
}

// pictureElements are elements that are pictures by themselves.
var pictureElements = map[string]bool{"img": true, "svg": true, "canvas": true, "video": true}

// cropLength matches a height that bounds a picture: a length, not auto or a
// percentage of an unbounded parent.
var cropLength = regexp.MustCompile(`^(?:\d*\.?\d+)(?:px|rem|em|vh|svh|dvh|lvh|vw|cm|mm|in|pt)$|^(?:min|max|clamp|calc)\(`)

// pagePictures counts the page's pictures and the images shown whole.
func pagePictures(page string, root *element) (pictures int, uncropped []string) {
	styles := pageKit.Styles(page, root, SchemeLight)
	var walk func(e *element, hidden bool)
	walk = func(e *element, hidden bool) {
		if _, ok := e.Attrs["hidden"]; ok || e.Tag == "template" {
			hidden = true
		}
		switch {
		case hidden:
		case pictureElements[e.Tag], e.Attrs["role"] == "img", e.Attrs["role"] == "figure", isPictureBlock(e), isDataTable(e):
			pictures++
		}
		if e.Tag == "img" && !hidden && !cropped(styles, e) {
			uncropped = append(uncropped, imageName(e))
		}
		for _, child := range e.Children {
			walk(child, hidden)
		}
	}
	walk(root, false)
	return pictures, uncropped
}

func isPictureBlock(e *element) bool {
	for _, class := range pictureBlocks {
		if e.Classes[class] {
			return true
		}
	}
	return false
}

// isDataTable reports a table with at least two body rows: a list or a matrix
// laid out to compare.
func isDataTable(e *element) bool {
	if e.Tag != "table" {
		return false
	}
	rows := 0
	var count func(n *element)
	count = func(n *element) {
		for _, child := range n.Children {
			if child.Tag == "tr" {
				rows++
			}
			if child.Tag != "table" {
				count(child)
			}
		}
	}
	count(e)
	return rows >= 3 // a header row and two rows
}

// cropped reports an image shown at a bounded height: a height or max-height
// that is a length, an aspect-ratio, or a height attribute.
func cropped(styles pagescheme.Styles, img *element) bool {
	for _, property := range []string{"max-height", "height"} {
		if cropLength.MatchString(strings.ToLower(strings.TrimSpace(strings.TrimSuffix(styles.Value(img, property), "!important")))) {
			return true
		}
	}
	if ratio := strings.TrimSpace(styles.Value(img, "aspect-ratio")); ratio != "" && ratio != "auto" {
		return true
	}
	_, sized := img.Attrs["height"]
	return sized
}

func imageName(img *element) string {
	if src := strings.TrimSpace(img.Attrs["src"]); src != "" && !strings.HasPrefix(src, "data:") {
		return src
	}
	if alt := strings.TrimSpace(img.Attrs["alt"]); alt != "" {
		return shorten(alt, 40)
	}
	return "an inline image"
}

// checkPictures reports a page that shows nothing, and pictures shown whole.
// A page the restating rule already refuses gets no second finding.
func checkPictures(explainer PacketExplainer, page *element, restates bool) []PacketWarning {
	pictures, uncropped := pagePictures(string(explainer.HTML), page)
	var warnings []PacketWarning
	if pictures == 0 && !restates {
		warnings = append(warnings, advice(WarnPageNoPicture, explainer.Field,
			"the page tells the decision in sentences and shows nothing: it has no image, no chart (role=\"img\"), no table and none of the kit's "+
				"decision blocks. Option cards and callouts repeat the cover. Show the situation instead: the numbers as .hub-proportion, the items "+
				"as .hub-groups, options against criteria as table.hub-matrix, the repeats as .hub-recurrence, a before/after as .hub-compare, "+
				"the steps as .hub-timeline, a command as .hub-command, or a cropped screenshot. Start from a starter in the skill's examples/ folder."))
	}
	if len(uncropped) >= minUncropped {
		warnings = append(warnings, advice(WarnPagePictureUncropped, explainer.Field, fmt.Sprintf(
			"%d pictures are shown whole (%s), so each takes a phone screen. Crop each to the part the decision is about: "+
				"img { height: 220px; object-fit: cover; object-position: bottom } (or top), or a cropped file, and show each picture once.",
			len(uncropped), strings.Join(uncropped, ", "))))
	}
	return warnings
}
