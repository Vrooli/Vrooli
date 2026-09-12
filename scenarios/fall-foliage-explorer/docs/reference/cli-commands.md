# CLI Commands

The installed command is `fall-foliage-explorer`; lifecycle wrappers should still be invoked through `make` or `vrooli scenario`.

## Lifecycle

```bash
make setup
make start
make status
make logs
make test
make stop
```

These map to [CODE: Makefile] and [CODE: .vrooli/service.json].

## Application CLI

```bash
fall-foliage-explorer --help
fall-foliage-explorer --version
fall-foliage-explorer regions
fall-foliage-explorer foliage status <region-id>
fall-foliage-explorer foliage predict <region-id>
fall-foliage-explorer foliage weather <region-id> --date YYYY-MM-DD
fall-foliage-explorer reports list --region <region-id>
fall-foliage-explorer reports submit --body-file report.json
fall-foliage-explorer trips list
fall-foliage-explorer trips save --body-file trip.json
```

Command registration lives in [CODE: cli/domains/domains.go]. Domain command implementations live in [CODE: cli/domains/regions/register.go], [CODE: cli/domains/foliage/register.go], [CODE: cli/domains/reports/register.go], and [CODE: cli/domains/trips/register.go].

## API Base

The CLI adapter uses the scenario command metadata in [CODE: cli/app.go]. Keep API output shapes aligned with [CODE: cli/internal/support/types.go].
