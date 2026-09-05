# Deployment

Fall Foliage Explorer deploys through the Vrooli scenario lifecycle rather than direct binary or Node process starts.

## Build Artifacts

- API binary: `api/fall-foliage-explorer-api`
- UI bundle: `ui/dist/index.html`, `ui/dist/app.js`, `ui/dist/styles.css`

Build steps are declared in [CODE: .vrooli/service.json].

## Resource Requirements

Deployment requires PostgreSQL, Redis, Ollama, and Qdrant resources as configured in [DOC: docs/reference/configuration.md#resources]. PostgreSQL schema initialization uses [CODE: api/internal/<domain>/schema.sql].

## Surface Expectations

The API and UI must both expose `/health`. The UI must be loadable through app-monitor iframe embedding, which depends on the bridge initialization in [CODE: ui/src/app.js].
