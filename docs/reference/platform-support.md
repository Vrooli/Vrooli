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
| `vrooli setup` minimal/development profiles | build-verified | build-verified | unqualified | `internal/setup`, `internal/runtime`, `internal/hostreqkit`, and `internal/cli/projectcli` tests cover typed setup results and user-local verified tools. Development protobuf tools are required because setup regenerates `packages/proto`; macOS uses the target user, but fresh-host hardware and reboot evidence remain outstanding. |
| Buf 1.37.0 | build-verified | build-verified | experimental | `internal/tools/buf/tool.json` declares checksum-pinned Linux/Darwin assets; generic runtime tests cover target selection, checksum failure, missing architecture, user-local discovery, and convergence. Windows intentionally uses winget fallback. |
| Runtime supervisor persistence | supported by existing systemd user-service path | build-verified | unqualified | The macOS contract is a per-user `com.vrooli.runtime-supervisor` LaunchAgent, not a system daemon. `internal/runtimesupervisor` tests render and plan its plist on Linux; a logged-in GUI session and a real reboot/lifecycle record are still required. |
| Bridge node agent | supported by existing lifecycle evidence | build-verified | experimental | The Bridge agent has its distinct `com.vrooli.bridge.vrooli-bridge-agent` LaunchAgent contract and independent service tests. A headless Mac requires an auto-logged-in user for LaunchAgents. |
| Bridge onboarding | build-verified | experimental | unqualified | `bootstrap_test.sh` covers paired source/binary transfer and maps only setup's structured `unsupported_platform` category to exit 3. The Intel `darwin/amd64` `minimouse` run proved working-tree transfer, prebuilt native artifacts, native CLI setup, LaunchDaemon installation, online heartbeat, protocol compatibility, and dispatchability; the separate minimal setup run also completed on that host. `darwin/arm64` remains cross-build evidence only. Reboot persistence, Keychain provisioning, and credential-store completion remain unqualified. |
| Bridge-managed protected cleanup | build-verified | experimental | unqualified | The Bridge cleanup domain freezes an attributed plan, requires target-bound operator authorization, and applies it through named typed helper operations. VCS1 X25519/AES-GCM sealing and local/agent refusal tests are covered; the Intel Mac node is enrolled and dispatchable, but target-bound break-glass provisioning still awaits the operator's one-time passphrase, so the cleanup preview/apply hardware ladder and SSH fallback hardware evidence remain outstanding. The supported operational path is Bridge-managed cleanup; the terminal procedure is emergency-only for a node that is unreachable and unmanageable. |
| Shared TypeScript package provisioning | build-verified | build-verified (Intel hardware path) | unqualified | Minimouse working-tree onboarding operation `3e6335e0-3b4e-49c7-978a-df1861a7e3e8` completed setup, Buf regeneration, package-local frozen installs, and native Darwin CLI compilation; Bridge run `7ab43630-6db4-4ebe-96e3-40b088af18bf` launched the structure-health Test Genie suite. Final Mac credential-store setup still requires an interactive passphrase. |
| Workspace Sandbox copy driver | build-verified | build-verified | build-verified | Copy-driver E2E and cross-build gates cover identity-layout behavior. |
| Workspace Sandbox containment | supported where bwrap is available | experimental | unqualified | macOS Seatbelt is intentionally partial: filesystem-write containment and network denial only; no PID namespace or `/workspace` path illusion. Follow the real-host shakeout before claiming support. |
| Device Control LAN discovery | supported on hosts with multicast UDP | experimental | unqualified | The host and target must share a multicast-reachable LAN segment; mDNS does not cross subnets or VLANs. The control plane reports firewall/interface failures as unreachable reasons. |
| `resource-minio` managed service | build-verified | build-verified | build-verified | Cross-build and checksum manifests cover Linux amd64/arm64, macOS amd64/arm64, and Windows amd64. Promotion to `supported` still requires a real Windows amd64 start/stop/readiness run with the native binary and a durable-data smoke test. |
| `resource-qdrant` managed service | build-verified | build-verified | build-verified | Cross-build and checksum manifests cover Linux amd64/arm64, macOS amd64/arm64, and Windows amd64. Promotion to `supported` still requires a real Windows amd64 collection-load/readiness run and a durable-storage recovery smoke test. |
| `resource-ollama` managed service | build-verified (conditional Linux amd64 native bundle) | experimental (conditional amd64/arm64 bundle; target smoke pending) | build-verified (conditional Windows amd64 native bundle) | `resources/ollama/resource.json` pins Ollama `0.30.10` server artifacts and declares the managed-service readiness contract. Linux amd64 native restart, GPU health, model preservation, and Search Hub consumer evidence are recorded; Windows is checksum-staged but has no real-host smoke evidence yet. |
| `resource-reranker` managed service | build-verified (conditional Linux amd64 native bundle) | unsupported | unsupported | `resources/reranker/resource.json` pins TEI `1.7.4` for Linux amd64 and declares `/health` readiness. GPU/CPU behavior remains host/model conditional; no native macOS or Windows bundle is staged. |
| `resource-whisper` managed service | build-verified (native whisper.cpp Linux amd64/arm64 acquisition and checksum contract) | unsupported | conditional/build-verified (native upstream Windows archives; target smoke pending) | `resources/whisper/resource.json` owns the managed native server and GGML model. macOS remains unsupported because the selected upstream release publishes an XCFramework rather than a server executable; Windows is not promoted beyond conditional until target smoke evidence exists. |
| `resource-sherpa-onnx` managed service | experimental (qualified Linux amd64 adapter build; signed release artifact not published) | unsupported | unsupported | `resources/sherpa-onnx/resource.json` contains the native TTS, streaming STT, speaker, and separation contracts, but every executable acquisition target remains explicitly unsupported until a target-native Vrooli adapter bundle is signed, published, and smoke-tested. Upstream runtime packages alone do not earn support. |
| `resource-postgres` `managed-service` | supported (amd64); build-verified (arm64) | build-verified (amd64/arm64) | build-verified (amd64); unsupported (arm64) | `resources/postgres/resource.json` declares `driver: managed-service` and `bundling: vendorable`. Linux stages a digest-pinned `library/postgres` filesystem tree with no container runtime; Linux amd64 has a live run on this host (install, start, readiness, serving on `127.0.0.1`, clean restart, and every `resource-postgres content` command against the managed process). macOS and Windows stage checksum-pinned upstream archives from `theseus-rs/postgresql-binaries` 16.15.0; the macOS archives are built from source on GitHub runners and carry OpenSSL via `@loader_path`, while the Windows archive repackages EnterpriseDB's official build and ships its ICU/OpenSSL/libxml2 DLLs beside the executables. **No macOS or Windows hardware run has been performed**; those rows rest on staging, checksum, tree-digest, and contract evidence only. Windows on ARM is unsupported because upstream publishes no `aarch64-pc-windows-msvc` archive. Windows shutdown relies on a console control event and is unproven on hardware; `pg_ctl stop -m fast` ships in the same archive as the known fallback. |
| `resource-redis` `managed-service` | supported (Linux amd64/arm64 OCI tree) | unsupported | build-verified (Windows amd64 native archive; ARM unsupported) | `resources/redis/resource.json` stages the digest-pinned official OCI tree on Linux and the checksum-pinned Redis 8.10.0 Windows MSYS2 archive for Windows amd64. The Windows path is native and self-contained; no Docker prerequisite is claimed. **No Windows hardware run has been performed; the Windows row remains build-verified pending target smoke evidence.** |
| `resource-searxng` managed service | build-verified (Linux amd64/arm64 composed trees) | build-verified (macOS amd64/arm64 composed trees; live readiness pending) | build-verified (Windows amd64 composed tree) | SearXNG is composed from a pinned standalone Python runtime, locked wheels, and source tree. The managed-service artifact carries per-target tree digests; Windows arm64 is explicitly unsupported until a compatible Granian wheel exists. |
| `storage-manager` census/device accounting | build-verified | build-verified | build-verified | Deterministic filesystem/device seam tests cover Linux privileged and least-privilege, macOS, and Windows-degraded verdicts; Darwin and Windows cross-builds are required evidence, not real-host qualification. |
| Host session invocation and Linux arm64 tool declarations | build-verified | build-verified | build-verified | Linux session behavior is unit-tested; macOS `launchctl asuser`, Windows typed-unsupported behavior, and Linux arm64 manifest routes are cross-build/test verified only. No macOS, Windows, or arm64 hardware qualification is claimed. |
| First-run credential authority and unattended encrypted-store wrap | build-verified | build-verified | build-verified | `internal/resources/securestore` proves copy/verify/commit reselection, Linux host-bound fallback, Darwin Keychain and Windows DPAPI cross-build paths, and typed unavailable results. Hardware reboot qualification is still required before a row becomes supported. |
| Cross-platform protobuf generation and descriptor publication | build-verified | build-verified | build-verified | Cross-build gate covers Linux amd64/arm64, macOS amd64/arm64, and Windows amd64; `protogen.TestProductionPipelineIsShellFree` is the shell-free source gate; `descriptorimage.TestSourceCachesAndReloadsByPortableStamp` is the portable `os.SameFile` stamp gate; publish tests cover atomic file rename and last-known-good reload behavior. |
| `system-monitor` host collection | supported (amd64); build-verified (arm64 pending Raspberry Pi 4 evidence) | build-verified (Apple Silicon) | build-verified (amd64) | Linux amd64 contract tests and live `/api/v1/disk-pressure` evidence cover the measured collectors; Linux arm64 has cross-build/contract evidence only until a real Raspberry Pi 4 run is recorded. Darwin and Windows are cross-build/contract evidence only. |

## Accelerator support per resource

Each resource declares its accelerator backends once, in the `acceleration`
block of its `resource.json`. This table records what each one declares and what
evidence stands behind it.

Read the two columns separately. **Declared** is what the manifest asks the
platform for. **Placement evidence** is whether the control plane has ever
verified, on a live host, that the resource landed there. They are not the same
claim, and a declaration is never evidence.

| Resource | Declared backends | `require` | Linux/CUDA | macOS/Metal | Windows |
|---|---|---|---|---|---|
| `ollama` | `cuda`, `metal`, `cpu` | preferred | **verified live** — `nvidia-smi` reports its `llama-server` child holding device memory | **unknown** — no live macOS host; the darwin release stages a native executable that carries Metal, and selection is fixture-tested, but placement is unverified | build-verified — the standalone Windows amd64 server archive is checksum-pinned and staged; no live Windows host qualification |
| `whisper` | `cuda`, `cpu` | preferred | **verified live** — currently reports `mode_drift` because upstream publishes no Linux CUDA asset | **not declared** — whisper.cpp v1.9.2 publishes an Apple XCFramework, not a `whisper-server` executable. There is no artifact to run, so no `metal` backend is declared | partial — a CUDA-predicated Windows target exists; no live Windows host |
| `reranker` | `cuda` | **required** | **verified live** — reports `observed_mode: cuda` | unsupported — the pinned TEI image has no native macOS executable bundle | unsupported — no native Windows executable bundle |
| `kokoro` | `cuda`, `cpu` | preferred | declared; not currently installed on the reference host | not declared | not declared |
| `kyutai-stt` | `cuda`, `cpu` | preferred | declared; not currently installed on the reference host | not declared | not declared |
| `speaker-verification` | `cuda`, `cpu` | preferred | declared; not currently installed on the reference host | not declared | not declared |
| `sherpa-onnx` | none | — | does no accelerated work; its ONNX runtime is CPU-only | — | — |

### Why ROCm is declared nowhere

`internal/hostinventory` detects ROCm and `internal/accel` selects and verifies
it — the reference host reports `accel.backends=cuda,rocm,vulkan,cpu` with
`/dev/kfd` present, and a fixture test proves an AMD host selects `rocm` through
the same code path CUDA uses. No resource declares it because none of the six
has a **pinnable upstream ROCm artifact**: every accelerated image and release
asset currently pinned by digest is a CUDA build. Declaring `rocm` without one
would produce a backend the resolver selects and cannot stage. The declaration
lands when an upstream ROCm artifact exists to pin.

### What "unknown" means here

No live macOS or Windows host was available. A fixture-driven selection test
proves the *decision* is right on those platforms; it is not a placement proof
and is not reported as one. Live placement verification on macOS and Windows is
recorded as `unknown`, and `vrooli resource status` reports it that way rather
than assuming agreement.

`ollama`'s `platforms.macos` stays `build-verified` for the same reason: the artifact
and the Metal support are both real, and neither has been exercised on a live
Apple Silicon machine by this work.

## Automated evidence at this execution start

- Git Control Tower baseline `macos-setup-cross-platform-qualification-baseline` is complete for `vrooli-bridge` (`20260716-230307-bd5ae870`) and `workspace-sandbox` (`20260716-230307-4822d785`).
- Required non-hardware validation includes `bash install/install_test.sh`, `bash scenarios/vrooli-bridge/bootstrap/bootstrap_test.sh`, focused setup/runtime tests, and static Darwin `amd64`/`arm64` builds.

Bridge's enrolled local operator session has a maximum 15-minute lifetime, so
that is the maximum documented revocation delay for locally minted sessions.
The current plan ships passphrase sealing rather than the transport-only
fallback. PostgreSQL and Redis declare the `managed-service` driver with
`bundling: vendorable`. PostgreSQL stages a digest-pinned OCI filesystem tree on
Linux, where the official image carries the shared libraries the server links
against, and checksum-pinned upstream archives on macOS and Windows, where the
published archives are self-contained; the acquisition kind is therefore
declared per target. Redis stages its OCI tree on Linux and a checksum-pinned
native Windows amd64 archive; macOS and Windows ARM are explicitly unsupported.
SearXNG uses the
same portable managed-service contract and its live readiness qualification
remains pending on this host. A resource can no longer declare a platform it has
no acquisition route for, or deny one it serves: `checkPlatformClaims` in
`internal/resources/fleet_contract.go` enforces both directions and runs in CI
through the `resource-contract` job. The project CLI health verdict remains intentional proto-first migration debt from
the predecessor work and is not silently reclassified as a platform regression.

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
- native `vrooli-bridge` lifecycle plus Docker-backed PostgreSQL/Redis evidence (including provider provenance, bootstrap, readiness, and durable-data observations);
- repeat/convergence and interrupted-install recovery evidence; and
- failures, exceptions, and the resulting tier for every matrix row.

Use [`scenarios/workspace-sandbox/docs/guides/macos-shakeout.md`](../../scenarios/workspace-sandbox/docs/guides/macos-shakeout.md)
for the Sandbox portion. The full qualification ladder is owned by the macOS
setup qualification plan; this matrix records its earned result.
