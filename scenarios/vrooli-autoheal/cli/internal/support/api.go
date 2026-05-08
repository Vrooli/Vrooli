package support

import "time"

type PlatformInfo struct {
	Platform                string `json:"platform"`
	SupportsRdp             bool   `json:"supportsRdp"`
	SupportsSystemd         bool   `json:"supportsSystemd"`
	SupportsLaunchd         bool   `json:"supportsLaunchd"`
	SupportsWindowsServices bool   `json:"supportsWindowsServices"`
	IsHeadlessServer        bool   `json:"isHeadlessServer"`
	HasDocker               bool   `json:"hasDocker"`
	IsWsl                   bool   `json:"isWsl"`
	SupportsCloudflared     bool   `json:"supportsCloudflared"`
}

type Summary struct {
	Total    int `json:"total"`
	OK       int `json:"ok"`
	Warning  int `json:"warning"`
	Critical int `json:"critical"`
}

type CheckResult struct {
	CheckID   string                 `json:"checkId"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
}

type StatusResponse struct {
	Status                          string        `json:"status"`
	Platform                        PlatformInfo  `json:"platform"`
	TickRunning                     bool          `json:"tickRunning"`
	TickStartedAt                   *time.Time    `json:"tickStartedAt"`
	LastCompletedTickAt             *time.Time    `json:"lastCompletedTickAt"`
	StatusFresh                     bool          `json:"statusFresh"`
	StatusAgeSeconds                int64         `json:"statusAgeSeconds"`
	StatusFreshnessThresholdSeconds int64         `json:"statusFreshnessThresholdSeconds"`
	StatusStaleReason               string        `json:"statusStaleReason"`
	Summary                         Summary       `json:"summary"`
	Checks                          []CheckResult `json:"checks"`
	Timestamp                       time.Time     `json:"timestamp"`
}

type TickResponse struct {
	Success   bool          `json:"success"`
	Status    string        `json:"status"`
	Summary   Summary       `json:"summary"`
	Results   []CheckResult `json:"results"`
	Warnings  []string      `json:"warnings,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

type CheckInfo struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Importance      string   `json:"importance"`
	Category        string   `json:"category"`
	IntervalSeconds int      `json:"intervalSeconds"`
	Platforms       []string `json:"platforms,omitempty"`
}

type CheckHistoryResponse struct {
	CheckID string        `json:"checkId"`
	History []CheckResult `json:"history"`
	Count   int           `json:"count"`
}

type RecoveryAction struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Dangerous   bool   `json:"dangerous"`
	Available   bool   `json:"available"`
}

type CheckActionsResponse struct {
	CheckID string           `json:"checkId"`
	Actions []RecoveryAction `json:"actions"`
}

type ActionResult struct {
	ActionID  string        `json:"actionId"`
	CheckID   string        `json:"checkId"`
	Success   bool          `json:"success"`
	Message   string        `json:"message"`
	Output    string        `json:"output,omitempty"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
}

type ActionLog struct {
	ID         int64  `json:"id"`
	CheckID    string `json:"checkId"`
	ActionID   string `json:"actionId"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMS int64  `json:"durationMs"`
	Timestamp  string `json:"timestamp"`
}

type ActionLogsResponse struct {
	Logs  []ActionLog `json:"logs"`
	Total int         `json:"total"`
}

type WatchdogStatus struct {
	LoopRunning          bool   `json:"loopRunning"`
	WatchdogType         string `json:"watchdogType"`
	WatchdogInstalled    bool   `json:"watchdogInstalled"`
	WatchdogEnabled      bool   `json:"watchdogEnabled"`
	WatchdogRunning      bool   `json:"watchdogRunning"`
	BootProtectionActive bool   `json:"bootProtectionActive"`
	CanInstall           bool   `json:"canInstall"`
	ServicePath          string `json:"servicePath,omitempty"`
	LastError            string `json:"lastError,omitempty"`
	ProtectionLevel      string `json:"protectionLevel"`
	LingeringEnabled     bool   `json:"lingeringEnabled"`
	Username             string `json:"username,omitempty"`
	IsUserService        bool   `json:"isUserService,omitempty"`
}

type WatchdogInstallStatus struct {
	Installed        bool   `json:"installed"`
	Enabled          bool   `json:"enabled"`
	Running          bool   `json:"running"`
	BootProtected    bool   `json:"bootProtected"`
	ServicePath      string `json:"servicePath,omitempty"`
	WatchdogType     string `json:"watchdogType"`
	CanInstall       bool   `json:"canInstall"`
	NeedsLinger      bool   `json:"needsLinger"`
	LingerCommand    string `json:"lingerCommand,omitempty"`
	ProtectionLevel  string `json:"protectionLevel"`
	LastChecked      string `json:"lastChecked"`
	RecommendedSetup string `json:"recommendedSetup"`
}

type WatchdogMutationResult struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	ServicePath   string `json:"servicePath,omitempty"`
	NeedsLinger   bool   `json:"needsLinger,omitempty"`
	LingerCommand string `json:"lingerCommand,omitempty"`
	Error         string `json:"error,omitempty"`
}

type TimelineEvent struct {
	CheckID   string                 `json:"checkId"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

type TimelineResponse struct {
	Events  []TimelineEvent `json:"events"`
	Count   int             `json:"count"`
	Summary map[string]int  `json:"summary"`
}

type Incident struct {
	ID               string                 `json:"id"`
	Fingerprint      string                 `json:"fingerprint"`
	Type             string                 `json:"type"`
	Severity         string                 `json:"severity"`
	Status           string                 `json:"status"`
	Title            string                 `json:"title"`
	Summary          string                 `json:"summary"`
	DetectedAt       string                 `json:"detectedAt"`
	LastSeenAt       string                 `json:"lastSeenAt"`
	UpdatedAt        string                 `json:"updatedAt"`
	BootID           string                 `json:"bootId,omitempty"`
	PreviousBootID   string                 `json:"previousBootId,omitempty"`
	SourceCheckIDs   []string               `json:"sourceCheckIds,omitempty"`
	Evidence         map[string]interface{} `json:"evidence,omitempty"`
	Recommendations  []string               `json:"recommendations,omitempty"`
	EventCount       int                    `json:"eventCount"`
	ObservationCount int                    `json:"observationCount"`
	OperatorNotes    string                 `json:"operatorNotes,omitempty"`
}

type IncidentsResponse struct {
	Incidents []Incident             `json:"incidents"`
	Total     int                    `json:"total"`
	Filters   map[string]interface{} `json:"filters"`
}

type HostInventoryResponse struct {
	Snapshot    *HostInventorySnapshot `json:"snapshot"`
	Fresh       bool                   `json:"fresh"`
	AgeSeconds  int64                  `json:"ageSeconds"`
	ProbeStatus map[string]string      `json:"probeStatus"`
}

type HostInventorySnapshot struct {
	ID            string                 `json:"id"`
	CollectedAt   string                 `json:"collectedAt"`
	Platform      string                 `json:"platform"`
	OS            string                 `json:"os"`
	Arch          string                 `json:"arch"`
	BootID        string                 `json:"bootId"`
	KernelRelease string                 `json:"kernelRelease"`
	Fingerprint   string                 `json:"fingerprint"`
	Inventory     map[string]interface{} `json:"inventory"`
}

type HostInventoryChange struct {
	ID             int64                  `json:"id"`
	FromSnapshotID string                 `json:"fromSnapshotId"`
	ToSnapshotID   string                 `json:"toSnapshotId"`
	ChangeType     string                 `json:"changeType"`
	Severity       string                 `json:"severity"`
	Summary        string                 `json:"summary"`
	Details        map[string]interface{} `json:"details,omitempty"`
	CreatedAt      string                 `json:"createdAt"`
}

type HostInventoryChangesResponse struct {
	Changes []HostInventoryChange `json:"changes"`
	Total   int                   `json:"total"`
}

type UptimeStats struct {
	TotalEvents      int     `json:"totalEvents"`
	OkEvents         int     `json:"okEvents"`
	WarningEvents    int     `json:"warningEvents"`
	CriticalEvents   int     `json:"criticalEvents"`
	UptimePercentage float64 `json:"uptimePercentage"`
	WindowHours      int     `json:"windowHours"`
}

type UptimeHistoryBucket struct {
	Timestamp time.Time `json:"timestamp"`
	Total     int       `json:"total"`
	OK        int       `json:"ok"`
	Warning   int       `json:"warning"`
	Critical  int       `json:"critical"`
}

type UptimeHistory struct {
	Buckets     []UptimeHistoryBucket `json:"buckets"`
	Overall     UptimeStats           `json:"overall"`
	WindowHours int                   `json:"windowHours"`
	BucketCount int                   `json:"bucketCount"`
}

type CheckTrend struct {
	CheckID        string   `json:"checkId"`
	Total          int      `json:"total"`
	OK             int      `json:"ok"`
	Warning        int      `json:"warning"`
	Critical       int      `json:"critical"`
	UptimePercent  float64  `json:"uptimePercent"`
	CurrentStatus  string   `json:"currentStatus"`
	RecentStatuses []string `json:"recentStatuses"`
	LastChecked    string   `json:"lastChecked"`
}

type CheckTrendsResponse struct {
	Trends      []CheckTrend `json:"trends"`
	WindowHours int          `json:"windowHours"`
	TotalChecks int          `json:"totalChecks"`
}
