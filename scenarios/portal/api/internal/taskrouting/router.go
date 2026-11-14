// Package taskrouting resolves Portal task requirements to an authorized
// provider route. It owns policy only: provider owners still own lifecycle,
// admission, and side effects.
package taskrouting

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/api-core/targetmodel"
)

// ProviderState is an observed provider state. It is intentionally separate
// from targetmodel.CapabilityState because lifecycle recovery is meaningful
// for a provider, while a capability fact alone cannot authorize it.
type ProviderState string

const (
	StateReady       ProviderState = "ready"
	StateAbsent      ProviderState = "absent"
	StateStopped     ProviderState = "stopped"
	StateUnhealthy   ProviderState = "unhealthy"
	StateUnsupported ProviderState = "unsupported"
	StateDenied      ProviderState = "denied"
	StateUnknown     ProviderState = "unknown"
	StateLost        ProviderState = "lost"
)

// TaskRequirements are the immutable facts that a route must preserve. A
// retry carries the same value, so recovery cannot silently change the task.
type TaskRequirements struct {
	ID                   string
	Target               targetmodel.TargetRef
	RequiredCapabilities []string
	AccountID            string
	DataResidency        string
	Authorization        string
	RequestedOutcome     string
	AcceptanceChecks     []string
}

// ProviderRoute is a provider owner's bounded route advertisement. Endpoint
// addresses and credentials deliberately do not appear here.
type ProviderRoute struct {
	ProviderID         string
	Version            string
	Target             targetmodel.TargetRef
	Capabilities       []string
	SupportedOutcomes  []string
	AcceptanceChecks   []string
	AccountID          string
	DataResidency      string
	Authorization      string
	State              ProviderState
	RecoveryAuthorized bool
}

type DecisionStatus string

const (
	StatusSelected    DecisionStatus = "selected"
	StatusUnavailable DecisionStatus = "unavailable"
	StatusRecoverable DecisionStatus = "recoverable"
	StatusRejected    DecisionStatus = "rejected"
	StatusRetry       DecisionStatus = "retry"
	StatusReconcile   DecisionStatus = "reconcile"
)

// Decision is safe to present in Portal status and is also suitable for a
// durable task-routing receipt. Route is populated only for a selected route.
type Decision struct {
	TaskID     string
	Status     DecisionStatus
	ReasonCode string
	NextAction string
	Route      *ProviderRoute
}

// Observe records screen or embedded-app text as data. It intentionally does
// not merge text into task requirements, route constraints, or authorization.
// A hostile surface can describe an action, but it cannot grant itself one.
func Observe(task TaskRequirements, content string) (TaskRequirements, string) {
	copy := task
	copy.RequiredCapabilities = append([]string(nil), task.RequiredCapabilities...)
	copy.AcceptanceChecks = append([]string(nil), task.AcceptanceChecks...)
	return copy, strings.TrimSpace(content)
}

// Lifecycle is implemented by the provider owner. Portal may request a
// recovery only after the route explicitly advertises that authority; it does
// not install packages or start arbitrary processes itself.
type Lifecycle interface {
	Start(context.Context, string) error
	Probe(context.Context, string) (ProviderRoute, error)
}

// Resolve evaluates all candidates deterministically and never silently
// changes a task's destination, account, residency, authority, outcome, or
// acceptance contract.
func Resolve(task TaskRequirements, routes []ProviderRoute) Decision {
	if err := validateTask(task); err != nil {
		return Decision{TaskID: task.ID, Status: StatusRejected, ReasonCode: "invalid_task", NextAction: err.Error()}
	}
	ordered := append([]ProviderRoute(nil), routes...)
	sort.SliceStable(ordered, func(i, j int) bool { return routeKey(ordered[i]) < routeKey(ordered[j]) })
	var stopped, absent, denied, unsupported, unhealthy, unknown, lost bool
	var inequivalent bool
	for _, route := range ordered {
		if err := validateRoute(route); err != nil {
			continue
		}
		if reason := constraintMismatch(task, route, nil); reason != "" {
			inequivalent = true
			continue
		}
		if !hasCapabilities(route.Capabilities, task.RequiredCapabilities) {
			continue
		}
		switch route.State {
		case StateReady:
			selected := route
			return Decision{TaskID: task.ID, Status: StatusSelected, ReasonCode: "route_ready", Route: &selected}
		case StateStopped:
			stopped = true
		case StateAbsent:
			absent = true
		case StateDenied:
			denied = true
		case StateUnsupported:
			unsupported = true
		case StateUnhealthy:
			unhealthy = true
		case StateUnknown:
			unknown = true
		case StateLost:
			lost = true
		}
	}
	status, reason, next := StatusUnavailable, "missing_capability", "provide the required provider and retry"
	switch {
	case stopped:
		status, reason, next = StatusRecoverable, "provider_stopped", "request owner lifecycle recovery and recheck readiness"
	case denied:
		status, reason, next = StatusRejected, "authority_denied", "request the missing provider authority"
	case unsupported:
		status, reason, next = StatusRejected, "provider_unsupported", "choose a supported provider or target"
	case unhealthy:
		status, reason, next = StatusRetry, "provider_unhealthy", "retain the task and retry after provider health recovers"
	case unknown:
		status, reason, next = StatusRetry, "provider_readiness_unknown", "probe provider readiness within the task deadline"
	case lost:
		status, reason, next = StatusReconcile, "provider_lost_after_effect", "reconcile the original receipt before retrying"
	case absent:
		status, reason, next = StatusUnavailable, "missing_capability", "install or attach the required provider, then retry"
	case inequivalent:
		status, reason, next = StatusRejected, "route_inequivalent", "choose a route preserving the exact target, account, residency, and authority"
	}
	return Decision{TaskID: task.ID, Status: status, ReasonCode: reason, NextAction: next}
}

// Recover asks the owner to start a stopped provider and then probes it. The
// refreshed route must still satisfy the original task before it is selected.
func Recover(ctx context.Context, task TaskRequirements, routes []ProviderRoute, lifecycle Lifecycle) (Decision, error) {
	if lifecycle == nil {
		return Decision{TaskID: task.ID, Status: StatusRejected, ReasonCode: "recovery_unavailable", NextAction: "use the provider owner's recovery action"}, nil
	}
	decision := Resolve(task, routes)
	if decision.Status != StatusRecoverable || decision.ReasonCode != "provider_stopped" {
		return decision, nil
	}
	for _, route := range routes {
		if route.State != StateStopped || !route.RecoveryAuthorized || constraintMismatch(task, route, nil) != "" || !hasCapabilities(route.Capabilities, task.RequiredCapabilities) {
			continue
		}
		if err := lifecycle.Start(ctx, route.ProviderID); err != nil {
			return Decision{TaskID: task.ID, Status: StatusRetry, ReasonCode: "provider_recovery_failed", NextAction: "retain the task and retry after the owner reports readiness"}, err
		}
		refreshed, err := lifecycle.Probe(ctx, route.ProviderID)
		if err != nil {
			return Decision{TaskID: task.ID, Status: StatusRetry, ReasonCode: "provider_probe_failed", NextAction: "retain the task and retry after the owner reports readiness"}, err
		}
		if refreshed.State != StateReady || constraintMismatch(task, refreshed, nil) != "" || !hasCapabilities(refreshed.Capabilities, task.RequiredCapabilities) {
			return Decision{TaskID: task.ID, Status: StatusRetry, ReasonCode: "provider_recovery_not_ready", NextAction: "retain the task until the owner reports an equivalent ready route"}, nil
		}
		return Decision{TaskID: task.ID, Status: StatusSelected, ReasonCode: "provider_recovered", Route: &refreshed}, nil
	}
	return decision, nil
}

// Equivalent reports whether candidate can replace original after a failure.
// Provider identity may change; task identity, target, account, residency,
// authority, outcome, and acceptance checks may not.
func Equivalent(task TaskRequirements, original, candidate ProviderRoute) (bool, string) {
	if err := validateTask(task); err != nil {
		return false, "invalid_task"
	}
	if err := validateRoute(original); err != nil {
		return false, "invalid_original_route"
	}
	if err := validateRoute(candidate); err != nil {
		return false, "invalid_candidate_route"
	}
	if reason := constraintMismatch(task, candidate, &original); reason != "" {
		return false, reason
	}
	return true, "equivalent_route"
}

func constraintMismatch(task TaskRequirements, route ProviderRoute, original *ProviderRoute) string {
	if task.Target.OwnerScenario != "" && route.Target != task.Target {
		return "target_mismatch"
	}
	if original != nil && route.Target != original.Target {
		return "target_mismatch"
	}
	if task.AccountID != "" && route.AccountID != task.AccountID {
		return "account_mismatch"
	}
	if task.DataResidency != "" && route.DataResidency != task.DataResidency {
		return "residency_mismatch"
	}
	if task.Authorization != "" && route.Authorization != task.Authorization {
		return "authority_mismatch"
	}
	if !containsExact(route.SupportedOutcomes, task.RequestedOutcome) {
		return "outcome_mismatch"
	}
	if !hasCapabilities(route.AcceptanceChecks, task.AcceptanceChecks) {
		return "acceptance_mismatch"
	}
	return ""
}

func validateTask(task TaskRequirements) error {
	if strings.TrimSpace(task.ID) == "" || len(task.ID) > 255 {
		return fmt.Errorf("task identity is required")
	}
	if task.Target.OwnerScenario != "" {
		if err := task.Target.Validate(); err != nil {
			return fmt.Errorf("task target: %w", err)
		}
	}
	if strings.TrimSpace(task.RequestedOutcome) == "" {
		return fmt.Errorf("requested outcome is required")
	}
	if len(task.RequiredCapabilities) > 64 || len(task.AcceptanceChecks) > 64 {
		return fmt.Errorf("task requirement list exceeds bound")
	}
	return nil
}

func validateRoute(route ProviderRoute) error {
	if strings.TrimSpace(route.ProviderID) == "" || len(route.ProviderID) > 128 || strings.TrimSpace(route.Version) == "" || len(route.Version) > 64 {
		return fmt.Errorf("provider identity is required")
	}
	if err := route.Target.Validate(); err != nil {
		return fmt.Errorf("route target: %w", err)
	}
	switch route.State {
	case StateReady, StateAbsent, StateStopped, StateUnhealthy, StateUnsupported, StateDenied, StateUnknown, StateLost:
		return nil
	default:
		return fmt.Errorf("invalid provider state")
	}
}

func hasCapabilities(offered, required []string) bool {
	for _, wanted := range required {
		found := false
		for _, have := range offered {
			if strings.EqualFold(strings.TrimSpace(wanted), strings.TrimSpace(have)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func containsExact(values []string, wanted string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == strings.TrimSpace(wanted) {
			return true
		}
	}
	return false
}

func routeKey(route ProviderRoute) string {
	return strings.Join([]string{route.ProviderID, route.Version, route.Target.OwnerScenario, route.Target.ResourceID, route.Target.HostNodeID}, "/")
}
