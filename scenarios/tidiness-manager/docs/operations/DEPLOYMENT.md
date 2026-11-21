# Deployment

Tidiness Manager is deployed as a Vrooli scenario with API, CLI, and UI surfaces.

## Local Deployment

Use lifecycle commands:

```bash
cd scenarios/tidiness-manager
make setup
make start
make status
```

## Runtime Dependencies

PostgreSQL is required. Redis, visited-tracker, resource-claude-code, resource-codes, and analyzer binaries are optional enhancements.

## Release Checks

Before treating a deployment as ready, run:

```bash
test-genie execute tidiness-manager --phases quality --json
test-genie execute tidiness-manager --phases tidiness --json
test-genie execute tidiness-manager --phases docs --json
```

Use `vrooli scenario test tidiness-manager` for full validation when preparing a broad release.

## Operational Boundary

Do not deploy Tidiness Manager as the source of static-quality policy. That role belongs to Quality Health.
