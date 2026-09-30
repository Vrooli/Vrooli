# Integration Hub

Integration Hub owns provider-neutral connection metadata, credential-authority
references, connection lifecycle, and scenario bindings. It deliberately does
not store or return credential values. The first connector is the smallest
useful API-key connector: OpenRouter can be created, probed, refreshed, bound,
revoked, and deleted while its value remains in the canonical credential
authority.

The API is the generated `common.v1.integrations.ConnectionService` Connect
surface. Every request requires `X-Vrooli-Identity` or a bearer identity, and
all reads are filtered by that owner. Durable state is `data/connections.json`
under the scenario-owned data root.
