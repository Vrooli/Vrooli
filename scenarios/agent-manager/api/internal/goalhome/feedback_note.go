package goalhome

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var feedbackNoteIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

var (
	ErrFeedbackNoteInvalid  = errors.New("invalid supervisor feedback entry")
	ErrFeedbackNoteConflict = errors.New("feedback ID already exists with different or resolved content")
	ErrFeedbackMalformed    = errors.New("FEEDBACK.md has malformed fenced content")
)

// AddSupervisorFeedback inserts one machine-readable supervisor feedback
// entry before ## Resolved. An identical open entry is a no-op; reusing an ID
// with changed content or after resolution is refused.
func AddSupervisorFeedback(data []byte, id, title, detail string, at time.Time) ([]byte, bool, error) {
	id = strings.TrimSpace(id)
	title = strings.TrimSpace(title)
	detail = strings.TrimSpace(strings.ReplaceAll(detail, "\r\n", "\n"))
	if !feedbackNoteIDPattern.MatchString(id) || title == "" || len(title) > 200 || strings.ContainsAny(title, "\r\n\x00") || !utf8.ValidString(detail) || detail == "" || len(detail) > 4096 {
		return nil, false, ErrFeedbackNoteInvalid
	}
	if at.IsZero() {
		return nil, false, fmt.Errorf("%w: timestamp required", ErrFeedbackNoteInvalid)
	}
	if strings.ContainsRune(detail, '\x00') {
		return nil, false, fmt.Errorf("%w: detail contains NUL", ErrFeedbackNoteInvalid)
	}
	if !balancedFeedbackFences(data) {
		return nil, false, ErrFeedbackMalformed
	}
	date := at.UTC().Format(time.RFC3339)
	body := feedbackNoteBody(detail)
	doc := parseDocument(data)
	resolvedAt := doc.offsetOf("resolved")
	matchingID := 0
	identicalOpenEntry := false
	for _, section := range doc.sections(3, 3) {
		words := strings.Fields(section.Title)
		sectionID := ""
		if len(words) > 0 {
			sectionID = words[0]
		}
		if sectionID != id {
			continue
		}
		matchingID++
		expectedPrefix := id + " — " + title + " ("
		sameTitle := false
		if strings.HasPrefix(section.Title, expectedPrefix) && strings.HasSuffix(section.Title, ", supervisor)") {
			dateText := strings.TrimSuffix(strings.TrimPrefix(section.Title, expectedPrefix), ", supervisor)")
			_, dateErr := time.Parse(time.RFC3339, dateText)
			sameTitle = dateErr == nil
		}
		sameOpenBody := strings.TrimSpace(section.bodyText()) == strings.TrimSpace(body)
		if resolvedAt >= 0 && section.Start >= resolvedAt {
			continue
		}
		parsed := ParseFeedback(data)
		open := false
		for _, entry := range parsed.Entries {
			if entry.Line == section.Line {
				open = entry.Open()
				break
			}
		}
		identicalOpenEntry = identicalOpenEntry || (sameTitle && sameOpenBody && open)
	}
	if matchingID > 1 {
		return nil, false, ErrFeedbackNoteConflict
	}
	if matchingID == 1 {
		if identicalOpenEntry {
			return data, false, nil
		}
		return nil, false, ErrFeedbackNoteConflict
	}
	entry := fmt.Sprintf("### %s — %s (%s, supervisor)\n\n%s", id, title, date, body)
	atOffset := len(data)
	if resolvedAt >= 0 {
		atOffset = resolvedAt
	}
	return splice(data, atOffset, atOffset, entry), true, nil
}

func feedbackNoteBody(detail string) string {
	var b strings.Builder
	for _, line := range strings.Split(detail, "\n") {
		b.WriteString("> ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("\nRecorded by `agent-manager effort feedback add`.\n\nStatus: open\n")
	return b.String()
}

func balancedFeedbackFences(data []byte) bool {
	fenced := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fenced = !fenced
		}
	}
	return !fenced
}
