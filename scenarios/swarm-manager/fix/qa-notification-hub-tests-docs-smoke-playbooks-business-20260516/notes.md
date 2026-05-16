## GCT Evidence

Scenario: notification-hub
Review job: 65bd8ba3-9bf2-49b7-9d5e-e4e97f49ff3b
Timestamp: 2026-05-16T10:02:13Z
Readiness: yellow
Checks: rules=completed, tests=completed, tidiness=skipped

Tests dimension:
- available=true
- passed=false
- total=11
- passedCount=6
- failedCount=5
- lastRun=2026-05-16T10:02:07.156418043Z

Failures:
- standards: standards violations exceed fail_on=high, highest=critical
- docs: Docs validation failed
- smoke: ui smoke failed with HTTP 500 at http://127.0.0.1:21086/api/v1/admin/profiles
- playbooks: registry not found at scenarios/notification-hub/bas/registry.json
- business: no requirement modules found

## Remediation Target

Fix docs validation, repair the admin profiles smoke path, regenerate bas/registry.json, and scaffold/link requirement modules. The standards failure is covered by fix/qa-notification-hub-standards-structure-api-base-20260516.

## Verification

Run git-control-tower review run notification-hub --json. Tests should report passed=true, total=11, passedCount=11, failedCount=0.