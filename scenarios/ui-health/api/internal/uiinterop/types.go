package uiinterop

// CheckContext provides everything a rule needs to inspect a scenario.
type CheckContext struct {
	ScenarioRoot string       // absolute path to scenario dir
	TechStack    []string     // detected components: ["React", "Vite", "iframe-bridge"]
	ScenarioName string       // scenario directory name
	Sources      []SourceFile // scanned production source files under ui/
	TestSources  []SourceFile // scanned test source files under ui/
}

// SourceFile is one scanned production source file under a UI tree.
type SourceFile struct {
	RelPath string // path relative to scenarioRoot, slash-separated (e.g. "ui/src/App.tsx")
	AbsPath string // absolute path on disk
	Content string // file contents
}

// SourceSet splits a single UI tree scan into production and test files.
type SourceSet struct {
	Production []SourceFile
	Tests      []SourceFile
}

// Violation represents a single issue found by a rule check.
type Violation struct {
	RuleID         string `json:"rule_id"`
	Severity       string `json:"severity"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	FilePath       string `json:"file_path,omitempty"`
	Line           int    `json:"line,omitempty"`
	CodeSnippet    string `json:"code_snippet,omitempty"`
	Recommendation string `json:"recommendation"`
}

// RuleResult is what a rule's check function returns.
type RuleResult struct {
	RuleID     string      `json:"rule_id"`
	Passed     bool        `json:"passed"`
	Skipped    bool        `json:"skipped,omitempty"`
	SkipReason string      `json:"skip_reason,omitempty"`
	Violations []Violation `json:"violations,omitempty"`
	Message    string      `json:"message"`
}

// RuleDef is parsed from the docstring — served to UI via API.
type RuleDef struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Why            string   `json:"why"`
	Category       string   `json:"category"`
	Severity       string   `json:"severity"`
	Slot           string   `json:"slot,omitempty"`
	SlotFile       string   `json:"slot_file,omitempty"`
	TechStack      []string `json:"tech_stack"`
	Recommendation string   `json:"recommendation"`
	GoodExample    string   `json:"good_example,omitempty"`
	BadExample     string   `json:"bad_example,omitempty"`
	Standard       string   `json:"standard,omitempty"`
	Enabled        bool     `json:"enabled"`
}

// CheckFunc is the function signature every rule implements.
type CheckFunc func(ctx CheckContext) RuleResult

// Rule combines parsed metadata with the check implementation.
type Rule struct {
	Def   RuleDef
	Check CheckFunc
}
