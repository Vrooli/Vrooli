# Configuration

The API reads no database-path environment variable. The storage resolver chooses the scenario data class and variant-aware namespace from the scenario id, so no inherited value can redirect it at another scenario's database. Lifecycle-provided API/UI ports must be used; hard-coding ports bypasses scenario routing.

Adapters are registered data, not environment toggles. A manual adapter and a file adapter require no secret. Future upstream credentials must enter through the platform secret boundary and must not be written to the journal or committed to this repository.

Crypto receipt acceptance uses `api-core/authn` and Scenario Authenticator's existing service-principal flow. Its defaults are issuer `scenario-authenticator`, audience `scenario-authenticator:money-ledger`, subject `treasury`, actor kind `service`, and capability `money-ledger:write`. JWKS discovery resolves Scenario Authenticator through the control plane. Missing credentials, invalid signatures, wrong audiences and unavailable verification fail closed.

The shared runtime bindings `VROOLI_AUTH_PROVIDERS`, `VROOLI_AUTH_SCENARIO_ISSUER`, `VROOLI_AUTH_SCENARIO_AUDIENCE` and `VROOLI_AUTH_SCENARIO_JWKS_URL` configure the verifier. A disposable isolated test instance may point at an ephemeral loopback JWKS fixture. Production uses the registered resource and the credential authority; the API does not accept a private shared token or a token-file bypass.

`treasury-base-usdc` is a reserved adapter id. Authenticated acceptance registers it within the receipt transaction. Manual registration and ordinary `IngestEvent` reject that id.
