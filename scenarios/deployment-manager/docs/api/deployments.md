# Deployment Endpoints

Endpoints for triggering and monitoring deployments.

> **Note**: Full deployment orchestration is partially implemented. The `package` CLI command provides the most complete packaging workflow today.

## POST /deploy/{profile_id}

Initiate a deployment for a profile.

**Request:**

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/deploy/profile-123" \
  -H "Content-Type: application/json" \
  -d '{
    "dry_run": false,
    "async": true
  }'
```

**Request Body:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dry_run` | boolean | false | Simulate without executing |
| `async` | boolean | false | Run in background |
| `validate_only` | boolean | false | Only run validation |

**Response (sync):**

```json
{
  "deployment_id": "deploy-1704067200",
  "profile_id": "profile-123",
  "status": "success",
  "tier": 2,
  "artifacts": {
    "bundle_manifest": "/path/to/bundle.json",
    "installers": [
      "dist-electron/app-1.0.0.msi",
      "dist-electron/app-1.0.0-mac.pkg",
      "dist-electron/app-1.0.0.AppImage"
    ]
  },
  "duration_ms": 180000
}
```

**Response (async):**

```json
{
  "deployment_id": "deploy-1704067200",
  "profile_id": "profile-123",
  "status": "queued",
  "logs_url": "/api/v1/deployments/deploy-1704067200/logs",
  "status_url": "/api/v1/deployments/deploy-1704067200"
}
```

**Response (dry_run):**

```json
{
  "mode": "dry-run",
  "profile_id": "profile-123",
  "tier": 2,
  "steps": [
    { "step": 1, "action": "validate_profile", "status": "would_execute" },
    { "step": 2, "action": "assemble_bundle_manifest", "status": "would_execute" },
    { "step": 3, "action": "invoke_packager", "packager": "scenario-to-desktop" },
    { "step": 4, "action": "build_installers", "platforms": ["win-x64", "darwin-arm64", "linux-x64"] },
    { "step": 5, "action": "generate_update_manifests" }
  ],
  "estimated_duration": "5-10 minutes"
}
```

---

## GET /deployments/{deployment_id}

Check the status of a deployment.

**Request:**

```bash
curl "http://localhost:${API_PORT}/api/v1/deployments/deploy-1704067200"
```

**Response (in-progress):**

```json
{
  "deployment_id": "deploy-1704067200",
  "profile_id": "profile-123",
  "scenario": "picker-wheel",
  "tier": 2,
  "status": "in-progress",
  "started_at": "2025-01-15T12:00:00Z",
  "current_step": {
    "number": 4,
    "action": "build_installers",
    "details": "Building Windows MSI installer"
  },
  "progress": {
    "completed_steps": 3,
    "total_steps": 5,
    "percentage": 60
  },
  "logs_url": "/api/v1/deployments/deploy-1704067200/logs"
}
```

**Response (success):**

```json
{
  "deployment_id": "deploy-1704067200",
  "profile_id": "profile-123",
  "scenario": "picker-wheel",
  "tier": 2,
  "status": "success",
  "started_at": "2025-01-15T12:00:00Z",
  "completed_at": "2025-01-15T12:03:00Z",
  "duration_ms": 180000,
  "artifacts": {
    "bundle_manifest": "/scenarios/picker-wheel/platforms/electron/bundle/bundle.json",
    "installers": {
      "windows": "dist-electron/picker-wheel-1.0.0.msi",
      "macos": "dist-electron/picker-wheel-1.0.0-mac.pkg",
      "linux": "dist-electron/picker-wheel-1.0.0.AppImage"
    },
    "update_manifests": {
      "windows": "dist-electron/latest.yml",
      "macos": "dist-electron/latest-mac.yml",
      "linux": "dist-electron/latest-linux.yml"
    }
  }
}
```

**Response (failed):**

```json
{
  "deployment_id": "deploy-1704067200",
  "profile_id": "profile-123",
  "status": "failed",
  "started_at": "2025-01-15T12:00:00Z",
  "failed_at": "2025-01-15T12:01:30Z",
  "error": {
    "step": "build_installers",
    "code": "BUILD_FAILED",
    "message": "electron-builder failed: ENOENT package.json",
    "details": "Ensure the Electron app directory exists at platforms/electron/"
  },
  "logs_url": "/api/v1/deployments/deploy-1704067200/logs"
}
```

---

## Deployment Status Values

| Status | Description |
|--------|-------------|
| `queued` | Waiting to start |
| `in-progress` | Currently executing |
| `success` | Completed successfully |
| `failed` | Encountered error |
| `cancelled` | Cancelled by user |

---

## Deployment Steps

Typical deployment executes these steps:

| Step | Action | Description |
|------|--------|-------------|
| 1 | `validate_profile` | Run pre-flight validation |
| 2 | `assemble_bundle_manifest` | Generate bundle.json |
| 3 | `invoke_packager` | Call scenario-to-* |
| 4 | `build_installers` | Run electron-builder |
| 5 | `generate_update_manifests` | Create latest.yml files |

---

## Error Codes

| Code | Description |
|------|-------------|
| `VALIDATION_FAILED` | Profile didn't pass validation |
| `MANIFEST_ASSEMBLY_FAILED` | Couldn't generate bundle.json |
| `PACKAGER_UNAVAILABLE` | Packager scenario not running |
| `BUILD_FAILED` | Installer build failed |
| `TIMEOUT` | Deployment exceeded time limit |

---

## Related

- [CLI Deployment Commands](../cli/deployment-commands.md) - CLI interface
- [Bundle Endpoints](bundles.md) - Manual manifest assembly
- [Desktop Workflow](../workflows/desktop-deployment.md) - Complete workflow
