# Vrooli Configuration

This is the operator-facing reference for configuring a Vrooli install: which scenarios are enabled, which resources are running, what secrets are wired, what host tools and safeguards are opted into. It is the **canonical contract**. The [`vrooli-onboarding`](../../scenarios/vrooli-onboarding/) scenario is one surface over this contract; secrets-manager is another. New surfaces must conform to what's documented here.

## Why this folder exists

Configuration was scattered: credentials in provider-specific stores and resource manifests, scenario state in service.json files, host-tool opt-ins inferred at runtime, and no single doc walked an operator (or AI agent) from "I want to use X" to "X is declared, stored, validated, and consumed." This folder is the unified description of the configuration substrate. If you're trying to figure out where a knob lives or how a setting reaches the running system, start here.

## What lives where

There are three categories of configuration data, and every file in the system fits exactly one. Mixing them is the failure mode the layering prevents.

| Category | Lives in | Edited by | Examples |
|---|---|---|---|
| **Declarative manifest** — what each thing *is* | `resources/<name>/resource.json`, `scenarios/<name>/.vrooli/service.json`, `internal/tools/<name>/tool.json`, `internal/safeguards/<name>/safeguard.json` | Humans, source-controlled | "This resource needs a Postgres dependency", "this safeguard writes to /etc/sysctl.d", "this scenario is system-required" |
| **Computed analysis** — what an analyzer figured out | scenario-dependency-analyzer reports, `.vrooli/schemas/resource-definitions.json` | Tools, regenerable | "This scenario needs 2 GB RAM", "tier-3-mobile fitness score 0.4" |
| **Operator state** — what this install *chose* | `.vrooli/operator-state.json` | The wizard, or hand | "swarm-manager is enabled", "kernel_config safeguard is opted in", "ollama resource is disabled" |

Generated lifecycle/runtime state is not configuration. Setup markers, resource-populated markers, runtime databases (including the SQLite port-claim registry), logs, and process records belong under `~/.vrooli/`; per-project setup/resource markers use `~/.vrooli/state/projects/<project-key>/` so multiple local checkouts stay disambiguated.

See [`architecture.md`](architecture.md) for the full source-of-truth table and the rules for resolving any specific value at runtime (manifest default vs. operator override).

## How to find what you need

If you want to…

- **Enable or disable a scenario** → [`scenarios.md`](scenarios.md)
- **Enable or disable a resource** → [`resources.md`](resources.md)
- **Add an API key (any third-party integration)** → [`secrets.md`](secrets.md)
- **Opt into a host tool or safeguard** → [`host/tools.md`](host/tools.md), [`host/safeguards.md`](host/safeguards.md)
- **Configure auto-restart, always-on, startup behavior** → [`operating-mode.md`](operating-mode.md)
- **Wire an external integration (OAuth, GitHub, coding-agent sign-in)** → [`integrations/`](integrations/)
- **Use a named profile (engineering / marketing / homelab)** → [`profiles.md`](profiles.md)

If you're a contributor adding new configurability:

- New schema field → declare it in the relevant `.vrooli/schemas/*.schema.json` first; the manifest is the source of truth.
- New operator choice → it goes in [`operator-state.schema.json`](../../.vrooli/schemas/operator-state.schema.json), not in onboarding state, not in a docs list, not hardcoded in UI.
- New integration → add a page to [`integrations/`](integrations/) describing what the operator needs to provide and how the system consumes it.

## Discipline

Two rules keep this folder honest over time:

1. **Manifests are the data.** Docs describe schemas and meanings; they never hold the canonical list of scenarios, resources, tools, or safeguards. Those lists are derived from filesystem manifests and are drift-protected by tests in `internal/runtime/manifests_test.go` (handlers vs. manifests) and by the schema validators.

2. **Onboarding is feature-complete when every integration here has a wizard step.** This is the explicit acceptance criterion in [`vrooli-onboarding/PRD.md`](../../scenarios/vrooli-onboarding/PRD.md). New configurability lands as a doc page first; the wizard step follows.

## Schema files referenced

- [`.vrooli/schemas/service.schema.json`](../../.vrooli/schemas/service.schema.json) — top-level project and per-scenario `service.json`
- [`.vrooli/schemas/resource.schema.json`](../../.vrooli/schemas/resource.schema.json) — per-resource `resource.json`
- [`.vrooli/schemas/tool.schema.json`](../../.vrooli/schemas/tool.schema.json) — per-host-tool `tool.json`
- [`.vrooli/schemas/safeguard.schema.json`](../../.vrooli/schemas/safeguard.schema.json) — per-safeguard `safeguard.json`
- [`.vrooli/schemas/common.schema.json`](../../.vrooli/schemas/common.schema.json) — shared primitives, including `secretDescriptor`
- [`.vrooli/schemas/operator-state.schema.json`](../../.vrooli/schemas/operator-state.schema.json) — `.vrooli/operator-state.json`
