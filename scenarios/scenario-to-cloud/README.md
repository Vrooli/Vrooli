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

