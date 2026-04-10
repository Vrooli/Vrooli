# Security Posture

## Current Controls

1. Input validation at handler boundaries.
2. Discovery-based service resolution rather than hardcoded addresses.
3. Separation between profile definitions (filesystem) and runtime state (database).

## Risks

1. Misconfigured CORS in shared environments.
2. Over-broad tool permissions in upstream agent profiles.

[CODE: api/pkg/handlers/tasks.go]
[CODE: api/pkg/discovery/scenarios.go]
[CODE: api/pkg/autosteer/profile_repository_fs.go]
