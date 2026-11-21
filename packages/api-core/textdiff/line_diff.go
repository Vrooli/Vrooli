// Package textdiff contains bounded line-oriented diagnostics for textual
// artifacts such as canonical graph fixtures.
package textdiff

import (
	"fmt"
	"strings"
)

// MaxLineChanges caps the number of changed lines emitted per side so a large
// artifact mismatch cannot produce an unbounded response body.
const MaxLineChanges = 400

// LineDiff trims the common prefix and suffix and renders the differing middle
// with two lines of context. Removed lines use "-" and added lines use "+".
func LineDiff(expected, actual string) string {
	e := strings.Split(expected, "\n")
	a := strings.Split(actual, "\n")

	prefix := 0
	for prefix < len(e) && prefix < len(a) && e[prefix] == a[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(e)-prefix && suffix < len(a)-prefix &&
		e[len(e)-1-suffix] == a[len(a)-1-suffix] {
		suffix++
	}

	const contextLines = 2
	var b strings.Builder
	start := prefix - contextLines
	if start < 0 {
		start = 0
	}
	for i := start; i < prefix; i++ {
		fmt.Fprintf(&b, "  %s\n", e[i])
	}

	emitted := 0
	for i := prefix; i < len(e)-suffix && emitted < MaxLineChanges; i++ {
		fmt.Fprintf(&b, "- %s\n", e[i])
		emitted++
	}
	emitted = 0
	for i := prefix; i < len(a)-suffix && emitted < MaxLineChanges; i++ {
		fmt.Fprintf(&b, "+ %s\n", a[i])
		emitted++
	}

	end := len(e) - suffix + contextLines
	if end > len(e) {
		end = len(e)
	}
	for i := len(e) - suffix; i < end; i++ {
		fmt.Fprintf(&b, "  %s\n", e[i])
	}

	return b.String()
}
