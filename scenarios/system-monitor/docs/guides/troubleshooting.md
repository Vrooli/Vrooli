# Troubleshooting

## Checking Health

```bash
system-monitor health
```

If the API is running, you will see uptime and status. If not, check that the scenario is started:

```bash
cd scenarios/system-monitor && make start
```

## Common Issues

### CLI `report` command returns 404

The CLI `report` command calls `/api/reports/generate` instead of the correct `/api/v1/reports/generate` (missing `/v1/` prefix). This is a known bug.

**Workaround**: Use curl directly:

```bash
curl -X POST http://localhost:8080/api/v1/reports/generate \
  -H "Content-Type: application/json" \
  -d '{"type": "daily"}'
```

### CLI `simulate` command fails

The `simulate` command calls `GET /api/test/anomaly/cpu`, which does not exist in the API. This test endpoint was never implemented.

### Data lost on restart

By default, the API uses in-memory storage. All metrics, investigations, and reports are lost when the API restarts.

**Fix**: Set `DATABASE_URL` to a PostgreSQL connection string to enable persistent storage. See [Configuration](../reference/configuration.md).

### CLI `--quiet` flag has no effect

The `--quiet` flag is parsed but never checked in any command implementation. It is effectively a no-op.

### Process kill silently fails

The UI's process kill dialog references `POST /api/v1/processes/{pid}/kill`, but this endpoint does not exist in the API router. The kill action completes without error in the UI but has no effect.

## Missing API Endpoints

The following endpoints are referenced by the UI but are not implemented:

| Endpoint | Referenced By |
|----------|--------------|
| `GET /api/v1/metrics/timeline` | Sparkline charts |
| `GET /api/v1/metrics/disk/details` | Disk detail view |
| `POST /api/v1/processes/{pid}/kill` | Process kill dialog |

## No Authentication

The API has no authentication middleware. All endpoints are publicly accessible. Do not expose the API port to untrusted networks.

## Logs

View scenario logs:

```bash
cd scenarios/system-monitor && make logs
```

## Restarting

```bash
cd scenarios/system-monitor && make stop && make start
```
