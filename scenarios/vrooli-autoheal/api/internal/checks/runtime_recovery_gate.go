package checks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// Hold bounds per class (see GateVerdict). Host pressure and a live starter are
// ordinary reasons to wait a few minutes; neither may hold a failing core
// service for hours. An unreadable gate or registry must not hold a repair for
// longer than it takes to notice (R6: a repair never routes through what it
// repairs).
const (
	gateEpochHold      = 10 * time.Minute
	gateStartingHold   = 10 * time.Minute
	gateMaintenanceMax = 10 * time.Minute
	gateUnreadableHold = 2 * time.Minute
	gateSchemaOverride = 2 * time.Minute
)

func epochVerdict(reason string) GateVerdict {
	return GateVerdict{Class: GateClassEpoch, Reason: reason, CriticalAfter: gateEpochHold, OverrideAfter: gateEpochHold}
}

func unreadableVerdict(reason string) GateVerdict {
	return GateVerdict{Class: GateClassUnreadable, Reason: reason, CriticalAfter: gateUnreadableHold, OverrideAfter: gateUnreadableHold}
}

// registryErrorVerdict classifies a failure to open or read the runtime
// registry. A registry newer than this binary is critical at once and names the
// fix; every unreadable-registry case is overridden after a short bound.
func registryErrorVerdict(err error) GateVerdict {
	var schema *scenarioruntime.SchemaCompatibilityError
	if errors.As(err, &schema) {
		return GateVerdict{
			Class: GateClassSchema, CriticalAfter: 0, OverrideAfter: gateSchemaOverride,
			Reason: fmt.Sprintf("runtime registry schema_version %d > supported %d: this vrooli-autoheal binary is older than the control plane; rebuild it with `vrooli scenario restart vrooli-autoheal`", schema.DatabaseVersion, schema.BinaryVersion),
		}
	}
	return unreadableVerdict(fmt.Sprintf("runtime recovery ownership unavailable: %v", err))
}

// RuntimeRecoveryGate reads the runtime registry's durable pressure epoch and
// the coordinated owner's admission fence. Denials are bounded deferrals that
// stay visible on the check (see GateVerdict), so losing coordination neither
// creates two recovery controllers nor blocks repair indefinitely.
type RuntimeRecoveryGate struct {
	HomeDir string
	// CoordinatedScenario identifies the lifecycle owner whose maintenance and
	// startup state must be observed before autoheal can restart it. Keeping the
	// owner configurable makes this gate reusable for another control-plane
	// owner without scattering scenario-name checks through the registry.
	CoordinatedScenario string
}

func (g RuntimeRecoveryGate) AllowsAutoHealRestart(ctx context.Context, checkID, actionID string) GateVerdict {
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: g.HomeDir})
	if err != nil {
		return registryErrorVerdict(err)
	}
	defer store.Close()
	epochs, err := store.ListPressureEpochs(ctx, 1)
	if err != nil {
		return registryErrorVerdict(err)
	}
	if len(epochs) > 0 {
		epoch := epochs[0]
		if epoch.Status == scenarioruntime.PressureEpochDetected || epoch.Status == scenarioruntime.PressureEpochRegressed || epoch.Status == scenarioruntime.PressureEpochGated {
			return epochVerdict(fmt.Sprintf("runtime pressure epoch %s owns recovery (%s since %s)", epoch.EpochID, epoch.Status, epoch.UpdatedAt.UTC().Format(time.RFC3339)))
		}
	}
	if _, gatedAction := gatedActionIDs[strings.TrimSpace(actionID)]; gatedAction {
		if verdict := g.allowsCoordinatedOwnerRestart(ctx, store, checkID); !verdict.Allowed {
			return verdict
		}
	}
	return GateAllowed
}

// allowsCoordinatedOwnerRestart observes the owner lifecycle record immediately
// before autoheal invokes a restart. A health result can be stale while the
// owner is still starting or while its maintenance fence is closed. In either
// case, launching another restart would interrupt the coordinated operation.
// An absent owner instance remains eligible for ordinary post-release repair.
func (g RuntimeRecoveryGate) allowsCoordinatedOwnerRestart(ctx context.Context, store *scenarioruntime.SQLiteStore, checkID string) GateVerdict {
	owner := strings.TrimSpace(g.CoordinatedScenario)
	if owner == "" {
		owner = "agent-manager"
	}
	if strings.TrimPrefix(strings.TrimSpace(checkID), "scenario-") != owner {
		return GateAllowed
	}
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{
		Scenario: owner, Variant: scenarioruntime.DefaultVariant, Statuses: scenarioruntime.ActiveInstanceStatuses(),
	})
	if err != nil {
		return registryErrorVerdict(fmt.Errorf("coordinated owner startup state: %w", err))
	}
	for _, instance := range instances {
		now := time.Now()
		if instance.Status == scenarioruntime.StatusStarting {
			// A live starter owns the transition. An expired lease whose recorded
			// starter is provably gone is ordinary stale state; let the normal
			// lifecycle repair path recover it instead of suppressing forever.
			if instance.HeartbeatDeadlineAt == nil || instance.HeartbeatDeadlineAt.After(now) || instance.OwnerPID == nil || *instance.OwnerPID <= 0 || process.IsPIDRunning(*instance.OwnerPID) {
				return GateVerdict{Class: GateClassStarting, Reason: fmt.Sprintf("coordinated owner %s is still starting", owner), CriticalAfter: gateStartingHold, OverrideAfter: gateStartingHold}
			}
		}
		if instance.Status == scenarioruntime.StatusRunning && instance.HeartbeatDeadlineAt != nil && !instance.HeartbeatDeadlineAt.After(now) && instance.OwnerPID != nil && *instance.OwnerPID > 0 && !process.IsPIDRunning(*instance.OwnerPID) {
			// A running row with an expired lease and a PID proven dead is
			// authoritative stale state. Do not let it suppress recovery forever
			// when its old API endpoint cannot be read.
			continue
		}
		claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{
			InstanceID: instance.InstanceID, Statuses: []string{scenarioruntime.ClaimStatusBound},
		})
		if err != nil {
			return registryErrorVerdict(fmt.Errorf("coordinated owner maintenance state: %w", err))
		}
		for _, claim := range claims {
			if !strings.EqualFold(strings.TrimSpace(claim.PortName), "API_PORT") && !strings.EqualFold(strings.TrimSpace(claim.EnvVar), "API_PORT") {
				continue
			}
			admission, status, readErr := readOwnerAdmission(ctx, claim.Port)
			if readErr != nil {
				// The owner's own admission endpoint is the thing being repaired. A
				// hung or dead owner cannot answer, so an unreadable fence is a
				// short deferral and then a restart, never a standing denial.
				return unreadableVerdict(fmt.Sprintf("coordinated owner %s admission unreadable (HTTP %d): %v", owner, status, readErr))
			}
			if *admission.Closed {
				return GateVerdict{Class: GateClassMaintenance, Reason: fmt.Sprintf("coordinated owner %s maintenance is closed", owner), CriticalAfter: gateMaintenanceMax}
			}
		}
	}
	return GateAllowed
}

type ownerAdmissionProjection struct {
	// A pointer distinguishes an explicit false from a missing field. The
	// admission endpoint is an ownership contract, so an incomplete or
	// wrong-service 200 response must fail closed rather than grant admission.
	Closed *bool `json:"closed"`
}

func readOwnerAdmission(ctx context.Context, port int) (ownerAdmissionProjection, int, error) {
	var projection ownerAdmissionProjection
	if port <= 0 {
		return projection, 0, fmt.Errorf("owner API port is invalid")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/v1/maintenance/admission?view=fence", port), nil)
	if err != nil {
		return projection, 0, err
	}
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.Transport.(*http.Transport).CloseIdleConnections()
	response, err := client.Do(req)
	if err != nil {
		return projection, 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return projection, response.StatusCode, fmt.Errorf("owner admission returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return projection, response.StatusCode, err
	}
	if err := json.Unmarshal(body, &projection); err != nil {
		return projection, response.StatusCode, err
	}
	if projection.Closed == nil {
		return projection, response.StatusCode, fmt.Errorf("owner admission omitted explicit boolean closed")
	}
	return projection, response.StatusCode, nil
}
