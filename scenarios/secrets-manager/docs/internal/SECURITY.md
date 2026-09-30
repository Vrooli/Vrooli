# Security — Secrets Manager

## Assets

The protected assets are vault keys, encrypted item payloads, usernames and
origins, provider bootstrap credentials, grants, assurance tokens, and audit
integrity. Metadata-only responses may identify an item but never contain its
secret fields.

## Boundaries

- Production management requests require the configured owner authentication
  boundary. Database-less in-memory mode exists only for unit tests.
- Vault payloads use AES-GCM with nonce and item-bound associated data.
- Lock blocks reveal and writes. Missing key material fails closed and does not
  remint a key.
- Reveal requires fresh, operation-bound assurance and consumes it once.
- Grants separate use, reveal, export, injection, and signing. Current-snapshot
  membership is the default; dynamic membership must be explicit.
- Request digests prevent an approval for one scope from authorizing another.
- Activity records are metadata-safe and bounded.

## Threat responses

Prompt injection or arbitrary metadata cannot broaden a grant because the
authority evaluates typed operations, item scope, target, selector, and expiry.
Replay fails through one-time assurance and terminal request state. A stale
client revision receives a conflict. A locked or unavailable authority does
not return an empty success. A copied runtime-injected value remains the
consumer's responsibility and cannot be recalled by local revoke.

## Claims deliberately withheld

This product does not claim zero knowledge against the self-hosted authority
administrator, independent security audit completion, Safari support, mobile
native support, passkey-provider behavior, or payment-card autofill. Browser,
native-host, provider, backup-recovery, and commercial claims remain pending
until their owning validation produces evidence.

## Test evidence

Synthetic tests cover envelope authentication, exact-byte preservation,
field-scoped reveal, lock failure, revision conflicts, current-snapshot grants,
approval digest binding, terminal decisions, and SQLite persistence. Tests must
never use real account values or copy secrets into logs, screenshots, or generic
diagnostics.
