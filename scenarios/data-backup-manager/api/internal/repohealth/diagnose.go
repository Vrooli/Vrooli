package repohealth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"data-backup-manager/internal/engine"
)

// Cause is the diagnosed reason backups to a destination cannot proceed.
type Cause string

const (
	// CauseNone means every read-only check passed: the failure has cleared.
	CauseNone Cause = "none"
	// CauseVolumeUnmounted: the drive is not mounted. The existing volume
	// recovery owns this; no second question is raised.
	CauseVolumeUnmounted Cause = "volume_unmounted"
	// CauseVolumeReadOnly: mounted, but not writable.
	CauseVolumeReadOnly Cause = "volume_read_only"
	// CauseRepositoryUnopenable: kopia cannot open the repository at all.
	CauseRepositoryUnopenable Cause = "repository_unopenable"
	// CauseSnapshotsUnlistable: kopia opens it but cannot list its snapshots.
	CauseSnapshotsUnlistable Cause = "snapshots_unlistable"
	// CauseLeftoverTempUnreadable: kopia opens and lists everything, and the only
	// unreadable entries are leftover temporary upload files.
	CauseLeftoverTempUnreadable Cause = "leftover_temp_unreadable"
	// CauseContentUnreadable: at least one unreadable entry is committed
	// repository content or something that is not a temporary file.
	CauseContentUnreadable Cause = "content_unreadable"
	// CauseUsageUnexplained: usage cannot be measured and no unreadable entry
	// explains it.
	CauseUsageUnexplained Cause = "usage_unexplained"
)

// VolumeState is what the host reports about the drive holding a destination.
type VolumeState struct {
	Known    bool   `json:"known"`
	Mounted  bool   `json:"mounted"`
	ReadOnly bool   `json:"read_only"`
	Label    string `json:"label,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// PlanFacts is one plan's stake in the destination.
type PlanFacts struct {
	PlanID             string   `json:"plan_id"`
	PlanName           string   `json:"plan_name"`
	Enabled            bool     `json:"enabled"`
	Schedule           string   `json:"schedule,omitempty"`
	TargetCount        int      `json:"target_count"`
	UnprotectedTargets int      `json:"unprotected_targets"`
	UnprotectedNames   []string `json:"unprotected_names,omitempty"`
}

// Diagnosis is the read-only account of why a destination is failing.
type Diagnosis struct {
	CheckedAt       time.Time `json:"checked_at"`
	DestinationID   string    `json:"destination_id"`
	DestinationName string    `json:"destination_name"`
	DriveName       string    `json:"drive_name"`
	Location        string    `json:"location"`
	RepositoryRoot  string    `json:"repository_root"`
	Cause           Cause     `json:"cause"`

	Volume VolumeState `json:"volume"`

	RepositoryOpens   bool      `json:"repository_opens"`
	OpenError         string    `json:"open_error,omitempty"`
	SnapshotsListed   bool      `json:"snapshots_listed"`
	SnapshotCount     int       `json:"snapshot_count"`
	SnapshotError     string    `json:"snapshot_error,omitempty"`
	LatestSnapshotAt  time.Time `json:"latest_snapshot_at,omitempty"`
	UsageMeasurable   bool      `json:"usage_measurable"`
	UsageError        string    `json:"usage_error,omitempty"`
	ScanRan           bool      `json:"scan_ran"`
	ScanError         string    `json:"scan_error,omitempty"`
	ScanFiles         int64     `json:"scan_files"`
	ScanBytes         int64     `json:"scan_bytes"`
	ScanSeconds       float64   `json:"scan_seconds"`
	ReadableTempFiles int64     `json:"readable_temp_files"`

	// Unreadable are the entries nothing yet covers; Accepted are covered by an
	// already recorded exception.
	Unreadable []Entry `json:"unreadable,omitempty"`
	Accepted   []Entry `json:"accepted,omitempty"`

	LastSuccessAt     time.Time `json:"last_success_at,omitempty"`
	LastSuccessSource string    `json:"last_success_source,omitempty"`
	PriorRepairAt     time.Time `json:"prior_repair_at,omitempty"`

	Plans PlanSummary `json:"plans"`
}

// PlanSummary is the destination's stake across every plan that writes to it:
// how many distinct targets those plans cover and how many of them no other
// enabled, recently successful plan protects.
type PlanSummary struct {
	Plans            []PlanFacts `json:"plans,omitempty"`
	TargetTotal      int         `json:"target_total"`
	Unprotected      int         `json:"unprotected"`
	UnprotectedNames []string    `json:"unprotected_names,omitempty"`
}

// Signature identifies the condition so the same condition raises the same
// question and a changed one raises a new one: the cause plus the exact set of
// unreadable paths.
func (d Diagnosis) Signature() string {
	paths := make([]string, 0, len(d.Unreadable))
	for _, e := range d.Unreadable {
		paths = append(paths, e.RelPath+"|"+string(e.Problem)+"|"+string(e.Class))
	}
	sort.Strings(paths)
	sum := sha256.Sum256([]byte(d.DestinationID + "\x00" + string(d.Cause) + "\x00" + strings.Join(paths, "\n")))
	return hex.EncodeToString(sum[:12])
}

// DaysStopped is whole days since the last known success, or -1 when unknown.
func (d Diagnosis) DaysStopped() int {
	if d.LastSuccessAt.IsZero() {
		return -1
	}
	days := int(d.CheckedAt.Sub(d.LastSuccessAt).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// RepoProbe is the read-only engine surface diagnosis uses. It is the usage
// engine, so an approved exception is honoured here exactly as in preflight.
type RepoProbe interface {
	RepoStatus(ctx context.Context, repo string) (engine.RepoStatus, error)
	SnapshotList(ctx context.Context, repo, path string) ([]engine.Snapshot, error)
	RepoStats(ctx context.Context, repo string) (engine.RepoStats, error)
}

// DiagnoseInput carries everything diagnosis needs, as seams.
type DiagnoseInput struct {
	Now             time.Time
	DestinationID   string
	DestinationName string
	Location        string
	RepositoryRoot  string
	Volume          func(context.Context) (VolumeState, error)
	Engine          RepoProbe
	Scan            func(context.Context, string) (Scan, error)
	Scanner         *Scanner
	Exceptions      ExceptionStore
	LastSuccess     func(context.Context, []engine.Snapshot) (at time.Time, source string)
	PriorRepairAt   func(context.Context) time.Time
	Plans           func(context.Context) PlanSummary
}

// Diagnose runs the read-only checks in order of cost and stops at the first
// finding that explains the failure. It writes nothing.
func Diagnose(ctx context.Context, in DiagnoseInput) (d Diagnosis) {
	d = Diagnosis{
		CheckedAt: in.Now.UTC(), DestinationID: in.DestinationID, DestinationName: in.DestinationName,
		Location: in.Location, RepositoryRoot: in.RepositoryRoot,
	}
	d.DriveName = in.DestinationName
	if in.Volume != nil {
		vol, err := in.Volume(ctx)
		if err != nil {
			vol = VolumeState{Detail: err.Error()}
		}
		d.Volume = vol
		if strings.TrimSpace(vol.Label) != "" {
			d.DriveName = vol.Label
		}
	}
	if in.Plans != nil {
		d.Plans = in.Plans(ctx)
	}
	if in.PriorRepairAt != nil {
		d.PriorRepairAt = in.PriorRepairAt(ctx)
	}

	var snapshots []engine.Snapshot
	defer func() {
		if in.LastSuccess != nil {
			d.LastSuccessAt, d.LastSuccessSource = in.LastSuccess(ctx, snapshots)
		}
	}()

	if d.Volume.Known && !d.Volume.Mounted {
		d.Cause = CauseVolumeUnmounted
		return d
	}
	if d.Volume.Known && d.Volume.ReadOnly {
		d.Cause = CauseVolumeReadOnly
		return d
	}
	if _, err := in.Engine.RepoStatus(ctx, in.DestinationName); err != nil {
		d.OpenError = err.Error()
		d.Cause = CauseRepositoryUnopenable
		return d
	}
	d.RepositoryOpens = true
	snaps, err := in.Engine.SnapshotList(ctx, in.DestinationName, "")
	if err != nil {
		d.SnapshotError = err.Error()
		d.Cause = CauseSnapshotsUnlistable
		return d
	}
	snapshots = snaps
	d.SnapshotsListed, d.SnapshotCount = true, len(snaps)
	for _, s := range snaps {
		if t, err := time.Parse(time.RFC3339Nano, s.StartTime); err == nil && t.After(d.LatestSnapshotAt) {
			d.LatestSnapshotAt = t.UTC()
		}
	}
	if _, err := in.Engine.RepoStats(ctx, in.DestinationName); err == nil {
		d.UsageMeasurable = true
		d.Cause = CauseNone
		return d
	} else {
		d.UsageError = err.Error()
	}

	if strings.TrimSpace(in.RepositoryRoot) == "" {
		d.Cause = CauseUsageUnexplained
		return d
	}
	scan := in.Scan
	if scan == nil {
		scan = in.Scanner.Walk
	}
	result, err := scan(ctx, in.RepositoryRoot)
	if err != nil {
		d.ScanError = err.Error()
		d.Cause = CauseUsageUnexplained
		return d
	}
	d.ScanRan = true
	d.ScanFiles, d.ScanBytes, d.ScanSeconds = result.Files, result.Bytes, result.Duration.Seconds()
	d.ReadableTempFiles = result.TempFiles

	var exceptions []Exception
	if in.Exceptions != nil {
		exceptions, _ = in.Exceptions.List(ctx, in.DestinationID)
	}
	verdict := Evaluate(in.Scanner.fsys(), in.RepositoryRoot, result.Unreadable, exceptions)
	d.Unreadable, d.Accepted = verdict.Blocking, verdict.Excused
	switch {
	case len(verdict.Blocking) == 0:
		d.Cause = CauseUsageUnexplained
	case allLeftoverTemp(verdict.Blocking):
		d.Cause = CauseLeftoverTempUnreadable
	default:
		d.Cause = CauseContentUnreadable
	}
	return d
}

func allLeftoverTemp(entries []Entry) bool {
	for _, e := range entries {
		if e.Class != ClassLeftoverTemp {
			return false
		}
	}
	return len(entries) > 0
}

// SnapshotSourceMatches reports whether a kopia snapshot source belongs to a DBM
// target registered as owner/name. DBM records snapshots under a logical source
// of the form dbm://<owner>/<name>, which kopia stores as a path ending in
// "dbm:/<owner>/<name>".
func SnapshotSourceMatches(source, owner, name string) bool {
	return strings.HasSuffix(source, "dbm:/"+owner+"/"+name)
}

// DescribeProblem is a plain-language phrase for a read failure.
func DescribeProblem(p Problem) string {
	switch p {
	case ProblemIO:
		return "a drive read error"
	case ProblemPermission:
		return "a permissions error"
	default:
		return "an unknown error"
	}
}

func (d Diagnosis) String() string {
	return fmt.Sprintf("%s on %s", d.Cause, d.DestinationName)
}
