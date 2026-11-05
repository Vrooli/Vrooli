# Architecture — Secrets Manager

Secrets Manager has two cooperating product surfaces. The password-manager
domain owns vault metadata, encrypted item envelopes, grants, access requests,
assurance tokens, and metadata-only audit events. The existing deployment
domain owns resource coverage, scanning, strategy resolution, and deployment
manifest consumers. Both are exposed by one lifecycle-managed API, but their
secret custody contracts remain separate.

## Trust topology

```text
Human browser / typed client
        │ authenticated management request
        ▼
Secrets Manager API ── metadata and policy ── PostgreSQL or desktop SQLite
        │ secret-bearing operation after assurance
        ▼
Credential authority key ── AES-GCM item envelope ── vault payload
        │ bounded approved use
        ├── broker or runtime injection owner
        └── Bridge / deployment / extension owners

Agent Manager runs use a separate boundary. Secrets Manager sends the opaque
`X-Agent-Identity-Token` to Agent Manager's live verification endpoint and
derives the actor from the verified run ID (`agent-run:<run-id>`). That actor
can create an access request or use an approved broker session. It cannot use
the agent path for grant administration, approvals, vault mutation, or owner
authentication.
```

The API never returns encrypted payloads in list or detail responses. The
`reveal` route accepts an assurance token whose operation is exactly
`reveal:<item-id>` and consumes it once. Provider sources are registered by
explicit owner action; an external source requires an HTTPS endpoint and a
bootstrap reference, and remains unverified until its adapter supplies a
receipt. Scenario Authenticator is the production relying party for human
sessions. Sensitive action assurance requires a recent Authenticator-issued
JWT session, while the owner token remains a local operator fallback for
non-sensitive administration and isolated deployments.

Workspace roles are enforced below route visibility: owner and administrator
roles manage membership and grants, members can manage ordinary vault items,
viewers can read metadata only, and security operators can inspect audit
metadata without receiving secret fields. A removed member loses both its
membership and active owner-token records. Local desktop authority uses its
own enrolled operator context; it does not mint or imply a remote account
session.

## Storage ownership

`api/internal/vault/schema.sql` is the domain-owned schema. It stores item
metadata, encrypted payloads, revisions, bounded history, grants, access
requests, assurance records, and audit metadata. `api-core/database.RoutedDB`
keeps production and test-pool routing identical. Desktop mode applies the
same domain schema to its scenario-private SQLite database.

The envelope is AES-GCM with a fresh nonce and associated data binding the
format version, vault ID, and item ID. A missing or malformed 32-byte key is a
recovery-required state. Startup never remints a key for an existing vault.

## Authorization model

Selectors default to `current_snapshot`, recording the reviewed member IDs at
grant creation. `dynamic` is explicit and means future matching members can
qualify. Grants list operations independently; use authority never implies
reveal or export authority. Access requests bind the grant, item, operation,
and requester into a digest. The authority rechecks status, expiry, selector,
and operation at decision and use time.

## UI and lifecycle

`ui/src/PasswordManagerApp.tsx` is the current product shell. It provides vault
inventory, field-scoped reveal, access authoring, safe activity, recovery
status, and settings. It keeps an owner token in memory when a local authority
requires one and never writes it to local storage. The legacy deployment API
remains available to its existing consumers while the visible shell moves to
the password-manager workflow.

All processes start through the scenario lifecycle. The API port, UI port,
database route, and health checks remain owned by the Vrooli control plane.
