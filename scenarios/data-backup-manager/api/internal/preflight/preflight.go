// Package preflight evaluates shared backup dependencies once before target
// fan-out. It is intentionally read-only and returns grouped incidents so a
// missing destination credential cannot become one opaque failure per target.
package preflight

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/failures"
	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/sysmounts"

	"github.com/vrooli/api-core/schedule"
)

type (
	Plan   struct{ TargetIDs, DestinationIDs []string }
	Target struct {
		ID      string
		Kind    sources.SourceKind
		Locator string
	}
)

type (
	Destination struct {
		ID          string
		Name        string
		BackendKind string
		Location    string
	}
	TargetLookup interface {
		TargetForRun(context.Context, string) (Target, error)
	}
)

type DestinationLookup interface {
	DestinationForRun(context.Context, string) (Destination, error)
}

type Input struct {
	Plan             Plan
	Targets          TargetLookup
	Destinations     DestinationLookup
	Engine           engine.KopiaEngine
	Sources          *sources.Registry
	Clock            schedule.Clock
	CheckSourcePaths bool
	Readiness        DestinationReadiness
}

// DestinationReadiness is the read-only mounted-volume diagnosis used by
// destination registration and backup preflight. Keeping it as a seam lets
// preflight share the production rules without taking ownership of host repair.
type DestinationReadiness interface {
	Analyze(context.Context, destinationreadiness.AnalyzeInput) (destinationreadiness.Report, error)
}

type Result struct {
	CheckedAt           time.Time
	Ready               bool
	Blocked             bool
	Incidents           []failures.Cause
	BlockedDestinations map[string]failures.Cause
	BlockedTargets      map[string]failures.Cause
}

func (r Result) Summary() string {
	if len(r.Incidents) == 0 {
		return "preflight ready"
	}
	parts := make([]string, 0, len(r.Incidents))
	for _, incident := range r.Incidents {
		scope := string(incident.Scope)
		if len(incident.TargetIDs) > 0 {
			scope = fmt.Sprintf("%s (%d target(s))", scope, len(incident.TargetIDs))
		}
		parts = append(parts, string(incident.Code)+" ["+scope+"]: "+incident.Message)
	}
	return strings.Join(parts, "; ")
}

// Check performs only metadata, repository-status, adapter, and existence
// checks. It never captures a source or writes to a repository.
func Check(ctx context.Context, in Input) Result {
	now := time.Now().UTC()
	if in.Clock != nil {
		now = in.Clock.Now().UTC()
	}
	r := Result{CheckedAt: now, Ready: true, BlockedDestinations: map[string]failures.Cause{}, BlockedTargets: map[string]failures.Cause{}}
	if len(in.Plan.DestinationIDs) == 0 {
		r.add(failures.Cause{Code: failures.DestinationMissing, Category: failures.CategoryDestination, Scope: failures.ScopePlan, Message: "plan has no destination", NextAction: "configure at least one approved destination"})
	}
	if len(in.Plan.TargetIDs) == 0 {
		r.add(failures.Cause{Code: failures.SourceMissing, Category: failures.CategorySource, Scope: failures.ScopePlan, Message: "plan has no target", NextAction: "configure at least one approved target"})
	}

	// Resolve and probe each destination exactly once, then attach the same
	// cause to all affected targets in one grouped incident.
	for _, id := range unique(in.Plan.DestinationIDs) {
		dest, err := in.Destinations.DestinationForRun(ctx, id)
		if err != nil {
			cause := failures.Classify(err)
			cause.DestinationID, cause.Scope = id, failures.ScopeDestination
			if cause.Code == failures.Unknown {
				cause.Code, cause.Category = failures.DestinationMissing, failures.CategoryDestination
			}
			cause.Message = "destination could not be resolved"
			cause.NextAction = "inspect the destination catalog and readiness before retrying"
			r.add(cause)
			continue
		}
		if in.Engine == nil {
			r.add(failures.Cause{Code: failures.RepositoryInvalid, Category: failures.CategoryRepository, Scope: failures.ScopeDestination, DestinationID: id, Message: "backup engine is unavailable", NextAction: "restore resource-kopia availability before retrying"})
			continue
		}
		if _, err := in.Engine.RepoStatus(ctx, dest.Name); err != nil {
			cause := failures.Classify(err)
			cause.DestinationID, cause.Scope = id, failures.ScopeDestination
			if cause.Code == failures.Unknown {
				cause.Code, cause.Category = failures.RepositoryInvalid, failures.CategoryRepository
			}
			cause.TargetIDs = append([]string(nil), in.Plan.TargetIDs...)
			r.add(cause)
		}
		if in.Readiness != nil && dest.BackendKind == "filesystem" && strings.TrimSpace(dest.Location) != "" {
			report, err := in.Readiness.Analyze(ctx, destinationreadiness.AnalyzeInput{Location: dest.Location})
			if err != nil {
				cause := failures.Classify(err)
				cause.DestinationID, cause.Scope = id, failures.ScopeDestination
				if cause.Code == failures.Unknown {
					cause.Code, cause.Category = failures.DestinationInaccessible, failures.CategoryDestination
				}
				cause.TargetIDs = append([]string(nil), in.Plan.TargetIDs...)
				r.add(cause)
			} else if report.OverallSeverity == destinationreadiness.SeverityFail {
				cause := readinessFailure(report)
				cause.DestinationID = id
				cause.TargetIDs = append([]string(nil), in.Plan.TargetIDs...)
				r.add(cause)
			}
		}
	}

	for _, id := range unique(in.Plan.TargetIDs) {
		target, err := in.Targets.TargetForRun(ctx, id)
		if err != nil {
			cause := failures.Classify(err)
			cause.TargetIDs, cause.Scope = []string{id}, failures.ScopeTarget
			if cause.Code == failures.Unknown {
				cause.Code, cause.Category = failures.SourceMissing, failures.CategorySource
			}
			cause.Message = "target could not be resolved"
			cause.NextAction = "restore the source path or remove the stale target"
			r.add(cause)
			continue
		}
		if in.Sources == nil {
			r.add(failures.Cause{Code: failures.SourceUnsupported, Category: failures.CategorySource, Scope: failures.ScopeTarget, TargetIDs: []string{id}, Message: "source adapters are unavailable", NextAction: "restore source adapter registration"})
			continue
		}
		if _, err := in.Sources.Capturer(target.Kind); err != nil {
			cause := failures.Classify(err)
			cause.TargetIDs, cause.Scope = []string{id}, failures.ScopeTarget
			cause.Code, cause.Category = failures.SourceUnsupported, failures.CategorySource
			cause.Message = "source kind is not available"
			r.add(cause)
			continue
		}
		if in.CheckSourcePaths && (target.Kind == sources.KindFilesystem || target.Kind == sources.KindSQLite) && filepath.IsAbs(target.Locator) {
			if _, err := os.Stat(target.Locator); err != nil {
				cause := failures.Classify(err)
				cause.TargetIDs, cause.Scope = []string{id}, failures.ScopeTarget
				cause.Code, cause.Category = failures.SourceMissing, failures.CategorySource
				cause.Message = "source path is missing or inaccessible"
				cause.NextAction = "restore the source path or deregister the stale target"
				r.add(cause)
			}
		}
	}
	return r
}

func readinessFailure(report destinationreadiness.Report) failures.Cause {
	for _, check := range report.Checks {
		if check.Severity != destinationreadiness.SeverityFail {
			continue
		}
		cause := failures.Cause{
			Code:       failures.DestinationInaccessible,
			Category:   failures.CategoryDestination,
			Scope:      failures.ScopeDestination,
			Message:    "destination readiness check failed",
			NextAction: "inspect destination readiness and remediate the native filesystem state before retrying",
		}
		switch check.Code {
		case "mounted_read_write":
			// The blocking condition is read-only; the attributed cause decides
			// the remediation. Keeping the code stable while sharpening the
			// action means existing consumers keep working and the operator
			// stops being handed a menu of unrelated fixes.
			cause.Code = failures.DestinationReadOnly
			cause.Message = readOnlyMessage(report.ReadOnlyCause)
			cause.NextAction = readOnlyNextAction(report.ReadOnlyCause)
		case "destination_dirty":
			cause.Code = failures.DestinationDirty
			cause.Message = "destination filesystem reports a dirty or needs-check state"
			cause.NextAction = "check and repair the destination filesystem under explicit confirmation, then recheck readiness"
		case "destination_missing":
			cause.Code = failures.DestinationUnmounted
			cause.Message = "destination is not mounted or its path is absent"
			cause.NextAction = "inspect and repair the destination with native operating-system tools, then recheck identity"
		case "directory_inaccessible":
			cause.Message = "destination directory is inaccessible"
			cause.NextAction = "restore directory access and re-run destination readiness"
		}
		return cause
	}
	return failures.Cause{
		Code:       failures.DestinationInaccessible,
		Category:   failures.CategoryDestination,
		Scope:      failures.ScopeDestination,
		Message:    "destination readiness check failed",
		NextAction: "inspect destination readiness and remediate the native filesystem state before retrying",
	}
}

// readOnlyMessage states the blocking condition together with its cause, so a
// persisted run outcome carries enough to act on without re-running readiness.
func readOnlyMessage(cause sysmounts.ReadOnlyCause) string {
	switch cause {
	case sysmounts.CauseDeviceWriteProtected:
		return "destination is mounted read-only because its block device is write-protected"
	case sysmounts.CauseFilesystemDirty:
		return "destination is mounted read-only because its filesystem carries a dirty or needs-check flag"
	case sysmounts.CauseMountOption:
		return "destination is mounted read-only because read-only was explicitly requested for this mount"
	default:
		return "destination is mounted read-only for an unattributed cause"
	}
}

func readOnlyNextAction(cause sysmounts.ReadOnlyCause) string {
	switch cause {
	case sysmounts.CauseDeviceWriteProtected:
		return "clear the block-device write protection; filesystem repair cannot restore writes"
	case sysmounts.CauseFilesystemDirty:
		return "check and repair the destination filesystem under explicit confirmation, then remount read/write and recheck readiness"
	case sysmounts.CauseMountOption:
		return "change the declared mount options if this destination is meant to be writable"
	default:
		return "attribute the read-only cause before acting; an unexplained read-only mount must not be repaired blindly"
	}
}

func (r *Result) add(c failures.Cause) {
	if c.FirstObserved.IsZero() {
		c.FirstObserved = r.CheckedAt
	}
	c.LastObserved = r.CheckedAt
	for i := range r.Incidents {
		if r.Incidents[i].Code != c.Code || r.Incidents[i].DestinationID != c.DestinationID || r.Incidents[i].Scope != c.Scope {
			continue
		}
		r.Incidents[i].TargetIDs = mergeIDs(r.Incidents[i].TargetIDs, c.TargetIDs)
		if len(r.Incidents[i].TargetIDs) > 0 {
			c.TargetIDs = r.Incidents[i].TargetIDs
		}
		if c.DestinationID != "" {
			r.BlockedDestinations[c.DestinationID] = r.Incidents[i]
		}
		if c.Scope == failures.ScopeTarget || (c.DestinationID == "" && len(c.TargetIDs) > 0) {
			for _, id := range c.TargetIDs {
				r.BlockedTargets[id] = r.Incidents[i]
			}
		}
		r.Ready, r.Blocked = false, true
		return
	}
	r.Incidents = append(r.Incidents, c)
	if c.DestinationID != "" {
		r.BlockedDestinations[c.DestinationID] = c
	}
	if c.Scope == failures.ScopeTarget || (c.DestinationID == "" && len(c.TargetIDs) > 0) {
		for _, id := range c.TargetIDs {
			r.BlockedTargets[id] = c
		}
	}
	r.Ready, r.Blocked = false, true
	sort.Slice(r.Incidents, func(i, j int) bool { return r.Incidents[i].Code < r.Incidents[j].Code })
}

func unique(ids []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func mergeIDs(a, b []string) []string {
	return unique(append(append([]string(nil), a...), b...))
}
