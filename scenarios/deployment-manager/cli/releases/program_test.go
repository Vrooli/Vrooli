package releases

import (
	"bytes"
	"os/exec"
	"testing"
)

// Exercise the shipped program with owner-shaped responses, without making a
// release mutation. Program Runtime projects singular protobuf fields as meta.
func TestRecoveryProgramChecksOwnerReceipt(t *testing.T) {
	const probe = `
import json, runpy, sys

class Handle:
    def __init__(self, meta): self.value = meta
    def meta(self): return self.value

class Program:
    def __init__(self): self.calls = []
    def inputs(self): return {"review_key":"review", "release_id":"release", "candidate_id":"candidate", "destination_revision_id":"destination", "action":"halt", "execute":True, "confirmation":"halt release", "dry_run_reference":"preview"}
    def fail(self, status, klass, detail, where):
        envelope = self.env["envelope"]
        envelope["status"] = status
        envelope["errors"].append({"class":klass, "detail":str(detail), "where":where})
        return "report"
    def run(self, states, state):
        self.env = states[state].__globals__
        while state: state = states[state]()

class Review:
    def get(self, **kwargs): return Handle({"status":"approved"})

class Releases:
    def __init__(self, receipt): self.receipt = receipt; self.calls = []
    def recover(self, **kwargs):
        self.calls.append(kwargs)
        if kwargs["dry_run"]:
            preview = dict(self.receipt, dryRun=True, outcome="preview")
            if sys.argv[2] == "preview-mismatch": preview["candidateId"] = "other"
            return Handle({"dryRun":True, "receipt":preview})
        actual = dict(self.receipt)
        if sys.argv[2] == "mismatch": actual["candidateId"] = "other"
        return Handle({"receipt":actual})

class Deployment:
    def __init__(self, receipt): self.readiness_reviews = Review(); self.releases = Releases(receipt)

receipt = {"releaseId":"release", "candidateId":"candidate", "destinationRevisionId":"destination", "deploymentId":"deployment", "action":"halt", "outcome":"halted", "externalReceipt":"owner-1"}
program, deployment = Program(), Deployment(receipt)
runpy.run_path(sys.argv[1], init_globals={"program":program, "deployment_manager":deployment})
assert deployment.releases.calls[0]["_confirm"] is True and deployment.releases.calls[0]["dry_run"] is True
assert len(deployment.releases.calls) == (1 if sys.argv[2] == "preview-mismatch" else 2)
`
	for _, tc := range []struct {
		name   string
		status string
	}{
		{name: "valid", status: `"status": "ok"`},
		{name: "mismatch", status: `"class": "ambiguous_response"`},
		{name: "preview-mismatch", status: `"class": "ambiguous_response"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := exec.Command("python3", "-c", probe, "../../.vrooli/program-runtime/release-recover.py", tc.name).CombinedOutput()
			if err != nil {
				t.Fatalf("program probe: %v\n%s", err, out)
			}
			if !bytes.Contains(out, []byte(tc.status)) {
				t.Fatalf("program output %s does not contain %s", out, tc.status)
			}
		})
	}
}

func TestReadinessProgramRejectsNonStringTargets(t *testing.T) {
	const probe = `
import runpy, sys
class Program:
    def inputs(self): return {"scenario":"s", "profile_id":"p", "candidate_commit":"c", "artifact_digest":"d", "channel":"c", "targets":[1]}
    def fail(self, status, klass, detail, where):
        envelope = self.env["envelope"]
        envelope["status"] = status
        envelope["errors"].append({"class":klass, "detail":str(detail), "where":where})
        return "report"
    def run(self, states, state):
        self.env = states[state].__globals__
        while state: state = states[state]()
runpy.run_path(sys.argv[1], init_globals={"program":Program()})
`
	out, err := exec.Command("python3", "-c", probe, "../../.vrooli/program-runtime/readiness-review.py").CombinedOutput()
	if err != nil || !bytes.Contains(out, []byte(`"detail": "targets must be an array of strings"`)) {
		t.Fatalf("readiness program: %v\n%s", err, out)
	}
}
