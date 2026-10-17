package goalhome

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// HandoffMaxBytes caps the one handoff (decision P-06). Census prose belongs
// in ## Censuses or archive/.
const HandoffMaxBytes = 4096

// ErrConcurrentChange means the file changed between read and write.
var ErrConcurrentChange = errors.New("QUEUE.md changed while the handoff was being written; read it again and retry")

// HandoffError is a refused handoff replacement.
type HandoffError struct {
	Code   string
	Detail string
}

func (e *HandoffError) Error() string { return e.Code + ": " + e.Detail }

var topHeadingPattern = regexp.MustCompile(`^#{1,2}\s`)

// SetHandoff returns queue with its single "## Handoff" section holding body.
// It creates the section after ## Needs operator (or before the first level-2
// section) when there is none. changed is false when the section already
// holds body. It refuses a body over HandoffMaxBytes, a body with its own
// level-1/2 or handoff heading, and a queue with more than one handoff: older
// handoffs are history, so they move to archive/ by hand instead of being
// deleted here.
func SetHandoff(queue []byte, body string) ([]byte, bool, error) {
	body = normalizeHandoff(body)
	if body == "" {
		return nil, false, &HandoffError{Code: "handoff-empty", Detail: "the handoff text is empty"}
	}
	if len(body) > HandoffMaxBytes {
		return nil, false, &HandoffError{Code: "handoff-size", Detail: fmt.Sprintf("the handoff is %d bytes; the limit is %d. Move census and history prose to ## Censuses or archive/", len(body), HandoffMaxBytes)}
	}
	for _, l := range strings.Split(body, "\n") {
		if topHeadingPattern.MatchString(l) || (headingPattern.MatchString(l) && handoffWord.MatchString(l)) {
			return nil, false, &HandoffError{Code: "handoff-content", Detail: fmt.Sprintf("the handoff may not contain the heading %q", strings.TrimSpace(l))}
		}
	}
	doc := parseDocument(queue)
	var handoffs []section
	for _, s := range doc.sections(2, 3) {
		if handoffWord.MatchString(s.Title) {
			handoffs = append(handoffs, s)
		}
	}
	block := "## Handoff\n\n" + body + "\n"
	switch {
	case len(handoffs) > 1:
		described := make([]HandoffSection, 0, len(handoffs))
		for _, s := range handoffs {
			described = append(described, HandoffSection{Title: s.Title, Level: s.Level, Line: s.Line})
		}
		return nil, false, &HandoffError{Code: "handoff-count", Detail: "QUEUE.md has " + fmt.Sprint(len(handoffs)) + " handoff sections: " + describeHandoffs(described) + ". Move the old ones to archive/ first"}
	case len(handoffs) == 1 && handoffs[0].Level != 2:
		return nil, false, &HandoffError{Code: "handoff-count", Detail: fmt.Sprintf("the only handoff is the level-%d heading %q (line %d); move it to archive/ first", handoffs[0].Level, handoffs[0].Title, handoffs[0].Line)}
	case len(handoffs) == 1:
		current := handoffs[0]
		if current.Title == "Handoff" && current.bodyText() == body {
			return queue, false, nil
		}
		return splice(queue, current.Start, current.End, block), true, nil
	}
	at := doc.offsetOf("needs operator")
	if at >= 0 {
		needs, _ := doc.section("needs operator")
		at = needs.End
	} else if sections := doc.sections(2, 2); len(sections) > 0 {
		at = sections[0].Start
	} else {
		at = len(queue)
	}
	return splice(queue, at, at, block), true, nil
}

// normalizeHandoff trims the text and drops a leading "## Handoff" heading.
func normalizeHandoff(body string) string {
	body = strings.TrimSpace(strings.ReplaceAll(body, "\r\n", "\n"))
	if first, rest, _ := strings.Cut(body, "\n"); strings.EqualFold(strings.TrimSpace(first), "## Handoff") {
		body = strings.TrimSpace(rest)
	}
	return body
}

// splice replaces queue[start:end] with block, keeping one blank line between
// the block and its neighbours.
func splice(queue []byte, start, end int, block string) []byte {
	before := strings.TrimRight(string(queue[:start]), "\n")
	after := strings.TrimLeft(string(queue[end:]), "\n")
	var out strings.Builder
	if before != "" {
		out.WriteString(before + "\n\n")
	}
	out.WriteString(block)
	if after != "" {
		out.WriteString("\n" + after)
	}
	return []byte(out.String())
}

// WriteIfUnchanged atomically replaces path with updated (temporary file and
// rename, keeping the file mode) only while the file still holds original.
func WriteIfUnchanged(path string, original, updated []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(updated); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), info.Mode().Perm())
	}
	if err != nil {
		return err
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, original) {
		return ErrConcurrentChange
	}
	return os.Rename(tmp.Name(), path)
}
