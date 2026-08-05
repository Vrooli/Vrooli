# Overview: Bundled Offline Desktop Apps

## Recommended: Bundled Offline Mode (Default)

**Bundled mode is the strongly recommended and default option for scenarios
whose dependency plan is bundle-ready.** It packages the scenario services and
runtime supervisor for offline use. It resolves every required resource from
its `resource.json` target profile and stages declared bundled artifacts only
from a signed release directory (`resource_artifact_root`); it never builds a
resource on the end-user target. A conditional/degraded requirement is shown
before runtime, while an unsupported route is a blocking error.

### What bundled mode provides
- Complete offline operation when every required capability has an offline
  bundle route; cloud/remote resources remain network-dependent by design
- Runtime supervisor that manages bundled scenario services plus a verified
  `resource-deployment-plan.json` describing selected resource artifacts
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

See [Hello Desktop Tutorial](../../deployment-manager/docs/tutorials/hello-desktop-walkthrough.md) for a complete walkthrough.

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

## Where this fits

scenario-to-desktop is the Tier 2 ramp. It owns build, packaging, signing, publishing, and running the artifact on desktop targets. `deployment-manager` owns the approval gate and the release record, and this pipeline asks it for permission before publishing. See [API Contract](reference/api-contract.md#release-authority).

## What's not yet automated
- The smoke-test journey captures a deterministic startup and interaction evidence trail (recording, screenshots, window geometry, and a durable step list). It is not a replacement for arbitrary `bas/` workflow assets; those remain a separate browser validation concern.
- `cloud-api` mode remains a stub for future SaaS deployments
- Electron is the supported desktop framework.
- Auto-updates, signing, and app-store submissions remain optional/manual
- Resource artifact execution is limited to the declared bundled modes. Docker,
  compose, and native-host-tool resources remain explicit host prerequisites
  until their runtime adapter is selected by a resource profile.

## Related docs
- Bundled desktop tutorial: `../../deployment-manager/docs/tutorials/hello-desktop-walkthrough.md`
- Runtime supervisor details: `runtime/README.md`
- Choosing deployment modes: `docs/concepts/deployment-modes.md`
- Build/troubleshoot: `docs/guides/build-and-packaging.md`, `docs/guides/debugging-windows.md`, `docs/guides/wine-installation.md`
- Bundled runtime logging: `docs/guides/logging-bundled-desktop.md`
- Resource target/deployment contract: [../../../docs/resources/deployment-contract.md](../../../docs/resources/deployment-contract.md)
