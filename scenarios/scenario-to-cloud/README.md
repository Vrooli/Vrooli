# scenario-to-cloud

Deploy a single Vrooli scenario and its analyzer-derived dependencies to cloud targets. P0 focuses on **Ubuntu VPS** using **SSH + scp tarball** and a “mini Vrooli” install (native resources, no Docker).

This scenario is designed to be invoked by `deployment-manager` (mirroring the “scenario-to-* packager” pattern used by `scenario-to-desktop`).

## Quickstart (local dev)

```bash
cd scenarios/scenario-to-cloud
make start
```

- UI: `http://localhost:<UI_PORT>/`
- API health: `http://localhost:<API_PORT>/health`

## CLI (via Vrooli lifecycle)

```bash
# Validate manifest
scenario-to-cloud manifest-validate manifest.json

# Preflight + bundle + VPS setup (upload + extract + setup + autoheal scope)
scenario-to-cloud preflight manifest.json
scenario-to-cloud bundle-build manifest.json
scenario-to-cloud vps-setup-plan manifest.json /path/to/bundle.tar.gz
scenario-to-cloud vps-setup-apply manifest.json /path/to/bundle.tar.gz

# Deploy/start (Caddy + TLS + fixed ports + health verification)
scenario-to-cloud vps-deploy-plan manifest.json
scenario-to-cloud vps-deploy-apply manifest.json

# Inspect (status + logs over SSH)
scenario-to-cloud vps-inspect-plan manifest.json
scenario-to-cloud vps-inspect-apply manifest.json
```

## Docs

- PRD: `scenarios/scenario-to-cloud/PRD.md`
- Requirements: `scenarios/scenario-to-cloud/requirements/`
- Research: `scenarios/scenario-to-cloud/docs/RESEARCH.md`
- Problems/Risks: `scenarios/scenario-to-cloud/docs/PROBLEMS.md`

## P0 Deployment Intent (VPS)

P0 will:
- Use `scenario-dependency-analyzer` to compute required scenarios + resources for a target scenario (plus always include `vrooli-autoheal`).
- Build a “mini Vrooli” tarball containing required `scenarios/`, required `resources/`, and all `packages/`.
- `scp` the tarball to the VPS and run Vrooli setup + start required resources + start the scenario.
- Force fixed ports at start time: `UI_PORT=3000`, `API_PORT=3001`, `WS_PORT=3002`.
- Configure Caddy + Let’s Encrypt to expose the UI over HTTPS (DNS is manual prerequisite in P0).
