package trailers

import "strings"

// Work is the set of work links rendered as generated trailers.
type Work struct {
	EffortRef      string
	EffortRevision string
	// Epoch is "E<n>"; it renders self-contained as "<effort ref>#E<n>".
	Epoch      string
	RunIDs     []string
	Plans      []string
	Continues  string
	ProposalID string
}

// NormalizeEpoch returns "E<n>" for "E27", "e27" or "27".
func NormalizeEpoch(epoch string) string {
	trimmed := strings.TrimSpace(epoch)
	if trimmed == "" {
		return ""
	}
	if trimmed[0] == 'e' || trimmed[0] == 'E' {
		trimmed = trimmed[1:]
	}
	return "E" + trimmed
}

// WorkEntries returns the generated trailers for work links in registry
// order. Values are rendered as given; Validate reports grammar errors.
func WorkEntries(work Work) []Entry {
	var entries []Entry
	add := func(key, value string) {
		if value = strings.TrimSpace(value); value != "" {
			entries = append(entries, Entry{Key: key, Value: value, Supported: true})
		}
	}
	effort := strings.TrimSpace(work.EffortRef)
	if effort != "" {
		value := effort
		if revision := strings.TrimSpace(work.EffortRevision); revision != "" {
			value += "@" + revision
		}
		add("Vrooli-Effort", value)
	}
	if epoch := NormalizeEpoch(work.Epoch); epoch != "" {
		add("Vrooli-Epoch", effort+"#"+epoch)
	}
	for _, run := range work.RunIDs {
		add("Vrooli-Run", strings.ToLower(run))
	}
	for _, plan := range work.Plans {
		add("Vrooli-Plan", plan)
	}
	add("Vrooli-Continues", work.Continues)
	add("Vrooli-Proposal", work.ProposalID)
	return entries
}

// Normalize gives registry keys their canonical case, trims values and drops
// exact duplicates, keeping the first. It never reorders entries.
func Normalize(entries []Entry) []Entry {
	result := make([]Entry, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		if entry.Key == "" {
			result = append(result, entry)
			continue
		}
		canonical, supported := CanonicalKey(entry.Key)
		normalized := Entry{Key: canonical, Value: strings.TrimSpace(entry.Value), Supported: supported, Raw: entry.Raw}
		identity := strings.ToLower(canonical) + "\x00" + normalized.Value
		if _, duplicate := seen[identity]; duplicate {
			continue
		}
		seen[identity] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

// Render builds the exact commit message: subject, a blank line, the body,
// a blank line and the trailer block. The body is already in the form git's
// default whitespace cleanup produces (no trailing spaces, no repeated blank
// lines), so the committed message equals the rendered text.
func Render(subject, body string, entries []Entry) string {
	var builder strings.Builder
	builder.WriteString(strings.TrimSpace(subject))
	if cleaned := CleanBody(body); cleaned != "" {
		builder.WriteString("\n\n")
		builder.WriteString(cleaned)
	}
	if len(entries) > 0 {
		builder.WriteString("\n")
		for _, entry := range entries {
			builder.WriteString("\n")
			builder.WriteString(renderEntry(entry))
		}
	}
	return builder.String()
}

func renderEntry(entry Entry) string {
	if entry.Raw != "" && rawStillMatches(entry) {
		return entry.Raw
	}
	if entry.Key == "" {
		return entry.Value
	}
	return entry.Key + ": " + strings.TrimSpace(entry.Value)
}

// rawStillMatches reports whether a parsed entry's original text still
// expresses its current key and value, so unedited folding is kept.
func rawStillMatches(entry Entry) bool {
	lines := strings.Split(entry.Raw, "\n")
	if entry.Key == "" {
		return entry.Raw == entry.Value
	}
	key, value, ok := splitTrailerLine(lines[0])
	if !ok {
		return false
	}
	for _, line := range lines[1:] {
		value = strings.TrimSpace(value + " " + strings.TrimSpace(line))
	}
	// A raw key in non-canonical case re-renders in canonical case.
	return key == entry.Key && value == strings.TrimSpace(entry.Value)
}

// CleanBody applies git's whitespace cleanup to a body: CRLF becomes LF,
// trailing whitespace is removed and runs of blank lines collapse to one.
func CleanBody(body string) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	var kept []string
	blank := false
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			blank = true
			continue
		}
		if blank && len(kept) > 0 {
			kept = append(kept, "")
		}
		blank = false
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
