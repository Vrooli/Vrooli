package control

import (
	"context"
	"time"

	auditdomain "device-control/internal/audit"
	devicedomain "device-control/internal/devices"
	executiondomain "device-control/internal/execution"
	sessionsdomain "device-control/internal/sessions"
	"device-control/strategy"
)

type Device struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Kind         string                `json:"kind"`
	Serial       string                `json:"serial,omitempty"`
	Model        string                `json:"model,omitempty"`
	OSVersion    string                `json:"os_version,omitempty"`
	StrategyID   string                `json:"strategy_id"`
	Status       string                `json:"status"`
	Health       string                `json:"health,omitempty"`
	HealthReason string                `json:"health_reason,omitempty"`
	HostNodeID   string                `json:"host_node_id,omitempty"`
	Transport    string                `json:"transport,omitempty"`
	Capabilities []strategy.Capability `json:"capabilities"`
	ObservedAt   time.Time             `json:"observed_at"`
	FirstSeenAt  time.Time             `json:"first_seen_at,omitempty"`
	LastSeenAt   time.Time             `json:"last_seen_at,omitempty"`
}

func deviceFromRecord(record devicedomain.Record) Device {
	capabilities := make([]strategy.Capability, len(record.Capabilities))
	copy(capabilities, record.Capabilities)
	return Device{ID: record.ID, Name: record.Name, Kind: record.Kind, Serial: record.Serial, Model: record.Model, OSVersion: record.OSVersion, StrategyID: record.StrategyID, Status: record.Status, Health: record.Health, HealthReason: record.HealthReason, HostNodeID: record.HostNodeID, Transport: record.Transport, Capabilities: capabilities, ObservedAt: record.ObservedAt, FirstSeenAt: record.FirstSeenAt, LastSeenAt: record.LastSeenAt}
}

type (
	WebViewAttachment = strategy.WebViewEndpoint
	AttachedDevice    struct {
		ID, Name, HostNodeID, Kind, Transport, Serial, OSVersion, TrustState, Reachability, HealthReason string
	}
	AttachedReader interface {
		List(context.Context) ([]AttachedDevice, error)
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
	RunResult  = executiondomain.RunResult
	AgentRun   struct {
		ID, Goal, DeviceID, Actor, State, Skill string
		Result                                  RunResult `json:"result"`
		CreatedAt                               time.Time `json:"created_at"`
	}
)
