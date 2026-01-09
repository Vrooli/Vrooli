# Problems & Risks

## Open Issues
- Rule execution can be slow if it runs builds across many modules; keep rules scoped, time-bounded, and explicitly toggled.
- Some scenarios may intentionally deviate from templates; rules should provide “why” and remediation without assuming a single layout.

## Deferred Ideas
- Emit a stable JSON contract consumable by `scenario-auditor` as an external rule pack (P1).
- Add TS/Jest/Vitest harness validation rules and lifecycle invariant checks (P1).
