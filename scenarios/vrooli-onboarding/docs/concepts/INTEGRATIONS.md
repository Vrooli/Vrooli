# Integrations

## Current ownership

Scenario requirements are declared in `service.json#integrations[]`. Integration
Hub owns connection metadata and explicit scenario/context bindings. Its shipped
connector is an OpenRouter API-key pilot; provider OAuth, device flows, and
automatic manifest-driven onboarding remain unimplemented. See its owner guide
at [`scenarios/integration-hub/docs/START-HERE.md`](../../../../scenarios/integration-hub/docs/START-HERE.md)
and the user-facing [connection contract](../../../../docs/configuration/integrations/connections.md).

Onboarding reads declared requirements in
[`api/readiness.go`](../../api/readiness.go#L394). It emits a deferred readiness
item containing the connector name, purpose, requested scopes, required flag,
and multi-connection note. It does not query Integration Hub, compare a
manifest's requested scopes with a connection, create a binding, or prevent
scenario startup. No connection binding is stored in Onboarding's
`operator-state.json`.

This behavior is covered by
[`TestV2ReadinessReportsDeclaredIntegrationsWithoutFabricatingProviders`](../../api/v2_readiness_test.go#L397).
The UI presents the deferred status and any declared requirement. The current
Web Console consumer uses Integration Hub directly through
[`integration_hub_client.go`](../../../../scenarios/web-console/api/integration_hub_client.go).

## Deferred onboarding flow

The deferred step is intentional: it reports what a scenario declares but does
not invent a connection or imply that a supported provider flow exists. The Hub
pilot and the onboarding step have different scopes. A caller may use the Hub's
current OpenRouter operations directly, but onboarding does not turn other
connector declarations into Hub bindings.

For the manifest field and its present readiness behavior, see
[`docs/configuration/scenarios.md`](../../../../docs/configuration/scenarios.md#scenario-integration-declarations).
For resource-owned credential behavior, see
[`docs/configuration/secrets.md`](../../../../docs/configuration/secrets.md).
Do not change the step from deferred until the owner implements and validates an
explicit manifest-to-connection workflow, including scope and multi-binding
semantics.
