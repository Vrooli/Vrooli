## Tools focus: Scenario to Desktop

Convert any Vrooli scenario into a cross-platform desktop application (Windows, macOS, Linux) using the scenario-to-desktop CLI. The tool creates Electron wrappers that connect to your running scenario.

---

### **1. When to Use This Tool**

| Goal | Command |
|------|---------|
| Build desktop app | `scenario-to-desktop pipeline run <scenario> --platforms linux --clean --wait` |
| Build + deploy to LPBS | `scenario-to-desktop pipeline run <scenario> --deploy-to <lpbs> --remote-profile <tag> --app-key <key> --wait` |
| Download installer | `scenario-to-desktop download <scenario> <platform>` |
| Collect telemetry | `scenario-to-desktop telemetry ingest <scenario> --file <path>` |
| Configure signing | `scenario-to-desktop signing set <scenario> --config <json>` |

**Scope boundaries:**
- **In scope:** Scenarios with web UIs, Windows/macOS/Linux installers, code signing, telemetry, LPBS deployment
- **Out of scope:** API-only scenarios, mobile apps, bundled/offline mode (not ready)

---

### **2. Command Reference**

The CLI uses **subcommand groups**. Run `<group> help` for detailed options:

| Group | Purpose | Help command |
|-------|---------|--------------|
| `pipeline` | Build pipeline operations | `scenario-to-desktop pipeline help` |
| `telemetry` | Deployment telemetry | `scenario-to-desktop telemetry help` |
| `signing` | Code signing config | `scenario-to-desktop signing help` |
| `deploy-target` | Manage LPBS deploy targets | `scenario-to-desktop deploy-target help` |
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

Always use `--clean` to ensure fresh builds, and `--wait` for synchronous execution (blocks until complete, proper exit codes):

```bash
# Build desktop app for Linux
scenario-to-desktop pipeline run {{TARGET}} --platforms linux --clean --wait
```

**Pipeline stages:** `bundle` → `preflight` → `generate` → `build` → `smoketest` → (optional: `deploy`)

**Common options:**
- `--platforms win,mac,linux` - Target platforms (default: all)
- `--stages generate,build` - Run specific stages only
- `--timeout 900` - Max wait in seconds (default: 600)

**Deploy options** (all optional — deploy stage is skipped if none provided):
- `--deploy-target <name>` - Saved deploy target (see section 4)
- `--deploy-to <scenario>` - LPBS scenario name (inline, requires `--remote-profile`)
- `--remote-profile <tag>` - Remote profile tag on LPBS (inline)
- `--app-key <key>` - Download app key (required when deploying)

**Build + deploy example:**
```bash
scenario-to-desktop pipeline run {{TARGET}} --platforms linux --clean --wait \
  --deploy-to landing-page-business-suite --remote-profile prod --app-key my-app
```

---

### **4. Deploy Workflow**

The deploy stage uploads built artifacts to a remote LPBS instance and derives an auto-update URL. It is **always optional** — if no deploy flags are provided, the stage is silently skipped.

**Only deploy when the user explicitly requests it and provides target info.**

#### Two approaches

**Inline (one-off, preferred):**
```bash
scenario-to-desktop pipeline run {{TARGET}} --platforms linux --clean --wait \
  --deploy-to landing-page-business-suite --remote-profile prod --app-key {{APP_KEY}}
```

**Saved target (repeated deploys):**
```bash
# Save a target once
scenario-to-desktop deploy-target add prod \
  --scenario landing-page-business-suite \
  --profile prod \
  --label "Production"

# Use it in pipeline runs
scenario-to-desktop pipeline run {{TARGET}} --platforms linux --clean --wait \
  --deploy-target prod --app-key {{APP_KEY}}
```

#### Deploy target management

```bash
scenario-to-desktop deploy-target list
scenario-to-desktop deploy-target add <name> --scenario <s> --profile <p> [--label <l>]
scenario-to-desktop deploy-target remove <name>
scenario-to-desktop deploy-target test <name> [--require-service-auth]
```

Selector note:
- Use the target key (`<name>`, for example `prod`) when testing/running.
- `deploy-target list` shows both key and label to avoid key/label confusion.

#### Prerequisites for deploy

1. `LPBS_SERVICE_SECRET` env var set (same secret as the LPBS instance)
2. Local LPBS running (discovered automatically via api-core/discovery)
3. Remote profile on local LPBS with an active session (logged in)
4. Download app registered on the remote LPBS with the given `app_key`

See `landing-page-deploy-setup` for LPBS prerequisite setup, and `landing-page-desktop-upload` for end-to-end orchestration.

---

### **5. Telemetry Workflow**

Collect and analyze telemetry from deployed desktop apps:

```bash
# Ingest telemetry file from user
scenario-to-desktop telemetry ingest {{TARGET}} --file user-telemetry.jsonl

# View summary and AI insights
scenario-to-desktop telemetry summary {{TARGET}}
scenario-to-desktop telemetry insights {{TARGET}}
```

**Telemetry file locations:**
- Windows: `%APPDATA%\<App Name>\deployment-telemetry.jsonl`
- macOS: `~/Library/Application Support/<App Name>/deployment-telemetry.jsonl`
- Linux: `~/.config/<App Name>/deployment-telemetry.jsonl`

---

### **6. Code Signing Workflow**

```bash
# Check available signing tools
scenario-to-desktop signing prerequisites

# Discover existing certificates
scenario-to-desktop signing discover linux

# For Linux, generate GPG key if needed
scenario-to-desktop signing generate-key {{TARGET}} --name "My Company" --email "dev@example.com"

# Set and validate signing config
scenario-to-desktop signing set {{TARGET}} --config @signing.json
scenario-to-desktop signing validate {{TARGET}}
```

---

### **7. Cross-Platform Build Matrix**

Building from Linux (recommended):

| Target | Format | Notes |
|--------|--------|-------|
| Linux | AppImage, DEB | Native |
| Windows | NSIS (.exe) | Via Wine (use `wine check`/`wine install`) |
| macOS | ZIP | Native; DMG/PKG requires actual macOS |

---

### **8. Preflight Prerequisites**

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

Long-tail preflight and deploy failures are consolidated in `Troubleshooting & Edge Cases`.

---

### **9. Troubleshooting & Edge Cases**

Use this section for failure diagnosis and recovery beyond the standard happy path.

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `service binary validation failed` | Binaries not compiled | Verify `platforms/electron/bin/...` paths | Run `make build` in scenario dir |
| `critical validation checks failed` | UI assets missing | Check `platforms/electron/ui/dist/index.html` | Run `pnpm build` in the UI dir |
| `missing asset` during preflight | Bundle manifest points to non-existent paths | Inspect bundle manifest and referenced files | Rebuild scenario assets or fix manifest paths |
| Deploy stage skipped unexpectedly | No deploy flags provided | Confirm pipeline args include deploy selector + app key | Re-run with `--deploy-target <name> --app-key <key>` or inline deploy flags |
| Deploy target test fails | Missing/expired remote profile session on LPBS | `scenario-to-desktop deploy-target test <name>` and LPBS remote profile status | Re-run LPBS remote profile login/test via `landing-page-deploy-setup` |
| Deploy target auth check fails | LPBS service auth disabled or `LPBS_SERVICE_SECRET` missing/mismatched | `scenario-to-desktop deploy-target test <name> --require-service-auth` | Re-sync LPBS runtime service auth and `LPBS_SERVICE_SECRET`, then re-test |
| Deploy fails with service auth 401/403 | `LPBS_SERVICE_SECRET` missing or mismatched | Confirm `LPBS_SERVICE_SECRET` in deploy shell and LPBS runtime config | Re-sync the secret with LPBS runtime and retry |

Use CLI group help when needed:

```bash
scenario-to-desktop pipeline help
scenario-to-desktop deploy-target help
scenario-to-desktop signing help
scenario-to-desktop telemetry help
```

---

### **10. Guardrails**

**Do:**
- Use `--wait` for all scripted/agent workflows
- Verify target scenario is running before generation (preflight checks this)
- Use bundles mode for production
- Use environment variables for `LPBS_SERVICE_SECRET` (never embed in commands)

**Do NOT:**
- Use thin client mode unless specified
- Deploy unless the user explicitly requests it and provides target info
- Use deploy flags speculatively or "just in case"

---

### **11. Success Artifacts**

After successful pipeline run:

| Platform | File |
|----------|------|
| Linux | `<scenario>.AppImage` |
| Windows | `<scenario>-Setup.exe` |
| macOS | `<scenario>.zip` (contains .app) |

Download via: `scenario-to-desktop download <scenario> <platform> --output <path>`
