# Resource Architecture

This page defines the target implementation architecture for active resources.

It exists to lock the design center before broad migration work proceeds, so future resources and migrated resources converge on the same shape rather than drifting between manifest-only, shell-first, and partially native patterns.

## Purpose

The target resource architecture should make these things true:

- `resource.json` remains the declarative authority
- `vrooli resource ...` remains the canonical operator surface
- runtime/storage behavior is owned by shared Go control-plane code where possible
- per-resource code exists only where specialization is real
- retained shell code is treated as transitional compatibility, not as the default model

## Design Center

Resources are infrastructure/control-plane units, not scenario APIs.

That means new and migrated resource implementations should optimize for:

- manifest-backed configuration
- platform-aware orchestration
- honest portability classification
- shared runtime/storage logic
- minimal per-resource specialization

They should not optimize around historical shell wrappers or repo-local path conventions.

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
      install/
      runtime/
      status/
      health/
      env/
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

- `config/`
- `lib/`
- `cli.sh`

These may remain during migration, but they are not part of the target architecture center.

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

It should describe behavior, not be bypassed by shadow shell-era contracts.

### `cli/`

`cli/` is the resource-local executable entrypoint layer.

It should contain:

- the Go main package for the resource entrypoint when resource-local execution is needed
- resource-local command wiring
- install/bootstrap entry behavior only when resource-specific logic genuinely belongs here

It should not become a dumping ground for all implementation logic.

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

### `external-cli`

Use when the resource is centered on a host executable.

Expected design:

- shared control plane handles discovery, install probes, version checks, and status surfaces
- per-resource code handles only genuinely unique config or credential flows

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
- treat shell as compatibility-only unless a migration step explicitly requires it
- keep storage/runtime roots outside the repo

## Transitional Shell Labeling

Retained shell-era code should be labeled and treated consistently.

### Allowed transitional roles

- compatibility shim
- migration adapter
- bootstrap fallback during phased cutover

### Not allowed as target authority

- primary source of runtime path policy
- hidden contract that overrides `resource.json`
- default implementation model for new resources

### Documentation expectation

If a retained resource still relies materially on shell-era code, its docs should say so explicitly and describe:

- what shell code still owns
- why it still exists
- what the intended migration destination is

## Resource Storage Relationship

This architecture document assumes the resource storage policy in:

- [storage.md](storage.md)

Short version:

- resources should not standardize on `api-core/storage`
- resources should converge on a resource-specific shared storage/runtime layer
- repo-local `data/` is transitional, not target architecture

## Scenario Relationship

This architecture is intentionally separate from scenario runtime architecture.

Recommended split:

- scenarios use `api-core/storage`
- resources use a resource-specific shared control-plane storage/runtime layer

That separation preserves clean domain boundaries while still allowing similar class vocabulary and portability expectations.

