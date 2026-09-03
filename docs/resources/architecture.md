# Resource Architecture

This page defines the target implementation architecture for active resources.

It exists to lock the design center before broad migration work proceeds, so future resources and migrated resources converge on the same shape rather than drifting between manifest-only, shell-first, and partially native patterns.

## Purpose

The target resource architecture should make these things true:

- `resource.json` remains the declarative authority
- `vrooli resource ...` remains the canonical operator surface
- runtime/storage behavior is owned by shared Go control-plane code where possible
- per-resource code exists only where specialization is real
- mature resources have no Bash dependency in their operator, lifecycle, configuration, or test paths
- any retained shell code is explicitly tracked migration debt, never an accepted mature-state implementation

## Design Center

Resources are infrastructure/control-plane units, not scenario APIs.

That means new and migrated resource implementations should optimize for:

- manifest-backed configuration
- platform-aware orchestration
- artifact-based deployment instead of source builds on end-user machines
- honest portability classification
- shared runtime/storage logic
- minimal per-resource specialization

They should not optimize around historical shell wrappers or repo-local path conventions.

## Maturity and Portability

A resource is mature only when its normal operation is implemented through the
manifest and Go-native control plane. Bash is not portable to native Windows
and therefore cannot be part of a mature resource's required execution path.

This applies to lifecycle operations, configuration, diagnostics, and tests:

- `vrooli resource ...` and the resource's Go CLI are the supported operator
  surface.
- Resource-specific behavior is implemented in Go or a declared external
  runtime, not in sourced shell libraries.
- Platform-specific behavior is isolated behind explicit platform gates or
  implementation files; unsupported platforms fail early with an actionable
  message.
- A shell installer or compatibility shim may exist only during a bounded
  migration. It must not be required after installation, and it must have a
  removal criterion.

"Cross-platform" does not require every resource to run on every operating
system. It requires an honest contract: supported systems work through the
same operator surface, and unsupported systems are declared and rejected
cleanly rather than failing inside Linux-oriented shell code.

## Canonical Resource Layout

Target resource structure:

```text
resources/
  <name>/
    resource.json
    README.md
    docs/
    api/internal/<domain>/
    cli/
      main.go
      internal/
        ...
```

### Required

- `resource.json`
  - canonical declarative contract
- `README.md`
  - operator-facing overview
- `cli/`
  - executable entrypoint for resource-specific command handling when needed

### Common but optional

- `docs/`
  - deeper resource-local documentation
- `api/internal/<domain>/`
  - seeded config/assets/bootstrap content when the resource needs repo-owned bootstrap material
- `internal/`
  - resource-specific Go code

### Retired legacy shapes

- `lib/`, `cli.sh`, and shell-owned configuration files are not valid resource
  surfaces. Mature resources keep typed configuration under `cli/internal/`
  and expose lifecycle through the Go control plane.

A declarative `config/capabilities.yaml`, static schema, or repo-owned template
asset may be part of a mature resource when each entry has implementation and
test evidence.

## Ownership Boundaries

### `resource.json`

`resource.json` is the authoritative declarative artifact for:

See [authoring-a-resource.md](authoring-a-resource.md) for the decision tables
and validation path used when adding a resource.

- name and display metadata
- driver choice
- portability tier
- platform support
- ports
- health checks
- environment exports
- orchestration behavior
- runtime volume/env metadata
- deployment profiles, artifact delivery, host requirements, limitations, and fallbacks (target contract; see [deployment-contract.md](deployment-contract.md))

It should describe behavior, not be bypassed by shadow shell-era contracts.

### `cli/`

`cli/` is the resource-local executable entrypoint layer.

It should contain:

- the Go main package for the resource entrypoint when resource-local execution is needed
- resource-local command wiring
- resource-specific command/configuration behavior only when it genuinely belongs here

It should not become a dumping ground for all implementation logic.

For a source checkout, `cli-installer` may build this module. A desktop/release
target receives a prebuilt, signed artifact instead; a resource-local shell
installer is not part of the target architecture.

The preferred posture is:

- thin entrypoint in `cli/`
- shared platform/control-plane code in common packages
- only real specialization implemented per resource

### Shared Go control-plane packages

Shared resource platform code should own:

- storage/path resolution
- env rendering and export resolution
- driver implementations
- common health-check execution
- orchestration primitives
- common logs/status behavior
- portability validation and classification helpers

Recommended target area:

- `internal/resources/runtime/...`

That shared code is the real architecture center for native resource migration.

### Per-resource `internal/`

Per-resource `internal/` code should exist only for specialization that cannot be expressed through:

- manifest metadata
- shared drivers
- shared runtime helpers

Examples:

- resource-specific install translation
- resource-specific config generation
- resource-specific health probes
- resource-specific hosted API behavior

If multiple resources need the same logic, that logic should move out of the resource and into shared platform code.

## Driver Responsibilities

Driver choice should reflect actual integration shape.

### Retired container integration

Use when one primary container runtime contract is sufficient.

Docker is not the default merely because it is convenient. It requires a
container runtime/daemon and is consequently a poor fit for many bundled
desktop experiences. Prefer a cloud API, external CLI, or native/managed
runtime when that better matches the capability. Choose Docker when the
container is genuinely the supported and supportable runtime contract.

Expected design:

- declarative container runtime in `resource.json`
- shared driver owns install/start/stop/status/logs behavior
- per-resource code only for true configuration specialization

### Managed-service integrations

Use when the resource is genuinely a multi-service graph.

Expected design:

- compose graph is explicit and honest
- shared control plane owns orchestration and readiness policy
- service dependencies are modeled, not hidden behind sleeps or shell sequencing hacks

Use this only for a real multi-service graph. It is not a substitute for a
portable local runtime design.

### `external-cli`

Use when the resource is centered on a host executable.

Expected design:

- shared control plane handles discovery, install probes, version checks, and status surfaces
- per-resource code handles only genuinely unique config or credential flows

### `native-cli`

Use when the resource is a repo-owned Go binary with a real operator command surface.

Expected design:

- `resource.json` still owns install/invoke/freshness/runtime metadata
- `cli/main.go` stays thin
- `cli/internal/app` owns command registration and app wiring
- resource-local Go packages under `cli/internal/...` own the actual implementation
- the control plane treats the installed binary as the managed interface instead of pretending the resource is a third-party host executable

### Managed local service (target archetype)

Some resources are neither a hosted API nor a good Docker/Compose fit: Vrooli
owns a local service process and should install, configure, supervise, and
health-check it natively. The `managed-service` contract supplies the common
cross-platform process, configuration, health, and provider-authority model.
It permits reuse only of a verified Vrooli-owned service; arbitrary external
endpoints remain attach-only.

#### Declared acquisition

Managed-service and vendorable-tool bytes arrive through one declared
acquisition contract. A manifest provides an ordered list of targets, each
with a URL, composed step list, or pinned OCI image, download verification, artifact layout, and a
`when` predicate over host facts such as `os`, `arch`, `accel.backend`, or
`accel.cuda_compute`. The runtime installer, release stager, and desktop bundler
all resolve that same declaration; none may carry a per-resource fetch table.

### The accelerator contract

A resource that runs work on a device other than the CPU declares it once, in
the `acceleration` block of its `resource.json`:

```json
"acceleration": {
  "backends": ["cuda", "metal", "cpu"],
  "require": "preferred",
  "cuda": { "min_compute": "8.9", "compose_overlay": "docker/docker-compose.gpu.yml" },
  "metal": {},
  "cpu": {},
  "claim": { "resource_kind": "vram", "preferred_bytes": 0, "...": "..." }
}
```

`backends` is an ordered preference drawn from the closed set `cuda`, `metal`,
`rocm`, `vulkan`, `cpu`; the last entry is the floor. `require` is `required`,
`preferred` (the default) or `none`. The capacity claim lives inside the block,
so a resource cannot reserve VRAM without declaring the backend it needs it on.

**`internal/accel` owns accelerator truth.** A resource declares what it wants;
the control plane decides where it goes, verifies where it landed, and reports
both:

| Question | Answered by |
|---|---|
| What can this host reach? | `internal/hostinventory` publishes `accel.backends`, `accel.backend`, `accel.cuda_compute`, `accel.vram_bytes` and `accel.vendor` |
| Which backend does this resource get? | `accel.Readiness` — the first declared backend the host can reach, with a per-backend verdict for the ones it skipped |
| Where did the process actually land? | `accel.VerifyPlacement` over a `PlacementTarget` that is a host process, a container, or a compose service |
| Is that a problem? | `mode_drift` on the resource status: `running: true`, `serving: true`, `healthy: false` |

Two rules keep this honest, and both are enforced by tests rather than by
convention:

- `internal/accel` runs no command. Every host observation arrives through a
  `FactSource` reading `internal/hostinventory`, and every container probe
  through an injected `ContainerProbe`. `hostinventory` keeps sole ownership of
  vendor-tool calls, so a second copy of `nvidia-smi` parsing cannot grow.
- Placement is read from the host or reported as `unknown`. It is never inferred
  from configuration, from an environment variable, or from a log line. A
  resource is never trusted about its own placement.

The same code path selects and verifies `cuda` on Linux, `metal` on macOS and
`rocm` on an AMD host. Where no live host was available to verify placement, the
status reports `unknown` rather than assuming agreement — see
[platform-support.md](../reference/platform-support.md).

The host fact provider supplies runtime facts. Directory artifacts are verified
by a tree digest and launch only their declared entry path. This includes
interpreted-runtime services such as SearXNG, where a relocatable Python
runtime, locked wheels, and application source are composed into one verified
tree before launch. Release staging supplies only
build-time facts (`os` and `arch`), so a vendorable item may not predicate on a
GPU or other target-machine property. Directory artifacts are verified by a
tree digest and launch only their declared entry path. A single-file artifact
is verified byte-for-byte immediately before execution. The operator can use
`vrooli resource acquisition explain <name>` to see the facts, candidates,
rejections, and selected target.

### Control-plane host ownership

The control plane keeps host observation, host requirement resolution, and
host actuation separate. The package boundaries are:

| Package family | Owns | Does not own |
|---|---|---|
| `internal/hostfacts` | Cached, low-level host facts and their freshness | Requirement policy or remediation |
| `internal/hostinventory` | OS/tool/device probes and normalized inventory | Resource-specific policy or privileged writes |
| `internal/hostlifecycle` | Host lifecycle observations and service-manager seams | Scenario lifecycle orchestration |
| `internal/hostpresentation` | Presentation/session capability classification | Desktop policy or UI rendering |
| `internal/hostpressure` | Memory, thermal, and pressure observations | Capacity reservations or process admission |
| `internal/hostsession` | Session identity and host-session facts | Credential storage or operator authorization |
| `internal/hostreqspec` | Declarative host-requirement vocabulary | Host probing and execution |
| `internal/hostreq` | Requirement resolution and eligibility verdicts | Privileged host mutation |
| `internal/hostreqcheck` | Static manifest/handler consistency checks | Runtime host observation |
| `internal/hostreqkit` | Shared handler and host-actuation interfaces | Scenario-owned repair implementations |
| `internal/hostreqrun` | Resolve-then-ensure orchestration | Defining requirements or bypassing the control plane |

`packages/hostreq` is the public façade over `internal/hostreq`; the `hostreq*`
packages remain distinct because their inputs and side effects differ as shown
above. Repository-root discovery has one authority in
`packages/repo-contract-go`; application services and safeguards delegate to
that resolver rather than walking for private marker files.

Use when Vrooli does not own the service runtime, only the local representation and validation of a hosted capability.

Expected design:

- no fake lifecycle ownership of the hosted service
- shared control plane handles credential/config validation and connectivity probing
- resource-local code handles provider-specific behavior where necessary

### Retired desktop integration

Use when the resource is a native desktop application dependency.

Expected design:

- platform support is stated honestly
- unsupported platforms fail clearly
- manual/host-specific installation is documented instead of hidden behind weak automation

### Retired fallback integration

Use when Vrooli can describe and partially validate a resource but does not own its lifecycle fully.

Expected design:

- docs and validation are primary
- automation promises remain intentionally limited
- the control plane should not pretend full ownership exists

## Template-Kind Expectations

Phase 2 defines what each canonical template kind is expected to converge toward.

### Managed-service baseline

- minimal resource-local code
- shared driver owns most lifecycle behavior
- resource-local config generation only when required
- resource storage rooted outside repo

### Managed-service extension

- compose graph is first-class
- readiness and dependency order are explicit
- shared orchestration code owns most execution behavior

### `external-cli`

- binary detection and install guidance are first-class
- version detection is separate from auth/config validation
- shared probing/install behavior is preferred over resource-local shell glue

### `native-cli`

- repo-owned Go binary is the design center
- `cli/internal/app` owns the operator command surface
- `cli/internal/<domain>` owns resource-local implementation logic
- the manifest still owns command/install/invoke/freshness metadata

### `cloud-api`

- endpoint + credentials + validation are central
- no runtime ownership fiction
- secrets are referenced, not embedded

### Native-cli desktop extension

- host path detection and unsupported behavior are explicit
- platform-specific limitations are documented honestly
- generated scaffolds should not hide manual prerequisites

### Managed-service fallback

- validation/doc clarity over lifecycle automation
- limits are explicit
- control plane support remains intentionally partial

### Migration-only adapter pattern

`legacy-adapter` remains a migration pattern, not a design-center template kind.

Use it only when:

- a resource still depends materially on shell-era behavior
- migration needs a bounded compatibility phase
- the retained adapter is explicitly documented as transitional

Do not treat `legacy-adapter` as the normal starting point for new resources.

## Native-Go Convergence Rules

For future resource work:

- do not start from copying an old `resources/<name>/` directory
- start from blueprint -> template -> implementation
- prefer shared Go control-plane packages over resource-local reinvention
- keep `cli/` thin when possible
- do not introduce Bash into a mature resource
- when a migration temporarily retains shell, document the owner, callers,
  replacement, and removal criterion
- keep OS-specific behavior behind explicit Go platform boundaries or declared
  unsupported-platform gates
- keep storage/runtime roots outside the repo

## Transitional Shell Labeling

Retained shell-era code should be labeled and treated consistently.

### Allowed transitional roles

- compatibility shim
- migration adapter
- bootstrap fallback during phased cutover

These are migration exceptions, not maturity levels. New resources must not
start with them.

### Not allowed as target authority

- primary source of runtime path policy
- hidden contract that overrides `resource.json`
- default implementation model for new resources

### Documentation expectation

If a retained resource still relies materially on shell-era code, its docs should say so explicitly and describe:

- what shell code still owns
- why it still exists
- what the intended migration destination is
- the bounded condition that permits removing the shell code

## Resource Migration Completion Gate

Do not call a resource migration complete merely because a Go CLI exists. A
migrated resource is complete when:

- fresh setup/configuration succeeds without Bash and preserves existing
  operator-owned configuration
- lifecycle, logs, status, health, and diagnostics use the Go-native surface
- resource-specific configuration is typed, validated, and migration-safe
- a hermetic test suite covers resource-specific behavior and an integration
  test covers the declared runtime
- at least one real consumer smoke test exists when the resource has consumers
- manifest capabilities correspond to implemented commands and tests
- no normal path sources legacy shared shell code
- platform support is tested or explicitly gated
- each claimed deployment target has an explicit delivery mode, host
  requirements, limitations/fallbacks, and validation evidence
- desktop/release operation starts from a verified native artifact or declared
  external runtime, never a local Go build or Bash path

For the repeatable assessment, scoring, archetype-selection, and planning
workflow, see [maturity-migration.md](maturity-migration.md).

For target-specific artifact, desktop, cloud/server, and degradation behavior,
see [deployment-contract.md](deployment-contract.md).

## Resource Storage Relationship

This architecture document assumes the resource storage policy in:

- [storage.md](storage.md)

Short version:

- resources should not standardize on `package:api-core/storage`
- resources should converge on a resource-specific shared storage/runtime layer
- repo-local `data/` is transitional, not target architecture

## Scenario Relationship

This architecture is intentionally separate from scenario runtime architecture.

Recommended split:

- scenarios use `package:api-core/storage`
- resources use a resource-specific shared control-plane storage/runtime layer

That separation preserves clean domain boundaries while still allowing similar class vocabulary and portability expectations.
