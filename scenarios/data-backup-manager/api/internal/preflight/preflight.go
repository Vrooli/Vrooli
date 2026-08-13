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

	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/failures"
	"data-backup-manager/internal/sources"

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
	Destination  struct{ ID, Name string }
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
