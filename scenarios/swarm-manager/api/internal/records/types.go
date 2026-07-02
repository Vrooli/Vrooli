// Package records is the write-side of the recursive-learning loop:
// narrative artifacts of completed work (bug fixes, features, refactors,
// investigations). Records sit alongside backlog and captures; they are
// immutable once their narrative is filled.
//
// DOC: docs/concepts/ARCHITECTURE.md
// DOC: docs/internal/SEAMS.md
package records

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// RecordKind mirrors backlog.BacklogKind by string identity. An anti-drift
// test asserts the two enums stay equal; see types_test.go.
type RecordKind string

const (
	KindIdea     RecordKind = "idea"
	KindResearch RecordKind = "research"
	KindFix      RecordKind = "fix"
	KindExecute  RecordKind = "execute"
	KindChore    RecordKind = "chore"
)

// AllKinds is the canonical enumeration.
var AllKinds = []RecordKind{KindIdea, KindResearch, KindFix, KindExecute, KindChore}

// kindAliases maps common improvised kind names onto a canonical RecordKind.
// ParseKind consults this after an exact match fails but before erroring, so an
// agent that types "feature", "bugfix", or "investigation" still lands on the
// right kind instead of getting a 400. Keep this list curated and finite — it
// is not meant to absorb arbitrary input (that is what the nearest-match
// suggestion in the error is for).
var kindAliases = map[string]RecordKind{
	"improvement":          KindExecute,
	"scenario-improvement": KindExecute,
	"implementation":       KindExecute,
	"feature":              KindExecute,
	"feat":                 KindExecute,
	"refactor":             KindExecute,
	"build":                KindExecute,
	"bug":                  KindFix,
	"bugfix":               KindFix,
	"bug-fix":              KindFix,
	"fixup":                KindFix,
	"hotfix":               KindFix,
	"investigation":        KindResearch,
	"spike":                KindResearch,
	"explore":              KindResearch,
	"task":                 KindChore,
	"maintenance":          KindChore,
	"cleanup":              KindChore,
}

// ParseKind resolves a raw kind string to a canonical RecordKind. It accepts
// the canonical kinds (case-insensitive), a curated set of aliases
// (kindAliases), and otherwise returns a self-correcting error that names the
// valid kinds and suggests the nearest known one.
func ParseKind(raw string) (RecordKind, error) {
	candidate := strings.ToLower(strings.TrimSpace(raw))
	for _, k := range AllKinds {
		if candidate == string(k) {
			return k, nil
		}
	}
	if canon, ok := kindAliases[candidate]; ok {
		return canon, nil
	}
	return "", fmt.Errorf("invalid record kind %q; valid kinds: %s%s",
		raw, joinKinds(AllKinds), suggestKind(candidate))
}

// joinKinds renders the canonical kinds as a comma-separated lowercase list.
func joinKinds(kinds []RecordKind) string {
	parts := make([]string, len(kinds))
	for i, k := range kinds {
		parts[i] = string(k)
	}
	return strings.Join(parts, ", ")
}

// suggestKind returns a ` (did you mean "x"?)` fragment when candidate is a
// near miss of a canonical kind or alias, or "" when nothing is close enough.
// Alias keys are scanned in sorted order so the suggestion is deterministic.
func suggestKind(candidate string) string {
	if candidate == "" {
		return ""
	}
	type token struct {
		text  string
		canon RecordKind
	}
	tokens := make([]token, 0, len(AllKinds)+len(kindAliases))
	for _, k := range AllKinds {
		tokens = append(tokens, token{string(k), k})
	}
	aliasKeys := make([]string, 0, len(kindAliases))
	for a := range kindAliases {
		aliasKeys = append(aliasKeys, a)
	}
	sort.Strings(aliasKeys)
	for _, a := range aliasKeys {
		tokens = append(tokens, token{a, kindAliases[a]})
	}
	best := RecordKind("")
	bestDist := -1
	for _, t := range tokens {
		if d := levenshtein(candidate, t.text); bestDist == -1 || d < bestDist {
			bestDist = d
			best = t.canon
		}
	}
	// Only suggest for genuine near-misses; garbage like "nope" gets no hint.
	if bestDist > 2 {
		return ""
	}
	return fmt.Sprintf(" (did you mean %q?)", best)
}

// levenshtein is a small edit-distance helper backing the nearest-kind
// suggestion. It is only ever run on the short kind vocabulary, so the simple
// two-row implementation is more than fast enough.
func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}

// Outcome classifies how the work concluded.
type Outcome string

const (
	OutcomeShipped   Outcome = "shipped"
	OutcomePartial   Outcome = "partial"
	OutcomeAbandoned Outcome = "abandoned"
	OutcomeDuplicate Outcome = "duplicate"
)

// AllOutcomes is the canonical enumeration.
var AllOutcomes = []Outcome{OutcomeShipped, OutcomePartial, OutcomeAbandoned, OutcomeDuplicate}

// outcomeAliases maps common improvised outcome words onto a canonical
// Outcome, mirroring kindAliases. Transcript analysis showed agents most often
// write "success", "done", or "green" where the enum expects "shipped"; keep
// this list curated and finite.
var outcomeAliases = map[string]Outcome{
	"success":     OutcomeShipped,
	"succeeded":   OutcomeShipped,
	"done":        OutcomeShipped,
	"complete":    OutcomeShipped,
	"completed":   OutcomeShipped,
	"finished":    OutcomeShipped,
	"fixed":       OutcomeShipped,
	"resolved":    OutcomeShipped,
	"green":       OutcomeShipped,
	"delivered":   OutcomeShipped,
	"implemented": OutcomeShipped,
	"wip":         OutcomePartial,
	"in-progress": OutcomePartial,
	"in_progress": OutcomePartial,
	"incomplete":  OutcomePartial,
	"cancelled":   OutcomeAbandoned,
	"canceled":    OutcomeAbandoned,
	"dropped":     OutcomeAbandoned,
	"aborted":     OutcomeAbandoned,
	"dup":         OutcomeDuplicate,
	"dupe":        OutcomeDuplicate,
}

// outcomeProseThreshold is the length past which an unrecognized outcome value
// is treated as misplaced narrative rather than a typo'd enum word. The
// transcript corpus put prose outcomes at a median of 260 chars and short
// typos well under this bound.
const outcomeProseThreshold = 40

// ParseOutcome resolves a raw outcome string to a canonical Outcome. It
// accepts the canonical outcomes (case-insensitive) and a curated set of
// aliases (outcomeAliases). Unrecognized values get a self-correcting error:
// prose-shaped input (the dominant failure in transcript analysis — agents
// passing their whole validation summary) is told which fields narrative
// belongs in, and short near-misses get a did-you-mean hint.
func ParseOutcome(raw string) (Outcome, error) {
	candidate := strings.ToLower(strings.TrimSpace(raw))
	for _, o := range AllOutcomes {
		if candidate == string(o) {
			return o, nil
		}
	}
	if canon, ok := outcomeAliases[candidate]; ok {
		return canon, nil
	}
	if len(candidate) > outcomeProseThreshold || strings.ContainsAny(candidate, " \t\n") {
		return "", fmt.Errorf("invalid outcome: value looks like a narrative (%d chars); --outcome is a category, one of %s — put validation results in --evidence and the build story in --approach",
			len(raw), joinOutcomes(AllOutcomes))
	}
	return "", fmt.Errorf("invalid outcome %q (expected one of %s)%s",
		raw, joinOutcomes(AllOutcomes), suggestOutcome(candidate))
}

// joinOutcomes renders the canonical outcomes as a comma-separated list.
func joinOutcomes(outcomes []Outcome) string {
	parts := make([]string, len(outcomes))
	for i, o := range outcomes {
		parts[i] = string(o)
	}
	return strings.Join(parts, ", ")
}

// suggestOutcome returns a ` (did you mean "x"?)` fragment when candidate is a
// near miss of a canonical outcome or alias, or "" when nothing is close.
func suggestOutcome(candidate string) string {
	if candidate == "" {
		return ""
	}
	type token struct {
		text  string
		canon Outcome
	}
	tokens := make([]token, 0, len(AllOutcomes)+len(outcomeAliases))
	for _, o := range AllOutcomes {
		tokens = append(tokens, token{string(o), o})
	}
	aliasKeys := make([]string, 0, len(outcomeAliases))
	for a := range outcomeAliases {
		aliasKeys = append(aliasKeys, a)
	}
	sort.Strings(aliasKeys)
	for _, a := range aliasKeys {
		tokens = append(tokens, token{a, outcomeAliases[a]})
	}
	best := Outcome("")
	bestDist := -1
	for _, t := range tokens {
		if d := levenshtein(candidate, t.text); bestDist == -1 || d < bestDist {
			bestDist = d
			best = t.canon
		}
	}
	if bestDist > 2 {
		return ""
	}
	return fmt.Sprintf(" (did you mean %q?)", best)
}

// Record is a narrative artifact of completed work. Once Stub flips false,
// every field except SupersededBy is immutable; amendments require a new
// record with Supersedes set to this record's ID.
type Record struct {
	ID           string     `json:"id"`
	Kind         RecordKind `json:"kind"`
	Scenario     string     `json:"scenario"`
	BacklogRef   string     `json:"backlog_ref,omitempty"`
	InitiativeID string     `json:"initiative_id,omitempty"`
	Supersedes   string     `json:"supersedes,omitempty"`
	SupersededBy string     `json:"superseded_by,omitempty"`
	Trigger      string     `json:"trigger"`
	Approach     string     `json:"approach"`
	RuledOut     []string   `json:"ruled_out,omitempty"`
	Evidence     string     `json:"evidence,omitempty"`
	Commit       string     `json:"commit,omitempty"`
	FilesChanged []string   `json:"files_changed,omitempty"`
	Outcome      Outcome    `json:"outcome"`
	Stub         bool       `json:"stub"`
	CreatedAt    time.Time  `json:"created_at"`
	CreatedBy    string     `json:"created_by,omitempty"`
	NarrativeAt  time.Time  `json:"narrative_at,omitempty"`
}

// ErrStubLocked is returned when an attempt is made to fill or edit a
// record's narrative after the stub has already been filled.
var ErrStubLocked = errors.New("record narrative is already filled; create a superseding record to amend")

// ErrSupersedeCycle is returned when a Supersede operation would create a
// cycle in the supersede chain.
var ErrSupersedeCycle = errors.New("supersede would create a cycle")

// ErrAlreadySuperseded is returned when a record already has SupersededBy set.
var ErrAlreadySuperseded = errors.New("record is already superseded")

// hasNarrative reports whether the record has non-empty narrative content.
// A record with empty Trigger AND empty Approach AND empty RuledOut is a
// stub; once any of these are non-empty it is considered filled.
func (r *Record) hasNarrative() bool {
	return strings.TrimSpace(r.Trigger) != "" ||
		strings.TrimSpace(r.Approach) != "" ||
		len(r.RuledOut) > 0
}

// EmbeddingText composes the text used for semantic indexing.
func (r *Record) EmbeddingText() string {
	parts := []string{r.Trigger, r.Approach}
	parts = append(parts, r.RuledOut...)
	if r.Evidence != "" {
		parts = append(parts, r.Evidence)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}
