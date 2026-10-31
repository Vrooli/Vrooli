package main

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	eligibilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/eligibility"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
)

func TestManifestCoversTestGenieProtoSurface(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, manifestBytes, runsv1.File_test_genie_v1_runs_runs_proto, "RunsService")
	cliapp.RequireProtoServiceCoverage(t, manifestBytes, eligibilityv1.File_test_genie_v1_eligibility_eligibility_proto, "EligibilityService")
	cliapp.RequireProtoServiceCoverage(t, manifestBytes, validationv1.File_test_genie_v1_validation_validation_proto, "ValidationService")
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

func TestEvidenceProductionIsGovernedAndConfirmationRequired(t *testing.T) {
	m, err := cliapp.ParseManifest(manifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range m.Groups {
		if group.Name != "validation" {
			continue
		}
		for _, command := range group.Commands {
			if command.Name == "produce-evidence" {
				g := command.Governance
				if !g.RunEligible || g.Effect != "destructive" || g.RequiresConfirmation == nil || !*g.RequiresConfirmation {
					t.Fatalf("producer must be callable only as a confirmation-required destructive binding: %+v", g)
				}
				return
			}
		}
	}
	t.Fatal("missing validation produce-evidence binding")
}

func TestEvidenceProductionArgumentsResolveToTypedRequestFields(t *testing.T) {
	m, err := cliapp.ParseManifest(manifestBytes)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	request := (&validationv1.CreateEvidenceProductionRequest{}).ProtoReflect().Descriptor()
	want := map[string]string{
		"idempotency-key":             "idempotency_key",
		"provider":                    "provider",
		"producer":                    "producer",
		"candidate-scenario":          "candidate_scenario",
		"expected-candidate-identity": "expected_candidate_identity",
		"caller-scenario":             "caller_scenario",
		"caller-execution-id":         "caller_execution_id",
		"plan-id":                     "plan_id",
	}
	for _, group := range m.Groups {
		if group.Name != "validation" {
			continue
		}
		for _, command := range group.Commands {
			if command.Name != "produce-evidence" {
				continue
			}
			schema, err := cliapp.ManifestArgs(command)
			if err != nil {
				t.Fatalf("resolve producer argument schema: %v", err)
			}
			for argument, field := range want {
				resolved, err := cliapp.ResolveArgField(request, argument, schema)
				if err != nil {
					t.Fatalf("program-facing argument %q is not resolvable: %v", argument, err)
				}
				if len(resolved.Path) != 1 || string(resolved.Path[0].Name()) != field {
					t.Errorf("argument %q resolves to %v, want request.%s", argument, resolved.Path, field)
				}
			}
			for _, flag := range command.Flags {
				if flag.Name == "request-file" && (!flag.LocalOnly || flag.Required || flag.Bind != nil) {
					t.Errorf("request-file must remain an optional CLI-only input: %+v", flag)
				}
			}
			return
		}
	}
	t.Fatal("missing validation produce-evidence command")
}
