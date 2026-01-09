# Quick Start (Thin Client)

Use the lifecycle system so ports, logging, and process names stay consistent.

1. **Start the scenario**
   ```bash
   cd scenarios/scenario-to-desktop
   make start        # preferred; or: vrooli scenario start scenario-to-desktop
   ```
   Note the UI/API ports from `make logs` or `vrooli scenario status scenario-to-desktop`.

2. **Open the web UI**
   - Visit `http://localhost:<UI_PORT>` and go to **Scenario Inventory → Generate Desktop**.
   - Paste the Cloudflare/app-monitor proxy URL (or LAN URL) for the target scenario.
   - Keep `Deployment Mode = Thin Client (external-server)` unless you are testing a bundle stub.

3. **Generate installers**
   - Select Windows/macOS/Linux; the service runs `npm install`, `npm run build`, and `npm run dist`.
   - Health telemetry is written inside the generated app at `deployment-telemetry.jsonl`.

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

### CLI-only path (manual config)
```bash
scenario-to-desktop generate <scenario> \
  --deployment-mode external-server \
  --server-type external \
  --server-url https://app-monitor.<domain>/apps/<scenario>/proxy/ \
  --api-url https://app-monitor.<domain>/apps/<scenario>/proxy/api/ \
  --auto-manage-vrooli=false
```

### Preconditions
- Target scenario is already running and reachable (LAN or Cloudflare).
- UI build exists at `ui/dist` for the target scenario.
- For Windows installers on Linux, follow `docs/WINE_INSTALLATION.md`.
