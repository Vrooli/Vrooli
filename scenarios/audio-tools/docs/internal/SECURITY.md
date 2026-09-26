# Security — Audio Tools

This is the product-specific security target and known qualification boundary,
not a security certificate. PRD OT-P0-009 governs private bounded voice handling.
Published documentation does not enable multi-user or hosted access safely.

## Purpose Of This Document

Define sensitive data, trust boundaries, required controls and remaining evidence.
Use [DATA.md](../concepts/DATA.md) for storage ownership and retention; use
[MONETIZATION.md](../business/MONETIZATION.md) for shared financial authority.

## Data Sensitivity

| Data | Classification and handling | Owner |
| --- | --- | --- |
| Microphone PCM and transcripts | Potentially private speech; explicit capture/transmission permission, bounded retention, scoped recovery and deletion | Shared browser journal and Audio Tools session/corpus owners |
| Speaker enrollment and identity material | Sensitive voice/identity data; separately consented enrollment and resource deletion | STT metadata and speaker resource |
| BYOK keys and encryption material | Secrets; protected storage, scoped retrieval, redacted errors and no general-purpose logs | BYOK store and runtime secret owner |
| Session/usage/account metadata | Potentially linkable personal or financial data; minimum required attribution | Audio usage history and shared monetization owners |
| Test corpus and diagnostics | Only consented/licensed corpus; metadata-only diagnostics, no recordings, transcripts or credentials in learning records | Corpus, Test Genie and diagnostic owners |

## Auth And Authorization

The API/service boundary must authorize account-scoped credentials, sessions,
retained data, experiments and billing. A UI-selected account or route is not
authorization. Before multi-user deployment, qualify identity propagation and
cross-account denial for reads, streams, recovery, deletion and settlement.

Browser consumers use their same-origin integration adapter. Qualify stream
origin/authentication, expired/revoked credentials and reconnect authorization;
a stable session ID is not a bearer grant. Local-only policy must prevent
external transmission even on error. A remote or paid fallback requires prior
user policy, not a toast after it occurs.

## Secrets

BYOK credentials already exist; they are not template data. The existing store
persists ciphertext. Verify key-file permissions, backup/restore, rotation,
account isolation and redaction before deployment; encryption-at-rest alone
does not establish these controls. Never copy real keys into fixtures, PRD
answers, program output, screenshots or work records.

Use the configured secret owners and metadata-only credential commands. Do not
add a second consumer credential store or journal raw upstream error bodies.

## Threat Model

| Risk | Required control and proof | Qualification state |
| --- | --- | --- |
| Unintended microphone capture or remote upload | Gesture/permission and actual-route disclosure; denial, background and fallback tests | Required; per-device proof needed |
| Cross-account access to credentials, retained audio or wallet | Server-owned identity and authorization; wrong-account and revoked-session tests | Required before multi-user/owned launch |
| Unbounded retention, quota eviction or oversized audio | Size/duration/resource bounds, explicit reduced durability and terminal coverage, user deletion | Existing journal controls need full-path qualification |
| Duplicate or forged charge | Durable shared admission/delivery/settlement identity; replay, concurrency and lost-response controls | Owned-service implementation/qualification open |
| Test fixture reaches production | Explicit owner-routed lease and test identity; reject missing/expired/wrong-account context | Audio monetization fixture wiring open |
| Private data leaks into diagnostics or exports | Metadata allowlist and redaction on success/error paths, including provider logs | Existing safe-turn export is narrower than complete pipeline proof |
| Malformed media or provider responses | Bounded decoding, cancellation and provider conformance tests | Qualify each claimed adapter/profile |

## Security Gaps

Current source and historical tests do not establish an end-to-end multi-user
security review. Before deployment, resolve retention durations and deletion
coverage across journals, sessions, corpus, resource embeddings and backups;
qualify credentials and cross-account stream/recovery authorization; verify
remote consent and fixture isolation; and prove billing recovery with shared
owners. These are acceptance debt, not claims that every listed control is absent.

## Cross-References

- [DATA.md](../concepts/DATA.md) — storage and retention
- [INTEGRATIONS.md](../concepts/INTEGRATIONS.md) — provider and owner boundaries
- [ERROR-HANDLING.md](ERROR-HANDLING.md) — shared error semantics
- [TESTING.md](TESTING.md) — negative controls and evidence
- [PROBLEMS.md](PROBLEMS.md) — dated limitations
