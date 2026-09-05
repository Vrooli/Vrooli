# Runbook

## Start Or Stop The Scenario

```bash
cd scenarios/tidiness-manager
make start
make status
make stop
```

## Validate The Ownership Boundary

```bash
test-genie execute tidiness-manager --phases quality --json
test-genie execute tidiness-manager --phases tidiness --json
```

Quality failures usually mean this scenario's own config drifted from Quality Health contracts. Tidiness failures usually mean the maintainability scan endpoint or mapping is broken.

## Investigate A Scan Failure

1. Run `tidiness-manager scan <scenario> --type tidiness --json`.
2. Check API logs with `make logs`.
3. Verify the target scenario path resolves safely.
4. Confirm PostgreSQL is healthy through `/api/v1/health`.

## Investigate A Campaign Failure

1. List campaigns with `tidiness-manager campaigns list`.
2. Inspect the campaign with `tidiness-manager campaigns get <id>`.
3. Check optional visited-tracker availability if prioritization or handoff failed.
4. Resume or terminate only after recording the reason in issue or campaign notes.
