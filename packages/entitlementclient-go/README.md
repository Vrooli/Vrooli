# entitlementclient-go

Verifies LPBS-signed entitlement leases (Go package `entitlementclient`) and provides a
cache-first client for untrusted scenario processes.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).

## Restart-persistent offline leases

`Client.SetStore` enables an optional store for the original LPBS-signed token. `FileLeaseStore` writes a private, atomic record under an application-owned data directory. A new client re-verifies the token signature, expiry, user identity, and requested binding before using it. Verification keys come separately through `IssuerKeyStore`; the mutable lease file cannot supply its own trust root. Access and refresh credentials are not stored by this package.

Applications can back `IssuerKeyStore` with the OS credential authority. Online JWKS responses from the configured LPBS authority are saved there, keyed by issuer origin, and reused when the app restarts offline. Remote issuer URLs must use HTTPS; plain HTTP is accepted only for localhost or loopback development authorities. Sign-out clears lease records while retaining public issuer trust keys. If the independent trust store is unavailable, offline restore fails closed. This protects against replacing a lease record and its embedded JWKS together; it does not make client-side state authoritative against a user who can control both the app process and credential authority. Server-side metered operations still require LPBS authority.

## App-scoped effective grants

New leases may carry `app_entitlements`, with one unique record per application and its effective status, rank, features, and limits. LPBS must expand an individual application offer to that application's record and a bundle to the union of its included application records. When this field is present, consumers must ignore global feature/rank/limit values for authorization; legacy bundle-only leases without it retain their existing behavior. Duplicate or incomplete app records are rejected during signing and verification. These records define the signed client contract; they do not implement LPBS offer, Stripe, overlap, upgrade, cancellation, refund, or recovery lifecycle logic.
