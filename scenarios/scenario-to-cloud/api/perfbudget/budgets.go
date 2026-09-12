// Package perfbudget owns the phase-23 performance and bounded-operation
// contract of scenario-to-cloud: the frozen numeric budgets, a measurement
// record that never turns a small sample into a statistical claim, a
// comparison that yields within / exceeded / insufficient_samples, a load
// harness that drives an http.Handler or an explicit base URL while sampling
// the management process, bounded retry arithmetic, and the alert fact shape
// consumed by notification-hub.
//
// Nothing here talks to a target host. Every measurement against a real
// target requires an explicit --target argument; there is no default.
package perfbudget

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"scenario-to-cloud/execplan"
)

// Budgets is certification/budgets.json. Only the keys this package consumes
// are typed; the rest of the document is preserved as Raw for reporting.
type Budgets struct {
	SchemaVersion  int            `json:"schema_version"`
	FrozenAt       string         `json:"frozen_at"`
	MachineProfile MachineProfile `json:"machine_profile"`
	Qualification  Qualification  `json:"qualification"`
	Phase23        Phase23        `json:"phase_23"`

	Raw json.RawMessage `json:"-"`
}

// MachineProfile is the declared target machine the budgets are tied to.
type MachineProfile struct {
	VCPU      int `json:"vcpu"`
	MemoryGiB int `json:"memory_gib"`
	DiskGiB   int `json:"disk_gib"`
}

// Qualification carries the phase-1 numeric budgets that phase 23 measures.
type Qualification struct {
	HealthAlertDetectionSecondsMax int `json:"health_alert_detection_seconds_max"`
	SoakHoursMin                   int `json:"soak_hours_min"`
	BackupRecoveryPointSecondsMax  int `json:"backup_recovery_point_seconds_max"`
	IdleManagementOverhead         struct {
		CPUPercentMax float64 `json:"cpu_percent_max"`
		RSSMiBMax     float64 `json:"rss_mib_max"`
	} `json:"idle_management_overhead"`
	APIReadLatency struct {
		P95MsMax          float64 `json:"p95_ms_max"`
		ConcurrentReaders int     `json:"concurrent_readers"`
	} `json:"api_read_latency"`
}

// Phase23 is the block frozen by phase 23.
type Phase23 struct {
	FrozenAt                 string                     `json:"frozen_at"`
	MinSamplesForPercentiles int                        `json:"min_samples_for_percentiles"`
	ReaderConcurrency        int                        `json:"reader_concurrency"`
	FixtureProfiles          map[string]FixtureProfile  `json:"fixture_profiles"`
	ManagementProcess        ManagementProcess          `json:"management_process"`
	DeploymentQueue          DeploymentQueue            `json:"deployment_queue"`
	Retries                  map[string]RetryBudget     `json:"retries"`
	OperationTimeouts        map[string]json.RawMessage `json:"operation_timeouts_seconds"`
	RetentionBudgets         RetentionBudgets           `json:"retention_budgets"`
	Soak                     Soak                       `json:"soak"`
	Detection                Detection                  `json:"detection"`
}

// FixtureProfile is the per-workload latency / error / resource budget.
type FixtureProfile struct {
	FixtureRef      string          `json:"fixture_ref"`
	ReadEndpoints   []string        `json:"read_endpoints"`
	Latency         LatencyBudget   `json:"latency_ms"`
	ErrorRateMax    float64         `json:"error_rate_max"`
	TargetResources ResourceCeiling `json:"target_resources"`
}

// LatencyBudget is in milliseconds.
type LatencyBudget struct {
	P50Max float64 `json:"p50_max"`
	P95Max float64 `json:"p95_max"`
	P99Max float64 `json:"p99_max"`
}

// ResourceCeiling bounds a process or host.
type ResourceCeiling struct {
	CPUPercentMax float64 `json:"cpu_percent_max"`
	RSSMiBMax     float64 `json:"rss_mib_max"`
	DiskGiBMax    float64 `json:"disk_gib_max,omitempty"`
	GoroutinesMax int     `json:"goroutines_max,omitempty"`
	OpenFDsMax    int     `json:"open_fds_max,omitempty"`
}

// GrowthCeiling bounds post-cleanup growth relative to the baseline phase.
type GrowthCeiling struct {
	RSSMiBMaxDelta     float64 `json:"rss_mib_max_delta"`
	GoroutinesMaxDelta int     `json:"goroutines_max_delta"`
	OpenFDsMaxDelta    int     `json:"open_fds_max_delta"`
}

// ManagementProcess bounds the cloud management process itself.
type ManagementProcess struct {
	Idle              ResourceCeiling `json:"idle"`
	ActiveOperation   ResourceCeiling `json:"active_operation"`
	PostCleanupGrowth GrowthCeiling   `json:"post_cleanup_growth"`
}

// DeploymentQueue bounds admission and concurrency.
type DeploymentQueue struct {
	DepthMax                            int `json:"depth_max"`
	EffectfulOperationsPerDeploymentMax int `json:"effectful_operations_per_deployment_max"`
	EffectfulOperationsPerHostMax       int `json:"effectful_operations_per_host_max"`
	QueueDelaySecondsMax                int `json:"queue_delay_seconds_max"`
	WorkerPoolSize                      int `json:"worker_pool_size"`
}

// RetryBudget is one bounded retry class.
type RetryBudget struct {
	MaxAttempts           int     `json:"max_attempts"`
	InitialBackoffSeconds float64 `json:"initial_backoff_seconds"`
	BackoffMultiplier     float64 `json:"backoff_multiplier"`
	BackoffCapSeconds     float64 `json:"backoff_cap_seconds"`
}

// RetentionBudgets are the retention windows per artifact family.
type RetentionBudgets struct {
	ReleaseArtifacts struct {
		KeepLatestPerScenario int `json:"keep_latest_per_scenario"`
		Days                  int `json:"days"`
	} `json:"release_artifacts"`
	Logs struct {
		Days int `json:"days"`
	} `json:"logs"`
	OperationReceipts struct {
		Days int `json:"days"`
	} `json:"operation_receipts"`
	RecoveryPoints struct {
		KeepLast int `json:"keep_last"`
		Days     int `json:"days"`
	} `json:"recovery_points"`
	HealthObservations struct {
		Days int `json:"days"`
	} `json:"health_observations"`
}

// Soak is the 24-hour staging soak contract.
type Soak struct {
	DurationHoursMin                    int    `json:"duration_hours_min"`
	ScheduledObservationIntervalSeconds int    `json:"scheduled_observation_interval_seconds"`
	ControlledReboots                   int    `json:"controlled_reboots"`
	SafeUpdates                         int    `json:"safe_updates"`
	ExternalInput                       string `json:"external_input"`
}

// Detection bounds alerting.
type Detection struct {
	AlertDetectionSecondsMax       int `json:"alert_detection_seconds_max"`
	StaleObservationAfterSeconds   int `json:"stale_observation_after_seconds"`
	CertificateExpiryWarningDays   int `json:"certificate_expiry_warning_days"`
	BackupRPOSecondsMax            int `json:"backup_rpo_seconds_max"`
	RecoveryNotificationSecondsMax int `json:"recovery_notification_seconds_max"`
}

// RequiredRetryClasses are the retry classes that must carry a finite budget:
// every execplan retry contract plus the two transport-level classes this
// package owns.
func RequiredRetryClasses() []string {
	return []string{
		execplan.RetrySafeReplay,
		execplan.RetryObserveThenReplay,
		execplan.RetryRecover,
		"transport_receipt_read",
		"alert_delivery",
	}
}

// RequiredFixtureProfiles are the certified workload shapes of the support
// policy that need a numeric budget.
func RequiredFixtureProfiles() []string {
	return []string{"stateless-web", "headless-api", "sql-uploads"}
}

// Load reads and validates a budgets file.
func Load(path string) (*Budgets, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read budgets: %w", err)
	}
	return Parse(data)
}

// Parse decodes and validates budgets JSON.
func Parse(data []byte) (*Budgets, error) {
	var b Budgets
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("decode budgets: %w", err)
	}
	b.Raw = append(json.RawMessage(nil), data...)
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return &b, nil
}

// Validate refuses a budgets document that is not usable as a frozen
// contract: every required profile, retry class and ceiling must be present,
// numeric and finite.
func (b *Budgets) Validate() error {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	if b.SchemaVersion != 1 {
		add("schema_version %d unsupported", b.SchemaVersion)
	}
	if b.MachineProfile.VCPU <= 0 || b.MachineProfile.MemoryGiB <= 0 || b.MachineProfile.DiskGiB <= 0 {
		add("machine_profile must declare vcpu, memory_gib and disk_gib")
	}
	p := b.Phase23
	if strings.TrimSpace(p.FrozenAt) == "" {
		add("phase_23.frozen_at is required")
	}
	if p.MinSamplesForPercentiles <= 0 {
		add("phase_23.min_samples_for_percentiles must be positive")
	}
	if p.ReaderConcurrency <= 0 {
		add("phase_23.reader_concurrency must be positive")
	}
	for _, name := range RequiredFixtureProfiles() {
		fp, ok := p.FixtureProfiles[name]
		if !ok {
			add("fixture profile %q missing", name)
			continue
		}
		if fp.Latency.P50Max <= 0 || fp.Latency.P95Max <= 0 || fp.Latency.P99Max <= 0 {
			add("fixture profile %q latency budgets must be positive", name)
		}
		if fp.Latency.P50Max > fp.Latency.P95Max || fp.Latency.P95Max > fp.Latency.P99Max {
			add("fixture profile %q latency budgets must be monotonic p50 <= p95 <= p99", name)
		}
		if fp.ErrorRateMax < 0 || fp.ErrorRateMax >= 1 {
			add("fixture profile %q error_rate_max must be in [0,1)", name)
		}
		if fp.TargetResources.CPUPercentMax <= 0 || fp.TargetResources.RSSMiBMax <= 0 || fp.TargetResources.DiskGiBMax <= 0 {
			add("fixture profile %q target_resources must be positive", name)
		}
		if float64(b.MachineProfile.MemoryGiB)*1024 < fp.TargetResources.RSSMiBMax {
			add("fixture profile %q rss budget exceeds the machine profile", name)
		}
		if float64(b.MachineProfile.DiskGiB) < fp.TargetResources.DiskGiBMax {
			add("fixture profile %q disk budget exceeds the machine profile", name)
		}
		if len(fp.ReadEndpoints) == 0 {
			add("fixture profile %q declares no read endpoints", name)
		}
	}
	if p.ManagementProcess.Idle.CPUPercentMax <= 0 || p.ManagementProcess.Idle.RSSMiBMax <= 0 {
		add("management_process.idle ceilings must be positive")
	}
	if p.ManagementProcess.Idle.RSSMiBMax != b.Qualification.IdleManagementOverhead.RSSMiBMax ||
		p.ManagementProcess.Idle.CPUPercentMax != b.Qualification.IdleManagementOverhead.CPUPercentMax {
		add("management_process.idle must equal qualification.idle_management_overhead (one number, one owner)")
	}
	if p.ManagementProcess.Idle.GoroutinesMax <= 0 || p.ManagementProcess.Idle.OpenFDsMax <= 0 {
		add("management_process.idle must bound goroutines and open fds")
	}
	if p.ManagementProcess.PostCleanupGrowth.RSSMiBMaxDelta <= 0 || p.ManagementProcess.PostCleanupGrowth.GoroutinesMaxDelta <= 0 || p.ManagementProcess.PostCleanupGrowth.OpenFDsMaxDelta <= 0 {
		add("management_process.post_cleanup_growth deltas must be positive")
	}
	q := p.DeploymentQueue
	if q.DepthMax <= 0 || q.QueueDelaySecondsMax <= 0 || q.WorkerPoolSize <= 0 {
		add("deployment_queue depth, delay and pool size must be positive")
	}
	if q.EffectfulOperationsPerDeploymentMax != 1 {
		add("deployment_queue.effectful_operations_per_deployment_max must be 1 (one effective writer per deployment)")
	}
	if q.EffectfulOperationsPerHostMax < 1 {
		add("deployment_queue.effectful_operations_per_host_max must be at least 1")
	}
	for _, class := range RequiredRetryClasses() {
		rb, ok := p.Retries[class]
		if !ok {
			add("retry class %q has no budget", class)
			continue
		}
		if err := rb.Validate(); err != nil {
			add("retry class %q: %v", class, err)
		}
	}
	r := p.RetentionBudgets
	if r.ReleaseArtifacts.KeepLatestPerScenario < 2 {
		add("retention_budgets.release_artifacts.keep_latest_per_scenario must keep active + predecessor (>= 2)")
	}
	if r.ReleaseArtifacts.Days <= 0 || r.Logs.Days <= 0 || r.OperationReceipts.Days <= 0 || r.RecoveryPoints.Days <= 0 || r.HealthObservations.Days <= 0 {
		add("retention_budgets days must be positive for every family")
	}
	if r.RecoveryPoints.KeepLast <= 0 {
		add("retention_budgets.recovery_points.keep_last must be positive")
	}
	if p.Soak.DurationHoursMin < 24 {
		add("soak.duration_hours_min must be at least 24")
	}
	if p.Soak.DurationHoursMin != b.Qualification.SoakHoursMin {
		add("soak.duration_hours_min must equal qualification.soak_hours_min")
	}
	if p.Soak.ScheduledObservationIntervalSeconds <= 0 || p.Soak.ControlledReboots < 1 || p.Soak.SafeUpdates < 1 {
		add("soak must schedule observations and include one controlled reboot and one safe update")
	}
	d := p.Detection
	if d.AlertDetectionSecondsMax <= 0 || d.AlertDetectionSecondsMax != b.Qualification.HealthAlertDetectionSecondsMax {
		add("detection.alert_detection_seconds_max must equal qualification.health_alert_detection_seconds_max")
	}
	if d.StaleObservationAfterSeconds <= 0 || d.CertificateExpiryWarningDays <= 0 || d.RecoveryNotificationSecondsMax <= 0 {
		add("detection stale / certificate / recovery bounds must be positive")
	}
	if d.BackupRPOSecondsMax <= 0 || d.BackupRPOSecondsMax != b.Qualification.BackupRecoveryPointSecondsMax {
		add("detection.backup_rpo_seconds_max must equal qualification.backup_recovery_point_seconds_max")
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("budgets invalid: %s", strings.Join(problems, "; "))
	}
	return nil
}

// Validate refuses an unbounded retry class.
func (r RetryBudget) Validate() error {
	if r.MaxAttempts < 1 {
		return fmt.Errorf("max_attempts must be at least 1 (got %d)", r.MaxAttempts)
	}
	if r.MaxAttempts > 10 {
		return fmt.Errorf("max_attempts %d is not a bounded ceiling", r.MaxAttempts)
	}
	if r.InitialBackoffSeconds < 0 || r.BackoffCapSeconds < 0 {
		return fmt.Errorf("backoff seconds must not be negative")
	}
	if r.BackoffMultiplier < 1 {
		return fmt.Errorf("backoff_multiplier must be at least 1")
	}
	if r.MaxAttempts > 1 && r.BackoffCapSeconds <= 0 {
		return fmt.Errorf("a class with more than one attempt needs a positive backoff_cap_seconds")
	}
	if r.InitialBackoffSeconds > r.BackoffCapSeconds {
		return fmt.Errorf("initial backoff exceeds the cap")
	}
	return nil
}

// Profile returns the named fixture profile.
func (b *Budgets) Profile(name string) (FixtureProfile, bool) {
	fp, ok := b.Phase23.FixtureProfiles[name]
	return fp, ok
}
