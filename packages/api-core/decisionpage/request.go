// Package decisionpage is the shared decision-page writer. A sender that holds
// the facts of an operator decision but cannot design a page (scenario code,
// a courier, a steward script) hands those facts and the paths of its evidence
// to the writer. The writer starts one Codex Luna run through Agent Manager
// with a tight brief that follows the notification-hub skill, waits for it
// within a bound, checks the page with Notification Hub's own packet check
// (CheckAsk, the check behind `conversations ask --dry-run`), gives the run
// one more attempt with the findings, and returns page.html or a typed
// failure. A sender never builds a page that only restates its question.
//
// The writer keeps one job folder per sender, subject key and request digest
// under Root, so asking again for the same facts is idempotent: it reads the
// job's state instead of starting another run. Advance moves a job one step
// and never blocks; Write loops Advance for callers that can wait.
package decisionpage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Request is everything the sender knows about one decision. The writer
// never invents facts: what the page shows comes from these fields and the
// files named in Evidence.
type Request struct {
	// Sender is who asks, as the operator reads it ("Research Desk").
	Sender string `json:"sender"`
	// SubjectKey is the decision's stable situation key, the same one the ask
	// carries (Hub rule: 1-128 of A-Z a-z 0-9 . _ : / @ -).
	SubjectKey string `json:"subject_key"`
	// Question is the ask's question, as the operator will read it.
	Question string `json:"question"`
	// BigPicture is the ask's big picture text, if the sender has one.
	BigPicture string `json:"big_picture,omitempty"`
	// Options is the answer menu, at least two.
	Options []Option `json:"options"`
	// Recommended is the recommended option key; empty for no recommendation.
	Recommended string `json:"recommended,omitempty"`
	// RecommendationReason is why, in one plain sentence.
	RecommendationReason string `json:"recommendation_reason,omitempty"`
	// IfNoAnswer is what happens when the operator does not answer.
	IfNoAnswer string `json:"if_no_answer,omitempty"`
	// Cause is why this decision is coming up at all: the gate, defect or
	// rule behind it, and what would stop it coming up again.
	Cause string `json:"cause,omitempty"`
	// History is what the sender knows of earlier asks on the same subject
	// ("third time since 13 Nov").
	History string `json:"history,omitempty"`
	// Commands are what the operator must run themselves for an answer.
	Commands []Command `json:"commands,omitempty"`
	// Facts is free text the sender holds that is not in a file: numbers,
	// lists, names. Markdown is fine.
	Facts string `json:"facts,omitempty"`
	// Evidence are absolute paths of files the writer reads for real numbers,
	// names and lists: the decision's packet, a table, a report.
	Evidence []string `json:"evidence,omitempty"`
	// TechnicalDetails is the ask's technical details, for the writer's
	// understanding only; the page shows no IDs or paths from it.
	TechnicalDetails string `json:"technical_details,omitempty"`
}

// Option is one answer. Effect is what happens next when it is chosen, from
// the sender's packet; the page needs it to say what each answer does.
type Option struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Effect string `json:"effect,omitempty"`
}

// Command is something the operator must run for an answer. The page shows
// it on its first screen: the exact command, where to run it, what it shows
// before it acts and how to undo it.
type Command struct {
	For   string `json:"for,omitempty"` // the option key that needs it
	Run   string `json:"run"`
	Where string `json:"where,omitempty"`
	Shows string `json:"shows,omitempty"`
	Undo  string `json:"undo,omitempty"`
}

var (
	subjectKeyRule = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/@-]{0,127}$`)
	optionKeyRule  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
)

// ErrInvalidRequest wraps every reason a request cannot be written.
var ErrInvalidRequest = errors.New("invalid decision-page request")

// Validate says why the request cannot be written, or nil.
func (r Request) Validate() error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidRequest, fmt.Sprintf(format, args...))
	}
	switch {
	case strings.TrimSpace(r.Sender) == "":
		return bad("sender is required")
	case !subjectKeyRule.MatchString(r.SubjectKey):
		return bad("subject_key %q does not follow the Hub's subject-key rule", r.SubjectKey)
	case strings.TrimSpace(r.Question) == "":
		return bad("question is required")
	case len(r.Options) < 2:
		return bad("at least two options are required")
	}
	seen := map[string]bool{}
	for _, o := range r.Options {
		if !optionKeyRule.MatchString(o.Key) {
			return bad("option key %q is not a plain key", o.Key)
		}
		if seen[o.Key] {
			return bad("option key %q appears twice", o.Key)
		}
		seen[o.Key] = true
		if strings.TrimSpace(o.Label) == "" {
			return bad("option %q has no label", o.Key)
		}
	}
	if r.Recommended != "" && !seen[r.Recommended] {
		return bad("recommended %q is not an option", r.Recommended)
	}
	for _, c := range r.Commands {
		if strings.TrimSpace(c.Run) == "" {
			return bad("a command has nothing to run")
		}
		if c.For != "" && !seen[c.For] {
			return bad("command for %q names no option", c.For)
		}
	}
	for _, p := range r.Evidence {
		if !filepath.IsAbs(p) {
			return bad("evidence path %q is not absolute", p)
		}
	}
	return nil
}

// Digest identifies the request's content: the same facts are the same job.
func (r Request) Digest() string {
	data, _ := json.Marshal(r)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Keys are the option keys in menu order.
func (r Request) Keys() []string {
	keys := make([]string, len(r.Options))
	for i, o := range r.Options {
		keys[i] = o.Key
	}
	return keys
}

// slug is a path-safe folder name for a sender or subject key.
func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	dash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		switch {
		case ok:
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 80 {
		sum := sha256.Sum256([]byte(s))
		out = strings.Trim(out[:69], "-") + "-" + hex.EncodeToString(sum[:5])
	}
	if out == "" {
		out = "unnamed"
	}
	return out
}
