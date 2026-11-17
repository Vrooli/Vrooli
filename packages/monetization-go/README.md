# monetization-go

Shared scenario-side paid-surface gates, durable usage outbox, and LPBS usage
transport (Go package `monetization`). Every lease is untrusted until
`entitlementclient` verifies its signature against LPBS's published JWKS.

## Gate and app-grant contracts

`NewGateForApp` binds the generated meter inventory to an app and bundle. It
fails closed on undeclared meters; `CachedMeter` accepts Class B only. Class A
usage must run through LPBS authority, so local limits cannot reserve or spend.

With app-scoped grants, unscoped `FeatureDecision`, `HasFeature`, and `Limit`
fail closed; use app-specific gate methods. Signed grant and legacy behavior
belong to [`entitlementclient-go`](../entitlementclient-go/README.md#app-scoped-effective-grants);
LPBS derives grants in [`deriveAppEntitlements`](../../scenarios/landing-page-business-suite/api/internal/commerce/account_service.go).
Identity remains account-wide while entitlements are per app.

`CredentialIssuerKeyStore` keeps public issuer keys outside the mutable lease
cache, keyed by authority origin. Configure it and declare
`vrooli/lpbs-issuer-trust:jwks-keysets` for durable offline leases. `SnapshotFromLease`
is display/local-capacity data only; `Gate` and LPBS own authorization and
Class A metering.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
