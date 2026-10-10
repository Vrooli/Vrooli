package main

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	installation "github.com/vrooli/vrooli/packages/finiteinstallation"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
	"os"
	"prompt-manager/handlers/heartbeat"
	"strings"
)

// Startup selection requests a fixed protected owner bundle; a flag establishes
// no authority. Default remains inert; descriptor, identities and live owners
// independently refuse before service graph effects.
func loadProtectedFiniteOwnerStartup(ctx context.Context) (*heartbeat.FiniteOwnerStartup, error) {
	phase := strings.ToLower(strings.TrimSpace(os.Getenv("VROOLI_FINITE_OWNER_PHASE")))
	if phase == "" || phase == "off" {
		return nil, nil
	}
	if phase != "commission" && phase != "live" {
		return nil, effortauthority.ErrRefused
	}
	enabled := strings.ToLower(strings.TrimSpace(os.Getenv("VROOLI_FINITE_ENABLED")))
	switch enabled {
	case "", "0", "false":
		if phase == "live" {
			return nil, effortauthority.ErrRefused
		}
	case "1", "true":
		if phase != "live" {
			return nil, effortauthority.ErrRefused
		}
	default:
		return nil, effortauthority.ErrRefused
	}
	plan, e := installation.Load("pm")
	if e != nil {
		return nil, e
	}
	bundle, e := plan.Snapshot()
	if e != nil {
		return nil, e
	}
	if bundle.Installation.Enabled != (phase == "live") {
		return nil, effortauthority.ErrRefused
	}
	handles, e := plan.AcquireExistingHandles()
	if e != nil {
		return nil, e
	}
	var witness *isolation.Witness
	if phase == "live" {
		witness, e = plan.CaptureContinuing(ctx)
		if e != nil {
			return nil, e
		}
	}
	var broker *effortauthority.BrokerClient
	if phase == "live" {
		broker = handles.Broker
	}
	return heartbeat.PrepareFiniteOwnerStartup(ctx, bundle.Installation, handles.Acceptance, handles.Profiles, broker, witness, handles.Publication, bundle.ScopeGrants)
}
