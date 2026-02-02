## Tools focus: Scenario to Desktop

Convert any Vrooli scenario into a cross-platform desktop application (Windows, macOS, Linux) using the scenario-to-desktop CLI. The tool creates Electron wrappers that connect to your running scenario.

**Current status:** Thin client mode (external-server) is production-ready. Bundled mode is under development.

---

### **1. When to Use This Tool**

| Goal | Command |
|------|---------|
| Build desktop app | `scenario-to-desktop pipeline run <scenario> --platforms linux --wait` |
| Download installer | `scenario-to-desktop download <scenario> <platform>` |
| Collect telemetry | `scenario-to-desktop telemetry ingest <scenario> --file <path>` |
| Configure signing | `scenario-to-desktop signing set <scenario> --config <json>` |

**Scope boundaries:**
- **In scope:** Scenarios with web UIs, Windows/macOS/Linux installers, code signing, telemetry
- **Out of scope:** API-only scenarios, mobile apps, bundled/offline mode (not ready)

---

### **2. Command Reference**

The CLI uses **subcommand groups**. Run `<group> help` for detailed options:

| Group | Purpose | Help command |
|-------|---------|--------------|
| `pipeline` | Build pipeline operations | `scenario-to-desktop pipeline help` |
| `telemetry` | Deployment telemetry | `scenario-to-desktop telemetry help` |
| `signing` | Code signing config | `scenario-to-desktop signing help` |
| `dist` | Distribution targets | `scenario-to-desktop dist help` |
| `wine` | Windows builds on Linux | `scenario-to-desktop wine help` |

**Global options** (all commands):
- `--api-base <url>`: Override API URL
- `--auto-start`: Start scenario if not running
- `--no-color` / `--color`: Control ANSI output

**Flat commands:**
- `status` - Check API health
- `templates` / `template <type>` - List/get desktop templates
- `download <scenario> <platform>` - Download built installer
- `records` / `records-move` / `records-delete` - Manage desktop records
- `configure` - CLI settings

---

### **3. Primary Workflow**

Always use `--wait` for synchronous execution (blocks until complete, proper exit codes):

```bash
# Build desktop app for Linux
scenario-to-desktop pipeline run my-scenario --platforms linux --wait

# Download the built installer
scenario-to-desktop download my-scenario linux --output my-scenario.AppImage
```

**Pipeline stages:** `bundle` → `preflight` → `generate` → `build` → (optional: `distribution`, `smoketest`)

**Common options:**
- `--platforms win,mac,linux` - Target platforms (default: all)
- `--stages generate,build` - Run specific stages only
- `--timeout 900` - Max wait in seconds (default: 600)

---

### **4. Telemetry Workflow**

Collect and analyze telemetry from deployed desktop apps:

```bash
# Ingest telemetry file from user
scenario-to-desktop telemetry ingest my-scenario --file user-telemetry.jsonl

# View summary and AI insights
scenario-to-desktop telemetry summary my-scenario
scenario-to-desktop telemetry insights my-scenario
```

**Telemetry file locations:**
- Windows: `%APPDATA%\<App Name>\deployment-telemetry.jsonl`
- macOS: `~/Library/Application Support/<App Name>/deployment-telemetry.jsonl`
- Linux: `~/.config/<App Name>/deployment-telemetry.jsonl`

---

### **5. Code Signing Workflow**

```bash
# Check available signing tools
scenario-to-desktop signing prerequisites

# Discover existing certificates
scenario-to-desktop signing discover linux

# For Linux, generate GPG key if needed
scenario-to-desktop signing generate-key my-scenario --name "My Company" --email "dev@example.com"

# Set and validate signing config
scenario-to-desktop signing set my-scenario --config @signing.json
scenario-to-desktop signing validate my-scenario
```

---

### **6. Cross-Platform Build Matrix**

Building from Linux (recommended):

| Target | Format | Notes |
|--------|--------|-------|
| Linux | AppImage, DEB | Native |
| Windows | NSIS (.exe) | Via Wine (use `wine check`/`wine install`) |
| macOS | ZIP | Native; DMG/PKG requires actual macOS |

---

### **7. Preflight Prerequisites**

Before running the pipeline, ensure the scenario is fully built. The preflight stage validates:

**Required before pipeline:**
1. **Service binaries** - All API/CLI binaries must be compiled for target platforms
2. **UI assets** - Run `pnpm build` in the UI directory to generate production assets
3. **Bundle manifest** - The `bundle/bundle.json` must reference valid file paths

**Quick pre-flight checklist:**
```bash
# Build scenario (creates binaries + UI)
cd scenarios/<scenario-name>
make build

# Verify binaries exist for your platform
ls -la platforms/electron/bin/api/linux-x64/
ls -la platforms/electron/bin/cli/linux-x64/

# Verify UI is built
ls -la platforms/electron/ui/dist/index.html
```

**Common preflight failures:**

| Error | Cause | Fix |
|-------|-------|-----|
| `service binary validation failed` | Binaries not compiled | Run `make build` in scenario dir |
| `critical validation checks failed` | UI assets missing | Run `pnpm build` in ui/ dir |
| `missing asset` | Bundle manifest references non-existent files | Rebuild or fix manifest paths |

---

### **8. Troubleshooting**

Pipeline errors include structured error codes, recovery hints, and step-by-step guidance.

**When a pipeline fails, the CLI shows:**
- Error code (e.g., `BUNDLE_INVALID`, `SMOKE_TEST_FAILED`)
- Recovery hint with specific fix instructions
- Manual steps to resolve the issue
- Auto-fix commands when available

**Example failure output:**
```
Pipeline failed: 5dd1ca6d-e8a6-8c18-6759-6839539a485c
Error code: BUNDLE_INVALID

Recovery: Add bundle to extraResources in package.json:
  "build": { "extraResources": [{ "from": "bundle", "to": "bundle" }] }

Manual steps:
  1. Check if the bundle directory exists
  2. Edit package.json to add bundle to extraResources
  3. Run the pipeline again after fixing the configuration
```

**Get detailed status for any pipeline:**
```bash
# View full error details including stage logs
scenario-to-desktop pipeline status <id> --verbose

# Get raw JSON for programmatic parsing
scenario-to-desktop pipeline status <id> --json
```

**Common error codes and recovery:**

| Error Code | Meaning | Recovery |
|------------|---------|----------|
| `PREFLIGHT_VALIDATION_FAILED` | Missing binaries or assets | Build scenario first: `make build` |
| `BUNDLE_INVALID` | Bundle config mismatch | Fix package.json extraResources |
| `PREFLIGHT_FAILED` | Scenario not ready | Ensure scenario is running |
| `SMOKE_TEST_FAILED` | App failed validation | Check platform deps (xvfb, etc.) |
| `BUILD_FAILED` | Electron build error | Review build logs |
| `GENERATION_FAILED` | Template error | Check scenario service.json |

**Recovery actions:**
- Transient failures: `scenario-to-desktop pipeline resume <id>`
- Configuration issues: Follow the recovery hint in error output
- Deeper issues: `cd scenarios/scenario-to-desktop && make logs`

**Debug mode:** Use `--debug` flag to see raw API responses:
```bash
scenario-to-desktop pipeline run my-scenario --wait --debug
```

---

### **9. Guardrails**

**Do:**
- Use `--wait` for all scripted/agent workflows
- Verify target scenario is running before generation (preflight checks this)
- Use thin client mode for production

**Do NOT:**
- Use bundled mode in production (not ready)
- Expect MSI builds to work reliably via Wine
- Build DMG/PKG on Linux (requires macOS)

---

### **10. Success Artifacts**

After successful pipeline run:

| Platform | File |
|----------|------|
| Linux | `<scenario>.AppImage` |
| Windows | `<scenario>-Setup.exe` |
| macOS | `<scenario>.zip` (contains .app) |

Download via: `scenario-to-desktop download <scenario> <platform> --output <path>`

---

### **11. Smoke Test Troubleshooting**

When smoke tests fail, check these in order:

| Error Kind | Meaning | First Check |
|------------|---------|-------------|
| artifact | Build artifact missing/corrupted | `ls -la <artifact>`, rebuild |
| platform | Missing system deps | Install xvfb (Linux), check signing (Mac) |
| execution | App crashed | Check stderr for stack traces |
| validation | App ran but didn't pass | Check app logs for connection errors |
| timeout | App too slow | Increase timeout or optimize startup |

**Lifecycle marker diagnosis:**

The smoke test protocol emits markers at each stage. Check `last_lifecycle_state` in error context:

| Last State | Meaning | Likely Cause |
|------------|---------|--------------|
| (empty) | App crashed before smoke test code ran | Electron initialization failure, missing deps |
| `init` | App started smoke test but crashed during initialization | Bundle validation failed, config error |
| `ready` | App initialized but didn't report result | Server connectivity issue, app logic error |
| `result` | App reported result but didn't exit cleanly | Cleanup error (usually non-fatal) |
| `exit` | App completed full lifecycle | Should be success - check for race condition |

**Structured error markers:**

When the app detects specific failure conditions, it emits `SMOKE_TEST_ERROR kind=<kind> msg="<message>"`:

| Kind | Meaning | Resolution |
|------|---------|------------|
| `config` | Missing/invalid configuration | Regenerate desktop app |
| `network` | Server unreachable | Verify server is running, check connectivity |
| `validation` | Bundle validation failed | Review errors, rebuild bundle |
| `runtime` | App crashed during execution | Check logs, verify platform compatibility |

**Example troubleshooting flow:**
```bash
# 1. Check smoke test status for error details
scenario-to-desktop pipeline status <id> --verbose

# 2. Look at last_lifecycle_state in error context
# - If empty: focus on Electron/platform deps
# - If "init": check bundle validation
# - If "ready": check server connectivity

# 3. If error includes app_error_kind, use that for targeted fix
# - config: regenerate with `pipeline run`
# - network: verify server with `curl -I <url>`
# - validation: review bundle validation errors
```
