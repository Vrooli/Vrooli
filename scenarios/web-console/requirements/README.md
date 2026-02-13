# Web Console Requirements Registry

This directory tracks rewrite-target requirements for `web-console` and links each requirement to an operational target in `PRD.md`.

## Structure

```text
requirements/
├── index.json   # Requirement registry (schema v1.0.0)
└── README.md
```

## Linkage Model

Each requirement must map to one operational target ID via `prd_ref`:

```json
"prd_ref": "OT-P0-001"
```

This enables PRD Control Tower to:
1. Parse operational targets from `PRD.md`
2. Match requirements to targets by ID
3. Detect missing coverage and stale linkage

## Requirement Status Model

- `pending`: planned, not yet delivered
- `in_progress`: partially implemented
- `complete`: implemented and validated

## Validation Metadata

Every requirement should include structured validation records:

```json
{
  "type": "test",
  "ref": "api/example_test.go",
  "phase": "integration",
  "status": "pending",
  "notes": "Describe what behavior this validation covers"
}
```

Use concrete test files for `type: test`; avoid non-specific placeholders once implementation begins.

## Auto-sync Notes

Use `[REQ:<ID>]` tags in tests for robust requirement sync (example: `[REQ:WC-P0-003]`).

## Commands

```bash
# Validate PRD structure + requirements linkage
prd-control-tower prd validate web-console

# Validate requirements linkage only
prd-control-tower requirements validate web-console -type scenario
```
