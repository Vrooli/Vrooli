# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2026-09-06
> **Status**: Implementation contract
> **Scenario**: secrets-manager

## 🎯 Overview

Secrets Manager is Vrooli's self-hosted password manager for people and
bounded software access. It keeps ordinary password-manager workflows and
deployment credential workflows in one authority-aware product.

**Purpose**: Provide a durable local capability for encrypted human vault
management and explicitly authorized software credential use.
**Primary users**: Operators, developers, and enrolled agents working in a
self-hosted Vrooli installation.
**Deployment surfaces**: Go API, React web UI, lifecycle-managed scenario, and
future native/extension consumers.
**Intelligence amplification**: A domain-owned authority gives other scenarios
typed grants and brokered operations without copying password policy into each
consumer.

The first release supports login, password, API credential, secure note, TOTP,
and SSH item types. Metadata is searchable without decrypting payloads. Secret
fields are encrypted before durable storage and are returned only from a
field-scoped reveal operation after fresh action-bound assurance.

The product has explicit authenticated, unlocked, approved, brokered, injected,
backed-up, and recovered states. A grant that permits use does not permit raw
reveal or export. Runtime injection is documented as raw exposure to the
receiving process and is never described as model-isolated.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Encrypted vault custody | Create, search, update, trash, restore, and reveal supported password-manager items with metadata-only ordinary reads.
- [x] OT-P0-002 | Security finding detection | Preserve the existing security scanning contract while the password-manager boundary is added.
- [x] OT-P0-003 | Tier-aware deployment manifest | Preserve deployment manifest generation without returning secret values.
- [x] OT-P0-004 | Guided operator journeys | Provide a usable first-use dashboard with visible lock, reveal, access, and recovery boundaries.

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | Guided secret provisioning | Preserve the existing authorized provisioning flow and connect it to the authority boundary.
- [x] OT-P1-002 | Operator dashboard | Provide API-backed vault, access, activity, recovery, and settings views.
- [x] OT-P1-003 | Validation history persistence | Keep activity and audit history durable and metadata-only.
- [x] OT-P1-004 | Automation-friendly command normalization | Preserve the existing CLI/API command contract.
- [ ] OT-P1-005 | Reproducible lifecycle and tests | Use scenario lifecycle commands and the Test Genie suite as the release gate.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Suggested remediation | Add safe remediation suggestions with explicit approval.
- [ ] OT-P2-002 | Posture trend analysis | Add historical posture and unusual-delta views.
- [ ] OT-P2-003 | Policy enforcement integration | Add launch decisions for configured deployment policies.

## 🧱 Tech Direction Snapshot

- **UI**: React and Vite with the scenario's existing component and test setup.
- **API**: Go HTTP service with domain-owned password-manager handlers.
- **Preferred UI stack**: React, TypeScript, Vite, and the existing Vrooli UI
  package conventions.
- **Preferred API stack**: Go, Gorilla mux, and routed database access through
  `api-core/database`.
- **Storage**: Per-domain schema embedded beside the vault implementation and
  routed through `api-core/database`; PostgreSQL is the production authority
  and SQLite is used for isolated persistence tests.
- **Custody**: AES-256-GCM envelopes with fresh nonces and item-bound AAD.
- **Delegation**: Current-snapshot or dynamic grants, digest-bound approval,
  one-time assurance, and origin-bound bounded broker sessions.
- **Non-goals for this release**: zero-knowledge claims, independent audit
  claims, mobile applications, Safari support, and passkey-provider claims.

## 🤝 Dependencies & Launch Plan

**Required resources**:

- PostgreSQL for durable metadata and encrypted payloads in production.
- Scenario lifecycle and Test Genie for startup, test ownership, and receipts.

**Credential bindings**:

- `SECRETS_MANAGER_VAULT_KEY` is a lifecycle-provided 32-byte base64 or
  64-character hex key. It is never reminted at startup.
- `SECRETS_MANAGER_OWNER_TOKEN` is the temporary local management boundary
  until Scenario Authenticator RP integration is enabled.

**Launch sequence**: focused API/UI validation, scenario-owned lifecycle suite,
then browser/native-host/provider and recovery evidence from their owning
scenarios. Missing evidence remains pending rather than becoming a claim.

## 🎨 UX & Branding

The replacement shell is dark by default, uses a restrained security palette,
and keeps lock state, secret-bearing actions, approval consequences, and
recovery status visible. Keyboard-accessible buttons, form labels, responsive
layout, metadata-first content, and WCAG 2.1 AA accessibility expectations are
required. **Accessibility commitments**: keyboard navigation, labeled controls,
visible focus, readable contrast, and no secret values in ordinary accessible
names. The browser holds the owner token in memory only.

## Trust, threat model, and support boundary

Production management routes require the configured owner authentication
boundary. A self-hosted administrator can observe service process state. The
product makes no zero-knowledge or independent-audit claim. Revocation stops
future authority use and reports pending remote purge; it cannot recall a value
already copied to a consumer.

Chromium MV3, Firefox WebExtensions, native messaging, external providers,
backup drills, and commercial packaging remain explicitly pending until their
own scenario receipts exist. Safari and mobile native apps are unsupported in
this release.

## Acceptance and evidence

The API tests cover encrypted custody, exact field fidelity, metadata-only
reads, action-bound reveal, lock failure, optimistic conflicts, SQL
idempotency, grant snapshots, approval digests, terminal decisions, broker
origin/session controls, redaction, one-time export handles, and native import.
The UI suite covers the replacement shell and existing operator components.
Scenario, browser, native-host, provider, backup-drill, and commercial proofs
remain pending until their owning runners produce receipts.
