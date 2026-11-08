# monetization-go

Shared client-side paid-surface decisions and a durable usage outbox (Go package
`monetization`). Every lease is untrusted until `entitlementclient` verifies its signature
against LPBS's published JWKS.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
