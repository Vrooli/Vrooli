# Bundle Pipeline Architecture

> **How deployment-manager orchestrates bundle creation across multiple scenarios.**
>
> This document explains the data flow between scenarios during bundle creation and identifies where manual intervention is currently required.

---

## Overview

Creating a bundled desktop app requires coordination between four scenarios:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Bundle Creation Pipeline                        │
└─────────────────────────────────────────────────────────────────────────┘

     User Request                                              Installer
         │                                                         ▲
         ▼                                                         │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    │
│  deployment-    │───▶│   scenario-     │───▶│  scenario-to-   │────┘
│    manager      │    │  dependency-    │    │    desktop      │
│                 │    │   analyzer      │    │                 │
│  (orchestrator) │    │  (DAG + sizing) │    │  (packager)     │
└────────┬────────┘    └─────────────────┘    └─────────────────┘
         │
         │    ┌─────────────────┐
         └───▶│ secrets-manager │
              │ (classification)│
              └─────────────────┘
```

---

## Scenario Responsibilities

### 1. deployment-manager (Orchestrator)

**Purpose**: Drives the workflow, stores profiles, assembles manifests

**Inputs**:
- User's scenario name and target tier
- Swap decisions from user
- Secrets configuration

**Outputs**:
- Deployment profile (persisted)
- Bundle manifest (`bundle.json`)
- Validation results

**Key endpoints**:
```
POST /api/v1/profiles              - Create profile
POST /api/v1/bundles/assemble      - Assemble manifest
POST /api/v1/bundles/export        - Export with checksum
GET  /api/v1/fitness/score         - Calculate fitness
```

**Location**: `/scenarios/deployment-manager/`

---

### 2. scenario-dependency-analyzer (DAG + Sizing)

**Purpose**: Compute the full dependency graph and resource requirements

**Inputs**:
- Scenario name
- Recursively reads `.vrooli/service.json` for each dependency

**Outputs**:
- Dependency DAG (directed acyclic graph)
- Resource requirements (RAM, disk, CPU, GPU)
- Fitness scores per tier
- Swap suggestions

**Key endpoints**:
```
GET /api/v1/analyze/{scenario}     - Full dependency analysis
GET /api/v1/dag/{scenario}         - Just the dependency graph
```

**Data flow**:
```
scenario-dependency-analyzer
         │
         ├── Reads: scenarios/<name>/.vrooli/service.json
         ├── Reads: resources/<name>/.vrooli/service.json
         │
         └── Returns:
              {
                "dependencies": {
                  "resources": ["postgres", "redis"],
                  "scenarios": ["secrets-manager"]
                },
                "requirements": {
                  "ram_mb": 1536,
                  "disk_mb": 4200
                },
                "fitness": {
                  "tier-2-desktop": 0.45
                },
                "swaps": [
                  {"from": "postgres", "to": "sqlite", "impact": "+25"}
                ]
              }
```

**Location**: `/scenarios/scenario-dependency-analyzer/`

---

### 3. secrets-manager (Classification)

**Purpose**: Classify secrets and generate bundle-safe plans

**Inputs**:
- Scenario name
- Target tier

**Outputs**:
- Secret classification (`infrastructure`, `per_install_generated`, `user_prompt`, `remote_fetch`)
- Generator configs for auto-generated secrets
- Prompt configs for user-provided secrets
- Validation that no infrastructure secrets leak

**Key endpoints**:
```
GET  /api/v1/secrets/{scenario}           - List all secrets
GET  /api/v1/secrets/{scenario}/bundle    - Bundle-safe secrets only
POST /api/v1/secrets/classify             - Classify a secret
```

**Data flow**:
```
secrets-manager
         │
         └── Returns for tier-2-desktop:
              {
                "secrets": [
                  {
                    "id": "jwt_secret",
                    "class": "per_install_generated",
                    "generator": {"type": "random", "length": 32}
                  },
                  {
                    "id": "postgres_password",
                    "class": "infrastructure",
                    "action": "exclude"  // Must be swapped out
                  }
                ]
              }
```

**Location**: `/scenarios/secrets-manager/`

---

### 4. scenario-to-desktop (Packager)

**Purpose**: Generate Electron wrapper and build installers

**Inputs**:
- Bundle manifest (`bundle.json`)
- Pre-compiled binaries (currently manual)
- UI bundle (currently manual)

**Outputs**:
- Electron app with runtime supervisor
- Platform installers (MSI, PKG, AppImage, DEB)
- Update manifests (latest.yml, etc.)

**Key endpoints**:
```
POST /api/v1/desktop/generate/quick    - Generate thin client
POST /api/v1/desktop/generate/bundled  - Generate bundled app (planned)
GET  /api/v1/desktop/packages          - List built installers
```

**Components**:
```
scenario-to-desktop/
├── api/                  # Generation API
│   ├── bundle_packager.go    - Stages binaries/assets
│   └── handlers_desktop.go   - HTTP handlers
├── runtime/              # Bundled runtime supervisor
│   ├── supervisor.go         - Service orchestration
│   ├── manifest/             - Manifest parsing
│   ├── health/               - Health monitoring
│   ├── secrets/              - Secret injection
│   └── ports/                - Port allocation
└── templates/            # Electron templates
    └── vanilla/
        ├── main.ts           - Electron main process
        └── package.json.template
```

**Location**: `/scenarios/scenario-to-desktop/`

---

## Data Flow: Complete Bundle Creation

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         Bundle Creation Sequence                          │
└──────────────────────────────────────────────────────────────────────────┘

1. USER creates profile
   │
   └──▶ deployment-manager POST /api/v1/profiles
        Returns: profile-123

2. deployment-manager queries dependencies
   │
   └──▶ scenario-dependency-analyzer GET /api/v1/analyze/picker-wheel
        Returns: DAG, requirements, fitness scores, swap suggestions

3. USER applies swaps via CLI
   │
   └──▶ deployment-manager PUT /api/v1/profiles/profile-123
        Updates: swaps = {"postgres": "sqlite"}

4. deployment-manager queries secrets
   │
   └──▶ secrets-manager GET /api/v1/secrets/picker-wheel/bundle?tier=desktop
        Returns: Bundle-safe secrets with generators/prompts

5. deployment-manager assembles manifest
   │
   └──▶ deployment-manager POST /api/v1/bundles/assemble
        Combines: DAG + swaps + secrets + service definitions
        Returns: bundle.json

6. deployment-manager exports manifest
   │
   └──▶ deployment-manager POST /api/v1/bundles/export
        Returns: bundle.json + checksum

   ════════════════════════════════════════════════════════════════
   │  MANUAL STEPS CURRENTLY REQUIRED                              │
   ════════════════════════════════════════════════════════════════

7. USER builds binaries (MANUAL)
   │
   └──▶ go build for each platform (GOOS/GOARCH)

8. USER places assets (MANUAL)
   │
   └──▶ Copy binaries, UI bundle, seeds to bundle/ directory

9. USER triggers packaging
   │
   └──▶ scenario-to-desktop POST /api/v1/desktop/generate/bundled
        OR: pnpm run dist:all

10. scenario-to-desktop builds installers
    │
    └──▶ electron-builder produces MSI/PKG/AppImage/DEB
```

---

## Current Gaps in Pipeline

### Gap 1: Profile → Manifest Disconnect

**Issue**: `bundles/assemble` takes a scenario name, not a profile ID. Swaps configured in the profile aren't automatically applied to the manifest.

**Current workaround**: Swaps are passed separately in the request body.

**Future fix**: `POST /api/v1/bundles/assemble` should accept `profile_id` and read swaps/settings from the profile.

---

### Gap 2: No Automated Binary Compilation

**Issue**: The packager expects pre-compiled binaries but doesn't build them.

**Current workaround**: Manual `go build` for each platform.

**Future fix**: scenario-to-desktop should:
1. Read manifest to find Go source paths
2. Cross-compile for all target platforms
3. Stage binaries in bundle directory

---

### Gap 3: No Asset Procurement

**Issue**: Assets like Chromium (for Playwright) or model files must be manually downloaded.

**Current workaround**: Download/build assets before packaging.

**Future fix**: Manifest should reference asset URLs with checksums; packager downloads and verifies.

---

### Gap 4: deployment-manager → scenario-to-desktop Handoff

**Issue**: No direct integration between these scenarios. Bundle manifest must be manually copied.

**Current workaround**:
1. Export manifest from deployment-manager
2. Copy to `scenarios/<name>/platforms/electron/bundle/`
3. Trigger scenario-to-desktop

**Future fix**:
- Option A: `deployment-manager package` calls scenario-to-desktop API directly
- Option B: Shared filesystem location both scenarios watch
- Option C: Message queue for build requests

---

## API Contract Summary

### deployment-manager → scenario-dependency-analyzer

```
GET /api/v1/analyze/{scenario}

Response: {
  "scenario": "picker-wheel",
  "dependencies": {
    "resources": ["postgres", "redis"],
    "scenarios": []
  },
  "requirements": {...},
  "fitness": {...},
  "swaps": [...]
}
```

### deployment-manager → secrets-manager

```
GET /api/v1/secrets/{scenario}/bundle?tier=tier-2-desktop

Response: {
  "secrets": [
    {
      "id": "jwt_secret",
      "class": "per_install_generated",
      "generator": {...},
      "target": {"type": "env", "name": "JWT_SECRET"}
    }
  ],
  "excluded": [
    {"id": "postgres_password", "reason": "infrastructure"}
  ]
}
```

### deployment-manager → scenario-to-desktop (Future)

```
POST /api/v1/desktop/package

Request: {
  "profile_id": "profile-123",
  "manifest_path": "/path/to/bundle.json",
  "platforms": ["win-x64", "darwin-arm64", "linux-x64"],
  "compile_binaries": true,
  "sign": false
}

Response: {
  "build_id": "build-456",
  "status": "queued",
  "logs_url": "/api/v1/builds/build-456/logs"
}
```

---

## Environment Variables

Each scenario uses environment variables to locate others:

| Variable | Used By | Default |
|----------|---------|---------|
| `ANALYZER_BASE_URL` | deployment-manager | Auto-detected via lifecycle |
| `SECRETS_MANAGER_URL` | deployment-manager | Auto-detected via lifecycle |
| `DESKTOP_PACKAGER_URL` | deployment-manager | Auto-detected via lifecycle |

Auto-detection uses: `vrooli scenario port <scenario> API_PORT`

---

## Related Documentation

- [DEPLOYMENT-GUIDE.md](../DEPLOYMENT-GUIDE.md) - User-facing deployment guide
- [Desktop Workflow](../workflows/desktop-deployment.md) - Step-by-step instructions
- [Bundle Manifest Schema](../guides/bundle-manifest-schema.md) - Manifest reference
- [ROADMAP.md](../ROADMAP.md) - Implementation status
- [Bundled Runtime Plan](/docs/plans/bundled-desktop-runtime-plan.md) - Technical architecture
