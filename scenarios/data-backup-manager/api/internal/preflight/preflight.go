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

	"data-backup-manager/internal/capacity"
	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/failures"
	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/sysmounts"

	"github.com/vrooli/api-core/schedule"
)

type (
	Plan struct {
		TargetIDs, DestinationIDs []string
		// RetainedSnapshots is the maximum snapshots the plan's retention policy
		// can keep. It is a count, never a byte multiplier.
		RetainedSnapshots int
		// FirstCopy is true when at least one selected target has no successful
		// snapshot on the destination. The first copy pays the full logical cost.
		FirstCopy bool
		// MeasuredIncrementalBytes is the last successful run's physical repo
		// growth. Zero means no usable history and the estimate is forecast-only.
		MeasuredIncrementalBytes int64
		MeasuredDedupRatio       float64
		ReserveBytes             int64
		CapacityPolicy           capacity.Policy
		AllowWriteProbe          bool
	}
	Target struct {
		ID      string
		Kind    sources.SourceKind
		Locator string
	}
)

type (
	Destination struct {
		ID             string
		Name           string
		BackendKind    string
		Location       string
		DeviceIdentity *destinationreadiness.DeviceIdentity
	}
	TargetLookup interface {
		TargetForRun(context.Context, string) (Target, error)
	}
)

type DestinationLookup interface {
	DestinationForRun(context.Context, string) (Destination, error)
	// WithResolvedDestinationForRead resolves a destination for repository
	// probes. Preflight is read-only: it must not demand write-ready filesystem
	// state, or a mounted-but-unknown volume can never be inspected.
	WithResolvedDestinationForRead(context.Context, string, func(Destination) error) error
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
	r := Result{
		CheckedAt: now, Ready: true,
		BlockedDestinations: map[string]failures.Cause{},
		BlockedTargets:      map[string]failures.Cause{},
	}
	if len(in.Plan.DestinationIDs) == 0 {
		r.add(failures.Cause{Code: failures.DestinationMissing, Category: failures.CategoryDestination, Scope: failures.ScopePlan, Message: "plan has no destination", NextAction: "configure at least one approved destination"})
	}
	if len(in.Plan.TargetIDs) == 0 {
		r.add(failures.Cause{Code: failures.SourceMissing, Category: failures.CategorySource, Scope: failures.ScopePlan, Message: "plan has no target", NextAction: "configure at least one approved target"})
	}

	// Estimate every selected local source before destination readiness so its
	// capacity check uses the full source set and plan retention. If any source
	// cannot be enumerated (or has no local estimator), capacity remains unknown
	// rather than passing on a partial sum.
	resolvedTargets := make(map[string]Target, len(in.Plan.TargetIDs))
	sourceEstimateErrors := make(map[string]error)
	selectedTargetBytes := int64(0)
	capacityEstimateComplete := in.CheckSourcePaths
	for _, id := range unique(in.Plan.TargetIDs) {
		if in.Targets == nil {
			capacityEstimateComplete = false
			continue
		}
		target, err := in.Targets.TargetForRun(ctx, id)
		if err != nil {
			capacityEstimateComplete = false
			continue
		}
		resolvedTargets[id] = target
		if !in.CheckSourcePaths || !filepath.IsAbs(target.Locator) ||
			(target.Kind != sources.KindFilesystem && target.Kind != sources.KindSQLite) {
			capacityEstimateComplete = false
			continue
		}
		bytes, supported, estimateErr := sources.EstimateLocalSourceBytes(target.Kind, target.Locator)
		if estimateErr != nil {
			// A source that cannot be enumerated is blocked on its own below
			// (source_missing, scope target) and will not be captured, so it adds
			// nothing to what this run writes. It must not make the destination's
			// capacity unknown: that read as "destination filesystem readiness is
			// unknown" and stopped every other target of the plan.
			sourceEstimateErrors[id] = estimateErr
			continue
		}
		if !supported {
			capacityEstimateComplete = false
			continue
		}
		selectedTargetBytes += bytes
	}
	if !capacityEstimateComplete {
		selectedTargetBytes = 0
	}

	// Resolve each destination, revalidate at the repository-probe boundary,
	// then attach shared causes to affected targets in grouped incidents.
	for _, id := range unique(in.Plan.DestinationIDs) {
		dest, err := in.Destinations.DestinationForRun(ctx, id)
		if err != nil {
			cause := failures.Classify(err)
			cause.DestinationID, cause.Scope = id, failures.ScopeDestination
			if cause.Code == failures.Unknown {
				cause.Category = failures.CategoryDestination
			}
			if cause.Code == failures.DestinationInaccessible {
				cause.Message = "destination identity, writable state, or filesystem health could not be proven"
				cause.NextAction = "inspect destination identity and filesystem readiness before retrying"
			} else {
				cause.Message = "destination could not be resolved"
				cause.NextAction = "inspect the destination catalog and readiness before retrying"
			}
			r.add(cause)
			continue
		}
		if dest.BackendKind == "filesystem" && temporaryDestination(dest.Location) {
			r.add(failures.Cause{Code: "destination_ephemeral", Category: failures.CategoryDestination, Scope: failures.ScopeDestination, DestinationID: id, TargetIDs: append([]string(nil), in.Plan.TargetIDs...), Message: "backup destination uses temporary storage; durable protection is unavailable", NextAction: "select a permanent destination; preserve existing repository bytes and credentials"})
			continue
		}
		gateErr := in.Destinations.WithResolvedDestinationForRead(ctx, id, func(current Destination) error {
			if current.ID != dest.ID || current.Name != dest.Name || current.BackendKind != dest.BackendKind {
				return fmt.Errorf("destination identity changed during preflight")
			}
			dest = current
			if in.Engine == nil {
				r.add(failures.Cause{Code: failures.RepositoryInvalid, Category: failures.CategoryRepository, Scope: failures.ScopeDestination, DestinationID: id, Message: "backup engine is unavailable", NextAction: "restore resource-kopia availability before retrying"})
				return nil
			}
			if _, err := in.Engine.RepoStatus(ctx, current.Name); err != nil {
				cause := failures.Classify(err)
				cause.DestinationID, cause.Scope = id, failures.ScopeDestination
				if cause.Code == failures.Unknown {
					cause.Code, cause.Category = failures.RepositoryInvalid, failures.CategoryRepository
				}
				cause.TargetIDs = append([]string(nil), in.Plan.TargetIDs...)
				r.add(cause)
			} else if _, err := in.Engine.RepoStats(ctx, current.Name); err != nil {
				// Capacity and retention safety depend on a complete repository usage
				// read. A passing status and snapshot list do not prove that Kopia can
				// enumerate the content/index shards used by stats; do not start source
				// capture when the destination's current footprint cannot be measured.
				message := "repository usage could not be measured"
				next := "preserve repository bytes and credentials; resolve the repository usage error before writing backups"
				if owner, ok := in.Engine.(engine.RepoDamageScanner); ok {
					if scan, scanErr := owner.RepoDamageScan(ctx, current.Name); scanErr == nil && len(scan.Unreadable) > 0 {
						message = summarizeRepositoryDamage(scan)
						next = "preserve repository bytes and credentials; use Kopia's typed damage evidence to select a verified recovery path before writing backups"
					}
				}
				r.add(failures.Cause{
					Code: failures.RepositoryInvalid, Category: failures.CategoryRepository,
					Scope: failures.ScopeDestination, DestinationID: id,
					TargetIDs:  append([]string(nil), in.Plan.TargetIDs...),
					Message:    message,
					NextAction: next,
				})
			} else if _, err := in.Engine.SnapshotList(ctx, current.Name, ""); err != nil {
				// A repository status proves Kopia can open the repository and its
				// credential. Enumerating manifests catches missing or unreadable
				// snapshot metadata before we capture sources or attempt a new write
				// into a damaged repository. The engine bounds the inventory by
				// bytes, not by an arbitrary manifest count, so a repository with
				// hundreds of legitimate snapshots remains enumerable.
				r.add(failures.Cause{
					Code: failures.RepositoryInvalid, Category: failures.CategoryRepository,
					Scope: failures.ScopeDestination, DestinationID: id,
					TargetIDs:  append([]string(nil), in.Plan.TargetIDs...),
					Message:    "repository snapshot metadata could not be enumerated",
					NextAction: "preserve repository bytes and credentials; inspect repository integrity before retrying",
				})
			}
			return nil
		})
		if gateErr != nil {
			cause := failures.Classify(gateErr)
			cause.DestinationID, cause.Scope = id, failures.ScopeDestination
			cause.TargetIDs = append([]string(nil), in.Plan.TargetIDs...)
			cause.Message = "destination changed or readiness failed during repository preflight"
			cause.NextAction = "recheck destination identity and filesystem readiness before retrying"
			r.add(cause)
		}
		if in.Readiness != nil && dest.BackendKind == "filesystem" && strings.TrimSpace(dest.Location) != "" {
			readinessInput := destinationreadiness.AnalyzeInput{DestinationID: dest.ID, Location: dest.Location}
			if dest.DeviceIdentity != nil {
				readinessInput.ExpectedDevice = *dest.DeviceIdentity
			}
			if capacityEstimateComplete {
				readinessInput.Capacity = capacity.Input{
					LogicalSourceBytes:       selectedTargetBytes,
					RetainedSnapshots:        in.Plan.RetainedSnapshots,
					FirstCopy:                in.Plan.FirstCopy,
					MeasuredIncrementalBytes: in.Plan.MeasuredIncrementalBytes,
					MeasuredDedupRatio:       in.Plan.MeasuredDedupRatio,
					ReserveBytes:             in.Plan.ReserveBytes,
					Policy:                   in.Plan.CapacityPolicy,
				}
			}
			readinessInput.AllowWriteProbe = in.Plan.AllowWriteProbe
			report, err := in.Readiness.Analyze(ctx, readinessInput)
			if err != nil {
				cause := failures.Classify(err)
				cause.DestinationID, cause.Scope = id, failures.ScopeDestination
				if cause.Code == failures.Unknown {
					cause.Code, cause.Category = failures.DestinationInaccessible, failures.CategoryDestination
				}
				cause.TargetIDs = append([]string(nil), in.Plan.TargetIDs...)
				r.add(cause)
			} else if report.OverallSeverity == destinationreadiness.SeverityFail || report.OverallSeverity == destinationreadiness.SeverityUnknown {
				cause := readinessFailure(report)
				cause.DestinationID = id
				cause.TargetIDs = append([]string(nil), in.Plan.TargetIDs...)
				r.add(cause)
			}
		}
	}

	for _, id := range unique(in.Plan.TargetIDs) {
		target, ok := resolvedTargets[id]
		if !ok {
			r.add(failures.Cause{Code: failures.SourceMissing, Category: failures.CategorySource, Scope: failures.ScopeTarget, TargetIDs: []string{id}, Message: "target could not be resolved", NextAction: "restore the source path or remove the stale target"})
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
		if err := sourceEstimateErrors[id]; err != nil {
			cause := failures.ClassifyCapture(err)
			cause.TargetIDs, cause.Scope = []string{id}, failures.ScopeTarget
			r.add(cause)
			continue
		}
		if in.CheckSourcePaths && target.Kind == sources.KindWorkspace && filepath.IsAbs(target.Locator) {
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

func summarizeRepositoryDamage(scan engine.RepoDamageScan) string {
	counts := map[string]int{}
	for _, entry := range scan.Unreadable {
		counts[entry.Class+"/"+entry.Problem]++
	}
	parts := make([]string, 0, len(counts))
	for kind, count := range counts {
		parts = append(parts, fmt.Sprintf("%d %s", count, strings.ReplaceAll(kind, "/", " ")))
	}
	sort.Strings(parts)
	return "Kopia classified unreadable repository entries: " + strings.Join(parts, ", ")
}

func temporaryDestination(path string) bool {
	path = filepath.Clean(path)
	for _, root := range []string{"/tmp", "/var/tmp", "/run"} {
		if path == root || strings.HasPrefix(path, root+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func readinessFailure(report destinationreadiness.Report) failures.Cause {
	if report.OverallSeverity == destinationreadiness.SeverityUnknown {
		return failures.Cause{
			Code:       failures.DestinationInaccessible,
			Category:   failures.CategoryDestination,
			Scope:      failures.ScopeDestination,
			Message:    "destination filesystem readiness is unknown",
			NextAction: "obtain supported filesystem health evidence and re-run readiness before enabling backup writes",
		}
	}
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
		case "write_probe":
			cause.Code = failures.DestinationWriteProbe
			cause.Message = check.Message
			if check.NextAction != "" {
				cause.NextAction = check.NextAction
			}
		case "directory_inaccessible":
			cause.Message = "destination directory is inaccessible"
			cause.NextAction = "restore directory access and re-run destination readiness"
		case "capacity":
			cause.Code = failures.DestinationCapacity
			cause.Category = failures.CategoryCapacity
			cause.Message = check.Message
			if check.NextAction != "" {
				cause.NextAction = check.NextAction
			}
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
		if !sameIncident(r.Incidents[i], c) {
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

// sameIncident reports whether two causes describe the same blocking incident.
// Destination-scope causes are shared by every target the destination serves,
// so they merge. A target-scope cause is a fact about exactly one source: two
// unreadable targets must stay separate, or the merged incident keeps the
// first path's message and every other target is reported with the wrong cause.
func sameIncident(existing, incoming failures.Cause) bool {
	if existing.Code != incoming.Code || existing.DestinationID != incoming.DestinationID || existing.Scope != incoming.Scope {
		return false
	}
	if incoming.Scope == failures.ScopeTarget && incoming.DestinationID == "" {
		return oneTarget(existing.TargetIDs) && oneTarget(incoming.TargetIDs) && existing.TargetIDs[0] == incoming.TargetIDs[0]
	}
	return true
}

func oneTarget(ids []string) bool { return len(ids) == 1 }

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
