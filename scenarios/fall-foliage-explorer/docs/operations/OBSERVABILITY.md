# Observability

## Health

API and UI health endpoints are configured in [CODE: .vrooli/service.json]. The API health handler is wired in [CODE: api/main.go].

## Logs

Use lifecycle logs:

```bash
make logs
```

## Test Artifacts

Historical phase logs live under `test/artifacts/`. Coverage output lives under `coverage/` and API coverage files may be present under `api/coverage.out` and `api/coverage.html`.

## Known Signal Gap

The current unit phase can report passing Go tests while failing the coverage threshold. See [DOC: docs/internal/PROBLEMS.md#open-follow-up].
