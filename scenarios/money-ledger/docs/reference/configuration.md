# Configuration

The API reads no database-path environment variable. The storage resolver chooses the scenario data class and variant-aware namespace from the scenario id, so no inherited value can redirect it at another scenario's database. Lifecycle-provided API/UI ports must be used; hard-coding ports bypasses scenario routing.

Adapters are registered data, not environment toggles. A manual adapter and a file adapter require no secret. Future upstream credentials must enter through the platform secret boundary and must not be written to the journal or committed to this repository.

Crypto receipt acceptance uses `api-core/authn` and Scenario Authenticator's existing service-principal flow. Its defaults are issuer `scenario-authenticator`, audience `scenario-authenticator:money-ledger`, subject `treasury-money-ledger` (Treasury holds one principal per peer; the journal still records the actor as `treasury`), actor kind `service`, and capability `money-ledger:write`. JWKS discovery resolves Scenario Authenticator through the control plane. Missing credentials, invalid signatures, wrong audiences and unavailable verification fail closed.

The shared runtime bindings `VROOLI_AUTH_PROVIDERS`, `VROOLI_AUTH_SCENARIO_ISSUER`, `VROOLI_AUTH_SCENARIO_AUDIENCE` and `VROOLI_AUTH_SCENARIO_JWKS_URL` configure the verifier. A disposable isolated test instance may point at an ephemeral loopback JWKS fixture. Production uses the registered resource and the credential authority; the API does not accept a private shared token or a token-file bypass.

`treasury-base-usdc` is a reserved adapter id. Authenticated acceptance registers it within the receipt transaction. Manual registration and ordinary `IngestEvent` reject that id.

## Write access

Every write is gated (see [`../internal/SECURITY.md`](../internal/SECURITY.md)). Three host settings decide who passes:

- **The owner.** A person writes as `operator` only when the host's owner-binding policy (`~/.config/vrooli/owner-bindings.json`, or `VROOLI_AUTH_OWNER_BINDINGS` / `VROOLI_AUTH_OWNER_BINDINGS_FILE`) binds one of their verified provider subjects to their canonical subject with `money-ledger:write` or `*`. The console's local sign-in and the CLI's `--local-owner` present a `scenario_authenticator` identity, so that source needs a binding; the tunnel presents `cloudflare_access`. Bind both to the same canonical subject so the two sign-ins are one person. The API reads the policy at start, so restart Money Ledger after changing it.
- **Services.** `treasury-money-ledger` and `landing-page-business-suite-money-ledger-writer` are declared by their scenarios' `.vrooli/service-principals.json` and issued by the operator's `scenario-authenticator service-principals sync --apply`. Any other service principal is refused.
- **Browser session.** The console's session cookies (`vrooli_access_token`, `vrooli_refresh_token`, the names Scenario Authenticator's verifier reads by default; do not set `VROOLI_AUTH_SCENARIO_COOKIE` for Money Ledger, or the verifier stops reading the cookie the sign-in sets) are issued by Scenario Authenticator for the `money-ledger` resource through `api-core/browsersession`. They are HttpOnly and SameSite=Lax, and `Secure` when the request arrived over HTTPS.
