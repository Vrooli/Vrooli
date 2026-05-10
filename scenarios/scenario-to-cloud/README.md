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
scenario-to-cloud manifest validate manifest.json

# Generate starter manifest + inspect schema
scenario-to-cloud manifest init --scenario landing-page-business-suite --host 203.0.113.10 --domain example.com --out cloud-manifest.json
scenario-to-cloud manifest schema
scenario-to-cloud manifest doctor cloud-manifest.json
scenario-to-cloud manifest fix cloud-manifest.json --write

# Preflight + bundle + VPS setup (upload + extract + setup + autoheal scope)
scenario-to-cloud preflight run cloud-manifest.json
scenario-to-cloud bundle build cloud-manifest.json
scenario-to-cloud vps setup plan cloud-manifest.json /path/to/bundle.tar.gz
scenario-to-cloud vps setup apply cloud-manifest.json /path/to/bundle.tar.gz

# Deploy/start (Caddy + TLS + fixed ports + health verification)
scenario-to-cloud vps deploy plan cloud-manifest.json
scenario-to-cloud vps deploy apply cloud-manifest.json

# Inspect (status + logs over SSH)
scenario-to-cloud inspect plan cloud-manifest.json
scenario-to-cloud inspect status <deployment-id>
```

## Docs

- PRD: `scenarios/scenario-to-cloud/PRD.md`
- Requirements: `scenarios/scenario-to-cloud/requirements/`
- Research: `scenarios/scenario-to-cloud/docs/internal/RESEARCH.md`
- Problems/Risks: `scenarios/scenario-to-cloud/docs/internal/PROBLEMS.md`

## P0 Deployment Intent (VPS)

P0 will:
- Use `scenario-dependency-analyzer` to compute required scenarios + resources for a target scenario (plus always include `vrooli-autoheal`).
- Build a “mini Vrooli” tarball containing the required repo subset (`scenarios/`, `resources/`, shared `packages/`, and generated deployment metadata).
- Upload the tarball to the VPS, extract it, upload a deployment-local native `vrooli` binary to `<workdir>/.vrooli/bin/vrooli`, then run native setup + start required resources + start the scenario.
- Force fixed listener ports at start time from the scenario manifest, typically `UI_PORT=3000` and `API_PORT=3001`.
- Configure Caddy + Let’s Encrypt to expose the UI over HTTPS (DNS is manual prerequisite in P0).
