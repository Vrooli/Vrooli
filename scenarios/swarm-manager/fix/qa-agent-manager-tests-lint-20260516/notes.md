## GCT Evidence

Scenario: agent-manager
Review job: ac94dc89-88cd-41e8-8bd9-1056f2064612
Timestamp: 2026-05-16T04:03:22Z
Readiness: yellow
Checks: rules=completed, tests=completed, tidiness=skipped

Tests dimension:
- available=true
- passed=false
- total=11
- passedCount=10
- failedCount=1
- lastRun=2026-05-16T04:03:22.396317053Z

Failure:
- phase=lint
- error=lint validation failed
- classification=misconfiguration
- remediation=Fix lint/type issues and configure lint handlers for unmatched components before proceeding.

## Remediation Target

Resolve lint/type validation failures and any missing lint handlers for unmatched components.

## Verification

Run git-control-tower review run agent-manager --json. Tests should report passed=true, total=11, passedCount=11, failedCount=0.