package securestore

import "time"

type ScheduleStatus struct {
	Supported   bool      `json:"supported"`
	Enabled     bool      `json:"enabled"`
	Provider    string    `json:"provider"`
	State       string    `json:"state"`
	UpdatedAt   time.Time `json:"updated_at"`
	Remediation string    `json:"remediation,omitempty"`
}

// InstallCopySchedule installs or removes the native per-user refresh job.
// Unsupported platforms return degraded metadata without guessing a scheduler.
func InstallCopySchedule(executable string, interval time.Duration, enabled bool) (ScheduleStatus, error) {
	status := ScheduleStatus{Supported: nativeScheduleSupported, Enabled: enabled, Provider: nativeScheduleProvider, UpdatedAt: time.Now().UTC()}
	if !nativeScheduleSupported {
		status.Enabled = false
		status.State = "degraded"
		status.Remediation = "refresh manually with `vrooli credentials store copy scheduled --format json`"
		return status, nil
	}
	if err := installNativeCopySchedule(executable, interval, enabled); err != nil {
		status.State = "retryable_failure"
		status.Remediation = "correct the per-user scheduler session and retry"
		return status, err
	}
	status.State = "ready"
	return status, nil
}
