# Overview: Thin Client Today, Bundles Later

## What works today (v1 thin client)
- Generates Electron wrappers that point at a running scenario (LAN or Cloudflare/app-monitor proxy URL).
- Builds MSI/PKG/AppImage/DEB installers from the UI or CLI, using the scenario’s existing `ui/dist` output.
- Telemetry is written to `deployment-telemetry.jsonl` and can be collected via the UI or `scenario-to-desktop telemetry collect`.
- Optional `auto_manage_vrooli` lets the desktop wrapper run `vrooli setup/start/stop` locally when you explicitly opt in.

## What is stubbed / future
- `deployment_mode=bundled` and `cloud-api` are UX stubs; the runtime is under active development in `runtime/`.
- `server_type=static|node|executable` exist for future embedding experiments but are not production-supported.
- Alternative frameworks (Tauri/Neutralino) are placeholders; Electron is the only maintained path.
- Auto-updates, signing, and app-store submissions remain optional/manual; no automation shipped.

## How to deploy right now
1) Start the target scenario with lifecycle: `vrooli scenario start <name>` (or `make start`).  
2) Expose it via app-monitor/Cloudflare (preferred) or LAN; copy the proxy URL.  
3) In scenario-to-desktop UI → Scenario Inventory, choose Thin Client / external server, paste the proxy URL, select platforms, build, and download.  
4) Collect telemetry from testers and upload it for deployment-manager visibility.  
5) Stop the scenario when done.

## Where to track bundling progress
- Bundled runtime design and status: `runtime/README.md` (mermaid diagrams included).
- Deployment hub context: `../../docs/deployment/tiers/tier-2-desktop.md` and `../../docs/deployment/scenarios/scenario-to-desktop.md`.
- Seams for future embedding: `docs/SEAMS.md`.

## Related docs
- Thin-client quickstart: `docs/QUICKSTART.md`
- Choosing deployment/server types: `docs/deployment-modes.md`
- Build/troubleshoot: `docs/build-and-packaging.md`, `docs/DEBUGGING_WINDOWS.md`, `docs/WINE_INSTALLATION.md`
