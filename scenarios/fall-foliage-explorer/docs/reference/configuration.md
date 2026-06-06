# Configuration

## Ports

Ports are defined in [CODE: .vrooli/service.json]:

- API: `API_PORT`, range `15000-19999`
- UI: `UI_PORT`, range `20000-24999`

The UI fallback API port is `17175` in [CODE: ui/src/app.js]. Prefer lifecycle-provided proxy metadata when running under app-monitor.

## Resources

Required resources are declared in [CODE: .vrooli/service.json]:

- `postgres` for regions, weather, predictions, reports, and trips.
- `redis` for planned real-time weather caching.
- `ollama` with `llama3.2:latest` for advanced predictions.
- `qdrant` for generated embeddings.

## Environment Variables

- `API_PORT` and `UI_PORT` are assigned by lifecycle.
- `POSTGRES_*` values are consumed by the shared database helper.
- `OLLAMA_URL` overrides the default Ollama URL used by [CODE: api/main.go#generateFoliagePrediction].

## Health

Lifecycle health checks target `/health` on both API and UI. The API timeout is 5000 ms, satisfying the five-second guardrail. The UI timeout is 3000 ms.
