## Steer focus: Cross-Platform Readiness

Prioritize **preparing scenarios for deployment beyond Tier 1** (local Vrooli stack) in `scenarios/{{TARGET}}/`. This skill steers toward eliminating assumptions that break on desktop, mobile, cloud, or enterprise deployments.

Your goal is to ensure `{{TARGET}}` can be **bundled, distributed, and run** on platforms where the Vrooli monorepo, lifecycle system, and shared resources are not available. Changes should make the scenario self-contained and adaptable to different runtime contexts.

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve the scenario's portability.

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
- Filesystem path handling (sandbox-safe, cross-OS)
- Build configuration (CGO, static binaries, cross-compilation)
- Network/IPC patterns (localhost variants, port allocation)
- Data storage portability (postgres→sqlite swaps)
- Secret classification (infrastructure vs per-install)

**Out of scope:**
- Tier-specific UI implementation (Electron IPC, mobile native) → tier guides
- Code signing and distribution → scenario-to-desktop skill
- Database schema design → storage-steer skill
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
    return filepath.Join(os.Getenv("VROOLI_ROOT"), "scenarios", "my-scenario", "config")
}
// Crashes in desktop mode: VROOLI_ROOT is empty
```

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

**Desktop bundles:** The Electron main.ts must inject these before spawning:

```typescript
// In Electron main.ts (automatically handled by scenario-to-desktop)
runtimeEnv.API_PORT = String(PORTS.api.port);
runtimeEnv.UI_PORT = String(PORTS.ui.port);
runtimeEnv.VROOLI_LIFECYCLE_MANAGED = "true";
runtimeEnv.VROOLI_DESKTOP_MODE = "true";
```

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

| Resource | Tier 2 Desktop | Tier 3 Mobile | Tier 4 Cloud | Swap To |
|----------|----------------|---------------|--------------|---------|
| **PostgreSQL** | 0.3 (Wine/native) | 0.0 (impossible) | 0.95 | **SQLite** |
| **Redis** | 0.5 (embedded possible) | 0.3 (limited) | 0.9 | In-memory cache |
| **Ollama** | 0.0 (always-on server) | 0.0 | 0.95 | **OpenRouter API** |
| **Browserless** | 0.6 (bundled driver) | 0.0 | 0.9 | Bundled Playwright |
| **Neo4j** | 0.6 (heavy) | 0.0 | 0.95 | SQLite + JSON |
| **MinIO** | 0.7 (can embed) | 0.4 | 0.9 | Filesystem + cloud |
| **Qdrant** | 0.5 (resource-heavy) | 0.2 | 0.9 | SQLite FTS |

**Fitness scoring:** >= 0.9 = ship-ready, < 0.6 = swap required

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

See `resources/sqlite/README.md` for comprehensive SQLite guidance.

#### 3.3 Decision Tree: Resource Swapping

```
                    Does {{TARGET}} depend on this resource?
                                        │
                                       YES
                                        │
                        ┌───────────────┴───────────────┐
                        ▼                               ▼
                Is it storage?                    Is it AI/ML?
                (postgres, redis)                 (ollama, etc.)
                        │                               │
                        ▼                               ▼
            ┌───────────┴───────────┐         ┌───────┴───────┐
            │                       │         │               │
      Structured data?      Cache/session?   Local model?   API-based?
            │                       │         │               │
            ▼                       ▼         ▼               ▼
    → SQLite (pure Go)     → In-memory    → OpenRouter   → OK (portable)
      with modernc.org       or SQLite      or remote
                              for TTL        inference
```

#### 3.4 Swap Declaration in Bundle Manifest

When preparing for bundling, document swaps:

```json
{
  "swaps": [
    {
      "original": "postgres",
      "replacement": "sqlite",
      "reason": "Desktop bundle cannot run PostgreSQL server",
      "limitations": "Single-user file store; no concurrent writers"
    },
    {
      "original": "ollama",
      "replacement": "openrouter",
      "reason": "Desktop cannot run always-on LLM server",
      "limitations": "Requires internet; API costs"
    }
  ]
}
```

---

### 4. Filesystem Portability

#### 4.1 Path Handling Rules

| Pattern | Problem | Portable Alternative |
|---------|---------|---------------------|
| `~/.vrooli/` | Not expanded on Windows | `os.UserHomeDir()` + join |
| `/tmp/file` | Different on each OS | `os.TempDir()` |
| `./relative` | Depends on CWD | Relative to `BUNDLE_ROOT` or exe path |
| Hardcoded `/home/user` | Wrong user, wrong OS | `os.UserHomeDir()` |
| Unix path separators | Breaks on Windows | `filepath.Join()` |

```go
// ✅ CORRECT: Cross-platform path handling
func getTempPath(filename string) string {
    return filepath.Join(os.TempDir(), "{{TARGET}}", filename)
}

func getConfigPath() string {
    dataRoot := getDataRoot()
    return filepath.Join(dataRoot, "{{TARGET}}", "config.json")
}

// ❌ WRONG: Platform-specific assumptions
func getConfigPath() string {
    return "/home/user/.vrooli/my-scenario/config.json"
}
```

#### 4.2 Sandbox Compliance

Desktop apps (especially macOS with notarization) have restricted filesystem access:

```
ALLOWED (Desktop Sandbox):
├── APP_DATA_DIR/         # Electron sets this
│   └── {{TARGET}}/       # Scenario-specific data
├── BUNDLE_ROOT/          # Read-only bundled assets
└── User-selected paths   # Via file dialog only

FORBIDDEN (will fail notarization):
├── /usr/, /bin/, /etc/   # System directories
├── Other apps' data      # Privacy violation
└── Arbitrary user paths  # Without file dialog
```

**Steer:** All writes should go to `APP_DATA_DIR` or paths explicitly selected by the user via file dialogs.

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

// ✅ ALSO CORRECT: Request from allocator
port, err := portAllocator.Allocate("api", PortRange{Min: 18000, Max: 19000})

// ❌ WRONG: Hardcoded port
listener, _ := net.Listen("tcp", ":8080") // May conflict
```

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
cd scenarios/{{TARGET}}/api && CGO_ENABLED=0 go build ./... 2>&1

# Hardcoded paths
rg "~/.vrooli|/home/|/Users/" scenarios/{{TARGET}}/
rg "\"/tmp" scenarios/{{TARGET}}/ --type go

# Fixed ports without configuration
rg ":8080|:3000|:5000" scenarios/{{TARGET}}/ | grep -v "getEnv\|PORT"

# Resource dependencies in service.json
cat scenarios/{{TARGET}}/.vrooli/service.json | jq '.dependencies.resources'
```

#### 8.2 Red Flags Checklist

- [ ] Uses `VROOLI_ROOT` without `VROOLI_DESKTOP_MODE` check
- [ ] Uses `SCENARIO_ROOT` without fallback to `BUNDLE_ROOT`
- [ ] Uses `VROOLI_DATA` without `APP_DATA_DIR` fallback
- [ ] Depends on PostgreSQL without SQLite alternative
- [ ] Depends on Redis without in-memory fallback
- [ ] Depends on Ollama without remote inference option
- [ ] Uses `github.com/mattn/go-sqlite3` (CGO required)
- [ ] Has CGO imports that block static builds
- [ ] Hardcoded paths with `/home/`, `~/.vrooli/`, `/tmp/`
- [ ] Fixed port numbers without environment override
- [ ] `localhost` without `127.0.0.1` alternative in CORS
- [ ] Infrastructure secrets required at runtime (not swapped for desktop)

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
| Resource | Fitness (Desktop) | Swap Required? | Swap To |
|----------|-------------------|----------------|---------|
| | | | |

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

To ensure **systematic coverage without repetition**, use `visited-tracker`:

**At the start of each iteration:**
```bash
visited-tracker least-visited \
  --location scenarios/{{TARGET}} \
  --pattern "**/*.{go,ts}" \
  --tag cross-platform-readiness \
  --name "{{TARGET}} - Cross-Platform Readiness" \
  --limit 5
```

**After analyzing each file:**
```bash
visited-tracker visit <file-path> \
  --location scenarios/{{TARGET}} \
  --tag cross-platform-readiness \
  --note "<summary: env var fallbacks added, SQLite swap implemented, etc.>"
```

**When a file is irrelevant (no portability concerns):**
```bash
visited-tracker exclude <file-path> \
  --location scenarios/{{TARGET}} \
  --tag cross-platform-readiness \
  --reason "Pure business logic, no platform dependencies"
```

**Before ending your session:**
```bash
visited-tracker campaigns note \
  --location scenarios/{{TARGET}} \
  --tag cross-platform-readiness \
  --name "{{TARGET}} - Cross-Platform Readiness" \
  --note "<overall progress, remaining portability issues>"
```

---

### 10. Documentation and Memory Loop

#### 10.1 At Session Start

Read existing portability documentation:
- `scenarios/{{TARGET}}/.vrooli/service.json` - Resource dependencies
- `scenarios/{{TARGET}}/docs/internal/PORTABILITY_AUDIT.md` - Prior findings (if exists)
- `resources/sqlite/README.md` - If database swap is needed
- `scenarios/deployment-manager/docs/guides/fitness-scoring.md` - Fitness criteria

#### 10.2 At Session End

Update `scenarios/{{TARGET}}/docs/internal/PORTABILITY_AUDIT.md`:
- The code is the source of truth. Verify existing claims against actual code.
- Correct any inaccuracies discovered.
- Update tier compatibility status based on work completed.
- Note resource swaps implemented or still needed.
- Create the `docs/internal/` directory if needed.

---

### 11. Maintain Scenario Constraints

* Do **not** change `{{TARGET}}`'s core workflows, APIs, or business logic
* Do **not** introduce new features unrelated to portability
* Do **not** remove Tier 1 (local stack) functionality while adding portability
* Do **not** swap resources without documenting limitations
* Do **not** break existing tests - portability changes should be additive
* Prefer **feature detection** over tier-specific code paths
* Keep **abstractions minimal** - don't over-engineer for hypothetical platforms

---

### 12. Output Expectations

You may update in `scenarios/{{TARGET}}/`:
- Add environment variable fallback chains
- Add `VROOLI_DESKTOP_MODE` detection and handling
- Swap database drivers (postgres→sqlite with modernc.org driver)
- Add resource abstraction interfaces for swappable backends
- Update service.json with `offline_capable` flag and limitations
- Add path resolution helpers using `os.UserHomeDir()`, `os.TempDir()`
- Update CORS to accept both localhost variants
- Update Makefile/build scripts for static builds

You must:
- Preserve all Tier 1 (local stack) functionality
- Use `modernc.org/sqlite` for SQLite (not `go-sqlite3`)
- Ensure `CGO_ENABLED=0 go build` works after changes
- Document resource swap limitations in manifest
- Update `PORTABILITY_AUDIT.md` with changes made

You must NOT:
- Remove support for Vrooli lifecycle environment variables
- Hardcode desktop-specific paths (use `APP_DATA_DIR` env var)
- Add tier-specific code without feature detection pattern
- Remove resource dependencies without providing alternative
- Break Tier 1 testing to support other tiers

**Avoid superficial changes that rename variables or restructure code without materially improving cross-platform readiness.**
