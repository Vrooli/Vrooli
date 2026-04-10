# SSH Identity Greenfield Implementation Plan

> Created: 2026-02-08  
> Status: Draft (ready for implementation)  
> Scope: `scenario-to-cloud` SSH identity, bootstrap, deploy, and health coherence

## Required Reading

Run this exact command before implementation:

```bash
prompt-manager skill read screaming-architecture-audit utils-unification seam-discovery-and-enforcement documentation-health
```

## Hard Constraints (Non-Negotiable)

1. This plan is **greenfield**.
2. Implement with **no backward compatibility layers**.
3. Do **not** add legacy shims, dual paths, fallback mappers, or compatibility adapters for old identity behavior.
4. Remove/replace old key-status logic rather than preserving it.
5. Use a single canonical SSH identity model end-to-end.

## Problem Statement and Context

Current behavior can report:
- deploy/redeploy success (SSH transport works), while
- `ssh_key_auth` remains `unknown`.

Observed facts:
- Deployment can execute without explicit `-i` when manifest `key_path` is absent.
- Health key-auth logic depends on known key material presence for comparison in remote `authorized_keys`.
- Bootstrap key selection and deployment identity are not guaranteed to converge to one persisted source of truth.

Result: confusing operator experience and inconsistent automation signals.

## Target Outcome

After implementation:
1. Deploy, bootstrap, and health all use the same canonical identity model.
2. Health states are operationally meaningful and non-contradictory.
3. Identity provenance is explicit (pinned key vs agent/default transport).
4. Code structure clearly reflects domain intent (screaming architecture).

## Screaming Architecture Target

Add/refactor around explicit domain modules:

- `api/sshidentity/`
  - canonical model + state machine + resolution policy
- `api/orchestration/deploy/`
  - consume resolved identity and persist identity outcomes
- `api/orchestration/bootstrap/`
  - install/verify identity and persist canonical identity
- `api/observability/health/`
  - report identity state from canonical model only
- `api/infra/ssh/`
  - low-level SSH run/copy-key/authorized-keys inspection primitives

Rules:
- No identity decision logic in handlers.
- No direct key-state inference in health from ad hoc local variables.
- All identity transitions go through `sshidentity` package APIs.

## Canonical Domain Model

Introduce a single `DeploymentSSHIdentity` model:

- `key_path` (optional)
- `public_key_fingerprint` (optional)
- `auth_mode` (`explicit_key` | `agent` | `default_ssh` | `unknown`)
- `verification_state` (`authorized` | `unauthorized` | `unknown`)
- `last_verified_at` (timestamp)

This model is the only source of truth for:
- deploy transport identity
- bootstrap results
- health key-auth status rendering

## Implementation Plan (Greenfield Phases)

### Phase 1: Domain and Persistence Foundation

1. Add `DeploymentSSHIdentity` types and validation in `api/sshidentity/`.
2. Add persistence fields in deployment state to store canonical identity.
3. Remove direct old key-state interpretation paths from health internals.

Exit criteria:
- Deployment records can store and retrieve canonical SSH identity.

### Phase 2: Identity Resolver (Single Utility Path)

1. Implement `IdentityResolver` in `api/sshidentity/resolve.go`.
2. Resolver precedence:
   - explicit manifest key
   - successful tested key path
   - agent/default transport detection
3. Resolver must return a fully typed identity object (no loose maps).

Exit criteria:
- One resolver used by deploy, bootstrap, and health.

### Phase 3: Bootstrap Refactor

1. Bootstrap uses resolver-selected identity source.
2. On success, bootstrap persists canonical identity immediately.
3. Remove bootstrap-local identity assumptions not persisted to deployment state.

Exit criteria:
- Bootstrap output and stored identity are coherent and queryable.

### Phase 4: Deploy Refactor

1. Deploy pipeline loads canonical identity from resolver/state.
2. Successful deploy writes back final effective `auth_mode` + verification context.
3. Remove deploy-only implicit identity logic.

Exit criteria:
- Deploy uses and updates canonical identity only.

### Phase 5: Health/Observability Refactor

1. Health reads identity state from canonical model.
2. Replace ambiguous messages with explicit states:
   - `pass`: explicit key verified in `authorized_keys`
   - `warn`: connected via agent/default but not pinned
   - `fail`: explicit key expected but unauthorized, or SSH unreachable
3. Remove/retire old status mapping paths that produce contradictory outcomes.

Exit criteria:
- Health messaging is operationally clear and consistent with deploy/bootstrap.

### Phase 6: Cleanup and Enforcement

1. Remove dead/legacy identity helpers.
2. Enforce identity flow through interfaces only.
3. Add static checks/lints for prohibited imports/calls if needed.

Exit criteria:
- No parallel identity paths remain.

## Utilities Unification Plan

Unify these into shared, reusable components:

- key selection
- key-path normalization
- pubkey fingerprint extraction
- authorized-keys matching
- auth mode classification

Placement:
- pure/shared logic in `api/sshidentity/` and `api/infra/ssh/`
- no duplicate utility variants in CLI/API handlers

## Seam Design and Testability

Define explicit seams:

- `IdentityResolver`
- `SSHRunner`
- `AuthorizedKeysInspector`
- `DeploymentIdentityStore`

Requirements:
- core identity state transitions are pure and unit-testable
- infrastructure I/O is behind seam interfaces
- handlers only orchestrate; they do not perform domain decisions

## Test Strategy

### Unit Tests

1. Resolver precedence and edge cases.
2. State machine transitions (`unknown -> authorized`, etc.).
3. Health status mapping from canonical identity.

### Contract Tests

1. Bootstrap command output includes canonical identity state.
2. Health JSON contract reflects new explicit identity semantics.

### Integration Tests

1. Explicit key deploy path.
2. Agent/default deploy path.
3. Bootstrap -> deploy -> health convergence flow.

### Negative Tests

1. Explicit key configured but unauthorized.
2. Missing identity and no successful transport evidence.
3. Key mismatch after rotation.

## Documentation Work (Required)

Update:

1. `docs/concepts/architecture.md`
   - add SSH identity lifecycle and ownership boundaries
2. `docs/internal/SEAMS.md`
   - add seam ownership and dependency direction for identity flows
3. `docs/reference/configuration.md`
   - define canonical identity fields and semantics
4. `docs/reference/api-endpoints.md`
   - document health identity status payload and meanings
5. Add `// DOC:` references in new identity modules and orchestration call sites

## Acceptance Criteria

1. No code path exists where deploy succeeds and health reports ambiguous identity without clear auth mode context.
2. Bootstrap, deploy, and health all read/write the same identity model.
3. No legacy compatibility paths remain.
4. Tests pass for all identity matrices described above.
5. Documentation matches implemented behavior and is registered in manifest.

## Non-Goals

1. No VPS provisioning redesign.
2. No unrelated deployment pipeline feature changes.
3. No UI redesign beyond identity-state representation needed for correctness.

