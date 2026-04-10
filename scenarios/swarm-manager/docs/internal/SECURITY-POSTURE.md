# Security Posture

## Current Posture

Swarm Manager is designed for local Vrooli deployments with filesystem-backed state and controlled inter-scenario calls.

## Key Controls

1. Path safety checks for file operations and scenario/backlog paths.
2. Proto validation at API ingress for request contracts.
3. Controlled integration boundaries through discovery-based clients.
4. Confirmation gates for destructive scenario operations in the UI.

## Risks to Track

1. Misconfigured permissions on scenario folders.
2. Overly permissive execution mode defaults in autonomous contexts.
3. Stale assumptions about optional integration availability.

## Mitigations

1. Keep destructive confirmations enabled.
2. Use scheduled/manual modes where operational risk is high.
3. Keep path validation and request validation tests in CI.

## Security Code References

- [CODE: api/internal/pathutil/root.go]
- [CODE: api/internal/httputil/response.go]
- [CODE: api/internal/scenarios/handler.go]
- [CODE: ui/src/pages/ScenarioDetailsPage.tsx]
