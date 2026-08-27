package hygiene

import contractapp "github.com/vrooli/vrooli/internal/app/contract"

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Fixability string

const (
	FixabilityAutomatic Fixability = "automatic"
	FixabilityGuided    Fixability = "guided"
	FixabilityManual    Fixability = "manual"
	FixabilityUnsafe    Fixability = "unsafe"
)

type Check struct {
	Name     string   `json:"name"`
	Passed   bool     `json:"passed"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

type Finding struct {
	Severity    Severity   `json:"severity"`
	Code        string     `json:"code"`
	Path        string     `json:"path,omitempty"`
	Locations   []string   `json:"locations,omitempty"`
	Message     string     `json:"message"`
	Why         string     `json:"why,omitempty"`
	Fixability  Fixability `json:"fixability,omitempty"`
	NextActions []Action   `json:"next_actions,omitempty"`
}

type Action struct {
	Code       string     `json:"code"`
	Message    string     `json:"message"`
	Command    string     `json:"command,omitempty"`
	Fixability Fixability `json:"fixability,omitempty"`
}

type PlanCandidate struct {
	Path   string `json:"path"`
	Status string `json:"status,omitempty"`
	Reason string `json:"reason"`
}

type PlanFix struct {
	Source string        `json:"source"`
	Plan   HygienePlan   `json:"plan"`
	Action string        `json:"action,omitempty"`
	Mirror HygieneMirror `json:"mirror,omitempty"`
}

type PlanReconcileOutcome struct {
	Action                  string        `json:"action"`
	Source                  string        `json:"source,omitempty"`
	Plan                    HygienePlan   `json:"plan,omitempty"`
	Mirror                  HygieneMirror `json:"mirror,omitempty"`
	SourceUntouched         bool          `json:"source_untouched,omitempty"`
	SourceRetirementPlanned bool          `json:"source_retirement_planned,omitempty"`
	SourceRemoved           bool          `json:"source_removed,omitempty"`
	Error                   string        `json:"error,omitempty"`
}

type HygienePlan struct {
	ID        string `json:"id,omitempty"`
	Title     string `json:"title,omitempty"`
	Slug      string `json:"slug,omitempty"`
	Path      string `json:"path,omitempty"`
	Source    string `json:"source_path,omitempty"`
	Archived  bool   `json:"archived,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type HygieneMirror struct {
	Path   string `json:"path,omitempty"`
	Status string `json:"status,omitempty"`
}

type DependencyFreshnessStatus string

const (
	DependencyFreshnessStatusClean        DependencyFreshnessStatus = "clean"
	DependencyFreshnessStatusStaleModules DependencyFreshnessStatus = "stale-modules"
	DependencyFreshnessStatusStaleBuild   DependencyFreshnessStatus = "stale-build"
	DependencyFreshnessStatusError        DependencyFreshnessStatus = "error"
	DependencyFreshnessStatusSkipped      DependencyFreshnessStatus = "skipped"
)

type DependencyFreshnessScenario struct {
	Path       string                    `json:"path"`
	APIDir     string                    `json:"api_dir"`
	Status     DependencyFreshnessStatus `json:"status"`
	DiffPaths  []string                  `json:"diff_paths,omitempty"`
	BuildError string                    `json:"build_error,omitempty"`
	Error      string                    `json:"error,omitempty"`
	Replaces   []string                  `json:"replaces,omitempty"`
}

type DependencyFreshnessCompatReport struct {
	Clean             bool                          `json:"clean"`
	Root              string                        `json:"root"`
	Scenarios         []DependencyFreshnessScenario `json:"scenarios"`
	TouchedPackages   []string                      `json:"touched_packages,omitempty"`
	OnlyTouchedUsed   bool                          `json:"only_touched"`
	BuildChecked      bool                          `json:"build_checked"`
	FixApplied        bool                          `json:"fix_applied"`
	ModifiedTrackedOK bool                          `json:"modified_tracked_files,omitempty"`
	ElapsedMs         int64                         `json:"elapsed_ms"`
}

type Report struct {
	Success               bool                             `json:"success"`
	Root                  string                           `json:"root"`
	Checks                []Check                          `json:"checks"`
	Findings              []Finding                        `json:"findings,omitempty"`
	Actions               []Action                         `json:"actions,omitempty"`
	PlanCandidates        []PlanCandidate                  `json:"plan_candidates,omitempty"`
	FixesApplied          []PlanFix                        `json:"fixes_applied,omitempty"`
	PlanReconcileOutcomes []PlanReconcileOutcome           `json:"plan_reconcile_outcomes,omitempty"`
	ConfigFixes           []string                         `json:"config_fixes,omitempty"`
	Contract              contractapp.ValidationOutput     `json:"contract,omitempty"`
	SharedDrift           *DependencyFreshnessCompatReport `json:"shared_drift,omitempty"`
	BlockingFailures      int                              `json:"blocking_failures"`
	Warnings              int                              `json:"warnings"`
}

type Request struct {
	FixSafe                 bool
	Plans                   bool
	FailOn                  Severity
	IncludePlans            bool
	IncludeContract         bool
	IncludeDrift            bool
	IncludeFreshness        bool
	IncludeTidiness         bool
	RequireTidinessProvider bool
}
