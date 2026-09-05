# Scenario CLI Manifest Decision

## Status

- Accepted
- Date: 2026-04-15

## Why This Exists

Vrooli has partially migrated CLI handling away from filesystem inference and toward manifest-driven behavior, but the repo still contains mixed assumptions:

- some scenario CLI paths already resolve from `service.json`
- resource CLI handling still assumes `path:resources/<name>/cli`
- setup and validation code still infer behavior from layout in places

That mixed state is not an acceptable steady state. The platform contract must be explicit and uniform.

## Decision

CLI behavior is manifest-driven only.

### Scenarios

For scenarios, `.vrooli/service.json` is the sole CLI contract.

The top-level `cli` block is required on every scenario manifest, even when the scenario has no CLI. In that case, the manifest must still declare the block explicitly with `enabled: false`.

Scenario CLI behavior must be derived from declared manifest data, not from inferred filesystem structure.

### Resources

Resources will adopt a first-class top-level `cli` block parallel to the scenario model.

The resource manifest must declare CLI behavior explicitly. Resource CLI handling must not rely on implicit `path:resources/<name>/cli` conventions as part of the platform contract.

The long-term goal is one shared contract shape across scenarios and resources wherever practical.

### Supported Adapter Kind

Scenarios and resources use the `go_module` adapter. The control plane builds
and installs that declared module; CLI behavior is never inferred from nearby
filesystem layout.

### Invocation Policy

Invocation remains intentionally constrained to:

- `cli.invoke.kind = "installed_command"`

This is deliberate platform policy for now, not an accidental limitation.

The platform is therefore adapter-agnostic for implementation and install mechanics, but not fully invocation-agnostic.

### Command Naming

Every declared CLI must explicitly declare `cli.command`.

For resources, the expected convention is still that the declared command will normally be `resource-<name>`, but that value must be present in the manifest rather than inferred by runtime code.

Resource templates and generators must support this command as a real template variable, with the value rendered into the generated manifest and any CLI scaffolding that depends on it.

### Freshness Ownership

CLI freshness is manifest-driven.

That means:

- the manifest declares the freshness contract
- the adapter declares the required implementation files needed for that adapter kind
- no generic `app_root/cli`, `cli/go.mod`, `cli/*.go`, or `path:cli/<name>` inference is allowed unless the declared adapter makes those files relevant

Adapter-required files are part of the adapter contract. Everything else must be declared through manifest-owned freshness inputs.

### Explicitly Unsupported

The platform contract explicitly rejects these behaviors:

- implicit `cli/go.mod` inference
- implicit `path:cli/<scenario-name>` inference
- implicit `path:resources/<name>/cli` discovery
- generic layout-only freshness logic
- silent fallback from manifest-driven resolution to repo-layout guessing

## Consequences

- scenario and resource CLI discovery should converge on shared resolver logic
- resource manifests need a schema and Go type update
- setup freshness checks must become adapter-aware and manifest-aware
- validation and tests must describe remediation in terms of manifest fixes, not layout folklore
- template generation should reuse the scenario-style CLI variable model where possible, especially for resource command naming

## Hard-Cutover Rule

This is a greenfield hard cutover decision.

There is no migration-period contract where CLI omission or layout inference remains valid by policy. Existing code may take multiple implementation passes to converge, but the intended platform contract is already the manifest-only contract described above.

## Related Plans

- [Manifest-Driven CLI Contract Completion Plan](/home/matthalloran8/Vrooli/docs/plans/manifest-driven-cli-contract-completion-plan.md:1)
- [Scenario CLI Manifest Greenfield Migration Plan](/home/matthalloran8/Vrooli/docs/plans/scenario-cli-manifest-greenfield-migration-plan.md:1)
