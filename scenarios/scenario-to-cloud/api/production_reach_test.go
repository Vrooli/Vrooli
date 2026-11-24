package main

import (
	"context"
	"testing"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

func TestProductionReachDoesNotRouteSSHBoundTargets(t *testing.T) {
	router := newProductionTargetReach(nil)
	if router.SSH != nil {
		t.Fatal("production reach must not install an SSH adapter")
	}
	target := identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "host.example"}}
	_, err := router.Exec(context.Background(), target, reach.Command{Verb: "cloud-target release activate", RequiredScope: "vrooli:write", Effectful: true})
	if !reach.IsKind(err, reach.KindUnavailable) {
		t.Fatalf("SSH-bound effect error = %v, want fail-closed reach_unavailable", err)
	}
}
