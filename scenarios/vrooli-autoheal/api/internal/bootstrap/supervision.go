package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/coreset"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	checksvrooli "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/vrooli"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/userconfig"
)

const supervisionSourceCheckID = "vrooli-supervision-set-source"

// SupervisionSnapshot records whether a report came directly from the control
// plane or from the in-memory last-known-good copy. A source failure never
// narrows the active set to empty.
type SupervisionSnapshot struct {
	Report    coreset.Report
	Degraded  bool
	SourceErr string
	LoadedAt  time.Time
}

// SupervisionSource loads the one canonical supervision declaration and owns
// its last-known-good state.
type SupervisionSource struct {
	mu       sync.RWMutex
	executor checks.CommandExecutor
	lastGood *SupervisionSnapshot
	now      func() time.Time
}

func NewSupervisionSource(executor checks.CommandExecutor) *SupervisionSource {
	if executor == nil {
		executor = checks.DefaultExecutor
	}
	return &SupervisionSource{executor: executor, now: time.Now}
}

func (s *SupervisionSource) Load(ctx context.Context) (SupervisionSnapshot, error) {
	raw, err := s.executor.Output(ctx, "vrooli", "supervision-set", "--json")
	if err == nil {
		var report coreset.Report
		if decodeErr := json.Unmarshal(raw, &report); decodeErr != nil {
			err = fmt.Errorf("decode supervision set: %w", decodeErr)
		} else if validateErr := validateSupervisionReport(report); validateErr != nil {
			err = validateErr
		} else {
			snapshot := SupervisionSnapshot{Report: report, LoadedAt: s.now()}
			s.mu.Lock()
			s.lastGood = &snapshot
			s.mu.Unlock()
			return snapshot, nil
		}
	}

	if err == nil {
		err = fmt.Errorf("supervision set unavailable")
	}
	s.mu.RLock()
	lastGood := s.lastGood
	s.mu.RUnlock()
	if lastGood == nil {
		return SupervisionSnapshot{}, fmt.Errorf("load supervision set with no last-known-good state: %w", err)
	}
	fallback := *lastGood
	fallback.Degraded = true
	fallback.SourceErr = err.Error()
	return fallback, nil
}

func validateSupervisionReport(report coreset.Report) error {
	if len(report.Members) == 0 {
		return fmt.Errorf("supervision set is empty")
	}
	seen := make(map[string]struct{}, len(report.Members))
	for _, member := range report.Members {
		if strings.TrimSpace(member.Name) == "" {
			return fmt.Errorf("supervision member has an empty name")
		}
		if member.Kind != coreset.MemberKindScenario && member.Kind != coreset.MemberKindResource {
			return fmt.Errorf("supervision member %q has invalid kind %q", member.Name, member.Kind)
		}
		if member.SupervisionIntent != coreset.IntentMustStart && member.SupervisionIntent != coreset.IntentTryStart {
			return fmt.Errorf("supervision member %q has invalid intent %q", member.Name, member.SupervisionIntent)
		}
		key := member.Kind + ":" + member.Name
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("supervision member %q is duplicated", key)
		}
		seen[key] = struct{}{}
		if len(member.AttributionChain) == 0 || member.AttributionChain[len(member.AttributionChain)-1].Source != "core.seed" {
			return fmt.Errorf("supervision member %q has no complete core.seed attribution chain", key)
		}
	}
	return nil
}

// SupervisionController reconciles the canonical supervision set into the live
// check registry. Operator monitoring config is advisory only and cannot add
// a second, unreconciled supervision universe.
type SupervisionController struct {
	mu        sync.Mutex
	registry  *checks.Registry
	configMgr *userconfig.Manager
	source    *SupervisionSource
	managed   map[string]struct{}
}

func NewSupervisionController(registry *checks.Registry, configMgr *userconfig.Manager, source *SupervisionSource) *SupervisionController {
	return &SupervisionController{registry: registry, configMgr: configMgr, source: source, managed: make(map[string]struct{})}
}

func (c *SupervisionController) Refresh(ctx context.Context) (SupervisionSnapshot, error) {
	snapshot, err := c.source.Load(ctx)
	if err != nil {
		return SupervisionSnapshot{}, err
	}

	desired := make(map[string]checks.Check)

	supervised := make(map[string]string, len(snapshot.Report.Members))
	for _, member := range snapshot.Report.Members {
		critical := member.SupervisionIntent == coreset.IntentMustStart
		var check checks.Check
		var id string
		switch member.Kind {
		case coreset.MemberKindScenario:
			id = "scenario-" + member.Name
			check = checksvrooli.NewScenarioCheck(member.Name, critical, checksvrooli.WithScenarioSupervision(member.SupervisionIntent, member.AttributionChain))
		case coreset.MemberKindResource:
			if kept, dropped := PruneMissingResources([]string{member.Name}); len(kept) == 0 {
				if len(dropped) > 0 {
					log.Printf("vrooli-autoheal: canonical supervision member %q is not a repository resource; dropping it from active checks", member.Name)
				}
				continue
			}
			id = "resource-" + member.Name
			check = checksvrooli.NewResourceCheck(member.Name, checksvrooli.WithResourceSupervision(member.SupervisionIntent, member.AttributionChain))
		}
		desired[id] = check
		supervised[id] = member.SupervisionIntent
	}
	// Report stale operator entries once per reconciliation, but never register
	// them. This makes retirement observable without allowing the old additive
	// config to recreate deleted scenarios or resources.
	monitoring := c.configMgr.GetMonitoring()
	for name := range monitoring.Scenarios {
		if _, ok := supervised["scenario-"+name]; !ok {
			log.Printf("vrooli-autoheal: stale operator scenario override %q dropped", name)
		}
	}
	for _, name := range monitoring.Resources {
		if _, ok := supervised["resource-"+name]; !ok {
			log.Printf("vrooli-autoheal: stale operator resource override %q dropped", name)
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for id := range c.managed {
		if _, keep := desired[id]; !keep {
			c.registry.Unregister(id)
		}
	}
	ids := make([]string, 0, len(desired))
	for id := range desired {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c.registry.Register(desired[id])
	}
	c.managed = make(map[string]struct{}, len(desired))
	for id := range desired {
		c.managed[id] = struct{}{}
	}
	c.configMgr.SetSupervisedChecks(supervised)
	return snapshot, nil
}

// Reconcile is the narrow handler-facing form used after operator monitoring
// overrides change.
func (c *SupervisionController) Reconcile(ctx context.Context) error {
	_, err := c.Refresh(ctx)
	return err
}

type supervisionSourceCheck struct{ controller *SupervisionController }

func (c *supervisionSourceCheck) ID() string    { return supervisionSourceCheckID }
func (c *supervisionSourceCheck) Title() string { return "Supervision Set Authority" }
func (c *supervisionSourceCheck) Description() string {
	return "Reloads the canonical supervision set and reconciles active target checks"
}

func (c *supervisionSourceCheck) Importance() string {
	return "Prevents autoheal from drifting to an independent hardcoded core list"
}
func (c *supervisionSourceCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *supervisionSourceCheck) IntervalSeconds() int       { return 30 }
func (c *supervisionSourceCheck) Platforms() []platform.Type { return nil }
func (c *supervisionSourceCheck) Run(ctx context.Context) checks.Result {
	started := time.Now()
	result := checks.Result{CheckID: supervisionSourceCheckID, Timestamp: started, Details: make(map[string]interface{})}
	snapshot, err := c.controller.Refresh(ctx)
	result.Duration = time.Since(started)
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "canonical supervision set is unavailable and no last-known-good set exists"
		result.Details["error"] = err.Error()
		return result
	}
	result.Details["memberCount"] = len(snapshot.Report.Members)
	result.Details["source"] = snapshot.Report.Source
	result.Details["loadedAt"] = snapshot.LoadedAt
	result.Details["usingLastKnownGood"] = snapshot.Degraded
	if snapshot.Degraded {
		result.Status = checks.StatusWarning
		result.Message = "canonical supervision set is unavailable; retaining last-known-good checks"
		result.Details["sourceError"] = snapshot.SourceErr
		return result
	}
	result.Status = checks.StatusOK
	result.Message = "canonical supervision set loaded and active checks reconciled"
	return result
}
