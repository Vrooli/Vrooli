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
    initialization/
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
- `initialization/`
  - seeded config/assets/bootstrap content when the resource needs repo-owned initialization material
- `internal/`
  - resource-specific Go code

### Transitional only

- `lib/`
- `cli.sh`
- shell-owned files such as `config/defaults.sh`, `config/messages.sh`, or
  generated configuration mutated by shell scripts

A declarative `config/capabilities.yaml`, static schema, or repo-owned template
asset may be part of a mature resource. Shell-owned configuration is not; it
may remain only during migration and must have a named replacement/removal
milestone.

## Ownership Boundaries

### `resource.json`

`resource.json` is the authoritative declarative artifact for:

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

### `docker-service`

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

### `compose-service`

Use when the resource is genuinely a multi-service graph.

Expected design:

- compose graph is explicit and honest
- shared control plane owns orchestration and readiness policy
- service dependencies are modeled, not hidden behind sleeps or shell sequencing hacks

As with `docker-service`, use this only for a real multi-service graph. It is
not a substitute for a portable local runtime design.

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

### `cloud-api`

Use when Vrooli does not own the service runtime, only the local representation and validation of a hosted capability.

Expected design:

- no fake lifecycle ownership of the hosted service
- shared control plane handles credential/config validation and connectivity probing
- resource-local code handles provider-specific behavior where necessary

### `desktop-app`

Use when the resource is a native desktop application dependency.

Expected design:

- platform support is stated honestly
- unsupported platforms fail clearly
- manual/host-specific installation is documented instead of hidden behind weak automation

### `manual`

Use when Vrooli can describe and partially validate a resource but does not own its lifecycle fully.

Expected design:

- docs and validation are primary
- automation promises remain intentionally limited
- the control plane should not pretend full ownership exists

## Template-Kind Expectations

Phase 2 defines what each canonical template kind is expected to converge toward.

### `docker-service`

- minimal resource-local code
- shared driver owns most lifecycle behavior
- resource-local config generation only when required
- resource storage rooted outside repo

### `compose-service`

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

### `desktop-app`

- host path detection and unsupported behavior are explicit
- platform-specific limitations are documented honestly
- generated scaffolds should not hide manual prerequisites

### `manual-resource`

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
