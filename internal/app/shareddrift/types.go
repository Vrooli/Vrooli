package shareddrift

type ScenarioStatus string

const (
	StatusClean        ScenarioStatus = "clean"
	StatusStaleModules ScenarioStatus = "stale-modules"
	StatusStaleBuild   ScenarioStatus = "stale-build"
	StatusError        ScenarioStatus = "error"
	StatusSkipped      ScenarioStatus = "skipped"
)

type CheckRequest struct {
	Fix         bool
	OnlyTouched bool
	Build       bool
	Concurrency int
}

type ScenarioReport struct {
	Path       string         `json:"path"`
	APIDir     string         `json:"api_dir"`
	Status     ScenarioStatus `json:"status"`
	DiffPaths  []string       `json:"diff_paths,omitempty"`
	BuildError string         `json:"build_error,omitempty"`
	Error      string         `json:"error,omitempty"`
	Replaces   []string       `json:"replaces,omitempty"`
}

type Report struct {
	Clean             bool             `json:"clean"`
	Root              string           `json:"root"`
	Scenarios         []ScenarioReport `json:"scenarios"`
	TouchedPackages   []string         `json:"touched_packages,omitempty"`
	OnlyTouchedUsed   bool             `json:"only_touched"`
	BuildChecked      bool             `json:"build_checked"`
	FixApplied        bool             `json:"fix_applied"`
	ModifiedTrackedOK bool             `json:"modified_tracked_files,omitempty"`
	ElapsedMs         int64            `json:"elapsed_ms"`
}
