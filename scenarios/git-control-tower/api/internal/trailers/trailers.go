// Package trailers parses, validates and renders work-reference trailers
// without rewriting the original commit message or treating metadata as
// authorship proof.
//
// Parsing follows git's trailer rules (git interpret-trailers): the trailer
// block is the final paragraph of the message, never the title paragraph;
// every key is kept, not only Vrooli keys, so Co-Authored-By and
// Signed-off-by round-trip in order; whitespace-led lines continue the
// previous trailer and unfold to one space. A "Vrooli-Plan:" line inside an
// earlier paragraph is body text.
package trailers

import (
	"context"
	"fmt"
	"strings"
)

// Supported is the set of canonical registry keys. It is derived from
// Registry so a new kind is one registry row.
var Supported = func() map[string]struct{} {
	keys := make(map[string]struct{}, len(Registry))
	for _, spec := range Registry {
		keys[spec.Key] = struct{}{}
	}
	return keys
}()

// Entry is one line of a trailer block. A trailer has a Key; a non-trailer
// line kept inside a mixed git trailer block has an empty Key and its text in
// Value.
type Entry struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Supported bool   `json:"supported"`
	// Raw is the original text of the entry, including continuation lines.
	// Render emits Raw unchanged for parsed entries whose Key and Value were
	// not edited, so an operator's folding survives a round trip.
	Raw string `json:"raw,omitempty"`
}

// Message is a commit message split into title, body and trailer block.
type Message struct {
	Original string  `json:"original"`
	Subject  string  `json:"subject"`
	Body     string  `json:"body"`
	Entries  []Entry `json:"entries"`
}

// gitGeneratedPrefixes are the trailer prefixes git itself writes. A final
// paragraph with at least one of these needs only 25% trailer lines to count
// as a trailer block, as in git's trailer.c.
var gitGeneratedPrefixes = []string{"Signed-off-by: ", "(cherry picked from commit "}

// Parse splits a commit message with git's trailer rules.
func Parse(text string) Message {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(strings.TrimRight(normalized, " \t\n"), "\n")
	msg := Message{Original: text}
	if len(lines) == 0 || (len(lines) == 1 && strings.TrimSpace(lines[0]) == "") {
		return msg
	}
	paragraphs := splitParagraphs(lines)
	msg.Subject = strings.TrimSpace(paragraphs[0][0])
	var bodyParts []string
	if rest := paragraphs[0][1:]; len(rest) > 0 {
		bodyParts = append(bodyParts, strings.Join(rest, "\n"))
	}
	middle := paragraphs[1:]
	if len(paragraphs) > 1 {
		last := paragraphs[len(paragraphs)-1]
		if entries, ok := parseTrailerBlock(last); ok {
			msg.Entries = entries
			middle = paragraphs[1 : len(paragraphs)-1]
		}
	}
	for _, paragraph := range middle {
		bodyParts = append(bodyParts, strings.Join(paragraph, "\n"))
	}
	msg.Body = strings.TrimSpace(strings.Join(bodyParts, "\n\n"))
	return msg
}

// splitParagraphs groups non-blank lines separated by blank lines.
func splitParagraphs(lines []string) [][]string {
	var paragraphs [][]string
	var current []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if len(current) > 0 {
				paragraphs = append(paragraphs, current)
				current = nil
			}
			continue
		}
		current = append(current, strings.TrimRight(line, " \t"))
	}
	if len(current) > 0 {
		paragraphs = append(paragraphs, current)
	}
	return paragraphs
}

// parseTrailerBlock reports whether a paragraph is a git trailer block and
// returns its entries in order.
func parseTrailerBlock(paragraph []string) ([]Entry, bool) {
	var entries []Entry
	trailerLines, otherLines := 0, 0
	recognized := false
	for _, line := range paragraph {
		if isContinuation(line) && len(entries) > 0 {
			last := &entries[len(entries)-1]
			last.Raw += "\n" + line
			if last.Key != "" {
				last.Value = strings.TrimSpace(last.Value + " " + strings.TrimSpace(line))
			} else {
				last.Value += "\n" + line
			}
			continue
		}
		for _, prefix := range gitGeneratedPrefixes {
			if strings.HasPrefix(line, prefix) {
				recognized = true
			}
		}
		key, value, ok := splitTrailerLine(line)
		if !ok {
			otherLines++
			entries = append(entries, Entry{Value: line, Raw: line})
			continue
		}
		trailerLines++
		canonical, supported := CanonicalKey(key)
		entries = append(entries, Entry{Key: canonical, Value: value, Supported: supported, Raw: line})
	}
	if trailerLines == 0 {
		return nil, false
	}
	if otherLines == 0 {
		return entries, true
	}
	// git accepts a mixed block when a git-generated trailer is present and at
	// least a quarter of the lines are trailers.
	if recognized && trailerLines*3 >= otherLines {
		return entries, true
	}
	return nil, false
}

func isContinuation(line string) bool {
	return strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
}

// splitTrailerLine applies git's separator rule: a token of letters, digits
// and '-', optional whitespace, then ':'.
func splitTrailerLine(line string) (string, string, bool) {
	if line == "" || isContinuation(line) {
		return "", "", false
	}
	whitespace := false
	for i, r := range line {
		switch {
		case r == ':':
			if i == 0 {
				return "", "", false
			}
			return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
		case !whitespace && (r == '-' || ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || ('0' <= r && r <= '9')):
			continue
		case i > 0 && (r == ' ' || r == '\t'):
			whitespace = true
		default:
			return "", "", false
		}
	}
	return "", "", false
}

// CanonicalKey returns the registry spelling of a key matched without regard
// to case. Unknown keys are returned unchanged.
func CanonicalKey(key string) (string, bool) {
	trimmed := strings.TrimSpace(key)
	for _, spec := range Registry {
		if strings.EqualFold(spec.Key, trimmed) {
			return spec.Key, true
		}
	}
	return trimmed, false
}

// IsVrooliKey reports whether a key uses the Vrooli namespace.
func IsVrooliKey(key string) bool {
	return len(key) > len("Vrooli-") && strings.EqualFold(key[:len("Vrooli-")], "Vrooli-")
}

type ResolutionStatus string

const (
	Resolved      ResolutionStatus = "resolved"
	Unresolved    ResolutionStatus = "unresolved"
	Ambiguous     ResolutionStatus = "ambiguous"
	Inaccessible  ResolutionStatus = "inaccessible"
	Deleted       ResolutionStatus = "deleted"
	Legacy        ResolutionStatus = "legacy"
	NotApplicable ResolutionStatus = "not_applicable"
)

type Reference struct {
	Key        string           `json:"key"`
	Value      string           `json:"value"`
	Kind       string           `json:"kind"`
	OwnerID    string           `json:"ownerId,omitempty"`
	Status     ResolutionStatus `json:"status"`
	Confidence string           `json:"confidence"`
	Detail     string           `json:"detail,omitempty"`
}

// Resolver is deliberately owner-backed. Implementations must resolve exact
// owner references; filename or path similarity is not a valid fallback.
type Resolver interface {
	Resolve(ctx context.Context, key, value string) (ownerID string, status ResolutionStatus, detail string, err error)
}

// Resolve maps every trailer of a message to a reference. Resolution never
// fails a message: a resolver error becomes an unresolved reference.
func Resolve(ctx context.Context, message Message, resolver Resolver) ([]Reference, error) {
	if resolver == nil {
		return nil, fmt.Errorf("trailer resolver is required")
	}
	return ResolveEntries(ctx, message.Entries, resolver), nil
}

// ResolveEntries resolves trailer entries in order. Non-trailer lines are
// skipped. Unknown Vrooli keys stay legacy and are never sent to the
// resolver, so an old initiative key cannot become a current work item
// without owner evidence. Private or deleted items never carry an owner title.
func ResolveEntries(ctx context.Context, entries []Entry, resolver Resolver) []Reference {
	refs := make([]Reference, 0, len(entries))
	for _, entry := range entries {
		if entry.Key == "" {
			continue
		}
		ref := Reference{Key: entry.Key, Value: entry.Value, Kind: KindOf(entry.Key, entry.Value), Status: Unresolved, Confidence: "asserted-metadata"}
		switch {
		case !IsVrooliKey(entry.Key):
			ref.Status, ref.Detail = NotApplicable, "non-Vrooli trailer preserved as written"
		case !entry.Supported:
			ref.Status, ref.Detail = Legacy, "unknown Vrooli trailer preserved without owner resolution"
		case resolver == nil:
			ref.Detail = "no owner resolver is configured"
		default:
			ownerID, status, detail, err := resolver.Resolve(ctx, entry.Key, entry.Value)
			switch {
			case err != nil:
				ref.Detail = "owner resolution unavailable: " + err.Error()
			case status == Inaccessible:
				ref.Status, ref.Detail = Inaccessible, "item is private or inaccessible; title withheld"
			case status == Deleted:
				ref.Status, ref.Detail = Deleted, "owner reports the item deleted"
			default:
				ref.OwnerID, ref.Status, ref.Detail = ownerID, status, detail
				if ref.Status == "" {
					ref.Status = Unresolved
				}
			}
			if ref.Status == Resolved {
				ref.Confidence = "owner-resolved-reference"
			}
		}
		refs = append(refs, ref)
	}
	return refs
}
