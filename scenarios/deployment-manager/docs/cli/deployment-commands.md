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
| `deploy-desktop` | **Working** | Full end-to-end desktop deployment pipeline |
| `build` | **Working** | Cross-compile service binaries for all platforms |
| `validate` | **Working** | Full pre-flight validation |
| `deploy` | **Stub** | Generic deploy (use `deploy-desktop` for tier 2) |
| `deployment status` | **Stub** | Returns mock data |
| `package` | **Stub** | Deprecated shim; prints guidance to use `deploy-desktop` |
| `packagers list` | **Stub** | Deprecated; returns static packager hints |
| `packagers discover` | **Stub** | Same static output; no discovery performed |
| `logs` | **Working** | Full telemetry filtering |
| `estimate-cost` | **Partial** | AWS only; basic estimates |
| `bundle assemble` | **Working** | Assemble bundle manifest from scenario |
| `bundle export` | **Working** | Export production-ready manifest with checksum |
| `bundle validate` | **Working** | Validate bundle.json against schema |

> **Recommended for Desktop Deployment**: Use `deploy-desktop` - it orchestrates the entire pipeline (manifest, binaries, Electron wrapper, installers) in a single command.

---

## deploy-desktop

> **Status: Working** - Full end-to-end desktop deployment pipeline

Orchestrate the complete bundled desktop deployment workflow. This is the **recommended command** for deploying scenarios as desktop applications.

```bash
deployment-manager deploy-desktop --profile <profile-id> [flags]
```

**Flags:**
- `--profile <id>` - Profile ID (required)
- `--output <dir>` - Output directory for bundle (default: scenario/platforms/electron/bundle)
- `--platforms <list>` - Comma-separated platforms to build: win,mac,linux (default: all)
- `--mode <mode>` - Deployment mode: bundled, external-server, cloud-api (default: bundled)
- `--skip-build` - Skip service binary compilation
- `--skip-validation` - Skip profile validation
- `--skip-packaging` - Skip Electron wrapper generation (manifest + binaries only)
- `--skip-installers` - Skip building platform installers (MSI/PKG/AppImage)
- `--timeout <duration>` - Override orchestration timeout (default: 10m). Accepts Go durations (e.g., 5m, 15m).
- `--dry-run` - Show what would be done without executing
- `--format json` - Output as JSON

**Pipeline Steps:**

The command executes a 7-step pipeline:

1. **Load profile** - Load deployment profile configuration
2. **Validate profile** - Check fitness score, secrets, dependencies, and **blocker resolution**
3. **Assemble manifest** - Generate bundle.json with profile swaps applied
4. **Export manifest** - Write manifest to output directory
5. **Build binaries** - Cross-compile service binaries for all platforms
6. **Generate desktop wrapper** - Create Electron app via scenario-to-desktop
7. **Build platform installers** - Create MSI/PKG/AppImage/DEB packages

**Blocker Validation:**

Step 2 automatically checks for blocking dependencies that cannot be bundled directly:

| Blocking Dependency | Required Swap | Reason |
|---------------------|---------------|--------|
| `postgres` | `sqlite` | File-based storage without network dependency |
| `redis` | `in-process` | Memory-only cache without separate process |
| `browserless` | `playwright-driver` | Bundled Chromium for offline browser automation |
| `n8n` | `embedded-workflows` | Pre-compiled workflow execution |
| `qdrant` | `faiss-local` | File-based vector search |

If blockers exist without swaps, validation fails with actionable guidance:
```
Error: unresolved blockers for desktop deployment: postgres (swap to sqlite), redis (swap to in-process).
Run 'deployment-manager swaps list my-scenario' to see available swaps,
then apply with 'deployment-manager swaps apply <profile-id> <from> <to>'
```

Use `--skip-validation` to bypass blocker checking (not recommended for production).

**Example - Full Pipeline:**

```bash
# Create a profile first
deployment-manager profile create my-profile my-scenario --tier 2

# Run the full deployment
deployment-manager deploy-desktop --profile my-profile
```

**Output:**

```
✓ Desktop Deployment: success
  Profile:  my-profile
  Scenario: my-scenario
  Duration: 4m32s

Steps:
  ✓ Load profile - loaded profile for scenario my-scenario
  ✓ Validate profile - profile validation passed
  ✓ Assemble manifest - assembled manifest with 0 swaps
  ✓ Export manifest - wrote manifest to .../bundle.json
  ✓ Build binaries - built 1 service(s) for 5 platform(s)
  ✓ Generate desktop wrapper - generated Electron wrapper at .../platforms/electron
  ✓ Build platform installers - built installers for 3 platform(s)

Manifest: /home/user/Vrooli/scenarios/my-scenario/bundle.json
Binaries: 5/5 succeeded
Desktop Wrapper: /home/user/Vrooli/scenarios/my-scenario/platforms/electron

Installers:
  win:     .../dist-electron/My App Setup 1.0.0.exe
  mac:     .../dist-electron/My App-1.0.0-mac.zip
  linux:   .../dist-electron/My App-1.0.0.AppImage
```

**Example - Partial Pipeline:**

```bash
# Preview without executing
deployment-manager deploy-desktop --profile my-profile --dry-run

# Build manifest and binaries only (no Electron packaging)
deployment-manager deploy-desktop --profile my-profile --skip-packaging

# Build for specific platforms
deployment-manager deploy-desktop --profile my-profile --platforms win,linux

# Generate wrapper but skip final installer builds
deployment-manager deploy-desktop --profile my-profile --skip-installers
```

**Prerequisites:**
- deployment-manager API running
- scenario-to-desktop API running (for steps 6-7)
- Go 1.21+ installed (for binary compilation)
- Node.js 18+ installed (for Electron builds)
- pnpm installed (Electron builds prefer pnpm; falls back to npm if unavailable)

**Related:**
- [Hello Desktop Tutorial](../tutorials/hello-desktop-walkthrough.md) - Complete walkthrough with hello-desktop scenario
- [Desktop Deployment Workflow](../workflows/desktop-deployment.md) - Detailed phase documentation

---

## build

> **Status: Working** - Cross-compile service binaries for all platforms

Cross-compile service binaries for desktop bundling. This command is automatically called by `deploy-desktop`, but can be used standalone.

```bash
deployment-manager build --profile <profile-id> [flags]
```

**Flags:**
- `--profile <id>` - Profile ID (required unless --scenario specified)
- `--scenario <name>` - Scenario name (optional if profile specified)
- `--platforms <list>` - Comma-separated platforms: linux-x64,darwin-arm64,win-x64 (default: all)
- `--services <list>` - Comma-separated service IDs to build (default: all with build config)
- `--dry-run` - Show what would be built without building
- `--format json` - Output as JSON

**Supported Platforms:**
- `linux-x64` - Linux x86_64
- `linux-arm64` - Linux ARM64
- `darwin-x64` - macOS Intel
- `darwin-arm64` - macOS Apple Silicon
- `win-x64` - Windows x86_64

**Supported Build Types:**

Build configuration is read from each service's `build` field in service.json or bundle.json:

| Type | Description | Requirements |
|------|-------------|--------------|
| `go` | Go cross-compilation | Go 1.21+ |
| `rust` | Cargo build with target triple | Rust toolchain |
| `npm` | Node.js build (pkg, nexe, etc.) | Node.js 18+ |
| `custom` | Arbitrary shell command | Depends on command |

**Example Build Config:**

```json
{
  "build": {
    "type": "go",
    "source_dir": "api",
    "entry_point": "./cmd/api",
    "output_pattern": "bin/{{platform}}/api{{ext}}",
    "args": ["-ldflags", "-s -w"],
    "env": {"CGO_ENABLED": "0"}
  }
}
```

**Example:**

```bash
# Build all services for all platforms
deployment-manager build --profile my-profile

# Build for specific platforms
deployment-manager build --profile my-profile --platforms linux-x64,darwin-arm64

# Build specific services
deployment-manager build --profile my-profile --services api,worker

# Dry run
deployment-manager build --profile my-profile --dry-run
```

**Output:**

```
Build success for my-scenario
Duration: 45s

✓ Service: api
  ✓ linux-x64      bin/linux-x64/api
  ✓ linux-arm64    bin/linux-arm64/api
  ✓ darwin-x64     bin/darwin-x64/api
  ✓ darwin-arm64   bin/darwin-arm64/api
  ✓ win-x64        bin/win-x64/api.exe
```

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

> **Status: Stub (deprecated)** - Compatibility shim that no longer calls packager APIs. Use `deploy-desktop` for real bundling.

Invoke a legacy packager entry point. The command now returns a stub response to avoid 404s against missing packager endpoints and points users to the supported desktop pipeline.

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
  "status": "stubbed",
  "message": "Package command is legacy-only; use deploy-desktop for end-to-end bundling."
}
```

---

## packagers / packagers list

> **Status: Stub (deprecated)** - Returns static hints; discovery is not performed.

List the known packagers without hitting any API endpoints (kept for CLI compatibility).

```bash
deployment-manager packagers
# or
deployment-manager packagers list
```

**Output:**

```json
{
  "status": "stubbed",
  "message": "Packager discovery is deprecated here; use deploy-desktop or scenario-to-* CLIs directly.",
  "packagers": [
    "scenario-to-desktop (desktop bundler)",
    "scenario-to-ios (stub)",
    "scenario-to-cloud (stub)"
  ]
}
```

---

## packagers discover

> **Status: Stub (deprecated)** - Same static output as `packagers list`.

Discover command retained for compatibility; returns the same stub payload as `packagers list`.

```bash
deployment-manager packagers discover
```

**Output:** Same as `packagers list` (static stub payload).

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

## bundle

> **Status: Working** - All subcommands fully functional

Bundle manifest operations for desktop deployments.

```bash
deployment-manager bundle <subcommand> [arguments]
```

**Subcommands:**
- `assemble` - Generate bundle manifest from scenario
- `export` - Export production-ready manifest with checksum
- `validate` - Validate manifest file against schema

---

### bundle assemble

> **Status: Working** - Fully functional

Assemble a bundle manifest from a scenario.

```bash
deployment-manager bundle assemble <scenario> [--tier <tier>] [--include-secrets] [--output <file>]
```

**Arguments:**
- `<scenario>` - Name of the scenario to bundle

**Flags:**
- `--tier <tier>` - Target tier (default: desktop). Options: desktop, mobile, saas, enterprise
- `--include-secrets` - Include secret configuration in manifest (default: true)
- `--output <file>` - Write manifest to file instead of stdout
- `--format <fmt>` - Output format (json)

**Example:**

```bash
# Assemble and print to stdout
deployment-manager bundle assemble picker-wheel

# Assemble and save to file
deployment-manager bundle assemble picker-wheel --output manifest.json

# Assemble without secrets (for debugging)
deployment-manager bundle assemble picker-wheel --include-secrets=false
```

**Output:**

```json
{
  "status": "assembled",
  "schema": "desktop.v0.1",
  "manifest": {
    "schema_version": "v0.1",
    "target": "desktop",
    "app": {"name": "picker-wheel", "version": "1.2.3"},
    "services": [...],
    "secrets": [...],
    "swaps": [...]
  }
}
```

---

### bundle export

> **Status: Working** - Fully functional

Export a production-ready bundle manifest with SHA256 checksum.

```bash
deployment-manager bundle export <scenario> [--tier <tier>] [--output <file>] [--manifest-only]
```

**Arguments:**
- `<scenario>` - Name of the scenario to export

**Flags:**
- `--tier <tier>` - Target tier (default: desktop)
- `--include-secrets` - Include secret configuration (default: true)
- `--output <file>` - Write manifest to file (recommended)
- `--manifest-only` - Output only the manifest, without metadata wrapper
- `--format <fmt>` - Output format (json)

**Example:**

```bash
# Export to file (recommended for bundled apps)
deployment-manager bundle export picker-wheel --output bundle.json

# Export manifest only (for piping to other tools)
deployment-manager bundle export picker-wheel --manifest-only > bundle.json

# Export with full metadata to stdout
deployment-manager bundle export picker-wheel
```

**Output (with --output):**

```
Bundle manifest exported to bundle.json
  Scenario:    picker-wheel
  Tier:        tier-2-desktop
  Schema:      desktop.v0.1
  Checksum:    a1b2c3d4e5f6...
  Generated:   2025-01-15T12:00:00Z
```

**Output (JSON):**

```json
{
  "status": "exported",
  "schema": "desktop.v0.1",
  "scenario": "picker-wheel",
  "tier": "tier-2-desktop",
  "manifest": {...},
  "checksum": "a1b2c3d4e5f6...",
  "generated_at": "2025-01-15T12:00:00Z"
}
```

---

### bundle validate

> **Status: Working** - Fully functional

Validate a bundle manifest file against the v0.1 schema.

```bash
deployment-manager bundle validate <file> [--format <fmt>]
```

**Arguments:**
- `<file>` - Path to the bundle.json manifest file

**Flags:**
- `--format <fmt>` - Output format (json)

**Example:**

```bash
# Validate a manifest file
deployment-manager bundle validate ./bundle.json

# Validate with JSON output
deployment-manager bundle validate ./bundle.json --format json
```

**Output (success):**

```
Manifest is valid
  File:   ./bundle.json
  Schema: desktop.v0.1
```

**Output (failure):**

```
validation failed: api error (400): bundle failed validation: services array is empty
```

**Validation checks:**
- Schema version is "v0.1"
- Target is "desktop"
- App name and version are present
- At least one service is defined
- Each service has valid health and readiness checks
- Secret classes are valid (per_install_generated, user_prompt, remote_fetch)
- IPC mode is "loopback-http"

---

## Related

- [Deployment Guide](../DEPLOYMENT-GUIDE.md) - Single entry point for deploying scenarios
- [Profile Commands](profile-commands.md) - Create and configure profiles
- [Desktop Workflow](../workflows/desktop-deployment.md) - Full desktop deployment guide
- [Bundle API Reference](../api/bundles.md) - REST API for bundle operations
- [Troubleshooting](../workflows/troubleshooting.md) - Common deployment issues
- [Roadmap](../ROADMAP.md) - Implementation status and planned work
