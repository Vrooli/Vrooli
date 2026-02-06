# Error Semantics & Recovery Paths

## Last Updated
[Date]

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
| [flow] | [what fails] | [what happens now] | [graceful degradation] |
