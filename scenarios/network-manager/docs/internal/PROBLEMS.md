# Problems — Network Manager

## What belongs here

Scenario-specific known issues, deferred decisions, architecture drift, and implementation blockers.

## What does NOT belong here

General Vrooli platform issues, unrelated bugs, or product ideas that belong in backlog.

## Entry template

```markdown
### YYYY-MM-DD — Title

- Signal:
- Impact:
- Next action:
```

## Entries

### 2026-06-23 — Product domain implementations are scaffolded

- Signal: P0 proto/API/CLI domains exist and template `notes` was removed, but domain behavior returns safe placeholder/read-only responses.
- Impact: Implementation agents can build behind stable contracts, but operators must not treat current responses as real network measurements or resolver changes.
- Next action: Implement the first real vertical slice, likely `snapshot run` with deterministic probe fakes and `[REQ:NM-P0-001]` tests.

### 2026-06-23 — AdGuard Home resource still needs implementation decision

- Signal: PRD selects AdGuard Home as the first resolver backend, but no resource or adapter has been implemented.
- Impact: P0 cannot manage filtering until a governed resource/adapter path exists.
- Next action: Plan AdGuard Home resource/adapter implementation through dependency/resource governance.

### 2026-06-23 — First router adapter not selected

- Signal: P0 intentionally uses manual router guidance; P1 needs one explicit router adapter.
- Impact: Router-enforced DNS rules and Wi-Fi/router changes remain manual until a platform is chosen.
- Next action: Select based on first real deployment environment.

## Architecture Drift

No drift yet. The template example domain has been removed; the main risk is leaving scaffold responses in place without requirement-tagged domain tests.

## Cross-references

- [`DECISIONS.md`](DECISIONS.md)
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)
