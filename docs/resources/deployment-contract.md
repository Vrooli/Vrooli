# Resource Deployment Contract

This document defines the target contract for taking a resource from a source
checkout to a supported deployment target, including native desktop bundles.
It is the design authority for schema, template, control-plane, and
`scenario-to-desktop` work. The `deployment` shape below is implemented by the
resource manifest parser and required for every active resource's desktop
profile.

## Why This Contract Exists

A Go implementation alone does not make a resource deployable. A resource may
still depend on Bash, a local Go toolchain, Docker, a GPU, a proprietary host
tool, a cloud account, or a Linux-only capability. Those facts must be visible
before a scenario is bundled, not discovered by an end user after install.

The target model separates these questions:

```text
Archetype  → Which deployment modes are plausibly feasible?
Resource   → Which modes does this particular resource support and require?
Readiness  → Is that claimed support implemented and evidenced today?
```

An archetype provides a feasible envelope, not an unconditional promise. For
example, `cloud-api` is normally a good desktop fit, but a provider may still
need network access, credentials, or a region-specific entitlement. A
`managed-service` may be usable on Windows with Docker Desktop, but that is not
the same claim as a native, self-contained Windows service.

## Terms

### Deployment target

A target is a context in which a resource may be used, such as `desktop`,
`server`, `cloud`, or `remote-client`. A target includes OS/architecture where
relevant; "desktop" is not one platform.

### Deployment mode

The mode says how the resource reaches and runs on that target:

| Mode | Meaning |
|---|---|
| `bundled-client` | A signed Go client/configuration component ships; no local service is owned. |
| `bundled-service` | A signed controller and a separately pinned server binary ship; the desktop runtime uses an app-private background component unless the embedding application has explicit shared-use consent and a broker-issued scoped binding. |
| `native-host-tool` | The resource uses a separately installed host executable after discovery/version preflight. |
| `docker-desktop` | The resource requires Docker Desktop or Docker Engine; this is an explicit conditional path. |
| `remote-service` | The bundle validates and connects to a user- or organization-managed endpoint. |
| `manual` | Vrooli can document/validate prerequisites but does not own lifecycle. |

### Host requirement axes

Every host tool, safeguard, and resource carries two independent declarations.
They are inputs to desktop admission; neither is an informal label.

| Axis | Values | Meaning |
|---|---|---|
| `privilege` | `none`, `user`, `elevated` | The maximum privilege needed to install or operate the requirement on a Vrooli-owned host. `elevated` work is confined to the explicit project setup boundary. |
| `bundling` | `vendorable`, `host-required`, `prohibited` | Whether a desktop application may supply the requirement itself, must discover it on the target, or must never ship it. |

`privilege` and `bundling` answer different questions. For example, a package
may require elevation to install on Linux yet still be vendorable; a host
safeguard is elevated and prohibited from bundles. A required resource that is
not eligible for a target is recorded as a named build limitation, while its
runtime readiness remains terminal rather than silently degraded.

### Support status

| Status | Meaning |
|---|---|
| `supported` | The mode is intended to work on this target and has required evidence. |
| `conditional` | It works only with explicit host requirements, permissions, hardware, or user setup. |
| `degraded` | The scenario can still run, but a declared capability/fallback changes. |
| `unsupported` | The target cannot satisfy a required resource at runtime. A validation artifact may still be built with an explicit limitation, but it is non-promotable and the runtime must refuse the unavailable service. |

### Deployment readiness versus maturity

Deployment readiness is an automatable target-specific admission check.
Resource maturity is broader: it also covers architecture, configuration/data
migration, capability correspondence, maintainability, and consumer evidence.

Do not infer one from the other. A resource can be M5 for a Linux server-only
contract while correctly reporting `unsupported` for native Windows desktop.

## Declared Inputs and Derived Verdicts

The manifest boundary is deliberately one-way: owners declare facts, and the
deployability resolver computes conclusions from those facts. A computed value
must not be copied back into a manifest as if it were an authored claim.

| Manifest surface | Declared input | Derived verdict/evidence |
|---|---|---|
| `resource.json` | `driver`, `platforms`, `bundling`, `privilege`, `requirements`, `deployment.profiles` | Per-resource deployability, supply eligibility, platform gaps, and provenance digest |
| `tool.json` / `safeguard.json` | `platforms`, `bundling`, capability and capability role, acquisition/handler data | Capability peer resolution and host admission verdict |
| `service.json` | dependencies, tier requirements, adaptations, secrets, artifacts, overrides | Tier/OS verdict, aggregate requirements, ordered reasons, and stale status |
| operator state | trusted-base and core-seed grants | Closure validation and trusted-base membership |
| native gate evidence | observed machine, OS, target, run result | Verified cross-OS evidence; never an input to the manifest |

`portability_tier`, tier `status`, tier `fitness_score`, tier `constraints`,
`aggregate_requirements`, analyzer identity, and analysis timestamps are
derived outputs. They are not resource or scenario manifest fields. Persisted
reports carry analyzer identity, computation time, and an input digest so a
consumer can distinguish a current prediction from stale evidence.

The capability ledger is a generated readout, never an authored manifest. Run
`vrooli capability ledger --json` to derive per-OS implementation, mechanism,
and peerless status from the checked-in tool and safeguard manifests. The
related `vrooli capability fleet` queries derive scenario blockers, Docker
requirements, peerless capability use, tier-upgrade candidates, and the
desktop-bundling verdict from the same resolver inputs. Do not copy those
results into `service.json`, resource manifests, or a prose table.

## Native Artifact Principle

Go is a build-time tool, not a required dependency of a deployed resource.

```text
CI/release environment (has Go)
  → cross-builds signed resource artifacts per OS/architecture
  → packages binary + sibling manifest + build metadata

Desktop device (does not need Go or Bash)
  → receives the matching verified artifact in the desktop bundle
  → desktop runtime starts it directly or uses its typed control contract
```

The current project bootstrap uses this same broad pattern for `vrooli`: a
small bootstrap selects a platform artifact, authenticates the signed release
manifest, verifies checksums, and installs a prebuilt native binary. The
current POSIX bootstrap uses `/bin/sh` and `curl`, but not Bash or Go. A native
desktop installer can avoid even that bootstrap because it contains the
verified artifacts itself.

`cli-installer` remains useful for source/developer workflows: it builds a Go
module, writes a sibling manifest and build metadata, and supports freshness
rebuilds. It is not the deployed-desktop installer. See
[`packages/cli-core/README.md`](../../packages/cli-core/README.md).

### One acquisition contract

The arrival of a native artifact is separate from the launch artifact. The
`managed_service.acquisition` block is the source contract: it declares an
ordered `targets` list, a URL or digest-pinned OCI image, download digest when
applicable, archive/layout, launch path, and a `when` predicate over host
facts. The `managed_service.artifact` block remains the launch gate and holds
the executable checksum or tree digest.

The same contract is consumed by `vrooli resource install`,
`vrooli-dist --resource-artifacts`, and the desktop bundler. `os` and `arch`
are build-time facts. GPU and other machine facts are runtime-only and cannot
be used by a `vendorable` item. This keeps signed releases deterministic while
allowing host-required resources to resolve a more precise target on the
machine where they run.

#### Per-target acquisition kind

`acquisition.kind` sets the default source mechanism, and a single target may
override it with its own `kind`. This exists because one item can have genuinely
different upstreams per platform. PostgreSQL is the worked example: on Linux it
stages a digest-pinned OCI filesystem tree, because the official image carries
the shared libraries the server links against, while on macOS and Windows it
stages a checksum-pinned upstream archive, because no equivalent image exists
and the published archives are self-contained. Declaring one acquisition-wide
kind would force one of those two to be wrong.

Use a per-target `kind` only when the upstreams really differ. A single kind for
every target remains the normal case and the easier contract to read.

#### Per-target launch and bootstrap differences

Three fields express platform variation without a `runtime.GOOS` branch in the
driver:

| Field | Replaces | Use when |
|---|---|---|
| `acquisition.targets[].bin_path` | `artifact.entry_path` | The staged tree roots the executable differently per target |
| `managed_service.arguments_by_platform` | `managed_service.arguments` | A launch argument is meaningless or harmful on a platform |
| `managed_service.bootstrap.executable_by_platform` and `bootstrap.arguments_by_platform` | the matching `bootstrap` fields | The first-run tool or its flags differ per target |

Keys are `os` or `os-arch`; an exact `os-arch` key wins over the bare `os`. Each
override replaces the declared default entirely rather than merging, so the
effective value is readable in one place. PostgreSQL on Windows uses two of
these: `-k` names a Unix-domain socket directory, and Windows PostgreSQL has no
Unix-domain sockets, while `--auth-local` writes a `pg_hba.conf` `local` line
that exists only on Unix.

Managed services may also declare `data_artifacts`. These are non-executable,
checksum-verified inputs such as model files staged below `RESOURCE_DATA_DIR`.
They use the same fact-predicated acquisition contract and are installed by
`vrooli resource install` before the service starts, but are never accepted as
the supervisor's launch artifact. Keeping model delivery explicit prevents a
native server archive from silently depending on hand-staged model bytes.

## Pinned Runtime Principle

The Native Artifact Principle pins managed-service binaries by checksum. The
same determinism requirement applies to the container archetypes: a mature
`managed-service` or `managed-service` resource references every pulled image
by an immutable reference — a version tag at minimum, a digest preferred.

Floating references (`latest`, `stable`, `latest-*`) are not acceptable in a
normal path. With a floating tag, `install` (a pull) can silently replace the
running engine, which makes regressions indistinguishable from upstream
changes and makes two hosts running "the same" resource behaviorally
different. Upgrades must be explicit manifest/compose edits reviewed like any
other change, never side effects of a lifecycle verb.

Locally built images (a compose `build:` directive) are exempt: their content
is pinned by the repository's own Dockerfile and base-image pins.

Enforcement:

- the closed resource-driver validator rejects `managed-service` declarations
  unless an explicit operator override carries both a reason and an ISO review
  date (`vrooli capability conformance --declarations-only`;
  `internal/deployability.CheckResourceDeclarations`); this keeps the Docker
  driver available for control-plane observation while preventing new resource
  runtime dependencies from silently acquiring a daemon
- manifest validation rejects floating `runtime.image` references for
  `managed-service` resources (`internal/resources/manifest`,
  `ValidatePinnedImageRef`)
- a fleet lint walks `managed-service` compose files and their GPU overlays
  and rejects floating pulled-image references
  (`internal/resources/manifest/runtime_pinning_realmanifest_test.go`);
  grandfathered debt is listed there explicitly and must shrink, not grow

## Target `resource.json` Shape

Existing manifest fields remain authoritative for runtime, ports, health,
storage, environment, and host requirements. The target adds a deployment
section rather than hiding delivery behavior in shell installers.

```json
{
  "name": "example-resource",
  "template": "managed-service",
  "driver": "managed-service",
  "template_version": "3",
  "managed_service": {
    "provider_policy": {
      "target_defaults": {
        "control-plane": "managed-shared",
        "desktop-bundle": "managed-private"
      },
      "allowed_modes": ["managed-private", "managed-shared", "attach-only", "remote-vrooli", "managed-discovered"],
      "shared_reuse_requires_consent": true,
      "external_management": "forbidden",
      "external_access_capabilities": ["read-only", "read-write"]
    },
    "artifact": {
      "path": "bin/example-service",
      "version": "1.0.0",
      "bundle_artifact": "example-service_${os}_${arch}",
      "sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    },
    "acquisition": {
      "kind": "url",
      "targets": [
        {
          "when": { "os": "linux", "arch": "amd64" },
          "url": "https://example.invalid/example-service-linux-amd64.tar.gz",
          "sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
          "archive": "tar.gz",
          "layout": "file",
          "bin_path": "example-service"
        }
      ]
    },
    "attach_health_path": "/v1/sys/health",
    "arguments": ["serve", "--config", "${VROOLI_RESOURCE_CONFIG_DIR}/service.json"]
  },
  "cli": {
    "command": "resource-example-resource",
    "adapter": { "kind": "go_module", "module_dir": "cli" },
    "artifacts": {
      "binary": "resource-example-resource",
      "manifest": { "location": "sibling" },
      "build_metadata": { "location": "sibling" }
    },
    "source_build": {
      "kind": "go_module"
    },
    "freshness": { "inputs": ["cli/**", "docs/**", "README.md", "resource.json"] },
    "distribution": {
      "kind": "prebuilt_artifact",
      "artifact_name": "resource-example-resource_${os}_${arch}"
    }
  },
  "deployment": {
    "profiles": {
      "desktop": {
        "windows": { "support": "conditional", "mode": "bundled-service", "architectures": ["amd64", "arm64"], "limitations": ["Requires a signed, target-specific service artifact and bundle smoke evidence."], "evidence": ["manifest-validation", "artifact-checksum"] },
        "macos": { "support": "conditional", "mode": "bundled-service", "architectures": ["amd64", "arm64"], "limitations": ["Requires a signed, target-specific service artifact and bundle smoke evidence."], "evidence": ["manifest-validation", "artifact-checksum"] },
        "linux": { "support": "conditional", "mode": "bundled-service", "architectures": ["amd64", "arm64"], "limitations": ["Requires a signed, target-specific service artifact and bundle smoke evidence."], "evidence": ["manifest-validation", "artifact-checksum"] }
      }
    }
  }
}
```

### Required information in a target profile

Every claimed profile must declare or derive:

- support status and deployment mode
- supported OS and architecture coverage
- delivery artifact or external dependency
- host requirements: Docker, GPU, microphone, administrator permission,
  entitlement, network, credentials, and so on
- capability-specific limitations and safe fallback/degradation behavior
- validation evidence required before the claim is admitted

The manifest must never use `supported` as shorthand for "might compile." A
support claim means the declared normal path is Go-native or a declared
external runtime, contains no Bash or source build requirement on the target,
and has the corresponding validation evidence.

## Archetype Baselines

Use these as starting policies. A resource profile narrows or qualifies them;
it cannot silently claim more than its runtime can deliver.

| Archetype | Best desktop mode when feasible | Typical limitation |
|---|---|---|
| `cloud-api` | `bundled-client` | Network, credentials, provider availability; not offline. |
| `native-cli` | `bundled-client` | Must cross-build and package the owned binary. |
| `managed-service` | `bundled-service` | The resource declares provider authority explicitly; a bundled runtime may only start a Vrooli-owned verified artifact. |
| `external-cli` | `native-host-tool` or bundled companion | Upstream binary, licensing, and version support determine feasibility. |
| `managed-service` | `docker-desktop` | Docker Desktop/Engine is a conditional host dependency. |
| `managed-service` | `docker-desktop` | Same host dependency plus multi-container resource cost. |
| `native-cli` | `native-host-tool` | Platform-specific install/discovery and permissions. |
| `native-cli` | `manual` | Never claim automatic lifecycle ownership. |

`managed-service` is the canonical contract for a Vrooli-owned local process.
It does not authorize a resource to adopt an arbitrary process or open port:
only a verified Vrooli-owned instance may be reused. An external endpoint is
always attach-only and may never be initialized, unsealed, stopped, or
rewritten by Vrooli.

### Bootstrap readiness for stateful managed services

Generic supervision proves only that a verified child process was started and
that its transport can be reached. A stateful resource must add its own
readiness transition before generic health can be reported to consumers. Vault
uses the following ordered states: `process-started`, `reachable`,
`uninitialized`, `sealed`, `unsealed`, and `usable`. An HTTP `501` from Vault
means `uninitialized`: it is reachable, but not healthy or consumable.

The resource-native bootstrap adapter owns initialization and recovery. It
stores recovery material only in supported platform secure storage, unseals an
already initialized service after restart, creates a scoped application
credential, and proves a harmless scoped operation before publishing the
service to a broker or application. Secure-storage failure, missing recovery
material, a sealed service that cannot be recovered, or a failed scoped probe
are terminal readiness failures; callers must not downgrade them to transport
health or silently expose a root credential.

Managed-service policies declare `target_defaults` rather than one static
default: `control-plane` selects the Vrooli-owned shared host, while
`desktop-bundle` selects an app-private verified artifact. An explicit provider
request may override the target default only when the mode is allowed; desktop
shared reuse also requires explicit consent. Non-service host-tool policies
continue to use `default_mode` because they have no desktop service lifecycle.

A shared-use lease carries no credential and never grants management authority.
After authorization, the resource-specific policy adapter may issue an
ephemeral credential for exactly that lease's app scope and expiry. The broker
persists ownership and leases, not bearer credentials or a shared root token.

`remote-vrooli` is not a direct resource endpoint mode. A thin desktop or
mobile client receives the scenario's API URL and leaves all resource calls on
the remote Vrooli server. A resource client running with this provider MUST
reject direct Vault endpoint operations rather than accepting `VAULT_ADDR` or
other endpoint overrides.

Attach-only use is explicit at runtime: the operator supplies a credential-free
HTTP(S) endpoint and the managed-service driver requests only the declared
`attach_health_path`. Successful validation does not register ownership or
grant start, stop, configuration, initialization, or unseal authority.

`managed-shared` means a verified Vrooli-owned **user resource host**, not a
network-shared endpoint. It owns bootstrap, recovery, upgrades, and the
credential-free broker state once per user. Its secure management material is
stored only through a credential-store adapter — a native platform store, or
the encrypted file store on a host that has none. The host must fail before
Vault initialization when no adapter can store a value. A mode-0600 plaintext
state file is not an acceptable fallback, because its protection is the file
mode alone. `managed-discovered` is reserved for a verified executable
candidate. It never adopts a running process or external
endpoint; after checksum/version or trusted-distribution verification, Vrooli
may supervise that executable with Vrooli-owned state and configuration.

### Credential-store adapters

Every supported platform has a native credential-store adapter, and every
adapter passes one shared conformance suite in
`internal/resources/securestore`:

| Platform | Adapter | Value path |
|---|---|---|
| Linux | libsecret (`secret-tool`) via the Secret Service | value on stdin |
| macOS | Security framework (`SecItemAdd`/`SecItemCopyMatching`/`SecItemUpdate`/`SecItemDelete`) via cgo | value in a `CFData` |
| Windows | Credential Manager (`CredWriteW`/`CredReadW`/`CredDeleteW`) | value in a `CREDENTIALW` |
| Any host with no native store | Encrypted file store (AES-256-GCM per entry) under a key wrapped by the host TPM or an operator passphrase | value sealed in the file, key held outside it |

Every native adapter requires a logged-in desktop session, so a headless host —
a server, a CI runner, a Raspberry Pi — has none of them. The encrypted file
store is the floor under that host class. It is selected only when the native
adapter reports `ErrAbsent`; where a native store works it stays the authority.
A native store that is merely unreachable (`ErrUnavailable`) never falls back,
because splitting credentials across two backends according to transient
session health is worse than an honest degraded state.

No adapter places a credential value in a process argument. The encrypted file
adapter never writes a credential value that is recoverable from the file alone.
An AES-256-GCM sealed entry whose data key is wrapped by the TPM or by an
operator passphrase satisfies that rule: reading the file yields ciphertext,
and the key that opens it is not in the file. Native stores have
platform-specific semantics: a GNOME passwordless `[keyring]` file can be
readable as a GKeyFile even though Secret Service is the authority, and
`vrooli credentials doctor` reports that caveat explicitly. A mode-0600
plaintext Vrooli fallback does not qualify, because any root process, backup,
disk image, or wrong file mode recovers the value from the file by itself.
Owner-only permissions stay required on every file the credential path writes;
they are no longer what the encrypted protection rests on.

This amends, and does not reverse, the Tier-1 decision recorded in swarm record
`rec-72cedb904accee1c`, which removed plaintext local-store provisioning from
Secrets Manager. Vrooli-owned fallback storage remains encrypted; native
platform semantics are reported rather than silently overstated. Environment
variables are injection targets for explicit deployment/runtime configuration,
not a durable credential authority or an implicit fallback.

A darwin build without cgo reports an absent provider rather than failing to
build, and therefore reaches the encrypted file store like any other host with
no native adapter.

Adapters must keep three conditions distinct, because each has a different
operator action: the backend answered and holds no value (`ErrNotFound`), the
backend exists but is unreachable (`ErrUnavailable`), and this host has no
adapter at all (`ErrAbsent`). Collapsing them is what once let an unreachable
keyring session read as an unset API key.

Credential state never blocks a start. A resource whose credential cannot be
resolved reports unhealthy with a named remediation while the scenario runs;
see [Secrets and credentials](../configuration/secrets.md). Write paths are the
exception and stay fail-closed: durable recovery material is only written after
`securestore.ProbeWritable` proves the backend accepts a write.

The generic user-resource host stays resource-agnostic: it owns secure-store
readiness, broker state, loopback ownership checks, and bootstrapper
registration. Resource-native initialization, recovery material, credential
issuers, and control wiring belong to that resource's adapter. Vault is the
reference implementation of this boundary; generic lifecycle code must not
gain Vault-specific branches.

An attach-only resource must declare `external_access_capabilities`. A
read-write capability allows application data writes only; it never converts
the external resource into a Vrooli-managed service.

### Bundled-service provenance and lifecycle

`bundled-service` stages independent artifacts: the resource controller
and the server selected through `managed_service.artifact.bundle_artifact`.
The release pipeline verifies the signed Vrooli checksum manifest and the
manifest-pinned server digest before copying either into a bundle. The runtime
verifies both again before launch. By default, it starts only the plan-selected
server as a background, app-private component. An embedding application can
instead supply a user-consented broker resolver. The resolver must return an
already-authorized, app-scoped binding; the desktop runtime does not discover
host services, select a broker scope, or receive management authority. If that
binding is unavailable or rejected, the runtime uses the verified private
artifact. Scoped connection settings are injected only into the bundle's own
service environment and are never reported through status, logs, or telemetry.
Private state and logs reside under the application data root and are visible
through the authenticated runtime status and log surfaces; the resource does
not create a launcher entry or adopt a host process.

## Native audio resource contract

The audio stack follows the same managed-service rules, but its native runtime
artifacts have an additional portability constraint: a server binary that
loads sibling shared libraries is one artifact tree, not one executable.

- `resources/whisper` supervises the checksum-pinned whisper.cpp server and
  GGML model. Its platform claim is earned independently for each upstream
  target; an XCFramework or an upstream archive that has not been smoke-tested
  as a server is not macOS evidence.
- `resources/sherpa-onnx` owns the Vrooli HTTP/WebSocket adapter for Kokoro
  TTS, streaming STT, speaker operations, and source separation. The server
  bundle contains `server/sherpa-onnx-server` plus the matching sherpa and ONNX
  Runtime libraries under `lib/`.
- A sherpa executable bundle must therefore declare `layout: "dir"`, an
  explicit `entry_path`, and a deterministic tree digest. Its acquisition
  target must also provide the archive layout and `artifact_sha256`; the
  upstream sherpa runtime package alone is not the Vrooli adapter and cannot
  be substituted for it.
- A target remains explicitly `unsupported` until a target-native Vrooli
  adapter bundle is signed, published, acquired into an empty artifact root,
  and smoke-tested through the managed lifecycle. Cross-compilation and
  library availability are build evidence only.

This boundary keeps optional model data (voices, punctuation, speaker, and
separation models) as independently checksum-pinned data artifacts while the
supervised executable tree remains immutable and lifecycle-owned.

## Canonical Mature Source Layouts

All mature resources use the common Go-first skeleton. Generated binaries,
user configuration, secrets, state, logs, and data do not live in this tree.

```text
resources/<name>/
  resource.json
  README.md
  docs/
    OPERATIONS.md
    CONFIGURATION.md              # when configuration exists
  config/
    capabilities.yaml             # declarative capability contract, if needed
  api/internal/<domain>/                 # repo-owned seed assets only, if needed
  cli/
    go.mod
    go.sum
    main.go
    main_test.go
    integration_test.go            # when a runtime exists
    internal/
      ... Go implementation and tests ...
```

The archetype determines the content below `cli/internal/`, not whether it is
allowed to reintroduce shell helpers.

| Archetype | Resource-local Go specialization | Runtime files that belong in source |
|---|---|---|
| `cloud-api` | typed provider client, auth, policy, validation | no daemon or container files required |
| `external-cli` | discovery, version/auth/config adaptation | only declarative host-tool metadata |
| `native-cli` | command domains and owned capability implementation | Go executable code and tests |
| `managed-service` | service, config, migration, health, platform adapters | Go service code; explicit `*_windows.go` gates where necessary |
| `managed-service` | config renderer and Docker API/controller adapter | declarative image/volume/env metadata in manifest |
| `managed-service` | topology/config translation and readiness policy | declarative graph/template assets only when genuinely required |
| `native-cli` | discovery, permissions, launch/status integration | platform adapter code and docs |
| `native-cli` | validation and diagnostics | instructions/validation data, not fake lifecycle scripts |

The target layout declares a Go module, its manifest contract, and only the
runtime assets genuinely required by the selected archetype.

## Examples

### Cloud API: portable client, not a fake local service

```json
{
  "name": "openrouter",
  "template": "cloud-api",
  "driver": "cloud-api",
  "cli": {
    "command": "resource-openrouter",
    "adapter": { "kind": "go_module", "module_dir": "cli" },
    "distribution": {
      "kind": "prebuilt_artifact",
      "artifact_name": "resource-openrouter_${os}_${arch}"
    }
  },
  "deployment": {
    "profiles": {
      "desktop": {
        "windows": { "support": "supported", "mode": "bundled-client", "requires": ["network", "openrouter_api_key"] },
        "macos": { "support": "supported", "mode": "bundled-client", "requires": ["network", "openrouter_api_key"] },
        "linux": { "support": "supported", "mode": "bundled-client", "requires": ["network", "openrouter_api_key"] }
      }
    }
  }
}
```

### Docker service: Go-native controller, conditional desktop support

```json
{
  "name": "example-docker-resource",
  "template": "managed-service",
  "driver": "managed-service",
  "cli": {
    "command": "resource-example-docker",
    "adapter": { "kind": "go_module", "module_dir": "cli" },
    "distribution": {
      "kind": "prebuilt_artifact",
      "artifact_name": "resource-example-docker_${os}_${arch}"
    }
  },
  "runtime": { "image": "registry.example.invalid/example:<pinned-version>" },
  "deployment": {
    "profiles": {
      "desktop": {
        "windows": { "support": "conditional", "mode": "docker-desktop", "requires": ["docker-desktop"] },
        "macos": { "support": "conditional", "mode": "docker-desktop", "requires": ["docker-desktop"] },
        "linux": { "support": "conditional", "mode": "docker-desktop", "requires": ["docker-engine"] }
      }
    }
  }
}
```

This example describes the remaining Docker-service archetype only; it is not
the SearXNG contract. SearXNG is now a `managed-service` with a `composed`
acquisition tree and a `bundled-service` desktop profile, so its desktop bundle
does not require Docker. The Docker requirement remains visible for resources
that still intentionally use this archetype.

## Desktop Bundle Decision Model

`scenario-to-desktop` must build from the scenario's declared dependency graph,
not from a raw list of names or an assumption that every resource can be
embedded. For every required resource and requested target it should:

1. resolve the resource's deployment profile for OS/architecture and tier
2. evaluate host requirements and artifact availability
3. select a declared fallback only when the scenario permits it
4. produce an explicit bundle plan and an operator-facing decision

| Result | Meaning | Packager behavior |
|---|---|---|
| ready | Every required capability has a supported, evidenced route. | Include selected artifacts/configuration. |
| warning | A conditional/degraded route is valid. | Require acknowledgement; include remediation and limitation text. |
| ineligible | A required resource has no valid runtime route on this target. | Build an explicitly non-promotable validation artifact, record the named limitation in `resource-deployment-plan.json`, and keep runtime readiness terminal. |
| error | The plan or its verified inputs are invalid. | Refuse to create any bundle. |

Example: a Windows desktop request that needs a `managed-service` should report
that Docker Desktop is required. A resource declared `unsupported` may still
produce a non-promotable validation artifact with its limitation recorded; it
must not reach a healthy runtime unless the scenario explicitly accepts a
compatible fallback. The system must not discover this by trying to source a
Linux shell script during desktop startup.

## Desktop Runtime Boundary

The desktop runtime/supervisor is a thin, generic local host. It owns:

- verified bundle/update installation
- authenticated local IPC
- process supervision, health, logs, ports, data paths, secrets, and recovery
- OS integration such as app lifecycle, keychain, notifications, and autostart
- preflight and reporting of declared requirements/degradation

It must not build Go code, invoke Bash/PowerShell resource scripts, contain
scenario business logic, or reimplement each resource's configuration and
lifecycle semantics. Resource implementations and their typed contracts remain
the source of truth.

## Readiness Evidence and Validation

Before a profile can claim `supported`, require all applicable evidence:

- manifest/schema validation and a static no-shell/no-source-build check
- target artifact exists for the claimed OS/architecture
- artifact signature/checksum verification in the bundle/release path
- unit tests for configuration, platform gates, and fallback selection
- integration test of the declared runtime or host-tool preflight
- target-platform smoke test exercising one real capability
- scenario consumer smoke test when the capability has consumers

For `conditional` or `degraded`, test the unavailable/unsupported path too:
the user must receive a structured, actionable explanation before a broken
operation occurs.

## Agent Workflow

An assessment or implementation agent should read this document together with
[architecture.md](architecture.md), [resource-templates.md](resource-templates.md),
and [maturity-migration.md](maturity-migration.md).

For an individual resource, the agent should:

1. identify its current archetype and all normal shell/source-build paths
2. select the cleanest feasible deployment mode per target
3. record explicit support, requirements, limitations, and fallbacks
4. distinguish current readiness from target M5 capability
5. propose artifact, schema, template, control-plane, and test changes in that
   order; do not delete old paths before compatibility validation

For a scenario deployment, the agent should:

1. resolve all required resource dependencies and the requested target
2. produce `ready`, `warning`, or `error` for each dependency
3. explain host requirements, unsupported platforms, and permitted fallbacks
4. refuse to call a bundle portable merely because its Go sources cross-compile

## Implementation Status and Follow-up Sequencing

The contract's schema/types, archetype baseline validation, canonical
templates (including `managed-service`), fleet profiles, signed prebuilt
release artifacts, and desktop admission/staging path are implemented.
`scenario-to-desktop` verifies the release signature and selected artifact
checksums before it writes
`resource-deployment-plan.json` into a bundle; it refuses bundled profiles
without a supplied signed artifact root.

### Resource release handoff

### Release trust versus installer signing

Release trust proves that every bundled resource executable and vendored tool
is the exact byte set approved for a Vrooli release. It is not Windows,
macOS, or Linux installer code signing. A staged release contains one
deterministic `release-manifest.json`, listing every shipped file, its SHA-256,
role, platform metadata where applicable, and upstream-provenance evidence.

`development-local` verifies that manifest and every staged file, but does not
require a Vrooli authority signature. Bundles built this way are explicitly
non-promotable and must not be published. `production` additionally requires
`release-manifest.sig.json`, a detached RSA/SHA-256 envelope signed by the
project-managed release authority. `vrooli release-authority` generates and
retains the private half in the native credential authority and publishes only
the public trust anchor at `install/vrooli-release.pub`; releases identify the
signing key with `key_id`. Missing or invalid production signatures fail before
packaging. Installer code-signing remains separately optional and is documented
by scenario-to-desktop's code-signing guide. See
[`docs/configuration/release-authority.md`](../configuration/release-authority.md).

Release automation may verify the staged directory independently before
bundling:

```bash
vrooli-dist --verify-release-manifest \
  --release-artifact-root /path/to/staged-release \
  --trust-mode production \
  --release-public-key /secure/trust/vrooli-release.pub
```

`vrooli-dist --resource-artifacts` is the source-build boundary. It stages the
controller artifacts, any separately pinned managed-service server artifacts,
and a deterministic `SHA256SUMS` manifest. It deliberately does **not** create
`SHA256SUMS.sig`: the release signer is a separate authority and must sign that
exact manifest using the key whose public half is distributed with Vrooli.

For example, a Linux release operator stages the artifacts, has the release
signing system produce `SHA256SUMS.sig`, and only then passes the directory to
the desktop pipeline:

```bash
go run ./cmd/vrooli-dist --root "$PWD" --out-dir /secure/release-stage --resource-artifacts
# Release signing authority writes /secure/release-stage/SHA256SUMS.sig.
scenario-to-desktop pipeline run <scenario> --platforms linux-amd64 \
  --deployment-mode bundled --resource-artifact-root /secure/release-stage --wait
```

An unsigned directory is useful for inspection but is intentionally not
deployable. This prevents a developer machine from substituting locally built
resource bytes for an authorized release.

The control plane now has a managed-service driver. It validates a relative,
manifest-pinned artifact, verifies its SHA-256 before direct argv execution,
and persists only credential-free instance/lineage state. It deliberately
refuses to manage attach-only or shared providers without their respective
broker paths. A target must also provide a process-ownership verification
adapter before it can claim managed lifecycle support; a PID, port, or marker
file is never enough evidence on its own.

The remaining work for a conditional bundled-service target should sequence as
follows:

1. run the target-host smoke test against the signed release artifact on every
   claimed OS/architecture pair, then promote only the verified profile to
   `supported`
2. exercise private, shared, remote, and attach-only provider paths through
   their declared consumer contracts
3. migrate the fleet using the maturity playbook and delete transitional shell
   paths only after validated replacement
