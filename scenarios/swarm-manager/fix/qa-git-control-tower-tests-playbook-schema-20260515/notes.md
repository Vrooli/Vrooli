## Problem

Git Control Tower readiness is yellow because the git-control-tower test dimension failed one BAS playbook. Fresh GCT job `6bbb52ab-0555-4f4b-a3f8-f5e78870cfc2` completed on 2026-05-15 with tests `total=11`, `passedCount=10`, `failedCount=1`.

The failed workflow is `scenarios/git-control-tower/bas/cases/01-health-check-endpoint/api/health.json`. Its `metadata.requirements` field is rejected by Browser Automation Studio / Vrooli Ascension request decoding with `proto: (line 1:196): unknown field "requirements"`.

## Top Violations

1. `scenarios/git-control-tower/bas/cases/01-health-check-endpoint/api/health.json` - playbook schema mismatch - `metadata.requirements` is not accepted by the current workflow request proto.
2. Same file - health-check playbook cannot execute, so `/health` assertions for service, readiness, dependencies.git, and dependencies.repository never run.

## Impact

GCT is the readiness gate runner used by Scenario QA. A broken self-test playbook means GCT cannot prove its own API health-check workflow, and downstream QA runs inherit weaker confidence in GCT test evidence.

## Reproduction

Run:

```sh
git-control-tower review run git-control-tower --details=10 --json
```

Observed evidence: job `6bbb52ab-0555-4f4b-a3f8-f5e78870cfc2`, readiness `yellow`, tests `10/11`, failure phase `playbooks`, error `unknown field "requirements"` for `bas/cases/01-health-check-endpoint/api/health.json`.

## Success Criteria

- The health-check BAS playbook conforms to the current workflow request schema.
- `git-control-tower review run git-control-tower --details=10 --json` reports tests `passed=true`, `total=11`, `passedCount=11`, `failedCount=0`.
- Standards remain green.

## Proposed Action

1. Inspect the current BAS workflow schema and remove, rename, or relocate `metadata.requirements` in the health-check playbook.
2. Re-run the single playbook or the full test suite to confirm the request is accepted.
3. Run the full GCT readiness command and verify the tests dimension turns green.
