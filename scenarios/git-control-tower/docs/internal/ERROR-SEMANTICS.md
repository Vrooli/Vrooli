# Error Semantics & Recovery Paths

## Last Updated
2026-07-14

## Error Categories

### Configuration Errors
- Recovery: [how to recover]
- User message pattern: [template]
- Machine hint: `{ category: 'CONFIG', ... }`

### Validation Errors
- Recovery: [how to recover]
- User message pattern: [template]
- Machine hint: `{ category: 'VALIDATION', field: '...', ... }`

### Connectivity Errors
- Recovery: [how to recover]
- User message pattern: [template]
- Retry strategy: [exponential backoff, etc.]

### Permission Errors
- Recovery: [how to recover]
- User message pattern: [template]

### Internal Logic Errors
- Recovery: [how to recover]
- Logging pattern: [what to log]

## Failure Modes
[From failure-topography-and-graceful-degradation]

| Flow | Failure Mode | Current Behavior | Desired Behavior |
|------|--------------|------------------|------------------|
| Baseline/collection finalizer | Client cancellation or detached-tail deadline | The durable snapshot intent remains pending; no baseline or false failed member is published. | Reattach once through the printed status command; the non-wait collection reconciliation path resumes terminal children from their durable Test Genie records. |
| Baseline/collection finalizer | Terminal Test Genie error | The individual intent records failure and the collection member becomes failed coverage. | Return truthful partial coverage so unaffected terminal members remain visible; failed coverage cannot become a clean collection diff. |
| Baseline commit | Two finalizers race for one durable run | Both may await the run, but terminal commit rechecks storage. | One owner-scoped pin and manifest persist; later finalizers return the stored anchor idempotently. |
