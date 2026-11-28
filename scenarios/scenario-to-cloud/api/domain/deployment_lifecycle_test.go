package domain

import (
	"errors"
	"testing"
)

func TestDeploymentTransitionStatus(t *testing.T) {
	states := []DeploymentStatus{StatusPending, StatusSetupRunning, StatusSetupComplete, StatusDeploying, StatusDeployed, StatusFailed, StatusStopped}
	allowed := map[[2]DeploymentStatus]bool{
		{StatusPending, StatusSetupRunning}:       true,
		{StatusPending, StatusDeploying}:          true,
		{StatusPending, StatusFailed}:             true,
		{StatusPending, StatusStopped}:            true,
		{StatusSetupRunning, StatusSetupComplete}: true,
		{StatusSetupRunning, StatusDeploying}:     true,
		{StatusSetupRunning, StatusDeployed}:      true,
		{StatusSetupRunning, StatusFailed}:        true,
		{StatusSetupRunning, StatusStopped}:       true,
		{StatusSetupRunning, StatusPending}:       true,
		{StatusSetupComplete, StatusSetupRunning}: true,
		{StatusSetupComplete, StatusDeploying}:    true,
		{StatusSetupComplete, StatusFailed}:       true,
		{StatusSetupComplete, StatusStopped}:      true,
		{StatusSetupComplete, StatusPending}:      true,
		{StatusDeploying, StatusDeployed}:         true,
		{StatusDeploying, StatusFailed}:           true,
		{StatusDeploying, StatusStopped}:          true,
		{StatusDeployed, StatusSetupRunning}:      true,
		{StatusDeployed, StatusDeploying}:         true,
		{StatusDeployed, StatusStopped}:           true,
		{StatusDeployed, StatusPending}:           true,
		{StatusFailed, StatusSetupRunning}:        true,
		{StatusFailed, StatusDeploying}:           true,
		{StatusFailed, StatusStopped}:             true,
		{StatusFailed, StatusPending}:             true,
		{StatusStopped, StatusSetupRunning}:       true,
		{StatusStopped, StatusDeploying}:          true,
		{StatusStopped, StatusPending}:            true,
	}
	for _, from := range states {
		if !from.Valid() {
			t.Errorf("declared state %q is invalid", from)
		}
		for _, to := range states {
			deployment := &Deployment{Status: from}
			err := deployment.TransitionStatus(to)
			wantAllowed := from == to || allowed[[2]DeploymentStatus{from, to}]
			if wantAllowed {
				if err != nil || deployment.Status != to {
					t.Errorf("%s -> %s error=%v state=%s", from, to, err, deployment.Status)
				}
				continue
			}
			if !errors.Is(err, ErrInvalidDeploymentTransition) || deployment.Status != from {
				t.Errorf("invalid %s -> %s changed state or returned wrong error: status=%q err=%v", from, to, deployment.Status, err)
			}
		}
	}
	for _, invalid := range []DeploymentStatus{"", "unknown"} {
		if invalid.Valid() || invalid.CanTransitionTo(StatusPending) || StatusPending.CanTransitionTo(invalid) {
			t.Errorf("invalid state %q was accepted", invalid)
		}
	}
}

func TestDeploymentTransitionRejectsNilReceiver(t *testing.T) {
	var deployment *Deployment
	if err := deployment.TransitionStatus(StatusPending); !errors.Is(err, ErrInvalidDeploymentTransition) {
		t.Fatalf("nil receiver error=%v", err)
	}
}
