package vps

import (
	"encoding/json"
	"strings"
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
)

// TestDataScheduleCompilesEffectfulTargetVerb [REQ:STC-P0-029] verifies the
// reviewed schedule is routed as an effectful target-local command.
func TestDataScheduleCompilesEffectfulTargetVerb(t *testing.T) {
	policy := `{"destination_name":"lpbs-r2","schedule":"24h","keep_daily":7,"keep_weekly":4,"keep_monthly":3,"recovery_drill_schedule":"168h"}`
	action := execplan.Action{ID: execplan.OpDataSchedule, OwnerOperation: execplan.OpDataSchedule, Inputs: map[string]string{
		"scenario": "landing-page-business-suite", "policy": policy,
		"binding_specs": "accounts-db:database:lpbs:postgres:scenario;uploads:dir:uploads:landing-page-business-suite:scenario",
	}}
	commands, err := ActionCommands(action, CommandContext{DeploymentID: "lpbs-stage", Identity: Identity{OperationID: "op-4", Fence: 8}, Manifest: domain.CloudManifest{Target: domain.ManifestTarget{VPS: &domain.ManifestVPS{Workdir: "/srv/Vrooli"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].Command.Verb != "cloud-target data schedule" || !commands[0].Command.Effectful || commands[0].Command.RequiredScope != "vrooli:write" {
		t.Fatalf("commands=%+v", commands)
	}
	argv := strings.Join(commands[0].Command.Args, " ")
	if !strings.Contains(argv, "--deployment lpbs-stage") || !strings.Contains(argv, "--scenario landing-page-business-suite") || !strings.Contains(argv, "--binding") || !strings.Contains(argv, "--policy b64:") {
		t.Fatalf("argv=%s", argv)
	}
	var got map[string]any
	for i, arg := range commands[0].Command.Args {
		if arg == "--policy" {
			raw, err := DecodeJSONArg(commands[0].Command.Args[i+1])
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if got["destination_name"] != "lpbs-r2" {
		t.Fatalf("policy=%v", got)
	}
}
