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
	"github.com/vrooli/api-core/demand"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	checksvrooli "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/vrooli"
	integration "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/integrations/vrooli"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/userconfig"
)

const supervisionSourceCheckID = "vrooli-supervision-set-source"

const (
	coreSupervisionConsumerID = "vrooli-autoheal:core-supervision"
	coreSupervisionRequestID  = "core-supervision"
	coreSupervisionLeaseTTL   = 2 * time.Minute
)

// SupervisionSnapshot records whether a report came directly from the control
// plane or from the in-memory last-known-good copy. A source failure never
// narrows the active set to empty.
type SupervisionSnapshot struct {
	Report    coreset.Report
	Degraded  bool
	SourceErr string
	LoadedAt  time.Time
	// DemandErrors reports lease synchronization failures without making the
	// health controller discard a valid supervision set. A core lease is a
	// safety hold; failure to refresh it must remain visible to operators.
	DemandErrors []string
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
	mu                     sync.Mutex
	registry               *checks.Registry
	configMgr              *userconfig.Manager
	source                 *SupervisionSource
	managed                map[string]struct{}
	coreDemand             demand.LeaseClient
	coreLeases             map[string]string
	resourceStatusProvider integration.ResourceStatusSnapshotProvider
}

// SetResourceStatusProvider installs the process-shared typed snapshot seam
// used by dynamically reconciled resource checks.
func (c *SupervisionController) SetResourceStatusProvider(provider integration.ResourceStatusSnapshotProvider) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resourceStatusProvider = provider
}

// NewSupervisionController accepts an optional demand client so unit tests can
// exercise core-hold reconciliation without invoking the control-plane CLI.
// Production wiring supplies a client; callers that do not supply one retain
// the historical check-only behavior.
func NewSupervisionController(registry *checks.Registry, configMgr *userconfig.Manager, source *SupervisionSource, coreDemand ...demand.LeaseClient) *SupervisionController {
	var leaseClient demand.LeaseClient
	if len(coreDemand) > 0 {
		leaseClient = coreDemand[0]
	}
	return &SupervisionController{
		registry:   registry,
		configMgr:  configMgr,
		source:     source,
		managed:    make(map[string]struct{}),
		coreDemand: leaseClient,
		coreLeases: make(map[string]string),
	}
}

func newSupervisionDemandClient(executor checks.CommandExecutor) demand.LeaseClient {
	return demand.Client{Runner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return executor.CombinedOutput(ctx, name, args...)
	}}
}

func (c *SupervisionController) Refresh(ctx context.Context) (SupervisionSnapshot, error) {
	snapshot, err := c.source.Load(ctx)
	if err != nil {
		return SupervisionSnapshot{}, err
	}

	desired := make(map[string]checks.Check)
	desiredCoreLeases := make(map[string]string)

	supervised := make(map[string]string, len(snapshot.Report.Members))
	for _, member := range snapshot.Report.Members {
		critical := member.SupervisionIntent == coreset.IntentMustStart
		var check checks.Check
		var id string
		switch member.Kind {
		case coreset.MemberKindScenario:
			id = "scenario-" + member.Name
			check = checksvrooli.NewScenarioCheck(member.Name, critical, checksvrooli.WithScenarioSupervision(member.SupervisionIntent, member.AttributionChain))
			if c.coreDemand != nil {
				leaseID := demand.StableLeaseID(coreSupervisionConsumerID, member.Name, coreSupervisionRequestID)
				desiredCoreLeases[leaseID] = member.Name
			}
		case coreset.MemberKindResource:
			if kept, dropped := PruneMissingResources([]string{member.Name}); len(kept) == 0 {
				if len(dropped) > 0 {
					log.Printf("vrooli-autoheal: canonical supervision member %q is not a repository resource; dropping it from active checks", member.Name)
				}
				continue
			}
			id = "resource-" + member.Name
			check = checksvrooli.NewResourceCheck(member.Name,
				checksvrooli.WithResourceSupervision(member.SupervisionIntent, member.AttributionChain),
				checksvrooli.WithResourceStatusProvider(c.resourceStatusProvider),
			)
		}
		desired[id] = check
		supervised[id] = member.SupervisionIntent
	}

	// Core supervision is an authority pin that composes with demand-managed
	// lifecycle. It does not start or stop anything itself; it only prevents a
	// demand-managed instance from being reaped while the canonical set still
	// names it. A degraded source retains the last-known-good set and therefore
	// must not release any existing pins.
	demandErrors := c.reconcileCoreLeases(ctx, desiredCoreLeases, snapshot.Degraded)
	snapshot.DemandErrors = demandErrors
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
	c.registry.SetSupervisedChecks(supervised)
	c.configMgr.SetSupervisedChecks(supervised)
	return snapshot, nil
}

func (c *SupervisionController) reconcileCoreLeases(ctx context.Context, desired map[string]string, degraded bool) []string {
	if c.coreDemand == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	errorsOut := make([]string, 0)
	desiredIDs := make([]string, 0, len(desired))
	for leaseID := range desired {
		desiredIDs = append(desiredIDs, leaseID)
	}
	sort.Strings(desiredIDs)
	for _, leaseID := range desiredIDs {
		scenario := desired[leaseID]
		var renewErr error
		if _, tracked := c.coreLeases[leaseID]; tracked {
			if _, err := c.coreDemand.Renew(ctx, leaseID, coreSupervisionLeaseTTL); err == nil {
				continue
			} else {
				// An expired lease cannot be revived by renew. Acquire below
				// re-establishes the same stable identity when possible.
				renewErr = err
			}
		}

		metadata, _ := json.Marshal(map[string]string{
			"source":  "core.seed",
			"purpose": "autoheal supervision",
		})
		if _, err := c.coreDemand.Acquire(ctx, demand.AcquireRequest{
			LeaseID:    leaseID,
			Scenario:   scenario,
			ConsumerID: coreSupervisionConsumerID,
			Kind:       demand.KindCore,
			RequestID:  coreSupervisionRequestID,
			Metadata:   string(metadata),
			TTL:        coreSupervisionLeaseTTL,
		}); err != nil {
			if renewErr != nil {
				errorsOut = append(errorsOut, fmt.Sprintf("renew core lease %s for %s: %v; reacquire: %v", leaseID, scenario, renewErr, err))
			} else {
				errorsOut = append(errorsOut, fmt.Sprintf("acquire core lease %s for %s: %v", leaseID, scenario, err))
			}
			continue
		}
		c.coreLeases[leaseID] = scenario
	}

	if degraded {
		return errorsOut
	}
	staleIDs := make([]string, 0, len(c.coreLeases))
	for leaseID := range c.coreLeases {
		if _, keep := desired[leaseID]; !keep {
			staleIDs = append(staleIDs, leaseID)
		}
	}
	sort.Strings(staleIDs)
	for _, leaseID := range staleIDs {
		scenario := c.coreLeases[leaseID]
		if _, err := c.coreDemand.Release(ctx, leaseID, "canonical supervision member removed"); err != nil {
			errorsOut = append(errorsOut, fmt.Sprintf("release core lease %s for %s: %v", leaseID, scenario, err))
			continue
		}
		delete(c.coreLeases, leaseID)
	}
	return errorsOut
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
	if len(snapshot.DemandErrors) > 0 {
		result.Details["demandLeaseErrors"] = append([]string(nil), snapshot.DemandErrors...)
	}
	if snapshot.Degraded {
		result.Status = checks.StatusWarning
		result.Message = "canonical supervision set is unavailable; retaining last-known-good checks"
		result.Details["sourceError"] = snapshot.SourceErr
		return result
	}
	if len(snapshot.DemandErrors) > 0 {
		result.Status = checks.StatusWarning
		result.Message = "canonical supervision set loaded; core demand leases need attention"
		return result
	}
	result.Status = checks.StatusOK
	result.Message = "canonical supervision set loaded and active checks reconciled"
	return result
}
