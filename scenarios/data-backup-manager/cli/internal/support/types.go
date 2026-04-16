package support

import (
	"encoding/json"
	"time"
)

// BackupJob mirrors the shape returned by /api/v1/backup/list and embedded in
// /api/v1/backup/status.active_jobs.
type BackupJob struct {
	ID               string     `json:"id"`
	Type             string     `json:"type"`
	Target           string     `json:"target"`
	TargetID         string     `json:"target_identifier,omitempty"`
	Status           string     `json:"status"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	SizeBytes        int64      `json:"size_bytes,omitempty"`
	CompressionRatio float64    `json:"compression_ratio,omitempty"`
	StoragePath      string     `json:"storage_path,omitempty"`
	Checksum         string     `json:"checksum,omitempty"`
	RetentionUntil   *time.Time `json:"retention_until,omitempty"`
	Description      string     `json:"description,omitempty"`
}

// BackupListResponse is the envelope returned by /api/v1/backup/list.
type BackupListResponse struct {
	Backups []BackupJob `json:"backups"`
	Total   int         `json:"total"`
}

// BackupCreateResponse is the result of POST /api/v1/backup/create.
type BackupCreateResponse struct {
	JobID             string   `json:"job_id"`
	EstimatedDuration string   `json:"estimated_duration,omitempty"`
	Status            string   `json:"status,omitempty"`
	Targets           []string `json:"targets,omitempty"`
}

// StorageUsageInfo is the nested storage stats block in backup status.
type StorageUsageInfo struct {
	UsedGB           float64 `json:"used_gb"`
	AvailableGB      float64 `json:"available_gb"`
	CompressionRatio float64 `json:"compression_ratio"`
}

// ResourceHealth is one entry in the backup status resource_health map.
type ResourceHealth struct {
	Status      string     `json:"status"`
	LastChecked *time.Time `json:"last_checked,omitempty"`
	Message     string     `json:"message,omitempty"`
}

// BackupStatusResponse is the response from /api/v1/backup/status.
type BackupStatusResponse struct {
	SystemStatus         string                    `json:"system_status"`
	ActiveJobs           []BackupJob               `json:"active_jobs"`
	LastSuccessfulBackup *time.Time                `json:"last_successful_backup,omitempty"`
	StorageUsage         StorageUsageInfo          `json:"storage_usage"`
	ResourceHealth       map[string]ResourceHealth `json:"resource_health"`
}

// BackupVerifyResponse is the response from /api/v1/backup/verify/{id}.
type BackupVerifyResponse struct {
	BackupID      string     `json:"backup_id"`
	Verified      bool       `json:"verified"`
	ChecksumMatch bool       `json:"checksum_match"`
	SizeMatch     bool       `json:"size_match"`
	VerifiedAt    *time.Time `json:"verified_at,omitempty"`
	Issues        []string   `json:"issues,omitempty"`
}

// RestoreCreateResponse is the result of POST /api/v1/restore/create.
type RestoreCreateResponse struct {
	RestoreID         string          `json:"restore_id"`
	EstimatedDuration string          `json:"estimated_duration,omitempty"`
	Status            string          `json:"status,omitempty"`
	ValidationResults json.RawMessage `json:"validation_results,omitempty"`
}

// Schedule is one entry returned from /api/v1/schedules.
type Schedule struct {
	ID             string     `json:"id"`
	Name           string     `json:"name,omitempty"`
	CronExpression string     `json:"cron_expression,omitempty"`
	BackupType     string     `json:"backup_type,omitempty"`
	Targets        []string   `json:"targets,omitempty"`
	RetentionDays  int        `json:"retention_days,omitempty"`
	Enabled        bool       `json:"enabled"`
	LastRun        *time.Time `json:"last_run,omitempty"`
	NextRun        *time.Time `json:"next_run,omitempty"`
}

// ScheduleListResponse is the envelope returned by GET /api/v1/schedules.
type ScheduleListResponse struct {
	Schedules []Schedule `json:"schedules"`
	Total     int        `json:"total"`
}
