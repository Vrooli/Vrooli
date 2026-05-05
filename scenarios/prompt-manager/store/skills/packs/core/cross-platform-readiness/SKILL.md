## Steer focus: Cross-Platform Readiness

Prioritize **preparing scenarios for deployment beyond Tier 1** (local Vrooli stack) in `scenarios/{{TARGET}}/`. This skill steers toward eliminating assumptions that break on desktop, mobile, cloud, or enterprise deployments.

Your goal is to ensure `{{TARGET}}` can be **bundled, distributed, and run** on platforms where the Vrooli monorepo, lifecycle system, and shared resources are not available. Changes should make the scenario self-contained and adaptable to different runtime contexts.

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve the scenario's portability.

Required reading:
- `prompt-manager skill read storage-steer visited-tracker-tools`

`storage-steer` is the storage-architecture authority — this skill picks *which* engine each tier needs; `storage-steer` decides *how* to architect it (per-domain schema, repository pattern, migration tier). The two skills share boundaries deliberately and cross-reference each other.

Optional reading:
- `prompt-manager skill read brand-manager` (draft — branding validation for deployment readiness)

---

### 0. Why This Skill Exists

Scenarios built for Tier 1 (local Vrooli stack) often make assumptions that fail catastrophically on other platforms:

**Environment Variable Assumptions:**
- `VROOLI_ROOT` not set in desktop bundles → Go binary crashes with "env var required"
- `SCENARIO_ROOT` assumed to exist → path resolution fails
- Hardcoded paths like `~/.vrooli/` → wrong location on Windows, sandboxed on mobile

**Resource Dependencies:**
- PostgreSQL assumed available → desktop users don't have database servers
- Redis for caching → mobile apps can't run Redis
- Ollama for AI → requires always-on local server (unacceptable for desktop)

**Filesystem Assumptions:**
- Write to arbitrary paths → fails on sandboxed platforms (macOS notarization, iOS)
- Assume Unix paths → breaks on Windows (`/` vs `\`)
- Assume home directory structure → different on every platform

**Network Assumptions:**
- localhost always works → not true in containers/mobile
- Fixed ports available → conflicts on shared systems
- Service discovery via Vrooli lifecycle → not available in bundles

**The Vrooli deployment tiers:**

| Tier | Platform | Key Constraints |
|------|----------|-----------------|
| **1** | Local Stack | Full Vrooli - all resources available |
| **2** | Desktop (Electron) | Self-contained bundle, no external dependencies |
| **3** | Mobile (iOS/Android) | Strict sandboxing, no server processes |
| **4** | Cloud/SaaS | Container isolation, managed services |
| **5** | Enterprise/Appliance | Air-gapped, custom resource mapping |

This skill ensures scenarios work across **all tiers** by eliminating tier-specific assumptions.

---

### 1. Scope Boundaries

**In scope:**
- Environment variable usage patterns (fallbacks, detection flags)
- Resource dependency analysis and swap recommendations
- Filesystem runtime storage portability via `package:api-core/storage`
- Build configuration (CGO, static binaries, cross-compilation)
- Network/IPC patterns (localhost variants, port allocation)
- Data storage portability (postgres→sqlite swaps)
- Secret classification (infrastructure vs per-install)

**Out of scope:**
- Tier-specific UI implementation (Electron IPC, mobile native) → tier guides
- Code signing and distribution → scenario-to-desktop skill
- Storage architecture (per-domain schema, repository pattern, migration tier) → storage-steer skill
- Performance optimization → performance skills
- Actual bundle creation → deployment-manager

---

### 2. Environment Variable Portability

#### 2.1 The Core Problem

Vrooli lifecycle injects environment variables that **do not exist** in bundled contexts:

| Variable | Tier 1 (Local) | Tier 2+ (Bundled) |
|----------|----------------|-------------------|
| `VROOLI_ROOT` | Monorepo path | **NOT SET** |
| `SCENARIO_ROOT` | Scenario path | **NOT SET** |
| `VROOLI_DATA` | `~/.vrooli/data` | `APP_DATA_DIR` (Electron sets this) |
| `API_PORT`, etc. | Lifecycle allocated | **Must be injected by wrapper** |
| `POSTGRES_*` | Resource defaults | **NOT AVAILABLE** (use SQLite) |

#### 2.2 Portable Environment Variable Pattern

```go
// ✅ CORRECT: Fallback chain with desktop mode detection
func getDataRoot() string {
    // Desktop bundles set APP_DATA_DIR via Electron
    if appData := os.Getenv("APP_DATA_DIR"); appData != "" {
        return appData
    }
    // Vrooli lifecycle sets VROOLI_DATA
    if vrooliData := os.Getenv("VROOLI_DATA"); vrooliData != "" {
        return vrooliData
    }
    // Fallback: user home directory
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".vrooli", "data")
}

// ✅ CORRECT: Desktop mode detection
func isDesktopMode() bool {
    return os.Getenv("VROOLI_DESKTOP_MODE") == "true"
}

func resolveScenarioAssets() string {
    if isDesktopMode() {
        // Bundled: assets are relative to bundle
        return os.Getenv("BUNDLE_ROOT")
    }
    // Local development: use monorepo paths
    return os.Getenv("SCENARIO_ROOT")
}
```

```go
// ❌ WRONG: Hard requirement on monorepo env vars
func getConfig() string {
    return requireEnv("VROOLI_ROOT")
}
// Crashes in desktop mode: VROOLI_ROOT is empty
```

When code genuinely needs canonical monorepo layout semantics in a Go runtime, use repo-contract-backed helpers instead of joining `VROOLI_ROOT` with `scenarios/...` manually. The canonical Go adapter is `path:packages/repo-contract-go`.

#### 2.3 Decision Tree: Environment Variable Usage

```
                    Does the code use VROOLI_ROOT or SCENARIO_ROOT?
                                        │
                    ┌───────────────────┴───────────────────┐
                   YES                                      NO
                    │                                        │
                    ▼                                        ▼
    What is it trying to locate?                    Is VROOLI_DATA used?
            │                                              │
    ┌───────┼───────┐                              ┌───────┴───────┐
    │       │       │                             YES              NO
    │       │       │                              │                │
    ▼       ▼       ▼                              ▼                ▼
  Assets  Config  Other                   Check for fallback    Continue
    │       │       │                              │
    │       │       │                     ┌────────┴────────┐
    │       │       │                    YES               NO
    │       │       │                     │                 │
    ▼       ▼       ▼                     ▼                 ▼
  Use     Use    Add              OK - portable       Add fallback:
BUNDLE_ROOT DATA_DIR VROOLI_DESKTOP_MODE                APP_DATA_DIR ||
  env var  env var   check                              VROOLI_DATA ||
                                                       ~/.vrooli/data
```

#### 2.4 Port Environment Variables

For scenarios with services that listen on ports:

```json
// service.json - ports with env_var names
"ports": {
  "api": { "env_var": "API_PORT", "port": 18700 },
  "ui": { "env_var": "UI_PORT", "port": 36400 }
}
```

**Desktop bundles:** The Electron main.ts injects these automatically (handled by scenario-to-desktop template generation - you don't write this code).

**Scenario code:** Should have sensible defaults, not hard requirements:

```go
// ✅ CORRECT: Default port with env override
port := getEnvInt("API_PORT", 18700)

// ❌ WRONG: Hard requirement with no default
port := requireEnv("API_PORT") // Crashes if not set
```

---

### 3. Resource Dependency Portability

#### 3.1 Resource Fitness by Tier

| Resource | Tier 2 Desktop | Tier 3 Mobile | Tier 4 Cloud | Portable Alternative |
|----------|----------------|---------------|--------------|----------------------|
| **PostgreSQL** | 0.3 (Wine/native) | 0.0 (impossible) | 0.95 | **SQLite** (pure Go) |
| **Redis** | 0.5 (embedded possible) | 0.3 (limited) | 0.9 | In-memory cache |
| **Ollama** | 0.0 (always-on server) | 0.0 | 0.95 | **OpenRouter** (as fallback) |
| **Browserless** | 0.6 (bundled driver) | 0.0 | 0.9 | Bundled Playwright |
| **Neo4j** | 0.6 (heavy) | 0.0 | 0.95 | SQLite + JSON |
| **MinIO** | 0.7 (can embed) | 0.4 | 0.9 | Filesystem + cloud |
| **Qdrant** | 0.5 (resource-heavy) | 0.2 | 0.9 | SQLite FTS |

**Fitness scoring:** >= 0.9 = ship-ready, < 0.6 = needs alternative

**Special case - Ollama:** Unlike databases, Ollama's value is being free and local. Don't replace it - make it **optional** with OpenRouter as a fallback for desktop/mobile where Ollama isn't available. Users who want free local AI can still install Ollama; others pay for OpenRouter convenience.

#### 3.2 SQLite as Portable Database

**Critical:** Use pure-Go SQLite driver for cross-compilation:

```go
// ✅ CORRECT: Pure Go driver (CGO_ENABLED=0 compatible)
import _ "modernc.org/sqlite"
db, err := sql.Open("sqlite", dbPath)

// ❌ WRONG: CGO-dependent driver
import _ "github.com/mattn/go-sqlite3"
db, err := sql.Open("sqlite3", dbPath) // Fails with CGO_ENABLED=0
```

**SQLite path resolution:**

```go
func getSQLitePath() string {
    dataRoot := getDataRoot() // Uses fallback chain from 2.2
    return filepath.Join(dataRoot, "{{TARGET}}", "database.db")
}
```

Use `modernc.org/sqlite` with the `sqlite` driver name for embedded scenario storage. For scenario runtime architecture, prefer embedded SQLite in the scenario rather than a standalone SQLite resource.

Once the engine is chosen, `storage-steer` covers the architecture: schema lives next to the code that interprets it (`internal/<dom>/schema.sql` embedded via `go:embed`, applied at boot via `database.EnsureSchemas` from `path:packages/api-core/database`), repository interfaces hide the engine, and the same per-domain rule applies to Qdrant collections and Redis namespaces.

#### 3.3 Full Replacement vs Runtime Swap

There are two strategies for handling non-portable resources:

| Strategy | Description | When to Use |
|----------|-------------|-------------|
| **Full Replacement** | Remove non-portable resource entirely, use only portable one | Portable resource is "good enough" for all use cases |
| **Runtime Swap** | Support both resources, select at runtime based on environment | Non-portable resource offers significant advantages you actually need |

**Decision Tree: Replacement vs Swap**

```
        Does the non-portable resource offer features you ACTUALLY USE
        that the portable alternative cannot provide?
                                │
                ┌───────────────┴───────────────┐
               NO                              YES
                │                               │
                ▼                               ▼
        FULL REPLACEMENT              What features specifically?
        (simpler, preferred)                    │
                                    ┌───────────┴───────────┐
                                    │                       │
                            Concurrent writes?      Advanced queries?
                            (multi-user)            (JSON ops, FTS)
                                    │                       │
                                    ▼                       ▼
                            Is this a              Can you work
                            multi-user app?        around it?
                                    │                       │
                            ┌───────┴───────┐       ┌───────┴───────┐
                           YES             NO      YES             NO
                            │               │       │               │
                            ▼               ▼       ▼               ▼
                      RUNTIME SWAP    FULL REPL  FULL REPL    RUNTIME SWAP
```

**Examples:**

| Scenario | Resource | Decision | Reasoning |
|----------|----------|----------|-----------|
| Simple CRUD app | PostgreSQL | **Full replacement** → SQLite | No concurrent writes needed, SQLite is fine |
| Multi-user SaaS + desktop | PostgreSQL | **Runtime swap** | Cloud needs concurrent writes, desktop is single-user |
| AI chat app | Ollama | **Runtime swap** (optional) | Ollama free+local is valuable; OpenRouter as fallback |
| Workflow storage | PostgreSQL | **Full replacement** → SQLite | Workflows are per-user, no need for postgres |

**Steer toward full replacement** unless you have a compelling reason to maintain both. Supporting multiple backends adds complexity, tests, and potential bugs. If SQLite handles your use case, just use SQLite everywhere.

#### 3.4 Decision Tree: Which Portable Alternative?

```
                    What type of resource is non-portable?
                                        │
                        ┌───────────────┴───────────────┐
                        ▼                               ▼
                Is it storage?                    Is it AI/ML?
                (postgres, redis)                 (ollama, etc.)
                        │                               │
                        ▼                               ▼
            ┌───────────┴───────────┐         Make Ollama OPTIONAL
            │                       │         with OpenRouter fallback
      Structured data?      Cache/session?    (preserves free+local value)
            │                       │
            ▼                       ▼
    → SQLite (pure Go)     → In-memory cache
      with modernc.org       or SQLite for
                              persistent sessions
```

#### 3.5 Documenting Resource Strategy

Document your resource strategy in service.json or bundle manifest:

```json
// For FULL REPLACEMENT - embed SQLite in the scenario
{
  "environment": {
    "MYAPP_SQLITE_PATH": "${SCENARIO_DATA_DIR}/myapp.db"
  },
  "notes": {
    "storage_strategy": "Embedded SQLite via modernc.org/sqlite; no standalone SQLite resource"
  }
}

// For RUNTIME SWAP - document both with selection logic
{
  "dependencies": {
    "resources": {
      "postgres": {
        "type": "postgres",
        "enabled": true,
        "required": false,
        "description": "Production storage (Tier 1/4)"
      }
    }
  },
  "environment": {
    "MYAPP_SQLITE_PATH": "${SCENARIO_DATA_DIR}/myapp.db"
  },
  "notes": {
    "storage_selection": "Uses STORAGE_BACKEND env var; defaults to embedded sqlite if postgres unavailable"
  }
}

// For OPTIONAL RESOURCE with fallback (Ollama pattern)
{
  "dependencies": {
    "resources": {
      "ollama": {
        "type": "ollama",
        "enabled": true,
        "required": false,
        "description": "Local AI inference (free, optional)"
      }
    }
  },
  "capabilities": {
    "ai_providers": ["ollama", "openrouter"],
    "ai_fallback": "openrouter",
    "ai_note": "Ollama preferred (free+local); OpenRouter used when Ollama unavailable"
  }
}
```

---

### 4. Filesystem Portability

#### 4.1 Canonical Filesystem Contract

For scenario runtime storage, adopt `github.com/vrooli/api-core/storage` instead of custom path logic.

Why:
- It encapsulates OS-aware roots (`auto`, `desktop`, `mobile`, `vps`)
- It separates mutable state from disposable deploy/app directories
- It provides traversal-safe relative path joins and atomic writes

#### 4.2 Required Pattern

```go
import "github.com/vrooli/api-core/storage"

resolver, err := storage.NewResolver(storage.ResolverConfig{
    AppID:   "vrooli",
    Profile: storage.ProfileAuto,
})
if err != nil {
    return err
}

_, err = storage.EnsureAllDirs(resolver, storage.Options{
    ScenarioID: "{{TARGET}}",
}, 0)
if err != nil {
    return err
}

path, err := resolver.Path(
    storage.Options{ScenarioID: "{{TARGET}}"},
    storage.ClassState,
    "runtime.json",
)
if err != nil {
    return err
}
return storage.WriteFileAtomic(path, payload, storage.DefaultFilePerm)
```

#### 4.3 Anti-Patterns

- Hardcoded absolute paths (`$HOME/...`, `os.TempDir()/...`, `C:\\...`)
- Scenario-local mutable writes (`./data`, `./state`) under app/deploy targets
- Hand-rolled `DATA_DIR` resolution or custom traversal checks when `package:api-core/storage` is available

#### 4.4 Tracked Source Assets

Do not confuse tracked scenario-authored source files with runtime state.

If a file is edited through a UI or tool but the intended result is a shared, reviewable change to scenario behavior, keep it in the repo in an explicit source directory such as:

- `config/`
- `policy/`

Use `package:api-core/storage` only for local runtime state, not for versioned source artifacts.

Reserve `.vrooli/` for repo or manifest metadata rather than as a generic bucket for checked-in configuration.

---

### 5. Build Portability

#### 5.1 Static Binary Requirements

Desktop bundles require static binaries:

```bash
# ✅ CORRECT: Static build for bundling
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o api-linux
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o api-darwin
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o api-windows.exe

# ❌ WRONG: CGO-dependent build
go build -o api  # May link against system libraries
```

#### 5.2 CGO Dependency Audit

Find CGO dependencies that block static builds:

```bash
# Check for CGO imports
rg "github.com/mattn/go-sqlite3" scenarios/{{TARGET}}/
rg "// #cgo" scenarios/{{TARGET}}/
rg "import \"C\"" scenarios/{{TARGET}}/

# Test static build
cd scenarios/{{TARGET}}/api && CGO_ENABLED=0 go build ./...
# If this fails, you have CGO dependencies to eliminate
```

#### 5.3 TypeScript/Frontend Build

Ensure UI builds are self-contained:

```bash
# Build should not reference external dev servers
rg "localhost:[0-9]+" scenarios/{{TARGET}}/ui/src --type ts
# Should only find conditional/configurable references, not hardcoded

# Check for dev-only dependencies in production build
rg "process.env.NODE_ENV" scenarios/{{TARGET}}/ui/src
# All dev checks should have proper fallbacks
```

---

### 6. Network Portability

#### 6.1 Localhost Variants

Different platforms resolve `localhost` differently:

```go
// ✅ CORRECT: Accept both localhost forms
allowedOrigins := []string{
    fmt.Sprintf("http://localhost:%d", uiPort),
    fmt.Sprintf("http://127.0.0.1:%d", uiPort),
}

// ❌ WRONG: Only one form
allowedOrigin := fmt.Sprintf("http://localhost:%d", uiPort)
// Breaks on Windows which sometimes uses 127.0.0.1
```

#### 6.2 Dynamic Port Allocation

Don't assume specific ports are available:

```go
// ✅ CORRECT: Configurable with sensible default
port := getEnvInt("API_PORT", 18700)

// ❌ WRONG: Hardcoded port
listener, _ := net.Listen("tcp", ":8080") // May conflict
```

**Note:** Port injection for bundled desktop apps is handled automatically by scenario-to-desktop's template generation. You don't need to write Electron code - just ensure your scenario code reads port values from environment variables with sensible defaults.

#### 6.3 Offline Capability

Mark whether the scenario works offline:

```json
// service.json
{
  "capabilities": {
    "offline_capable": false,
    "offline_limitations": "AI features require OpenRouter API"
  }
}
```

---

### 7. Secret Classification

Secrets must be classified for bundling:

| Class | Example | Bundleable? | Handling |
|-------|---------|-------------|----------|
| **Infrastructure** | DB password, API keys | **NO** | Reference only in manifest |
| **Per-install generated** | JWT secret, session key | YES | Generate on first launch |
| **User-prompted** | User's API key | YES | Prompt in UI, store locally |
| **Remote-fetch** | License key | Partial | Requires network |

```json
// Bundle manifest secret classification
{
  "secrets": [
    {
      "name": "JWT_SECRET",
      "class": "per_install_generated",
      "generator": "random_base64_32"
    },
    {
      "name": "OPENROUTER_API_KEY",
      "class": "user_prompted",
      "prompt": "Enter your OpenRouter API key (optional for AI features)"
    },
    {
      "name": "POSTGRES_PASSWORD",
      "class": "infrastructure",
      "note": "Not needed - using SQLite for desktop"
    }
  ]
}
```

---

### 8. Cross-Platform Readiness Audit

Before making changes, assess `{{TARGET}}`'s portability posture.

#### 8.1 Audit Commands

```bash
# Environment variable hard dependencies
rg "requireEnv|os\.Getenv\(\"VROOLI_ROOT" scenarios/{{TARGET}}/ --type go
rg "os\.Getenv\(\"SCENARIO_ROOT" scenarios/{{TARGET}}/ --type go
rg "process\.env\.VROOLI_ROOT" scenarios/{{TARGET}}/ --type ts

# Missing fallback patterns
rg "os\.Getenv\(" scenarios/{{TARGET}}/api --type go | grep -v "||"
# Lines without || likely have no fallback

# CGO dependencies
rg "mattn/go-sqlite3|\"C\"|#cgo" scenarios/{{TARGET}}/

# Test static build
cd scenarios/{{TARGET}}/api && CGO_ENABLED=0 go build ./...

# Filesystem storage contract adoption
rg "storage\\.NewResolver|storage\\.EnsureAllDirs|storage\\.EnsureClassDir|storage\\.WriteFileAtomic|\\.Path\\(" scenarios/{{TARGET}}/api --type go

# Direct filesystem writes / ad hoc path policy (anti-pattern)
rg "os\\.WriteFile|ioutil\\.WriteFile|os\\.Create\\(|os\\.OpenFile\\(" scenarios/{{TARGET}}/api --type go
rg "DATA_DIR|filepath\\.Join\\(\\s*\"\\.\"\\s*,\\s*\"data\"|/tmp|/home/|/Users/" scenarios/{{TARGET}}/

# Fixed ports without configuration
rg ":8080|:3000|:5000" scenarios/{{TARGET}}/ | grep -v "getEnv\|PORT"

# Resource dependencies in service.json
cat scenarios/{{TARGET}}/.vrooli/service.json | jq '.dependencies.resources'
```

#### 8.2 Red Flags Checklist

**Environment Variables:**
- [ ] Uses `VROOLI_ROOT` without `VROOLI_DESKTOP_MODE` check
- [ ] Uses `SCENARIO_ROOT` without fallback to `BUNDLE_ROOT`
- [ ] Uses `VROOLI_DATA` without `APP_DATA_DIR` fallback
- [ ] Port env vars with no default (crashes if not set)

**Resource Dependencies:**
- [ ] Depends on PostgreSQL with no portable alternative (SQLite)
- [ ] Depends on Redis with no in-memory fallback
- [ ] Depends on Ollama with no fallback (should be optional with OpenRouter)
- [ ] Runtime swap complexity where full replacement would suffice

**Build & Compilation:**
- [ ] Uses `github.com/mattn/go-sqlite3` (CGO required)
- [ ] Has CGO imports that block static builds
- [ ] `CGO_ENABLED=0 go build` fails

**Filesystem & Paths:**
- [ ] Runtime filesystem writes bypass `package:api-core/storage`
- [ ] Mutable files stored under scenario deploy/app directories
- [ ] Custom `DATA_DIR` policy used instead of shared storage resolver

**Network:**
- [ ] Fixed port numbers without environment override
- [ ] `localhost` without `127.0.0.1` alternative in CORS

**Secrets:**
- [ ] Infrastructure secrets required at runtime (DB passwords, etc.)

**Branding (deployment readiness):**
- [ ] Has a proper display name (not just the scenario slug)
- [ ] Has a logo (SVG + rasterized)
- [ ] Has a favicon (multi-size)
- [ ] Has a color system (primary, secondary, accent with WCAG validation)
- [ ] Has typography defined (heading, body, mono fonts)

#### 8.3 Document Findings

Record audit results in `scenarios/{{TARGET}}/docs/internal/PORTABILITY_AUDIT.md`:

```markdown
# {{TARGET}} Cross-Platform Readiness Audit

## Last Updated
[Date]

## Target Tiers
- [ ] Tier 2 Desktop (Electron)
- [ ] Tier 3 Mobile
- [ ] Tier 4 Cloud/SaaS
- [ ] Tier 5 Enterprise

## Environment Variable Status
| Variable | Usage | Fallback? | Desktop-Ready? |
|----------|-------|-----------|----------------|
| VROOLI_ROOT | | | |
| VROOLI_DATA | | | |
| SCENARIO_ROOT | | | |

## Resource Dependencies
| Resource | Fitness (Desktop) | Strategy | Alternative | Reasoning |
|----------|-------------------|----------|-------------|-----------|
| | | Full Replace / Runtime Swap / Optional | | |

## Build Status
- [ ] CGO_ENABLED=0 builds successfully
- [ ] Cross-compilation tested (linux, darwin, windows)
- [ ] No hardcoded platform-specific paths

## Network Status
- [ ] Ports configurable via environment
- [ ] CORS accepts localhost and 127.0.0.1
- [ ] Offline capability documented

## Issues Found
1. [File:line] - Issue description
2. ...

## Required Changes for Tier 2 (Desktop)
1. [Change description] - [Files affected]
2. ...

## Required Changes for Tier 3 (Mobile)
1. ...
```

---

### 9. Memory Management with Visited Tracker

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}` and TAG set to `cross-platform-readiness`.

---

### 10. Documentation and Memory Loop

#### 10.1 At Session Start

Read existing portability documentation:
- `scenarios/{{TARGET}}/.vrooli/service.json` - Resource dependencies
- `scenarios/{{TARGET}}/docs/internal/PORTABILITY_AUDIT.md` - Prior findings (if exists)
- `packages/api-core/README.md` - If shared SQLite driver/env guidance is needed
- `packages/api-core/docs/storage.md` - Filesystem runtime storage contract
- `scenarios/deployment-manager/docs/guides/fitness-scoring.md` - Fitness criteria

#### 10.2 At Session End

Update `scenarios/{{TARGET}}/docs/internal/PORTABILITY_AUDIT.md`:
- The code is the source of truth. Verify existing claims against actual code.
- Correct any inaccuracies discovered.
- Update tier compatibility status based on work completed.
- Note resource swaps implemented or still needed.
- Create the `path:docs/internal/` directory if needed.

---

### 11. Output Expectations

You may update in `scenarios/{{TARGET}}/`:
- Add environment variable fallback chains
- Add `VROOLI_DESKTOP_MODE` detection and handling
- Swap database drivers (postgres→sqlite with modernc.org driver)
- Add resource abstraction interfaces for swappable backends
- Update service.json with `offline_capable` flag and limitations
- Adopt `package:api-core/storage` for runtime filesystem paths
- Update CORS to accept both localhost variants
- Update Makefile/build scripts for static builds

You must:
- Preserve all Tier 1 (local stack) functionality
- Use `modernc.org/sqlite` for SQLite (not `go-sqlite3`)
- Ensure `CGO_ENABLED=0 go build` works after changes
- Route mutable runtime filesystem state through `package:api-core/storage`
- Document resource swap limitations in manifest
- Update `PORTABILITY_AUDIT.md` with changes made

You must NOT:
- Remove support for Vrooli lifecycle environment variables
- Hardcode custom runtime storage roots in scenario code
- Store mutable runtime files under scenario deploy/app directories
- Add tier-specific code without feature detection pattern
- Remove resource dependencies without providing alternative
- Break Tier 1 testing to support other tiers

**Avoid superficial changes that rename variables or restructure code without materially improving cross-platform readiness.**
