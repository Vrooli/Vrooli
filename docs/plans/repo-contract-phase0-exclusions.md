# Repo Contract Phase 0 Exclusions

This document is the explicit Phase 0 exclusion list for the repository contract.

These items may still exist in the repository or in active code paths, but they are not part of the future-state language-agnostic repo contract and new consumers must not depend on them.

## Explicit Path Exclusions

The following paths are excluded from the canonical contract:

- `cli/`
- `path:cli/commands/`
- `path:cli/lib/`
- `path:scripts/lib/`
- `path:scripts/manage.sh`
- project-level shell helper locations
- project-level shell dispatch assumptions

## Excluded Detection Rules

The following root-detection rules are excluded from the canonical contract:

- repository detection via `.git`
- repository detection via `pnpm-workspace.yaml`
- arbitrary ancestor walking with shallow fixed-depth limits
- `$HOME/Vrooli` as a canonical repository-root rule
- `APP_ROOT` as a canonical repo-root env var

These may appear as temporary compatibility behavior during migration, but they are not future-state contract semantics.

## Excluded Scenario Assumptions

The following scenario assumptions are excluded from the shared contract:

- scenario-private folders not used across the platform
- `coverage/` as a universal scenario path
- queue, research, workshop, execute, logs, prompt, profile, backlog, or other scenario-owned folder trees
- any scenario file or directory used by only one scenario family

The following path is explicitly deferred rather than accepted into the v1 contract:

- `.vrooli/metadata.json`

Reason:

- it is used by some scenarios, but it is not yet a stable repo-wide invariant

## Excluded Resource Assumptions

The following resource assumptions are excluded from the canonical contract:

- shell-first resource implementation details such as `cli.sh`, `config/defaults.sh`, `config/runtime.json`, `lib/*.sh`, and shell-only test layouts
- `.vrooli/resource.json` as a resource manifest location
- resource registry metadata under `.vrooli/resource-registry/` as a canonical source of truth

Reason:

- the future-state resource contract centers on `path:resources/<name>/resource.json` and Go-native control-plane behavior, not shell-era implementation details

## Excluded Bundle/Profile Inputs

The following are excluded from the contract as direct canonical rules:

- top-level `api/`, `src/`, `scripts/`, `platforms/`, and `assets/` being universally required root directories
- current hard-coded `scenario-to-cloud` include/exclude lists as normative contract behavior

Reason:

- they are current implementation inputs, not yet validated as stable future-state repository contract material

## Excluded Behavioral Logic

The repo contract must not encode behavior that belongs in code rather than in data. Excluded categories:

- storage policy
- health-check execution logic
- lifecycle execution logic
- runtime supervision behavior
- HTTP/API behavior
- CLI UX behavior
- fallback ordering that exists only for temporary migration compatibility

## Adoption Rules From This Exclusion Set

Phase 0 locks the following rules for follow-on work:

- no new repo-aware code should add fresh `$HOME/Vrooli` fallbacks
- no new repo-aware code should add fresh `.git` or `pnpm-workspace.yaml` root detection
- no new repo-aware code should invent new canonical scenario path composition logic outside the eventual shared adapter/helper layer
- no new repo-aware code should define its own glob semantics for root-relative repo matching

