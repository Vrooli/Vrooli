# Deployment Commands

Commands for validating, packaging, deploying, and monitoring scenarios.

---

## Command Implementation Status

Each command is marked with its current implementation status:

| Status | Meaning |
|--------|---------|
| **Working** | Fully functional |
| **Partial** | Works but with limitations documented below |
| **Stub** | Command exists but returns placeholder data |
| **Planned** | Not yet implemented; use workaround |

### Quick Status Overview

| Command | Status | Notes |
|---------|--------|-------|
| `validate` | **Working** | Full pre-flight validation |
| `deploy` | **Stub** | Dry-run works; actual deployment not orchestrated |
| `deployment status` | **Stub** | Returns mock data |
| `package` | **Partial** | Invokes packager but doesn't assemble manifest first |
| `packagers list` | **Working** | Returns configured packagers |
| `packagers discover` | **Stub** | Returns hardcoded list |
| `logs` | **Working** | Full telemetry filtering |
| `estimate-cost` | **Partial** | AWS only; basic estimates |
| `bundle export` | **Planned** | Use REST API workaround |
| `bundle assemble` | **Planned** | Use REST API workaround |
| `bundle validate` | **Planned** | Use REST API workaround |

---

## Planned Commands (Use REST API)

The following commands are documented in workflows but not yet available in the CLI:

| Planned Command | Purpose | Current Workaround |
|-----------------|---------|-------------------|
| `bundle export <profile-id>` | Export bundle manifest with checksum | Use REST API: `POST /api/v1/bundles/export` |
| `bundle assemble <profile-id>` | Assemble bundle manifest from profile | Use REST API: `POST /api/v1/bundles/assemble` |
| `bundle validate <path>` | Validate a bundle.json file | Use REST API: `POST /api/v1/bundles/validate` |

**REST API Example:**

```bash
# Get API port
API_PORT=$(vrooli scenario port deployment-manager API_PORT)

# Assemble bundle manifest
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/assemble" \
  -H "Content-Type: application/json" \
  -d '{"scenario": "picker-wheel", "tier": "tier-2-desktop"}'

# Export with checksum
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/export" \
  -H "Content-Type: application/json" \
  -d '{"scenario": "picker-wheel", "tier": "tier-2-desktop"}' > bundle.json

# Validate manifest
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/validate" \
  -H "Content-Type: application/json" \
  -d @bundle.json
```

See [Desktop Workflow - Phase 5](../workflows/desktop-deployment.md#phase-5-generate-bundle-manifest) for complete examples.

See [ROADMAP.md](../ROADMAP.md) for implementation tracking.

---

## validate

> **Status: Working** - Fully functional

Run pre-deployment validation checks on a profile.

```bash
deployment-manager validate <profile-id> [--verbose]
```

**Arguments:**
- `<profile-id>` - Profile ID to validate

**Flags:**
- `--verbose` - Show detailed check results

**Example:**

```bash
deployment-manager validate profile-123 --verbose
```

**Output:**

```json
{
  "profile_id": "profile-123",
  "valid": true,
  "checks": [
    {
      "name": "fitness_threshold",
      "status": "pass",
      "details": "Score 70 >= minimum 50 for tier 2"
    },
    {
      "name": "secrets_complete",
      "status": "pass",
      "details": "All 3 required secrets configured"
    },
    {
      "name": "licensing",
      "status": "pass",
      "details": "All dependencies OSS-compatible for commercial use"
    },
    {
      "name": "resource_limits",
      "status": "pass",
      "details": "RAM 256MB within desktop limit (2GB)"
    },
    {
      "name": "platform_binaries",
      "status": "pass",
      "details": "Binaries available for win-x64, darwin-arm64, linux-x64"
    },
    {
      "name": "dependency_swaps",
      "status": "pass",
      "details": "All blocking dependencies have swaps applied"
    }
  ],
  "ready_for_deployment": true
}
```

**Validation Checks:**

| Check | Description |
|-------|-------------|
| `fitness_threshold` | Overall fitness >= 50 for target tier |
| `secrets_complete` | All required secrets configured |
| `licensing` | No license conflicts for target distribution |
| `resource_limits` | RAM/disk within tier limits |
| `platform_binaries` | Required binaries exist for all platforms |
| `dependency_swaps` | All blockers resolved via swaps |

---

## deploy

> **Status: Stub** - Dry-run mode works; actual deployment orchestration not yet implemented

Deploy a profile to its target tier(s).

```bash
deployment-manager deploy <profile-id> [--dry-run] [--async] [--validate-only]
```

**Arguments:**
- `<profile-id>` - Profile ID to deploy

**Flags:**
- `--dry-run` - Simulate deployment without executing
- `--async` - Run deployment in background, return immediately
- `--validate-only` - Only run validation, same as `validate` command

**Example:**

```bash
deployment-manager deploy profile-123 --dry-run
```

**Output:**

```json
{
  "mode": "dry-run",
  "profile_id": "profile-123",
  "scenario": "picker-wheel",
  "tier": 2,
  "steps": [
    {"step": 1, "action": "validate_profile", "status": "would_execute"},
    {"step": 2, "action": "assemble_bundle_manifest", "status": "would_execute"},
    {"step": 3, "action": "invoke_packager", "packager": "scenario-to-desktop", "status": "would_execute"},
    {"step": 4, "action": "build_installers", "platforms": ["win-x64", "darwin-arm64", "linux-x64"], "status": "would_execute"},
    {"step": 5, "action": "generate_update_manifests", "status": "would_execute"}
  ],
  "estimated_duration": "5-10 minutes"
}
```

**Async Deployment:**

```bash
deployment-manager deploy profile-123 --async
```

```json
{
  "deployment_id": "deploy-1704067200",
  "status": "queued",
  "logs_url": "/api/v1/deployments/deploy-1704067200/logs",
  "check_status": "deployment-manager deployment status deploy-1704067200"
}
```

> **Note**: Full deployment orchestration is partially implemented. Use `package` command for direct packager invocation.

---

## deployment status

> **Status: Stub** - Returns mock data; real deployment tracking not yet implemented

Check the status of an async deployment.

```bash
deployment-manager deployment status <deployment-id>
```

**Arguments:**
- `<deployment-id>` - Deployment ID from `deploy --async`

**Output:**

```json
{
  "deployment_id": "deploy-1704067200",
  "profile_id": "profile-123",
  "status": "in-progress",
  "started_at": "2025-01-15T12:00:00Z",
  "current_step": "build_installers",
  "progress": {
    "completed": 3,
    "total": 5,
    "percentage": 60
  },
  "logs_url": "/api/v1/deployments/deploy-1704067200/logs"
}
```

**Status Values:**
- `queued` - Waiting to start
- `in-progress` - Currently executing
- `success` - Completed successfully
- `failed` - Encountered error

---

## package

> **Status: Partial** - Invokes the packager but does NOT assemble the bundle manifest first. You must generate the manifest via REST API before packaging.

Invoke a packager directly for a profile.

**Important limitation:** This command calls the packager but assumes a bundle manifest already exists. For bundled desktop apps, you must first:
1. Generate the manifest via `POST /api/v1/bundles/export`
2. Place it in the correct location
3. Then run `package`

See [Desktop Workflow - Phase 5 & 6](../workflows/desktop-deployment.md#phase-5-generate-bundle-manifest) for the complete workflow.

```bash
deployment-manager package <profile-id> --packager <packager-name> [--dry-run]
```

**Arguments:**
- `<profile-id>` - Profile ID to package

**Flags:**
- `--packager <name>` - Packager to use (required)
- `--dry-run` - Show what would be packaged without executing

**Example:**

```bash
deployment-manager package profile-123 --packager scenario-to-desktop
```

**Output:**

```json
{
  "profile_id": "profile-123",
  "packager": "scenario-to-desktop",
  "status": "success",
  "artifacts": {
    "bundle_dir": "/scenarios/picker-wheel/platforms/electron/bundle",
    "manifest": "/scenarios/picker-wheel/platforms/electron/bundle/bundle.json",
    "runtime_binaries": {
      "win-x64": "bundle/runtime/win-x64/runtime.exe",
      "darwin-arm64": "bundle/runtime/darwin-arm64/runtime",
      "linux-x64": "bundle/runtime/linux-x64/runtime"
    }
  },
  "next_steps": [
    "cd scenarios/picker-wheel/platforms/electron",
    "pnpm install",
    "pnpm run dist:all"
  ]
}
```

---

## packagers / packagers list

> **Status: Working** - Returns configured packagers

List available packagers.

```bash
deployment-manager packagers
# or
deployment-manager packagers list
```

**Output:**

```json
{
  "packagers": [
    {
      "name": "scenario-to-desktop",
      "status": "available",
      "tiers": [2],
      "description": "Generate Electron desktop apps (Windows/macOS/Linux)",
      "api_url": "http://localhost:8091"
    },
    {
      "name": "scenario-to-ios",
      "status": "not_running",
      "tiers": [3],
      "description": "Generate iOS applications"
    },
    {
      "name": "scenario-to-android",
      "status": "not_running",
      "tiers": [3],
      "description": "Generate Android applications"
    },
    {
      "name": "scenario-to-saas",
      "status": "not_implemented",
      "tiers": [4],
      "description": "Deploy to cloud infrastructure"
    }
  ]
}
```

---

## packagers discover

> **Status: Stub** - Returns hardcoded list; does not actually scan for packagers

Discover new packagers by scanning running scenarios.

```bash
deployment-manager packagers discover
```

**Output:**

```json
{
  "discovered": [
    {
      "name": "scenario-to-extension",
      "tiers": [2],
      "description": "Generate browser extensions",
      "api_url": "http://localhost:8095"
    }
  ],
  "total_available": 2
}
```

---

## logs

> **Status: Working** - Full telemetry filtering and search

Fetch deployment logs and telemetry for a profile.

```bash
deployment-manager logs <profile-id> [--level <level>] [--search <term>] [--format json|table]
```

**Arguments:**
- `<profile-id>` - Profile ID

**Flags:**
- `--level <level>` - Filter by level: `debug`, `info`, `warn`, `error`
- `--search <term>` - Search logs for term
- `--format` - Output format (default: json)

**Example:**

```bash
deployment-manager logs profile-123 --level error --format table
```

**Output:**

```
TIMESTAMP                 LEVEL   EVENT                    DETAILS
2025-01-15T12:00:00Z     error   migration_failed         Schema version mismatch
2025-01-15T12:01:30Z     error   dependency_unreachable   sqlite service not responding
2025-01-15T12:02:45Z     error   secrets_missing          SESSION_SECRET not provided
```

**JSON Output:**

```json
{
  "profile_id": "profile-123",
  "scenario": "picker-wheel",
  "logs": [
    {
      "timestamp": "2025-01-15T12:00:00Z",
      "level": "error",
      "event": "migration_failed",
      "details": {
        "expected_version": "2025-01-01",
        "actual_version": "2024-12-01",
        "migration": "add_user_preferences"
      }
    }
  ],
  "summary": {
    "total": 150,
    "errors": 3,
    "warnings": 12,
    "info": 135
  }
}
```

---

## estimate-cost

> **Status: Partial** - AWS estimates only; other providers not yet implemented

Estimate costs for SaaS/cloud deployments.

```bash
deployment-manager estimate-cost <profile-id> [--verbose]
```

**Arguments:**
- `<profile-id>` - Profile ID (must target tier 4 or 5)

**Flags:**
- `--verbose` - Show detailed breakdown

**Example:**

```bash
deployment-manager estimate-cost profile-456 --verbose
```

**Output:**

```json
{
  "profile_id": "profile-456",
  "tier": 4,
  "estimates": {
    "monthly": {
      "compute": 45.00,
      "storage": 10.00,
      "bandwidth": 15.00,
      "total": 70.00,
      "currency": "USD"
    },
    "annual": {
      "total": 840.00,
      "with_reserved": 672.00,
      "savings": 168.00
    }
  },
  "breakdown": {
    "compute": {
      "instance_type": "t3.medium",
      "vcpu": 2,
      "memory_gb": 4,
      "hours": 730,
      "rate_per_hour": 0.0416
    },
    "storage": {
      "type": "gp3",
      "size_gb": 100,
      "rate_per_gb": 0.10
    },
    "bandwidth": {
      "egress_gb": 150,
      "rate_per_gb": 0.10
    }
  },
  "recommendations": [
    "Consider reserved instances for 20% savings",
    "Enable auto-scaling to reduce off-peak costs"
  ]
}
```

> **Note**: Cost estimation is currently available for AWS. Additional providers coming soon.

---

## Related

- [Deployment Guide](../DEPLOYMENT-GUIDE.md) - Single entry point for deploying scenarios
- [Profile Commands](profile-commands.md) - Create and configure profiles
- [Desktop Workflow](../workflows/desktop-deployment.md) - Full desktop deployment guide
- [Bundle API Reference](../api/bundles.md) - REST API for bundle operations
- [Troubleshooting](../workflows/troubleshooting.md) - Common deployment issues
- [Roadmap](../ROADMAP.md) - Implementation status and planned work
