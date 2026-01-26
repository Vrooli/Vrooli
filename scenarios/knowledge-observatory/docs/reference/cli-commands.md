# CLI Commands

The CLI is a thin wrapper over the Knowledge Observatory API.

[CODE: cli/knowledge-observatory]

## status
Checks API and dependency health.

```bash
knowledge-observatory status
```

## search
Semantic search against the API.

```bash
knowledge-observatory search "agent workflows" --limit 10 --threshold 0.35
```

## ingest
Upsert a single record.

```bash
knowledge-observatory ingest --namespace docs --content "Knowledge Observatory entry" --visibility shared
```

## ingest-job
Enqueue an async document ingest job.

```bash
knowledge-observatory ingest-job --namespace docs --content "$(cat README.md)" --chunk-size 1200 --chunk-overlap 150
```

## job-status
Fetch status for an ingest job.

```bash
knowledge-observatory job-status <job_id>
```

## health / metrics
Returns knowledge health metrics (alias).

```bash
knowledge-observatory health
knowledge-observatory metrics
```

## graph
Generate a knowledge graph around a concept.

```bash
knowledge-observatory graph --center "ecosystem-manager" --depth 2
```

## configure
View or set CLI config values.

```bash
knowledge-observatory configure
knowledge-observatory configure api_base http://localhost:<API_PORT>
```

## version
Shows CLI version and API base.

```bash
knowledge-observatory version
```

## Environment overrides
The CLI respects environment variables derived from the name:
- `KNOWLEDGE_OBSERVATORY_API_BASE`
- `KNOWLEDGE_OBSERVATORY_API_TOKEN`

See also: [DOC: docs/reference/configuration.md#cli-configuration]
