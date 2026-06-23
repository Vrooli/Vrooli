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

### 2026-06-23 — Product scaffold is documentation-only

- Signal: Scenario was generated and PRD/requirements/docs were authored, but product code still contains template example behavior.
- Impact: Implementation agents must not treat current UI/API/CLI behavior as Network Manager functionality.
- Next action: Build the first real vertical slice, then remove template example domains with the detemplate flow.

### 2026-06-23 — AdGuard Home resource still needs implementation decision

- Signal: PRD selects AdGuard Home as the first resolver backend, but no resource or adapter has been implemented.
- Impact: P0 cannot manage filtering until a governed resource/adapter path exists.
- Next action: Plan AdGuard Home resource/adapter implementation through dependency/resource governance.

### 2026-06-23 — First router adapter not selected

- Signal: P0 intentionally uses manual router guidance; P1 needs one explicit router adapter.
- Impact: Router-enforced DNS rules and Wi-Fi/router changes remain manual until a platform is chosen.
- Next action: Select based on first real deployment environment.

## Architecture Drift

No drift yet. The main risk is accidentally extending template example domains instead of replacing them with Network Manager domains.

## Cross-references

- [`DECISIONS.md`](DECISIONS.md)
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)
