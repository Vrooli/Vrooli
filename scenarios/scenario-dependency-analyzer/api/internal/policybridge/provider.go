// Package policybridge publishes SDA's governed mutation capability as an
// agent-policy provider snapshot. The runtime owns all rollout semantics.
package policybridge

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type SnapshotSink interface {
	PublishProviderSnapshot(context.Context, []byte) error
}

func BuildSnapshot(now time.Time) ([]byte, error) {
	if now.IsZero() {
		return nil, errors.New("snapshot time is required")
	}
	return json.MarshalIndent(map[string]any{
		"provider_id": "scenario-dependency-analyzer", "version": "agent-policy/v1",
		"capabilities":   []map[string]any{{"id": "governed-dependency-mutation", "ideal_posture": "all package mutations use typed argv and frozen reproduction", "declared_maturity": "advisory", "supports_analysis": true, "supports_enforcement": true, "supports_repair": true}},
		"scope":          map[string]any{},
		"health":         map[string]any{"state": "healthy", "checked_at": now.UTC(), "expires_at": now.UTC().Add(time.Hour), "message": "typed install gateway available"},
		"readiness":      map[string]any{"state": "ready", "rollback_plan": "withdraw SDA snapshot and retain prior bundle"},
		"evidence_state": "clean", "captured_at": now.UTC(), "expires_at": now.UTC().Add(time.Hour),
		"rules":      []map[string]any{{"risk": "dependency_addition", "action": "route", "reason": "route package mutation through Scenario Dependency Analyzer"}, {"risk": "dependency_upgrade", "action": "route", "reason": "route package mutation through Scenario Dependency Analyzer"}, {"risk": "dependency_removal", "action": "route", "reason": "route package mutation through Scenario Dependency Analyzer"}, {"risk": "lifecycle_execution", "action": "deny", "reason": "lifecycle execution requires explicit governed reproduction"}},
		"provenance": map[string]string{"provider": "scenario-dependency-analyzer", "argv_only": "true", "scripts_disabled_by_default": "true"},
	}, "", "  ")
}

func Publish(ctx context.Context, sink SnapshotSink, now time.Time) error {
	if sink == nil {
		return errors.New("snapshot sink is required")
	}
	data, err := BuildSnapshot(now)
	if err != nil {
		return err
	}
	return sink.PublishProviderSnapshot(ctx, data)
}
