## GCT Evidence

Scenario: agent-inbox
Review job: 6d6c4951-f25e-4d05-8564-5d84f5631be6
Timestamp: 2026-05-16T04:05:04Z
Readiness: yellow
Checks: rules=completed, tests=completed, tidiness=skipped

Tests dimension:
- available=true
- passed=false
- total=11
- passedCount=8
- failedCount=3
- lastRun=2026-05-16T04:05:02.875061114Z

Failures:
- standards: standards violations exceed fail_on=high, highest=critical. Covered by fix/qa-agent-inbox-standards-storage-20260516.
- performance: page home performance below threshold.
- playbooks: bas/cases/01-inbox-list/ui/read-unread-toggle.json failed in phase wait. Execution 34e47c99-b79b-445e-accc-4c42524ccea1, node assert-unread-indicator, step 8 assert, expected element to be visible.

## Remediation Target

Raise home page performance above configured threshold and make the read-unread toggle workflow expose the unread indicator expected by the BAS playbook.

## Verification

Run git-control-tower review run agent-inbox --json. Tests should report passed=true, total=11, passedCount=11, failedCount=0.