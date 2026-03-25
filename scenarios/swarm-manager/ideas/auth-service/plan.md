# Auth Service — Implementation Plan

## Required Reading
```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read swarm-manager-backlog-tools
```

## 1. Purpose

Provide a centralized authentication and authorization service for the Vrooli ecosystem. Today, authentication is fragmented: `scenario-authenticator` offers JWT/OAuth/2FA/RBAC but operates independently, LPBS has its own admin session + user JWT stack, and other scenarios have no shared auth. This service would unify identity management so scenarios can share users, sessions, and permissions through a single source of truth.

## 2. Problem Statement

- **No unified identity**: Each scenario maintains independent user databases and auth logic
- **No cross-scenario SSO**: A user authenticated in one scenario cannot access another without re-authenticating
- **Duplicated auth code**: JWT validation, session management, and middleware are reimplemented per scenario
- **Keycloak unused**: An enterprise-grade IAM resource is enabled but not integrated
- **No service-to-service auth standard**: Inter-scenario communication uses ad-hoc bearer tokens

## 3. Scope

### In Scope
<!-- TBD — depends on workshop decisions about scope and approach -->

### Out of Scope
<!-- TBD -->

## 4. Current Technical Context

### Existing Auth Capabilities
- **scenario-authenticator**: Full auth suite (JWT RSA-signed, OAuth2 Google/GitHub, TOTP 2FA, RBAC, API keys, rate limiting, audit logging). PostgreSQL + Redis backed.
- **LPBS auth**: Admin sessions (HTTP-only cookies), user JWT, inter-scenario service tokens, encrypted API key storage. Independent PostgreSQL database.
- **secrets-manager**: Vault integration for secret discovery, security scanning, compliance scoring.

### Available Resources
- **PostgreSQL**: User accounts, sessions, audit logs
- **Redis**: Session storage, token blacklisting, rate limiting
- **Vault**: Secure secret storage (HashiCorp)
- **Keycloak**: Enterprise IAM — enabled but not integrated
- **Mailpit**: Email for password resets (disabled by default)

### Key Files
- `scenarios/scenario-authenticator/api/` — existing auth API
- `scenarios/landing-page-business-suite/api/auth.go` — LPBS admin sessions
- `scenarios/landing-page-business-suite/api/user_auth_service.go` — LPBS JWT
- `scenarios/secrets-manager/` — vault integration

## 5. Target End State
<!-- TBD — depends on whether this extends scenario-authenticator, wraps Keycloak, or is greenfield -->

## 6. Implementation Strategy
<!-- TBD — phased approach to be determined after workshop decisions -->

## 7. Contract Decisions

### API Surface
<!-- TBD — depends on approach (extend existing vs. new service) -->

### Data Model
<!-- TBD — unified user schema, session model, permission model -->

### Inter-Scenario Integration
<!-- TBD — middleware library, sidecar proxy, or direct API calls -->

## 8. Testing Plan
<!-- TBD -->

## 9. Rollout / Validation Checklist
<!-- TBD -->

## 10. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Migration complexity from existing auth systems | High | Phased rollout with backwards-compatible adapter |
| Session invalidation during migration | Medium | Dual-stack auth during transition period |
| Keycloak operational complexity | Medium | Evaluate managed vs. self-hosted; fallback to custom JWT |
| Breaking existing scenario auth flows | High | Maintain old endpoints with deprecation path |

## 11. Non-goals / Prohibited Patterns
<!-- TBD -->

## 12. Definition of Done
<!-- TBD — depends on scope decisions -->
