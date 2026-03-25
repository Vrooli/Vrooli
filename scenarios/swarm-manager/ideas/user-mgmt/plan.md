# User Management — Implementation Plan

## 1. Purpose

Add user management capabilities to the Vrooli ecosystem, enabling scenarios to authenticate users, manage accounts, and enforce access control. The existing `scenario-authenticator` provides a complete JWT-based auth service; this work defines how scenarios (starting with a target scenario TBD) integrate user management features.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

- `scenarios/scenario-authenticator/README.md` — existing auth service (JWT, OAuth2, 2FA, RBAC)
- `packages/api-base/` — shared API connectivity library

## 3. Problem Statement

Vrooli has a production-ready authentication service (`scenario-authenticator`) but no standardized pattern for scenarios to integrate user management. Individual scenarios either lack user awareness entirely or implement ad-hoc solutions. This creates:

- No consistent user identity across scenarios
- No reusable integration pattern for auth
- No role-based access control in scenarios that need it

## 4. Scope

### In Scope

<!-- TBD — depends on which scenario this targets and scope decisions from workshop -->

### Out of Scope

<!-- TBD -->

## 5. Current Technical Context

### Existing Infrastructure

- **scenario-authenticator**: Complete JWT auth with user management, session handling, RBAC, OAuth2 (Google/GitHub), 2FA (TOTP), audit logging, rate limiting
  - API: `/api/v1/auth/*`
  - DB: PostgreSQL + Redis session storage
  - Seed accounts: admin@vrooli.local, test@vrooli.local, demo@vrooli.local
- **packages/api-base**: Universal API connectivity — handles API resolution, WebSocket endpoints, proxy injection. No built-in auth handling.
- **landing-page-business-suite**: Has account service infrastructure as a reference implementation
- **agent-manager**: Currently no user management — clean slate

### Key Files

<!-- TBD — depends on target scenario -->

## 6. Target End State

<!-- TBD — depends on workshop decisions about scope and approach -->

## 7. Implementation Strategy

<!-- TBD — phased approach depends on scope decisions -->

### Phase 1: Core Integration

<!-- TBD -->

### Phase 2: Scenario-Specific Features

<!-- TBD -->

## 8. Contract Decisions

<!-- TBD — API/CLI/data model behavior depends on scope -->

## 9. Testing Plan

<!-- TBD -->

## 10. Rollout / Validation Checklist

<!-- TBD -->

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Scope creep — "user management" can expand indefinitely | High | Define clear boundaries in workshop |
| Auth integration complexity varies by scenario | Medium | Start with simplest integration pattern |
| Breaking existing scenario behavior | Medium | Additive changes only; no existing API changes |

## 12. Non-goals / Prohibited Patterns

- Do not fork or duplicate `scenario-authenticator` logic
- Do not implement custom auth when `scenario-authenticator` integration suffices
- Do not add user management to scenarios that don't need it

## 13. Definition of Done

<!-- TBD — depends on scope decisions -->
