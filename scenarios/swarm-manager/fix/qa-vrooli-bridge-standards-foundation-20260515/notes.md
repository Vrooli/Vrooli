# QA Evidence

Source: GCT readiness job `c2ef8e70-439f-441d-9ab2-96ed71b3c5e1` for `vrooli-bridge`, completed 2026-05-15T22:02:40Z. Overall readiness: red. Checks: rules=completed, tests=failed, tidiness=skipped.

Standards dimension: available=true, blockingViolations=4, warnings=9, totalViolations=13. Top critical findings:
- `Makefile`: Scenario Required Structure, recommendation: add required resource at Makefile.
- `ui/package.json`: Missing @vrooli/api-base, recommendation: run `pnpm add @vrooli/api-base` in ui/.
- `ui/package.json`: Missing @vrooli/iframe-bridge, recommendation: run `pnpm add @vrooli/iframe-bridge` in ui/.
- `requirements/index.json`: P0 target missing requirements, recommendation: link each P0/P1 operational target to at least one requirement before publishing.

Tests check reported failed but tests total=0, passedCount=0, failedCount=0. This heartbeat treated that as non-actionable for target-scenario backlog because the same zero-count failed-test symptom is already under GCT bug-investigation history.

Success target: rerun `git-control-tower review run vrooli-bridge --details=10 --json`; standards blockingViolations=0.