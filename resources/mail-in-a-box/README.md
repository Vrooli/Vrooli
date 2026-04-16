# Mail-in-a-Box Resource

Managed Mail-in-a-Box runtime for local mail, calendar, and contact workflows.

## Intent

- Resource ID: `mail-in-a-box`
- Category: `infrastructure`
- Driver: `compose-service`
- Portability tier: `platform-specific`

## Use Cases

- Run a self-hosted mail stack for testing or internal infrastructure workflows.
- Provide local SMTP, IMAP, webmail, and calendar endpoints for scenario integration.
- Experiment with mail-driven automations without relying on external providers.

## Architecture

This resource is being aligned to the updated `compose-service` structure.

- `resource.json` is the declarative authority for lifecycle, compose orchestration, ports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Mail-in-a-Box-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

The intended escalation path is:

1. express behavior in `resource.json` and `docker-compose.yml`
2. rely on the shared `vrooli resource ...` control plane
3. add Mail-in-a-Box-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/compose`: compose-specific runtime graph helpers
- `cli/internal/topology`: service dependency and readiness semantics
- `cli/internal/runtime`: runtime shaping helpers
- `cli/internal/health`: Mail-in-a-Box-specific readiness helpers
- `cli/internal/env`: environment export helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install mail-in-a-box

# Check status through the shared control plane
resource-mail-in-a-box status
```

Default access points:

- Webmail: `http://localhost:8881`
- SMTP: `localhost:25`
- Submission: `localhost:587`
- IMAPS: `localhost:993`

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for mail, account, or CalDAV workflows.
- Keep runtime state rooted in `${RESOURCE_*_DIR}` paths and compose-managed mounts rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/mail-in-a-box/docs/OPERATIONS.md) as the architecture boundary for future migrations.
