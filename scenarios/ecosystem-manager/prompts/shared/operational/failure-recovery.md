# Failure Recovery

## Purpose
Capture failures fast, choose the right response, and record what the next agent must know.

## Severity Snapshot
- **Level 1 – Trivial**: Cosmetic issues, non-blocking warnings, minor test flakes.
- **Level 2 – Minor**: Single feature broken, <20% performance dip, recoverable errors.
- **Level 3 – Major**: Core feature down, multiple failing tests, API contract break, data risk.
- **Level 4 – Critical**: Service will not start, data corruption, exposed vulnerability.
- **Level 5 – Catastrophic**: Active outage, data loss in progress, security breach.

## Response Decision Tree
```
Start
└─ Assess severity level (1-5)
    ├─ Levels 1-2 (Trivial/Minor)
    │     └─ Action: Keep iterating → apply targeted fix → rerun tests
    │            Documentation: Note quick fix in final summary (no formal failure log)
    ├─ Level 3 (Major)
    │     └─ Action: Stop feature work → stabilize scenario → verify regression removed
    │            Documentation: Add Failure Log (template below) + reference impacted PRD items
    └─ Levels 4-5 (Critical/Catastrophic)
          └─ Action: Stop immediately → follow Task Completion Protocol → do NOT attempt git rollback
                 Documentation: Complete Failure Log + highlight blockers + call out needed escalation
```

## Failure Log Template
Use when severity ≥3 or the issue remains unresolved at handoff.

```
### Failure Summary
- **Component**: [resource/scenario name]
- **Severity Level**: [1-5]
- **What Happened**: [Clear description]
- **Root Cause**: [If identified]

### What Was Attempted
1. [First fix attempt] — Result: [Failed/Partial/Success]
2. [Second fix attempt] — Result: [Failed/Partial/Success]

### Current State
- **Status**: [Working/Degraded/Broken]
- **Key Issues**: [Main problems]
- **Workaround**: [If any temporary fix was applied]

### Lessons Learned
- [What this failure teaches us]
- [How to prevent similar issues]
```

Refer back to `shared/operational/task-completion-protocol` for wrap-up steps once the log is captured.

## Key Principles
- Document everything — future agents rely on your trail.
- Edit forward — no git rollbacks; if it’s broken, stabilize or stop.
- Know when to halt — unresolved severity ≥3 means step back and report.
- Capture lessons — every failure should improve the ecosystem.
