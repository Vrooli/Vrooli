# Integration Hub

Integration Hub owns operator-scoped connection metadata and lifecycle for
external-service credentials. Its state stores credential-authority references,
connection metadata, and scenario/context bindings. Credential values stay in
the canonical credential authority and never appear in Hub responses.

## Current contract

The generated `common.v1.integrations.ConnectionService` contract in
`packages/proto/schemas/common/v1/integrations.proto` defines list/get, create,
probe, refresh, rotate, bind/unbind, revoke, and delete operations. The Hub
implements those operations in `api/hub.go`; `cli/main.go` is the thin command
surface. Requests require an explicit Vrooli identity or bearer identity, and
each identity sees only its own connections.

The only registered connector is OpenRouter with an API key and the `models`
scope. `connectors/openrouter/connector.json`, the dispatch in `api/hub.go`,
and `api/hub_test.go` define and guard this pilot. Create and rotate accept the
credential value as a write-only input. Probe and refresh check credential
authority status; they do not call a provider API. Binding checks only the
required scopes supplied in that request, then records a scenario slug and
optional context on the connection.

## Consumer boundary

Web Console calls the generated service server-to-server and projects only
metadata for its OpenRouter connection. The API adapter is
`scenarios/web-console/api/integration_hub_client.go`; its UI contract is
covered in `scenarios/web-console/ui/src/__tests__/integrations-panel.test.tsx`.
Consumers do not read Hub state or credential storage directly.

Onboarding currently reports scenario `integrations[]` declarations as
`deferred`; it does not query Hub or gate scenario startup. Callers must bind
connections directly through the Hub service or CLI. The project-level source
of truth for this boundary, including the manifest shape and onboarding
projection, is
[`docs/configuration/architecture.md`](../../../docs/configuration/architecture.md#how-scenario-integration-requirements-and-bindings-work-today).

OAuth, device-flow, scheduled provider checks, additional connectors, and
automatic manifest-to-binding onboarding are not implemented. Add provider
behavior behind the Hub service and keep each new capability tied to its actual
owner implementation and tests.

## Targeted validation

```bash
cd scenarios/integration-hub/api && GOWORK=off go test ./...
cd scenarios/integration-hub/cli && GOWORK=off go test ./...
vrooli scenario restart integration-hub
vrooli scenario status integration-hub --json
```
