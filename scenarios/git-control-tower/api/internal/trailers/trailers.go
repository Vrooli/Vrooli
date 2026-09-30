// Package trailers parses work-reference trailers without rewriting the
// original commit message or treating metadata as authorship proof.
package trailers

import (
	"context"
	"fmt"
	"strings"
)

var Supported = map[string]struct{}{"Vrooli-Plan": {}, "Vrooli-Backlog": {}, "Vrooli-Continues": {}}

type Entry struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Supported bool   `json:"supported"`
}

type Message struct {
	Original string  `json:"original"`
	Subject  string  `json:"subject"`
	Body     string  `json:"body"`
	Entries  []Entry `json:"entries"`
}

func Parse(text string) Message {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	subject := ""
	if len(lines) > 0 {
		subject = lines[0]
	}
	var body []string
	var entries []Entry
	for i := 1; i < len(lines); i++ {
		idx := strings.IndexByte(lines[i], ':')
		if idx <= 0 {
			body = append(body, lines[i])
			continue
		}
		key, value := strings.TrimSpace(lines[i][:idx]), strings.TrimSpace(lines[i][idx+1:])
		if !strings.HasPrefix(key, "Vrooli-") {
			body = append(body, lines[i])
			continue
		}
		entry := Entry{Key: key, Value: value}
		_, entry.Supported = Supported[key]
		for i+1 < len(lines) && (strings.HasPrefix(lines[i+1], " ") || strings.HasPrefix(lines[i+1], "\t")) {
			i++
			entry.Value += "\n" + strings.TrimSpace(lines[i])
		}
		entries = append(entries, entry)
	}
	return Message{Original: text, Subject: subject, Body: strings.TrimSpace(strings.Join(body, "\n")), Entries: entries}
}

type ResolutionStatus string

const (
	Resolved     ResolutionStatus = "resolved"
	Unresolved   ResolutionStatus = "unresolved"
	Ambiguous    ResolutionStatus = "ambiguous"
	Inaccessible ResolutionStatus = "inaccessible"
	Deleted      ResolutionStatus = "deleted"
	Legacy       ResolutionStatus = "legacy"
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

func Resolve(ctx context.Context, message Message, resolver Resolver) ([]Reference, error) {
	if resolver == nil {
		return nil, fmt.Errorf("trailer resolver is required")
	}
	refs := make([]Reference, 0, len(message.Entries))
	for _, entry := range message.Entries {
		ref := Reference{Key: entry.Key, Value: entry.Value, Kind: strings.TrimPrefix(entry.Key, "Vrooli-"), Status: Unresolved, Confidence: "asserted-metadata"}
		if !entry.Supported {
			ref.Status, ref.Detail = Legacy, "unknown Vrooli trailer preserved without owner resolution"
			refs = append(refs, ref)
			continue
		}
		ownerID, status, detail, err := resolver.Resolve(ctx, entry.Key, entry.Value)
		if err != nil {
			return nil, err
		}
		ref.OwnerID, ref.Status, ref.Detail = ownerID, status, detail
		if status == Resolved {
			ref.Confidence = "owner-resolved-reference"
		}
		refs = append(refs, ref)
	}
	return refs, nil
}
