# Landing Page Deployment Protocol

This guide explains how scenario-to-desktop deploys built artifacts to a Landing Page Business Suite (LPBS) instance for distribution and auto-updates.

## Overview

The deploy stage uploads desktop application artifacts to a remote LPBS instance through a local LPBS proxy. This replaces the previous self-contained S3 distribution system with a lightweight integration that reuses LPBS's existing download infrastructure.

```
scenario-to-desktop          Local LPBS              Remote LPBS (production)
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

Deploy targets are saved configurations stored in `.vrooli/deploy-targets.json`:

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

### Managing Targets via CLI

```bash
# List saved targets
scenario-to-desktop deploy-target list

# Add a target
scenario-to-desktop deploy-target add vrooli-production \
  --scenario landing-page-business-suite \
  --profile prod-server \
  --label "Vrooli.com Production"

# Remove a target
scenario-to-desktop deploy-target remove vrooli-production

# Test connectivity
scenario-to-desktop deploy-target test vrooli-production
```

## Running the Deploy Stage

### Using a saved target

```bash
scenario-to-desktop pipeline run my-scenario \
  --deploy-target vrooli-production \
  --app-key my-desktop-app \
  --wait
```

### Using inline parameters

```bash
scenario-to-desktop pipeline run my-scenario \
  --deploy-to landing-page-business-suite \
  --remote-profile prod-server \
  --app-key my-desktop-app \
  --wait
```

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
