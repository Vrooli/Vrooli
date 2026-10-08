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
	entries := []Entry{}
	for _, entry := range message.Entries {
		switch strings.ToLower(entry.Key) {
		case "vrooli-effort", "vrooli-plan", "vrooli-backlog", "vrooli-work":
			entries = append(entries, entry)
		}
	}
	entries = append(entries, Entry{Key: "Vrooli-Continues", Value: predecessor})
	return Render(subject, "", entries)
}

func validPredecessor(value string) bool {
	spec, _ := SpecFor("Vrooli-Continues")
	return spec.Grammar.MatchString(value)
}
