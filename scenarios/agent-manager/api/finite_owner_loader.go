package main

import (
	"agent-manager/internal/orchestration"
	"context"
	"github.com/vrooli/api-core/effortauthority"
	installation "github.com/vrooli/vrooli/packages/finiteinstallation"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
	"os"
	"strings"
)

// Startup selection requests a fixed protected owner bundle; a flag establishes
// no authority. Default remains inert; descriptor, identities and live owners
// independently refuse before service graph effects.
func loadProtectedFiniteOwnerStartup(ctx context.Context) (*orchestration.FiniteOwnerStartup, error) {
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
	plan, e := installation.Load("am")
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
	var unitOwner *isolation.FixedUnitOwner
	if phase == "live" {
		broker = handles.Broker
		unitOwner, e = isolation.NewFixedUnitOwner(isolation.FixedUnitOwnerConfig{Policy: bundle.Installation.Policy, Manifest: bundle.Manifest, Continuing: bundle.Continuing, Witness: witness, Authority: broker, Peers: bundle.UnitPeers})
		if e != nil {
			return nil, e
		}
	}
	return orchestration.PrepareFiniteOwnerStartup(ctx, bundle.Installation, broker, bundle.Manifest, witness, unitOwner, bundle.ProfileGrants, handles.Publication)
}
