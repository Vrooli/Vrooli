# QA Evidence

Source: GCT readiness job `1df564fe-514a-46ce-ac66-15b1c4eb6730` for `vrooli-emulator`, completed 2026-05-15T22:03:46Z. Overall readiness: yellow.

Tests dimension: available=true, passed=false, total=11, passedCount=7, failedCount=4, lastRun=2026-05-15T22:03:41.182532196Z. Failures:
- structure: schema validation failed for 1 file.
- standards: standards violations exceed fail_on=high, highest=critical. This is covered by companion item `fix/qa-vrooli-emulator-standards-requirements-20260515`.
- lint: lint validation failed.
- business: no requirement modules found; remediation says run `vrooli scenario requirements init` to scaffold P0/P1 modules.

Success target: rerun `git-control-tower review run vrooli-emulator --details=10 --json`; tests passed=true with total=11, passedCount=11, failedCount=0.