# Runbook

## Status

```bash
vrooli scenario status fall-foliage-explorer
```

Use this before and after changes to confirm lifecycle state and port ownership.

## Logs

```bash
make logs
```

Logs are lifecycle-owned. Do not tail directly started processes because direct starts bypass scenario tracking.

## Validation

```bash
scenario-completeness-scoring score get fall-foliage-explorer --json
scenario-auditor audit fall-foliage-explorer --timeout 240
vrooli scenario test fall-foliage-explorer all
vrooli scenario ui-smoke fall-foliage-explorer
```

Phase-specific tests can be run with `structure`, `dependencies`, `unit`, `integration`, `e2e`, `business`, or `performance`.

## Recovery

If the API or UI is stale:

```bash
make stop
make start
make status
```

If predictions degrade, verify Ollama is running. The API should still produce fallback peak dates when Ollama is unavailable.
