### Scenarios reviewed
Reviewed 3 scenarios with fresh GCT runs:
- `landing-page-business-suite`: red, job `4e091425-06c6-42f8-8f20-e35200e14d1d`
- `prompt-manager`: red, job `ddb5b39d-6974-4fdb-aef7-2050f06c8613`
- `swarm-manager`: red, job `f9d0b0b6-8493-4fc9-81fb-4d5330c8e696`

### Findings converted to backlog
No duplicate backlog items created. Updated existing Swarm Manager items:
- `fix/qa-lpbs-standards-20260514`
- `fix/qa-lpbs-tests-playbooks-20260514`
- `fix/qa-prompt-manager-standards-20260514`
- `fix/qa-swarm-manager-standards-20260408`
- `fix/qa-swarm-manager-tests-20260408`

### Dependencies wired
Wired QA gate dependencies onto 17 active direct target-scenario backlog items and repaired the initial failed bulk-loop dependency update. Remaining broad dependency sweep is incomplete; next run should continue from active `acceptance_allow` matches for `landing-page-business-suite`, `prompt-manager`, and `swarm-manager`.

### Skipped scenarios
None selected were skipped. Review queue file had no queued scenario IDs, so targets were selected from current QA/backlog context.

### Bugs filed
Filed `bug-inbox/unknown/gct-test-capability-inconsistent` as `knw-1778796308148709144`: GCT run output lost LPBS test failure detail and skipped prompt-manager/swarm-manager tests due `capabilities.test-genie=false`.

### Knowledge entries written
- `qa-run/landing-page-business-suite`: `knw-1778796495116743853`
- `qa-run/prompt-manager`: `knw-1778796495116903442`
- `qa-run/swarm-manager`: `knw-1778796495250980180`
- `reviewed-scenario/landing-page-business-suite`: `knw-1778796514951570351`
- `reviewed-scenario/prompt-manager`: `knw-1778796514950283774`
- `reviewed-scenario/swarm-manager`: `knw-1778796515148080094`
- `dependency-wiring/2026-05-14-gct-readiness`: `knw-1778796515148124364`

Operational friction to remember: heartbeat docs still show `prompt-manager team knowledge-add --by=...`, but CLI now rejects `--by`; use runtime attribution plus `--caller-note`.