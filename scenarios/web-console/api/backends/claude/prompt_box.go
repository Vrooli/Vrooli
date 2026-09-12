package claude

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"web-console/internal/backend"
)

// A Claude Code prompt box, as rendered by 2.1.x, ends in a hint line and
// lists its options above it, the selected one marked with "❯":
//
//	Do you want to proceed?
//	❯ 1. Yes
//	  2. Yes, and don't ask again for date commands in /work
//	  3. No, and tell Claude what to do differently (esc)
//	Enter to confirm · Esc to cancel
//
// Options may be numbered (permission prompts) or not (the workspace trust
// dialog); descriptions sit deeper than the option labels. The parser reads
// only the bottom of the screen and refuses anything it cannot place: a wrong
// parse is worse than "asking you something" with no content.

var (
	promptHint      = regexp.MustCompile(`(?i)enter to (confirm|select|submit)|esc to cancel|to navigate`)
	numberedOption  = regexp.MustCompile(`^(\d+)\.\s+(.+)$`)
	permissionAsk   = regexp.MustCompile(`(?i)do you want to (proceed|make this edit|create|allow|run)|trust this folder|permission`)
	freeTextOption  = regexp.MustCompile(`(?i)^(type something|other\b|no, and tell claude)`)
	tmuxStatusLine  = regexp.MustCompile(`^\[wc-[^\]]*\]`)
	ruleOnlyPattern = regexp.MustCompile(`^[\s─━═╌╍┄┅╭╮╰╯│┃-]+$`)
)

// promptScanLines bounds how far above the bottom the parser looks.
const promptScanLines = 40

// promptQuestionLines bounds the question text kept above the options.
const promptQuestionLines = 16

// parsePromptBox reads a prompt box from the bottom of the screen. A box is
// anchored by its hint line or, when it has none (the AskUserQuestion box in
// 2.1.268), by the cursor resting on its selected numbered option.
func parsePromptBox(all []string, cursorRow int) *backend.PendingPrompt {
	lines, offset := visibleTail(all, promptScanLines)
	hint := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if promptHint.MatchString(lines[i]) {
			hint = i
			break
		}
	}

	// The selected option marks the option column.
	selected := -1
	end := hint
	if hint >= 1 {
		for i := hint - 1; i >= 0; i-- {
			if strings.HasPrefix(strings.TrimLeftFunc(stripBorder(lines[i]), unicode.IsSpace), claudePromptGlyph) {
				selected = i
				break
			}
		}
	} else if row := cursorRow - offset; row >= 0 && row < len(lines) {
		body := strings.TrimLeftFunc(stripBorder(lines[row]), unicode.IsSpace)
		if strings.HasPrefix(body, claudePromptGlyph) && numberedOption.MatchString(strings.TrimSpace(strings.TrimPrefix(body, claudePromptGlyph))) {
			selected = row
			end = len(lines)
		}
	}
	if selected < 0 {
		return nil
	}
	selLine := stripBorder(lines[selected])
	glyphCol := strings.Index(selLine, claudePromptGlyph)
	labelCol := runeCol(selLine, glyphCol) + 2

	type optionLine struct {
		row   int
		label string
	}
	var options []optionLine
	first := selected
	// Walk up and down from the selected option collecting option rows; rows
	// indented deeper than the label column are descriptions and are skipped.
	isOption := func(line string) (string, bool) {
		line = stripBorder(line)
		if strings.TrimSpace(line) == "" {
			return "", false
		}
		if idx := strings.Index(line, claudePromptGlyph); idx >= 0 && strings.TrimSpace(line[:idx]) == "" {
			return strings.TrimSpace(line[idx+len(claudePromptGlyph):]), true
		}
		indent := leadingSpaces(line)
		if indent == labelCol {
			return strings.TrimSpace(line), true
		}
		return "", false
	}
	isDescription := func(line string) bool {
		line = stripBorder(line)
		return strings.TrimSpace(line) != "" && leadingSpaces(line) > labelCol
	}
	for i := selected; i >= 0; i-- {
		if label, ok := isOption(lines[i]); ok {
			options = append([]optionLine{{row: i, label: label}}, options...)
			first = i
			continue
		}
		if isDescription(lines[i]) {
			continue
		}
		break
	}
walkDown:
	for i := selected + 1; i < end; i++ {
		if label, ok := isOption(lines[i]); ok {
			options = append(options, optionLine{row: i, label: label})
			continue
		}
		body := strings.TrimSpace(stripBorder(lines[i]))
		switch {
		case body == "" || isDescription(lines[i]):
			continue
		case ruleOnlyPattern.MatchString(body):
			// A rule can separate the last option ("Chat about this").
			continue
		case hint < 0:
			// Without a hint the box ends at the first line that is not part of it.
			break walkDown
		default:
			// Anything else between the options and the hint is not a prompt box.
			return nil
		}
	}
	if len(options) < 2 {
		return nil
	}
	// Without a hint the box is AskUserQuestion's, which always ends its list
	// with a free-text row ("Type something."). A box without one is still
	// being drawn, or is not a question box: leave it at "asking you something"
	// rather than show a partial list as the whole question.
	if hint < 0 {
		complete := false
		for _, option := range options {
			label := option.label
			if m := numberedOption.FindStringSubmatch(label); m != nil {
				label = m[2]
			}
			if freeTextOption.MatchString(strings.TrimSpace(label)) {
				complete = true
				break
			}
		}
		if !complete {
			return nil
		}
	}

	prompt := &backend.PendingPrompt{Kind: "question"}
	for position, option := range options {
		key := strconv.Itoa(position + 1)
		label := option.label
		if m := numberedOption.FindStringSubmatch(label); m != nil {
			key, label = m[1], strings.TrimSpace(m[2])
		}
		prompt.Options = append(prompt.Options, backend.PromptOption{Key: key, Label: label, Selected: option.row == selected})
		if prompt.FreeTextHint == "" && freeTextOption.MatchString(label) {
			prompt.FreeTextHint = label
		}
	}

	// The question: the box body above the options, up to its top rule (or
	// promptQuestionLines lines), with paragraphs kept apart by one blank line.
	// A permission box's body names the tool call; its last line asks.
	var question []string
	lastBlank := false
	for i, kept := first-1, 0; i >= 0 && kept < promptQuestionLines; i-- {
		line := strings.TrimSpace(stripBorder(lines[i]))
		if line != "" && ruleOnlyPattern.MatchString(line) {
			break
		}
		if line == "" {
			if len(question) > 0 && !lastBlank {
				question = append([]string{""}, question...)
			}
			lastBlank = true
			continue
		}
		// AskUserQuestion heads its box with a checkbox and a short header.
		line = strings.TrimSpace(strings.TrimLeft(line, "☐☒✔"))
		question = append([]string{line}, question...)
		lastBlank = false
		kept++
	}
	for len(question) > 0 && question[0] == "" {
		question = question[1:]
	}
	prompt.Text = strings.Join(question, "\n")
	// A hint offering Escape ("Esc to cancel") means the box can be dismissed.
	prompt.Cancellable = hint >= 0 && strings.Contains(strings.ToLower(lines[hint]), "esc to cancel")
	// The trust dialog asks in its option ("Yes, I trust this folder").
	classified := prompt.Text
	for _, option := range prompt.Options {
		classified += "\n" + option.Label
	}
	if permissionAsk.MatchString(classified) {
		prompt.Kind = "permission"
	}
	return prompt
}

// visibleTail returns the last n lines with trailing blank rows and the tmux
// status line removed, and the index of its first line in all.
func visibleTail(all []string, n int) ([]string, int) {
	end := len(all)
	for end > 0 {
		line := strings.TrimSpace(all[end-1])
		if line == "" || tmuxStatusLine.MatchString(line) {
			end--
			continue
		}
		break
	}
	start := end - n
	if start < 0 {
		start = 0
	}
	return all[start:end], start
}

// stripBorder removes a box's vertical borders so bordered and unbordered
// prompts parse alike.
func stripBorder(line string) string {
	trimmed := strings.TrimRightFunc(line, unicode.IsSpace)
	trimmed = strings.TrimSuffix(trimmed, "│")
	if idx := strings.Index(trimmed, "│"); idx >= 0 && strings.TrimSpace(trimmed[:idx]) == "" {
		trimmed = strings.Repeat(" ", runeCol(trimmed, idx)) + trimmed[idx+len("│"):]
	}
	return trimmed
}

func leadingSpaces(line string) int {
	n := 0
	for _, r := range line {
		if r != ' ' {
			break
		}
		n++
	}
	return n
}

// runeCol converts a byte offset into a column count.
func runeCol(line string, byteOffset int) int {
	if byteOffset <= 0 {
		return 0
	}
	return len([]rune(line[:byteOffset]))
}
