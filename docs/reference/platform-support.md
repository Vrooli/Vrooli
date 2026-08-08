# Platform Support and Evidence Matrix

This is the canonical source for Vrooli platform capability claims. A row is
supported only when its stated evidence tier has been earned; cross-compilation
and unit tests prove build behavior, not real-host support.

## Evidence tiers

| Tier | Meaning |
|---|---|
| `supported` | Automated regressions and the required real-hardware acceptance ladder are recorded for the exact capability. |
| `build-verified` | Static cross-build and/or Linux-runnable contract tests pass; no real-host claim is implied. |
| `experimental` | Intended implementation exists but has incomplete automated or hardware evidence. |
| `unqualified` | No evidence has been recorded; do not use this as a support claim. |

## Current matrix

| Capability | Linux | macOS Intel / Apple Silicon | Windows | Evidence and limits |
|---|---|---|---|---|
| `vrooli` release binary | build-verified | build-verified | build-verified | Six targets are defined by `internal/buildinfo.DistributionTargets`; `internal/buildinfo/distribution_test.go`, `cmd/vrooli-dist`, and the release workflow guard asset names, fingerprints, and embedded version. macOS hardware execution remains unqualified. |
| Authenticated CLI/source installer | build-verified | build-verified | experimental | `bash install/install_test.sh` covers manifest signature, asset checksum, source digest pairing, source-directory refusal, repeat install, and absolute-path child setup. The POSIX path is the macOS installer; native Windows has a separate PowerShell installer but not a full setup lifecycle. |
| `vrooli setup` minimal/development profiles | build-verified | build-verified | unqualified | `internal/setup`, `internal/runtime`, `internal/hostreqkit`, and `internal/cli/projectcli` tests cover typed setup results and user-local verified tools. macOS uses the target user; hardware runs are still required. |
| Buf 1.37.0 | build-verified | build-verified | experimental | `internal/tools/buf/tool.json` declares checksum-pinned Linux/Darwin assets; generic runtime tests cover target selection, checksum failure, missing architecture, user-local discovery, and convergence. Windows intentionally uses winget fallback. |
| Runtime supervisor persistence | supported by existing systemd user-service path | build-verified | unqualified | The macOS contract is a per-user `com.vrooli.runtime-supervisor` LaunchAgent, not a system daemon. `internal/runtimesupervisor` tests render and plan its plist on Linux; a logged-in GUI session and a real reboot/lifecycle record are still required. |
| Bridge node agent | supported by existing lifecycle evidence | build-verified | experimental | The Bridge agent has its distinct `com.vrooli.bridge.vrooli-bridge-agent` LaunchAgent contract and independent service tests. A headless Mac requires an auto-logged-in user for LaunchAgents. |
| Bridge onboarding | build-verified | build-verified | unqualified | `bootstrap_test.sh` covers paired source/binary transfer and maps only setup's structured `unsupported_platform` category to exit 3. The original Mac-mini path must still be run without `--skip-setup`. |
| Workspace Sandbox copy driver | build-verified | build-verified | build-verified | Copy-driver E2E and cross-build gates cover identity-layout behavior. |
| Workspace Sandbox containment | supported where bwrap is available | experimental | unqualified | macOS Seatbelt is intentionally partial: filesystem-write containment and network denial only; no PID namespace or `/workspace` path illusion. Follow the real-host shakeout before claiming support. |
| `resource-minio` managed service | build-verified | build-verified | build-verified | Cross-build and checksum manifests cover Linux amd64/arm64, macOS amd64/arm64, and Windows amd64. Promotion to `supported` still requires a real Windows amd64 start/stop/readiness run with the native binary and a durable-data smoke test. |
| `resource-qdrant` managed service | build-verified | build-verified | build-verified | Cross-build and checksum manifests cover Linux amd64/arm64, macOS amd64/arm64, and Windows amd64. Promotion to `supported` still requires a real Windows amd64 collection-load/readiness run and a durable-storage recovery smoke test. |
| PostgreSQL and Redis via Docker Desktop | supported where Docker is available | unqualified | unqualified | Docker Desktop startup, ownership, and reboot behavior must be recorded on real macOS hardware before a macOS support claim. |
| `storage-manager` census/device accounting | build-verified | build-verified | build-verified | Deterministic filesystem/device seam tests cover Linux privileged and least-privilege, macOS, and Windows-degraded verdicts; Darwin and Windows cross-builds are required evidence, not real-host qualification. |
| Host session invocation and Linux arm64 tool declarations | build-verified | build-verified | build-verified | Linux session behavior is unit-tested; macOS `launchctl asuser`, Windows typed-unsupported behavior, and Linux arm64 manifest routes are cross-build/test verified only. No macOS, Windows, or arm64 hardware qualification is claimed. |

## Automated evidence at this execution start

- Git Control Tower baseline `macos-setup-cross-platform-qualification-baseline` is complete for `vrooli-bridge` (`20260716-230307-bd5ae870`) and `workspace-sandbox` (`20260716-230307-4822d785`).
- Required non-hardware validation includes `bash install/install_test.sh`, `bash scenarios/vrooli-bridge/bootstrap/bootstrap_test.sh`, focused setup/runtime tests, and static Darwin `amd64`/`arm64` builds.

## Structured setup result transport

Automation that needs setup's terminal classification must use the separate
result file, never parse setup's human output:

```bash
vrooli setup --result-file /secure/path/setup-result.json
```

The file is mode `0600` and contains the versioned `v1` fields `status`,
`category`, `stage`, `retryable`, `blocked_requirements`, and `remediation`.
Human progress remains on the normal output streams. Bridge uses this contract;
only `category=unsupported_platform` maps to bootstrap exit `3`.

## Required macOS hardware record

Do not upgrade any macOS row until an operator records one evidence document per
hardware class (Intel and Apple Silicon) containing:

- a non-secret host identifier, hardware model, architecture, macOS version and build;
- clean-state preparation and the exact minimal/development setup commands;
- Test Genie run IDs and Bridge onboarding operation IDs;
- installed paths, ownership, new-login-shell PATH result, and restart/reboot observations;
- runtime-supervisor and Bridge-agent `launchctl` state/log evidence;
- Workspace Sandbox Seatbelt write-denial, network-denial, approval, and provenance results;
- native `vrooli-bridge` lifecycle plus PostgreSQL/Redis Docker Desktop evidence;
- repeat/convergence and interrupted-install recovery evidence; and
- failures, exceptions, and the resulting tier for every matrix row.

Use [`scenarios/workspace-sandbox/docs/guides/macos-shakeout.md`](../../scenarios/workspace-sandbox/docs/guides/macos-shakeout.md)
for the Sandbox portion. The full qualification ladder is owned by the macOS
setup qualification plan; this matrix records its earned result.
