# System Monitor UI

## Purpose
The System Monitor UI renders the real-time dashboard, investigations workflow,
and reporting panels for the system-monitor scenario.

## Architecture
- `src/features`: Domain-first UI slices (metrics, monitoring, investigations, reports, settings).
- `src/shared`: Shared UI primitives and API helpers.
- `src/types`: Shared API/DTO typings used across features.

## Development
Use the scenario lifecycle to run the UI together with the API.

```bash
make start
# or
vrooli scenario start system-monitor
```

## Production Server
The production UI server entrypoint is `server.cjs`, which uses `@vrooli/api-base`
to serve `dist/` and proxy API calls.
