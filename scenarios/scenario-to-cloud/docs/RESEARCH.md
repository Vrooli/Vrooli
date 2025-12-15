# Research Packet

## Uniqueness Check

- Repo search: `rg -l "scenario-to-cloud" scenarios/`
- Result: `deployment-manager` already contains planning docs for `scenario-to-cloud` and treats it as future/stubbed. This scenario formalizes and implements that packager.

## Related Scenarios (in-repo)

- `scenarios/deployment-manager`: orchestrates tier workflows; will own profiles and invoke this packager.
- `scenarios/scenario-dependency-analyzer`: source of truth for scenario+resource dependency graphs.
- `scenarios/vrooli-autoheal`: must be included in “mini Vrooli” deployments to keep Tier-1 health guarantees.
- `scenarios/scenario-to-desktop`: reference implementation for “scenario-to-*” packagers (API/UI/CLI + requirements tracking).

## Key Decisions (P0)

- VPS-first, Ubuntu-only, SSH + scp tarball deploy.
- Native resources (no Docker) via “mini Vrooli” bundle + `./scripts/manage.sh setup`.
- Edge via Caddy + Let’s Encrypt; DNS is manual prerequisite.
- Fixed ports on VPS: UI 3000, API 3001, WS 3002 (overridden at start time).

## External References (starter list)

- Caddy HTTPS automation (Let’s Encrypt HTTP-01)
- SSH automation patterns (future P1/P2 hardening)
- Idempotent deployment patterns (future rollback/update work)

