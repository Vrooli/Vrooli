package deployments

import (
	"context"
	"strings"
	"testing"

	"deployment-manager/profiles"
	"deployment-manager/releases"
)

func TestResolveDesktopDestinationPlanSelectsExactLPBSBinding(t *testing.T) {
	revision := &releases.DestinationRevisionRecord{ID: "destination-staging", Revision: releases.DestinationRevision{
		Kind: "lpbs-channel", DestinationID: "staging-profile", ConfigurationDigest: "sha256:cfg", Channel: "beta", ExpectedChannelRevision: "rev-12",
	}}
	cfg := &profiles.LPBSReleaseConfig{LPBSAppKey: "aquila", LPBSRemoteProfile: "staging-profile", DefaultChannel: "beta", UpdateURL: "https://updates.example.test/aquila"}
	plan, err := resolveDesktopDestinationPlan(revision, cfg, "beta")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Kind != "lpbs" || plan.DestinationID != "staging-profile" || plan.RevisionID != "destination-staging" || plan.ConfigurationDigest != "sha256:cfg" || plan.Channel != "beta" || plan.ExpectedChannelRevision != "rev-12" || plan.LPBSAppKey != "aquila" || plan.LPBSRemoteProfile != "staging-profile" || plan.UpdateURL != cfg.UpdateURL {
		t.Fatalf("resolved plan = %+v", plan)
	}
}

func TestResolveDesktopDestinationPlanRejectsUnsupportedAndDriftedBindings(t *testing.T) {
	base := releases.DestinationRevisionRecord{ID: "destination-1", Revision: releases.DestinationRevision{Kind: "lpbs", DestinationID: "staging", ConfigurationDigest: "sha256:cfg", Channel: "stable"}}
	cfg := &profiles.LPBSReleaseConfig{LPBSAppKey: "app", LPBSRemoteProfile: "staging", DefaultChannel: "stable"}
	cases := []struct {
		name     string
		revision *releases.DestinationRevisionRecord
		config   *profiles.LPBSReleaseConfig
		channel  string
		want     string
	}{
		{name: "missing revision", want: "approved destination revision"},
		{name: "unsupported kind", revision: &releases.DestinationRevisionRecord{ID: "d", Revision: releases.DestinationRevision{Kind: "object-store", DestinationID: "staging", ConfigurationDigest: "sha256:cfg", Channel: "stable"}}, config: cfg, want: "no publisher adapter"},
		{name: "destination mismatch", revision: &base, config: &profiles.LPBSReleaseConfig{LPBSAppKey: "app", LPBSRemoteProfile: "other"}, want: "does not match"},
		{name: "channel mismatch", revision: &base, config: cfg, channel: "beta", want: "does not match approved destination channel"},
		{name: "missing config", revision: &base, want: "exact app key and remote profile"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveDesktopDestinationPlan(tc.revision, tc.config, tc.channel)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestUnsupportedDesktopDestinationNeverCallsOwnerPipeline(t *testing.T) {
	runner := &fakePublishPipelineRunner{}
	o := newOrch(nil, nil, &fakeLPBSConfigRepo{cfg: &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "app", LPBSRemoteProfile: "staging"}}, nil)
	o.releaseIdentity = &candidateIdentityFake{destination: &releases.DestinationRevisionRecord{ID: "destination-1", Revision: releases.DestinationRevision{
		Kind: "cloud", DestinationID: "staging", ConfigurationDigest: "sha256:cfg", Channel: "stable",
	}}}
	o.publishPipelineRunner = runner
	step := OrchestrationStep{Name: "Publish desktop artifacts"}
	response := &DeployDesktopResponse{}
	o.publishToDestination(context.Background(), &profiles.Profile{Scenario: "demo"}, DeployDesktopRequest{
		ProfileID: "p1", ReleaseID: "release-1", DestinationRevisionID: "destination-1", Channel: "stable",
	}, response, &step)
	if step.Status != "failed" || !strings.Contains(step.Error, "no publisher adapter") {
		t.Fatalf("step = %+v, want fail-closed unsupported destination", step)
	}
	if runner.calls != 0 {
		t.Fatalf("owner calls = %d, want 0", runner.calls)
	}
}
