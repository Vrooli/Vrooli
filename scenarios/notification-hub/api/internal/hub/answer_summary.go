package hub

import (
	"strings"
	"unicode/utf8"
)

const maxSummaryLabelRunes = 80

// AnswerSummary renders an operator's whole answer as plain text a requester
// can paste into its own log: the chosen option, the note, and each response
// to the answer form.
//
//	Answer: Approve with changes (approve)
//	Note: Keep the retention period short.
//	Layout: Card stack (cards)
//	Plan sections:
//	  - Retention: change — keep 30 days, not 90
//	  - Billing: drop
//	  - Other sections: approve
//	Notes on files:
//	  - character.glb: Hair is too glossy.
//
// A question's prompt, shortened and without its closing punctuation, is the
// line's label. A section review lists the sections whose verdict differs from
// the question's unanswered verdict, or that carry a note, and then names the
// verdict of all the others. Continuation lines of a multi-line note or text
// answer are indented two spaces (four inside a list). An ask that no operator has answered (still open, or
// closed by default, expiry, withdrawal, or dismissal) has no summary.
func AnswerSummary(record AskRecord) string {
	if record.State != AskAnswered || record.Answer == "" {
		return ""
	}
	lines := []string{"Answer: " + answerLine(record)}
	if record.Note != "" {
		lines = append(lines, "Note: "+indentContinuation(record.Note))
	}
	questions := make(map[string]AskQuestion, len(record.Questions))
	for _, question := range record.Questions {
		questions[question.ID] = question
	}
	for _, response := range record.Responses {
		question, known := questions[response.QuestionID]
		if !known {
			question = AskQuestion{ID: response.QuestionID, Prompt: response.QuestionID}
		}
		lines = append(lines, responseLines(question, response)...)
	}
	if len(record.AttachmentNotes) > 0 {
		lines = append(lines, "Notes on files:")
		for _, note := range record.AttachmentNotes {
			lines = append(lines, "  - "+note.target()+": "+indentListItem(note.Note))
		}
	}
	return strings.Join(lines, "\n")
}

func answerLine(record AskRecord) string {
	label := record.AnswerLabel
	if label == "" || label == record.Answer {
		return record.Answer
	}
	return label + " (" + record.Answer + ")"
}

func responseLines(question AskQuestion, response AskQuestionResponse) []string {
	label := summaryLabel(question.Prompt)
	if question.Type == QuestionSectionReview {
		return reviewLines(label, question, response)
	}
	var parts []string
	for _, key := range response.Selected {
		name := key
		if choice, ok := question.choice(key); ok {
			name = choice.Label
		}
		if name == key {
			parts = append(parts, key)
		} else {
			parts = append(parts, name+" ("+key+")")
		}
	}
	value := strings.Join(parts, ", ")
	if response.Text != "" {
		if value != "" {
			value += " — "
		}
		value += indentContinuation(response.Text)
	}
	return []string{label + ": " + value}
}

// summaryLabel turns a question's prompt into a short line label: whitespace
// collapsed, closing "?", ":" or "." removed, cut at maxSummaryLabelRunes.
func summaryLabel(prompt string) string {
	label := strings.TrimRight(strings.Join(strings.Fields(prompt), " "), "?:. ")
	if utf8.RuneCountInString(label) > maxSummaryLabelRunes {
		label = string([]rune(label)[:maxSummaryLabelRunes-1]) + "…"
	}
	return label
}

func indentContinuation(text string) string {
	return strings.ReplaceAll(text, "\n", "\n  ")
}

// reviewLines renders a section review: the sections that differ from the
// unanswered verdict (or carry a note) one per line, then the rest as one.
func reviewLines(label string, question AskQuestion, response AskQuestionResponse) []string {
	lines := []string{label + ":"}
	unanswered := question.unansweredVerdict()
	listed := 0
	for _, verdict := range response.Sections {
		if verdict.Verdict == unanswered && verdict.Note == "" {
			continue
		}
		listed++
		title := verdict.SectionKey
		if section, ok := question.section(verdict.SectionKey); ok {
			title = section.Title
		}
		line := "  - " + title + ": " + verdict.Verdict
		if verdict.Note != "" {
			line += " — " + indentListItem(verdict.Note)
		}
		lines = append(lines, line)
	}
	if len(question.Sections) > listed {
		lines = append(lines, "  - Other sections: "+unanswered)
	}
	if response.Text != "" {
		lines = append(lines, "  Note: "+indentContinuation(response.Text))
	}
	return lines
}

// indentListItem indents the continuation lines of text inside a list item.
func indentListItem(text string) string {
	return strings.ReplaceAll(text, "\n", "\n    ")
}
