## Tools focus: Scenario to Desktop

Convert any Vrooli scenario into a cross-platform desktop application (Windows, macOS, Linux) using the scenario-to-desktop CLI. The tool creates Electron wrappers that connect to your running scenario.

---

### **1. When to Use This Tool**

| Goal | Command |
|------|---------|
| Build desktop app | `scenario-to-desktop pipeline run <scenario> --platforms linux --clean --wait` |
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

Always use `--clean` to ensure fresh builds, and `--wait` for synchronous execution (blocks until complete, proper exit codes):

```bash
# Build desktop app for Linux
scenario-to-desktop pipeline run {{TARGET}} --platforms linux --clean --wait
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

### **5. Code Signing Workflow**

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

The CLI output provides all information needed for troubleshooting, organized as cleanly as possible.

---

### **9. Guardrails**

**Do:**
- Use `--wait` for all scripted/agent workflows
- Verify target scenario is running before generation (preflight checks this)
- Use bundles mode for production

**Do NOT:**
- Use thin client mode unless specified

---

### **10. Success Artifacts**

After successful pipeline run:

| Platform | File |
|----------|------|
| Linux | `<scenario>.AppImage` |
| Windows | `<scenario>-Setup.exe` |
| macOS | `<scenario>.zip` (contains .app) |

Download via: `scenario-to-desktop download <scenario> <platform> --output <path>`
