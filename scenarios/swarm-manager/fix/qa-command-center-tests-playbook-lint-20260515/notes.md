# QA Evidence

Source: GCT readiness job `0065e5f4-962f-40b7-9bad-8ad6df2240f1` for `command-center`, completed 2026-05-15T22:04:52Z. Overall readiness: yellow.

Tests dimension: available=true, passed=false, total=11, passedCount=8, failedCount=3, lastRun=2026-05-15T22:04:48.15941703Z. Failures:
- standards: standards violations exceed fail_on=high, highest=critical. Covered by companion item `fix/qa-command-center-standards-preflight-requirements-20260515`.
- lint: lint validation failed.
- playbooks: `bas/cases/01-foundation/broadcast-loads.json` failed during execute. Browser Automation Studio returned HTTP 400 INVALID_REQUEST with proto unknown field `expectedValue`.

Success target: rerun `git-control-tower review run command-center --details=10 --json`; tests passed=true with total=11, passedCount=11, failedCount=0, and the BAS playbook payload is accepted.