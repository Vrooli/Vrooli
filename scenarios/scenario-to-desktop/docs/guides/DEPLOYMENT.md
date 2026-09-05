# Landing Page Deployment Protocol

This guide explains how scenario-to-desktop deploys built artifacts to a Landing Page Business Suite (LPBS) instance for distribution and auto-updates.

## Implementation Map

- [CODE: api/pipeline/stage_deploy.go] - deploy stage execution, validation, artifact upload loop
- [CODE: api/deploy/lpbs_client.go] - LPBS discovery, proxy calls, presign/upload/commit/apply flow
- [CODE: api/deploy/targets.go] - saved deploy target persistence in the scenario-to-desktop config storage root
- [CODE: cli/domains/pipeline/commands.go] - generated Connect pipeline commands
- [CODE: ui/src/components/sections/deploy/DeploySection.tsx] - UI status and deploy result rendering

## Overview

The deploy stage uploads desktop application artifacts to a remote LPBS instance through a local LPBS proxy. This replaces the previous self-contained S3 distribution system with a lightweight integration that reuses LPBS's existing download infrastructure.

```
Desktop packager             Local LPBS              Remote LPBS (production)
┌──────────────┐   discovery  ┌──────────┐  proxy    ┌─────────────────────┐
│ Deploy Stage │────────────▶│ Admin API │─────────▶│ presign-upload      │
│              │  Bearer auth │          │          │ commit              │
│              │              │ Remote   │          │ apply               │
│              │  direct PUT  │ Profile  │          │                     │
│              │──────────────┼──────────┼─────────▶│ S3 (artifact files) │
└──────────────┘              └──────────┘          └─────────────────────┘
```

## Prerequisites

1. **Local LPBS running** — The local LPBS instance must be discoverable via `api-core/discovery`.
2. **Remote profile configured** — A remote profile must be set up in the local LPBS pointing to the production LPBS, with an active session (logged in).
3. **Service token** — Set the `LPBS_SERVICE_SECRET` environment variable to the same value configured in your LPBS instance.
4. **App registered** — The target desktop app must exist in the remote LPBS's download apps.

## Deploy Targets

Deploy targets are saved configurations stored in `deploy-targets.json` beneath the resolved scenario-to-desktop config storage root:

```json
{
  "schema_version": "1.0",
  "targets": {
    "vrooli-production": {
      "label": "Vrooli.com Production",
      "scenario_name": "landing-page-business-suite",
      "remote_profile": "prod-server"
    }
  }
}
```

### Managing targets

Deploy targets are administered through the generated Connect CLI and are
consumed by the pipeline configuration. The target’s name, remote scenario,
and remote profile are durable configuration; secrets remain local to the API
process and are never stored in the target or sent by the CLI.

```bash
scenario-to-desktop deploy-target add "vrooli-production" --scenario "landing-page-business-suite" --profile "prod-server" --label "Vrooli.com Production"
scenario-to-desktop deploy-target doctor "vrooli-production"
```

## Running the Deploy Stage

Start and inspect the durable packaging workflow through the generated CLI:

```bash
scenario-to-desktop pipeline run "my-scenario"
scenario-to-desktop pipeline list
```

## Stage Behavior

- The deploy stage depends on smoke test completion in pipeline order.
- The stage is skipped when no deploy config is provided.
- The stage fails fast when:
  - `LPBS_SERVICE_SECRET` is missing
  - no build artifacts are available
  - remote profile validation fails
  - any upload/commit/apply step fails

## Authentication

The deploy stage authenticates to the local LPBS using a service bearer token:

```
Authorization: Bearer <LPBS_SERVICE_SECRET>
```

This uses the same `LPBS_SERVICE_SECRET` environment variable that LPBS uses for service-to-service auth on usage endpoints. LPBS admin endpoints that s2d needs accept either admin session cookies (browser) or service bearer tokens (inter-scenario calls) via the `requireAdminOrService()` middleware.

## Required LPBS Endpoints

The deploy stage calls these endpoints on the local LPBS, which proxies them to the remote LPBS via the configured remote profile:

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/api/v1/admin/remote-profiles` | List remote profiles (for URL derivation) |
| `POST` | `/api/v1/admin/remote-profiles/{id}/test` | Validate remote profile session |
| `POST` | `/api/v1/admin/remote-profiles/{id}/proxy` | Forward requests to remote LPBS |

Through the proxy, these remote LPBS endpoints are called:

| Method | Path (proxied) | Purpose |
|--------|----------------|---------|
| `POST` | `/api/v1/admin/download-artifacts/presign-upload` | Get S3 presigned upload URL |
| `POST` | `/api/v1/admin/download-artifacts/commit` | Register uploaded artifact in DB |
| `POST` | `/api/v1/admin/download-assets/apply` | Link artifact to download asset |

## Upload Flow

For each platform artifact (e.g., `.exe`, `.dmg`, `.AppImage`):

1. **Presign** — Proxy a presign-upload request to the remote LPBS. Returns an S3 presigned URL, bucket, and object key.
2. **Upload** — PUT the artifact binary directly to the S3 presigned URL (bypasses the proxy for efficiency).
3. **Commit** — Proxy a commit request to register the uploaded artifact in the remote LPBS database.
4. **Apply** — Proxy an apply request to link the artifact to the download asset for the app.

## Update URL

After deployment, the deploy stage derives the auto-update URL from the remote profile's API base:

```
Remote profile api_base:  https://production.example.com/api/v1
App key:                  my-desktop-app

Update URL:               https://production.example.com/api/v1/updates/my-desktop-app
```

electron-updater then calls:
```
GET /api/v1/updates/{app_key}/{channel}/latest.yml
GET /api/v1/updates/{app_key}/{channel}/latest-mac.yml
GET /api/v1/updates/{app_key}/{channel}/latest-linux.yml
```

See [AUTO_UPDATES.md](./AUTO_UPDATES.md) for full auto-update configuration details.

## Troubleshooting

### Deploy stage is skipped

Ensure deploy config is provided via either:
- deploy target configuration is supplied through the UI/pipeline configuration
- inline target: `--deploy-to <scenario> --remote-profile <tag> --app-key <key>`

### "LPBS_SERVICE_SECRET environment variable not set"

Set `LPBS_SERVICE_SECRET` in the environment used by the scenario-to-desktop API process.

### "no built artifacts available for deployment"

Run build first and confirm the build stage produced at least one artifact.

### Remote profile test fails

Confirm the remote profile is active/logged in on LPBS before starting the pipeline.

### Service auth readiness fails

Use the deployment UI diagnostics for a triage report and exact next steps. If
service authentication fails, verify both scopes:
1) LPBS runtime scope (`landing-page-business-suite` scenario/deployment)
2) `scenario-to-desktop` scenario scope
