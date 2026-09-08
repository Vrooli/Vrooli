package control

import (
	"context"
	"strings"
	"time"

	auditdomain "device-control/internal/audit"
	devicedomain "device-control/internal/devices"
	executiondomain "device-control/internal/execution"
	identitydomain "device-control/internal/identity"
	sessionsdomain "device-control/internal/sessions"
	"device-control/strategy"
)

type Device struct {
	ID             string                         `json:"id"`
	IdentityKey    string                         `json:"identity_key,omitempty"`
	Claims         []identitydomain.IdentityClaim `json:"claims,omitempty"`
	IdentityReason string                         `json:"identity_reason,omitempty"`
	Name           string                         `json:"name"`
	Kind           string                         `json:"kind"`
	OnboardingKind string                         `json:"onboarding_kind,omitempty"`
	Serial         string                         `json:"serial,omitempty"`
	Endpoint       string                         `json:"endpoint,omitempty"`
	Model          string                         `json:"model,omitempty"`
	OSVersion      string                         `json:"os_version,omitempty"`
	StrategyID     string                         `json:"strategy_id"`
	Status         string                         `json:"status"`
	Health         string                         `json:"health,omitempty"`
	HealthReason   string                         `json:"health_reason,omitempty"`
	HostNodeID     string                         `json:"host_node_id,omitempty"`
	Transport      string                         `json:"transport,omitempty"`
	Capabilities   []strategy.Capability          `json:"capabilities"`
	Properties     []strategy.PropertyDescriptor  `json:"properties,omitempty"`
	Transports     []strategy.DeviceTransport     `json:"transports,omitempty"`
	ObservedAt     time.Time                      `json:"observed_at"`
	FirstSeenAt    time.Time                      `json:"first_seen_at,omitempty"`
	LastSeenAt     time.Time                      `json:"last_seen_at,omitempty"`
}

func deviceFromRecord(record devicedomain.Record) Device {
	capabilities := make([]strategy.Capability, len(record.Capabilities))
	copy(capabilities, record.Capabilities)
	kind := record.Kind
	// Retained identities can outlive the transport observation that created
	// them. Re-derive the stable kind from an unambiguous strategy/serial pair
	// so a disconnected emulator or desktop does not regress to a physical
	// device in the API after restart.
	if record.StrategyID == "android-adb" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(record.Serial)), "emulator-") {
		kind = "emulator"
	}
	if record.StrategyID == "host-desktop" {
		kind = "desktop"
	}
	onboardingKind := ""
	if record.StrategyID == "android-adb" {
		onboardingKind = "android"
	} else if record.StrategyID == "android-tv-remote" || record.StrategyID == "google-cast" {
		onboardingKind = "google-tv"
	}
	health, healthReason := devicedomain.AggregateHealth(record, time.Now().UTC(), 15*time.Minute)
	status := record.Status
	if status == "" || status == "available" {
		status = health
	}
	return Device{ID: record.ID, IdentityKey: record.IdentityKey, Claims: append([]identitydomain.IdentityClaim(nil), record.Claims...), IdentityReason: record.IdentityReason, Name: record.Name, Kind: kind, OnboardingKind: onboardingKind, Serial: record.Serial, Endpoint: record.Endpoint, Model: record.Model, OSVersion: record.OSVersion, StrategyID: record.StrategyID, Status: status, Health: health, HealthReason: healthReason, HostNodeID: record.HostNodeID, Transport: record.Transport, Capabilities: capabilities, Properties: append([]strategy.PropertyDescriptor(nil), record.Properties...), Transports: append([]strategy.DeviceTransport(nil), record.Transports...), ObservedAt: record.ObservedAt, FirstSeenAt: record.FirstSeenAt, LastSeenAt: record.LastSeenAt}
}

type (
	WebViewAttachment = strategy.WebViewEndpoint
	AttachedDevice    struct {
		ID, Name, HostNodeID, Kind, Transport, Serial, OSVersion, TrustState, Reachability, HealthReason string
	}
	AttachedReader interface {
		List(context.Context) ([]AttachedDevice, error)
	}
	AttachedRevocationReader interface {
		Get(context.Context, string) (AttachedDevice, error)
	}
)

type (
	Session    = sessionsdomain.Session
	Audit      = auditdomain.Record
	Step       = executiondomain.Step
	Flow       = executiondomain.Flow
	GapReport  = executiondomain.GapReport
	Chapter    = executiondomain.Chapter
	Resolution = executiondomain.Resolution
	Condition  = executiondomain.Condition
	RunBinding = executiondomain.RunBinding
	RunResult  = executiondomain.RunResult
	AgentRun   struct {
		ID, Goal, DeviceID, Actor, State, Skill string
		Result                                  RunResult `json:"result"`
		CreatedAt                               time.Time `json:"created_at"`
		DryRun                                  bool      `json:"dry_run,omitempty"`
		PlanningRole                            string    `json:"planning_role,omitempty"`
		PromptHash                              string    `json:"prompt_hash,omitempty"`
		PolicyHash                              string    `json:"policy_hash,omitempty"`
		PlanHashes                              []string  `json:"plan_hashes,omitempty"`
		PromotedFlowID                          string    `json:"promoted_flow_id,omitempty"`
		PlannedSteps                            []Step    `json:"-"`
	}
)
