# Runbook

## Start / Stop / Status

Use the scenario Makefile:

```bash
cd scenarios/knowledge-observatory
make start
make logs
make stop
```

## Common Incidents

| Incident | First check |
|---|---|
| Search unavailable | Qdrant health and configured URL |
| Metrics empty | PostgreSQL connectivity and collection presence |
| Healing unavailable | agent-manager and prompt-manager availability |
| Docs health noisy | Scenario `docs/manifest.json` contract and registered paths |

## Maintenance Tasks

Run documentation health and audit before large documentation changes.

## Escalation

If a resource is unavailable, validate the resource independently before
changing Knowledge Observatory code.
