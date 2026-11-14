package trailers

import (
	"regexp"
	"strconv"
	"strings"
)

var continuationPart = regexp.MustCompile(`^(.+)\s+([pP])([0-9]+)$`)

// Continue carries work relationships forward, never the previous proposal,
// run, epoch, or author attestations. The predecessor is an explicit Git link.
func Continue(text, predecessor string) string {
	message := Parse(text)
	if message.Subject == "" || !validPredecessor(predecessor) {
		return ""
	}
	subject := message.Subject
	if part := continuationPart.FindStringSubmatch(subject); part != nil {
		number, err := strconv.ParseUint(part[3], 10, 32)
		if err != nil || number == 1<<32-1 {
			return ""
		}
		subject = part[1] + " " + part[2] + strconv.FormatUint(number+1, 10)
	}
	return Render(subject, "", ContinuationEntries(message, predecessor))
}

// ContinuationEntries are the trailers a successor carries: the predecessor's
// work links, then the explicit Vrooli-Continues link.
func ContinuationEntries(predecessorMessage Message, predecessor string) []Entry {
	entries := []Entry{}
	for _, entry := range predecessorMessage.Entries {
		switch strings.ToLower(entry.Key) {
		case "vrooli-effort", "vrooli-plan", "vrooli-backlog", "vrooli-work":
			entries = append(entries, entry)
		}
	}
	return append(entries, Entry{Key: "Vrooli-Continues", Value: predecessor})
}

// ValidPredecessor reports whether value is a Vrooli-Continues commit reference.
func ValidPredecessor(value string) bool { return validPredecessor(value) }

func validPredecessor(value string) bool {
	spec, _ := SpecFor("Vrooli-Continues")
	return spec.Grammar.MatchString(value)
}
