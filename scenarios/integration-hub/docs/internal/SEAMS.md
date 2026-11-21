# Integration Hub seams

Integration Hub owns authenticated connection metadata and the lifecycle
operations exposed by `common.v1.integrations.ConnectionService`.

## Ownership

- `api/hub.go` owns connection state, connector dispatch, status mapping, and
  binding metadata.
- `api/credential_cli.go` is the production seam to the canonical credential
  authority. Provider values never belong in JSON state, protobuf responses, or
  logs.
- The current connector boundary is the source-controlled
  `connectors/openrouter/connector.json` plus its OpenRouter API-key lifecycle
  driver. New provider drivers must be added behind the Hub rather than in Web
  Console or another consumer.

## Safety and failure behavior

Requests require an explicit Vrooli identity or bearer identity. Bearer values
are hashed before they can become an owner identity. Responses contain
metadata only. Provider failures map to `PROVIDER_UNAVAILABLE`; absent or
unconfigured authority values map to `DISCONNECTED`.

State writes are bounded to the scenario data directory and use a temporary
file plus rename. The store is an implementation detail; consumers use the
generated Connect service or the thin `integration-hub` CLI.

## Continuation contract

The Hub may be unavailable while consumer scenarios remain usable in degraded
mode. Consumers must preserve local runtime-health sections and must not
invent a connected account from a commercial recommendation or a resource
credential declaration.

## Targeted validation

```bash
cd scenarios/integration-hub/api && GOWORK=off go test ./...
cd scenarios/integration-hub/cli && GOWORK=off go test ./...
vrooli scenario restart integration-hub
vrooli scenario status integration-hub --json
```
