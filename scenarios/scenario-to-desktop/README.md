# Scenario to Desktop

Scenario to Desktop packages a Vrooli scenario as a professional Electron desktop application for Linux, Windows, or macOS. It is the Tier 2 desktop deployment ramp. The generated application can run with a bundled runtime or connect to a managed external scenario.

Electron is the only supported desktop framework. Cross-ramp release approval, promotion, and deployment evidence belong to deployment-manager; this scenario owns the typed desktop generation, packaging, smoke-test, signing-configuration, and interactive-evidence workflows.

## Start here

Use the scenario lifecycle instead of running binaries directly:

```bash
make start
make logs
make stop
```

Open the generated scenario URL, select a source scenario and Electron template, then run the pipeline. The pipeline records the target decision, bundle/preflight evidence, generated wrapper, installer build, smoke-test evidence, and optional deployment handoff.

For command discovery, run:

```bash
scenario-to-desktop --help
```

## Deployment modes

- **Bundled runtime** is the default for offline desktop delivery. It packages the selected scenario's declared runtime and validates it before installer creation.
- **External server** is for a desktop shell that connects to a managed scenario endpoint. Validate the endpoint before release; do not describe it as offline.

The supported local evidence path starts a Linux desktop session on this host, launches the generated artifact, and exposes its VNC stream only through the API's loopback proxy. The typed evidence contract also represents a bridge-node target, but remote desktop execution is intentionally rejected until the bridge-owned remote-desktop protocol is implemented.

## Validation

Run the scenario-owned suite through Test Genie:

```bash
vrooli scenario test scenario-to-desktop
```

The run is server-owned. Reattach once with the run ID it returns:

```bash
test-genie runs wait --json scenario-to-desktop <run-id>
```

Do not poll the run. For focused local checks, use the surface-specific commands documented in [Testing](docs/internal/TEST-GENIE-BASELINE-20260726.md).

## Documentation

- [Overview](docs/OVERVIEW.md)
- [Quickstart](docs/QUICKSTART.md)
- [Deployment modes](docs/concepts/deployment-modes.md)
- [Desktop integration guide](docs/guides/desktop-integration.md)
- [Build and packaging guide](docs/guides/build-and-packaging.md)
- [Cross-platform builds](docs/guides/cross-platform-builds.md)
- [Interactive desktop evidence](docs/guides/interactive-desktop.md)
- [Code signing](docs/guides/code-signing.md)
- [API contract](docs/reference/api-contract.md)
- [CLI commands](docs/reference/cli-commands.md)
- [Security posture](docs/internal/SECURITY-POSTURE.md)

## Boundaries

Scenario to Desktop does not implement Tauri or Neutralino generation, public/LAN API authentication, app-store submission automation, or bridge-hosted remote desktop execution. Those are explicit follow-up capabilities, not implied promises of the desktop packaging flow.
