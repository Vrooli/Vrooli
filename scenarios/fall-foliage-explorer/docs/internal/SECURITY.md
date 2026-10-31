# Security

## Current Posture

The scenario is a local lifecycle-managed application with no authentication layer. It accepts report photo URLs and free-form descriptions, so future public deployment work should add input validation and abuse controls before exposing write endpoints broadly.

## Guardrails

- Keep health endpoints meaningful and bounded by lifecycle timeouts.
- Do not store secrets in repository files.
- Use lifecycle-provided resource configuration rather than hard-coded production credentials.

## Audit

Use:

```bash
scenario-auditor audit fall-foliage-explorer --timeout 240
```
