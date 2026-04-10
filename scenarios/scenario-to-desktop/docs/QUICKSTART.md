# Quick Start

## Bundled Desktop Apps (Recommended)

The recommended approach creates complete offline desktop applications:

```bash
# Create a deployment profile for your scenario
deployment-manager profile create my-profile my-scenario --tier 2

# Build the complete desktop app (binaries + Electron + installers)
deployment-manager deploy-desktop --profile my-profile
```

This creates installers for Windows, macOS, and Linux that work completely offline.

See [Hello Desktop Tutorial](../../deployment-manager/docs/tutorials/hello-desktop-walkthrough.md) for a complete walkthrough.

---

## Thin Client Mode (Alternative)

Use thin client only when you need multiple users connecting to a shared server.

### Using the UI

1. **Start scenario-to-desktop**
   ```bash
   cd scenarios/scenario-to-desktop
   make start        # preferred; or: vrooli scenario start scenario-to-desktop
   ```
   Note the UI/API ports from `make logs` or `vrooli scenario status scenario-to-desktop`.

2. **Open the web UI**
   - Visit `http://localhost:<UI_PORT>` and go to **Scenario Inventory → Generate Desktop**.
   - Select `Deployment Mode = Thin Client (external-server)`.
   - Paste the Cloudflare/app-monitor proxy URL (or LAN URL) for the target scenario.

3. **Generate installers**
   - Select Windows/macOS/Linux; the service runs `npm install`, `npm run build`, and `npm run dist`.
   - Telemetry is written inside the generated app at `deployment-telemetry.jsonl`.

4. **Distribute & collect telemetry**
   - Ask testers for the telemetry file and upload via the UI, or run:
     ```bash
     scenario-to-desktop telemetry collect \
       --scenario <name> \
       --file "<path-to>/deployment-telemetry.jsonl"
     ```

5. **Stop services when done**
   ```bash
   make stop     # or: vrooli scenario stop scenario-to-desktop
   ```

### CLI-only path (thin client)
```bash
scenario-to-desktop generate <scenario> \
  --deployment-mode external-server \
  --server-type external \
  --server-url https://app-monitor.<domain>/apps/<scenario>/proxy/ \
  --api-url https://app-monitor.<domain>/apps/<scenario>/proxy/api/ \
  --auto-manage-vrooli=false
```

### Thin client preconditions
- Target scenario is already running and reachable (LAN or Cloudflare).
- UI build exists at `ui/dist` for the target scenario.
- For Windows installers on Linux, follow `docs/guides/wine-installation.md`.

### Check build artifacts (CLI)
```bash
scenario-to-desktop desktop-status --name <scenario>
```
