package main

import (
	"github.com/vrooli/api-core/effortauthority"
	installation "github.com/vrooli/vrooli/packages/finiteinstallation"
	"os"
	"strings"
	"swarm-manager/internal/backlog"
)

func loadProtectedFiniteReadPublication() (*backlog.FiniteReadPublication, error) {
	phase := strings.ToLower(strings.TrimSpace(os.Getenv("VROOLI_FINITE_OWNER_PHASE")))
	if phase == "" || phase == "off" {
		return nil, nil
	}
	if phase != "commission" {
		return nil, effortauthority.ErrRefused
	}
	enabled := strings.ToLower(strings.TrimSpace(os.Getenv("VROOLI_FINITE_ENABLED")))
	if enabled != "" && enabled != "0" && enabled != "false" {
		return nil, effortauthority.ErrRefused
	}
	plan, e := installation.Load("acceptance")
	if e != nil {
		return nil, e
	}
	bundle, e := plan.Snapshot()
	if e != nil {
		return nil, e
	}
	publication, e := plan.AcquirePublication()
	if e != nil {
		return nil, e
	}
	target := bundle.CommissionTarget
	return backlog.PrepareFiniteReadPublication(bundle.Installation, map[string]backlog.FiniteCommissionTarget{bundle.Installation.Policy.Effort: {WorkShape: backlog.FiniteCommissionWorkShape(target.WorkShape), Kind: backlog.BacklogKind(target.Kind), Name: target.Name}}, bundle.ScopeGrants, publication)
}
