## GCT Evidence

Scenario: ecosystem-manager
Review job: 9d36d806-b28d-48ec-a6d0-47642e8ae5cf
Timestamp: 2026-05-16T10:02:33Z
Readiness: yellow
Checks: rules=completed, tests=completed, tidiness=skipped

Tests dimension:
- available=true
- passed=false
- total=11
- passedCount=8
- failedCount=3
- lastRun=2026-05-16T10:02:31.977368351Z

Failures:
- standards: standards violations exceed fail_on=high, highest=critical
- playbooks: registry not found at scenarios/ecosystem-manager/bas/registry.json
- business: no requirement modules found

## Remediation Target

Regenerate bas/registry.json and scaffold/link requirement modules. The standards failure is covered by fix/qa-ecosystem-manager-standards-structure-focus-20260516.

## Verification

Run git-control-tower review run ecosystem-manager --json. Tests should report passed=true, total=11, passedCount=11, failedCount=0.