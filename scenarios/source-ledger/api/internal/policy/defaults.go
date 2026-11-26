package policy

// Built-in and file-provided policy defaults.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type fileDefaults struct {
	Schema          string `json:"$schema"`
	FrontierTarget  int    `json:"frontierTarget"`
	WakeBudgetLines int    `json:"wakeBudgetLines"`
	WakeBudgetChars int    `json:"wakeBudgetChars"`
	MaxEntryLines   int    `json:"maxEntryLines"`
	MaxEntryChars   int    `json:"maxEntryChars"`
	// SummaryMaxOutputTokens is optional; omitted means the built-in default.
	SummaryMaxOutputTokens int `json:"summaryMaxOutputTokens"`
}

const (
	DefaultFrontierTarget = 16
	// DefaultWakeBudget bounds the whole ambient view in lines. It must be able
	// to hold every facet's resident budget at DefaultMaxEntryLines apiece, or
	// the declared residencies are unsatisfiable and the last facets in order
	// silently never appear.
	DefaultWakeBudget = 96
	// DefaultMaxEntryLines bounds one memory's contribution to the ambient view.
	// Wake is an index into memory, not a copy of it: the journal keeps every
	// line and recall returns them.
	DefaultMaxEntryLines = 2
	// DefaultWakeBudgetChars is the whole-view character backstop. Characters
	// bind for the single-line JSON documents written by the journal.
	DefaultWakeBudgetChars = 12000
	// DefaultMaxEntryChars keeps one verbose journal document from dominating
	// an agent's ambient context.
	DefaultMaxEntryChars = 200
	// DefaultSummaryMaxOutputTokens is the generation ceiling for a compaction
	// summary. It is a ceiling, not a target: the model stops on its own, so the
	// value only has to sit above the longest natural summary. Real compaction
	// prompts run through summarize.default with a 1024 ceiling stopped on their
	// own after 135-637 tokens (median 230, 14 of 14 above the former 128-token
	// cap, which cut every summary mid-sentence); 768 clears the longest with
	// margin and costs nothing for the summaries that finish sooner.
	DefaultSummaryMaxOutputTokens = 768
)

func BuiltInDefaults() Config {
	return Config{
		Scope:                  AgentMemory,
		FrontierTarget:         DefaultFrontierTarget,
		WakeBudget:             DefaultWakeBudget,
		WakeBudgetChars:        DefaultWakeBudgetChars,
		MaxEntryLines:          DefaultMaxEntryLines,
		MaxEntryChars:          DefaultMaxEntryChars,
		SummaryMaxOutputTokens: DefaultSummaryMaxOutputTokens,
		Origins: Origins{
			FrontierTarget: OriginBuiltIn, WakeBudget: OriginBuiltIn,
			WakeBudgetChars: OriginBuiltIn, MaxEntryLines: OriginBuiltIn,
			MaxEntryChars: OriginBuiltIn,
		},
	}
}

// DefaultPolicyPath finds the checked-in policy relative to the scenario root.
// Lifecycle starts normally from the scenario directory, while unit tests and
// developer tools may start from api/ or a repository subdirectory.
func DefaultPolicyPath() string {
	for _, start := range candidateRoots() {
		for dir := start; ; dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, PolicyFileName)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	return filepath.Join(".", PolicyFileName)
}

func candidateRoots() []string {
	roots := []string{}
	if wd, err := os.Getwd(); err == nil {
		roots = append(roots, wd)
	}
	if executable, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(executable))
	}
	return roots
}

// LoadDefaults validates the operator-controlled file before it enters the
// request path. A malformed or unsafe policy is a startup error, never a
// silently widened context budget.
func LoadDefaults(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read policy defaults %q: %w", path, err)
	}
	var file fileDefaults
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return Config{}, fmt.Errorf("parse policy defaults %q: %w", path, err)
	}
	if err := file.validate(); err != nil {
		return Config{}, fmt.Errorf("validate policy defaults %q: %w", path, err)
	}
	return Config{
		Scope:          fileScope,
		FrontierTarget: file.FrontierTarget, WakeBudget: file.WakeBudgetLines,
		WakeBudgetChars: file.WakeBudgetChars, MaxEntryLines: file.MaxEntryLines,
		MaxEntryChars: file.MaxEntryChars, SummaryMaxOutputTokens: file.SummaryMaxOutputTokens,
		Origins: Origins{
			FrontierTarget: OriginFileDefault, WakeBudget: OriginFileDefault,
			WakeBudgetChars: OriginFileDefault, MaxEntryLines: OriginFileDefault,
			MaxEntryChars: OriginFileDefault,
		},
	}, nil
}

var fileScope = Scope("defaults")

func (f fileDefaults) validate() error {
	values := map[string]int{
		"frontierTarget": f.FrontierTarget, "wakeBudgetLines": f.WakeBudgetLines,
		"wakeBudgetChars": f.WakeBudgetChars, "maxEntryLines": f.MaxEntryLines,
		"maxEntryChars": f.MaxEntryChars,
	}
	for name, value := range values {
		if value <= 0 {
			return fmt.Errorf("%s must be a positive integer", name)
		}
	}
	if f.SummaryMaxOutputTokens < 0 {
		return fmt.Errorf("summaryMaxOutputTokens must be a positive integer when set")
	}
	if f.MaxEntryLines > f.WakeBudgetLines {
		return fmt.Errorf("maxEntryLines cannot exceed wakeBudgetLines")
	}
	if f.MaxEntryChars > f.WakeBudgetChars {
		return fmt.Errorf("maxEntryChars cannot exceed wakeBudgetChars")
	}
	return nil
}

func normalizeDefaults(config Config) Config {
	if config.SummaryMaxOutputTokens <= 0 {
		config.SummaryMaxOutputTokens = DefaultSummaryMaxOutputTokens
	}
	for _, f := range []struct {
		value    *int
		fallback int
		origin   *string
	}{
		{&config.FrontierTarget, DefaultFrontierTarget, &config.Origins.FrontierTarget},
		{&config.WakeBudget, DefaultWakeBudget, &config.Origins.WakeBudget},
		{&config.WakeBudgetChars, DefaultWakeBudgetChars, &config.Origins.WakeBudgetChars},
		{&config.MaxEntryLines, DefaultMaxEntryLines, &config.Origins.MaxEntryLines},
		{&config.MaxEntryChars, DefaultMaxEntryChars, &config.Origins.MaxEntryChars},
	} {
		if *f.value <= 0 {
			*f.value = f.fallback
		}
		if *f.origin == "" {
			*f.origin = OriginBuiltIn
		}
	}
	return config
}

// DefaultSummaryInstruction is the agent-memory scope's opening line for the
// summarization prompt. It lives here rather than in the compaction engine
// because it names what this scope stores; a second scope replaces it without
// touching the engine.
const DefaultSummaryInstruction = "Summarize these episode memories for future agent context."

// ExpireOnResolution is the retention policy whose members leave the ambient
// frontier once marked resolved. The engine knows that some facets expire on
// resolution; only this package knows what that policy is called.
const ExpireOnResolution = "expire-on-resolution"

const (
	Retain         = "retain"
	Compact        = "compact"
	PinnedOrReview = "pinned-or-review"
)

// IsValidRetentionPolicy is the single vocabulary check shared by scope
// creation and facet-policy updates. Keeping the closed set here prevents a
// scope bootstrap path from accepting a value the update path cannot use.
func IsValidRetentionPolicy(value string) bool {
	switch strings.TrimSpace(value) {
	case Retain, Compact, ExpireOnResolution, PinnedOrReview:
		return true
	default:
		return false
	}
}
