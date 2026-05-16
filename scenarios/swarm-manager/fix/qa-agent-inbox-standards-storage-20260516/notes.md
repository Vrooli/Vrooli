## GCT Evidence

Scenario: agent-inbox
Review job: 6d6c4951-f25e-4d05-8564-5d84f5631be6
Timestamp: 2026-05-16T04:05:04Z
Readiness: yellow
Checks: rules=completed, tests=completed, tidiness=skipped

Standards dimension:
- available=true
- blockingViolations=1
- warnings=24
- totalViolations=25

Top violations:
- Makefile: Scenario Required Structure, severity=critical, recommendation: Add the required resource at Makefile.
- ui/src/components/layout/AppDialogs.tsx:96: Repo-local runtime storage, severity=high.
- ui/src/components/settings/Settings.tsx:26: Repo-local runtime storage, severity=high.
- ui/src/components/settings/Settings.tsx:211: Repo-local runtime storage, severity=high.
- ui/src/components/settings/useSettingsState.ts:11: Repo-local runtime storage, severity=high.

## Remediation Target

Restore required scenario structure and move mutable runtime storage off repo-local paths into github.com/vrooli/api-core/storage or declared durable resource-backed persistence in .vrooli/service.json.

## Verification

Run git-control-tower review run agent-inbox --json. Standards should report blockingViolations=0 and no high repo-local runtime storage findings.