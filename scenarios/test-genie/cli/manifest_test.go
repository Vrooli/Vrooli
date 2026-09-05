package main

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	eligibilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/eligibility"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

func TestManifestCoversTestGenieProtoSurface(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, manifestBytes, runsv1.File_test_genie_v1_runs_runs_proto, "RunsService")
	cliapp.RequireProtoServiceCoverage(t, manifestBytes, eligibilityv1.File_test_genie_v1_eligibility_eligibility_proto, "EligibilityService")
}

func TestManifestDeclaresExecuteDurableException(t *testing.T) {
	m, err := cliapp.ParseManifest(manifestBytes)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	for _, e := range m.Exceptions {
		if e.Command == "execute" {
			if e.Class != string(cliapp.ExceptionDurableRun) {
				t.Fatalf("execute exception class = %q, want %q", e.Class, cliapp.ExceptionDurableRun)
			}
			if e.Reason == "" {
				t.Fatal("execute exception must carry a reason")
			}
			return
		}
	}
	t.Fatal("manifest does not declare execute as a durable_run exception")
}
