# Invariants — Vrooli Bridge

This document records the non-negotiable invariants for the Machine and
EnrollmentAttempt model. Code and executable tests remain authoritative; this
contract is the review checklist for their implementation.

## Ownership Invariants

- Machine is the only owner of operator intent, ordered locators, lifecycle,
  desired policy, trust references, and Node lineage references.
- EnrollmentAttempt is the only owner of attempt state, input snapshot,
  checkpoints, correlation, retry lineage, and attempt diagnostics.
- Registry is the only owner of paired Node identity and approved scopes.
- Presence is the only owner of live health/liveness. Its facts are composed at
  read time and are never persisted as Machine or attempt fields.
- The trust store is the only owner of SSH private keys and known-host key
  material. Other stores retain only opaque references and public fingerprints.

## Identity and Lineage Invariants

- Machine UUID is stable. Identity resolution uses explicit Machine UUID first,
  current Node UUID second, SSH host-key fingerprint third, and normalized
  hostname last; it never uses username, display name, or Node name as identity.
- A Machine can exist before any contact and may have multiple prioritized
  locators. Repeated active locator evidence converges on the existing Machine;
  conflicting matches are an explicit repair/merge decision, never a guess.
- A Machine has at most one current Node, and a Node can be current in at most
  one Machine. Re-pair creates immutable lineage and explicitly supersedes the
  prior Node. The global partial unique index is installed only after migration
  reconciles legacy duplicates.
- `machines repair` reuses the Machine UUID and Bridge-managed key; `machines
  merge` is explicit, audited, archives the source, and preserves attempts,
  locators, and lineage history.

## Replay and Idempotency Invariants

- A terminal EnrollmentAttempt is immutable. Retry creates a distinct attempt
  with `retry_of_attempt_id`; it never resets, deletes, or changes a terminal
  row.
- Each checkpoint has a durable postcondition and is reusable only after that
  postcondition is revalidated. An ambiguous checkpoint produces typed repair,
  not a guess.
- Pairing correlation survives code redemption and later service-install or
  Presence failure. Replay/concurrent retry converges on one current Node and
  cannot make a burned code reusable.
- Archive, revoke, SSH removal, Machine removal, cleanup retry, and purge have
  separate idempotency keys and append-only `machine_audit_events` records.
  Local revocation commits before remote cleanup starts.

## Authorization and Secret Invariants

- Profiles and observed capabilities can propose setup or scopes but cannot
  authorize. Only explicit Registry approval grants scopes.
- Host-key mismatch fails closed. Server-host fingerprints and Bridge
  client-key fingerprints have distinct types and labels.
- Passwords and pairing codes are transient; private keys never enter a
  database, API response, CLI JSON, log, diagnostic, or audit record.
- The restricted Presence agent cannot dispatch, provision, approve scopes, or
  execute arbitrary shell through this model.
