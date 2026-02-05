# Overview: Bundled Offline Desktop Apps

## Recommended: Bundled Offline Mode (Default)

**Bundled mode is the strongly recommended and default option.** It creates complete offline desktop applications that include all services, resources, and the runtime supervisor - no server connection required.

### What bundled mode provides
- Complete offline operation - no internet or server required after installation
- Runtime supervisor that manages all bundled services (API, databases, etc.)
- Automatic service health monitoring and restart capabilities
- Secret management and secure configuration
- Dynamic port allocation to avoid conflicts
- Full telemetry for deployment insights
- Cross-platform installers (MSI/PKG/AppImage/DEB)

### How to deploy bundled apps
1) Create a deployment profile:
   ```bash
   deployment-manager profile create my-profile my-scenario --tier 2
   ```
2) Build everything (binaries, Electron wrapper, installers):
   ```bash
   deployment-manager deploy-desktop --profile my-profile
   ```
3) Distribute the installers to users - they work completely offline

See [Hello Desktop Tutorial](../deployment-manager/docs/tutorials/hello-desktop-walkthrough.md) for a complete walkthrough.

## Alternative: Thin Client Mode

Thin client mode is available for scenarios where you want a lightweight desktop shell that connects to an existing server. Use this when:
- You already have a Vrooli server running
- Multiple users need to connect to the same backend
- You want smaller installer sizes
- You need real-time data sharing between users

### Thin client limitations
- **Requires server** - API and resources must run elsewhere
- **No offline mode** - requires network connection to server
- **UI-only bundles** - copies `ui/dist` assets into the Electron wrapper

### Thin client workflow
1) Start the target scenario: `vrooli scenario start <name>`
2) Expose via app-monitor/Cloudflare or LAN
3) In scenario-to-desktop UI, choose Thin Client mode and paste the proxy URL
4) Build and distribute installers

## What's not yet automated
- `cloud-api` mode remains a stub for future SaaS deployments
- Alternative frameworks (Tauri/Neutralino) are placeholders; Electron is the maintained path
- Auto-updates, signing, and app-store submissions remain optional/manual

## Related docs
- Bundled desktop tutorial: `../deployment-manager/docs/tutorials/hello-desktop-walkthrough.md`
- Runtime supervisor details: `runtime/README.md`
- Choosing deployment modes: `docs/deployment-modes.md`
- Build/troubleshoot: `docs/build-and-packaging.md`, `docs/DEBUGGING_WINDOWS.md`, `docs/WINE_INSTALLATION.md`
- Bundled runtime logging: `docs/workflows/logging-bundled-desktop.md`
