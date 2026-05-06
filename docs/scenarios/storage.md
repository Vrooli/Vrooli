# Scenario Storage

This page defines the canonical storage policy for scenarios at the platform level.

It exists to make the target architecture explicit rather than leaving it implied across implementation examples, audits, and prompt-manager skills.

## Current Rule

Scenario runtime state must not be stored under the scenario source tree.

In particular:

- do not treat `path:scenarios/<name>/` as a runtime data root
- do not add new mutable runtime state under `path:scenarios/<name>/data/`
- do not rely on repo-local `../data/...` paths for scenario runtime state

Scenario source trees are deployable inputs. Mutable runtime state belongs outside the repo.

## Three File Categories

Scenario files should be classified into three categories:

- repo metadata
  - structural files that define scenario identity or repo-contract-facing layout
  - example: `path:scenarios/<name>/.vrooli/service.json`
- tracked scenario-authored assets
  - files intentionally edited by humans or scenario UIs and committed to git as shared defaults, policies, plans, or authored content
  - examples: `config/`, `policy/`, `initiatives/`, `ideas/`, `research/`, `requirements/`, `docs/`
- runtime mutable state
  - operational data created or changed while the scenario runs and not intended to be shared through git
  - examples: queues, runs, checkpoints, lockfiles, telemetry, local databases, caches

If a file is edited through the UI but the intended result is a shared, reviewable change to scenario behavior, treat it as a tracked scenario-authored asset, not runtime state.

## Canonical Runtime Storage Contract

For mutable filesystem state, scenarios should use:

- `github.com/vrooli/api-core/storage`

This is the canonical runtime storage seam for scenarios.

It provides:

- profile-aware storage roots
- classed storage directories
- safe relative path resolution
- atomic file write helpers
- a cross-platform path policy that keeps mutable state out of source trees

## Storage Classes

Scenarios should use the `package:api-core/storage` classes intentionally:

- `config`
  - durable operator/user-managed configuration
- `data`
  - primary mutable application data
- `cache`
  - rebuildable artifacts safe to evict
- `logs`
  - diagnostics and operational logs
- `state`
  - checkpoints, locks, transient runtime state

At runtime this resolves to class-scoped directories like:

- `<config-root>/vrooli/<scenario>/...`
- `<data-root>/vrooli/<scenario>/...`
- `<cache-root>/vrooli/<scenario>/...`
- `<logs-root>/vrooli/<scenario>/...`
- `<state-root>/vrooli/<scenario>/...`

See [packages/api-core/docs/storage.md](/home/matthalloran8/Vrooli/packages/api-core/docs/storage.md) for the package-level contract.

Tracked source-tree `config/` or `policy/` files are different from `package:api-core/storage` `config` class files:

- source-tree `config/` or `policy/`
  - versioned scenario defaults or policy authored in git
- storage `config` class
  - local operator or user-managed mutable configuration outside the repo

## Tracked Scenario Assets

Not all mutable-looking files are runtime state.

Some files are intentionally edited through tooling or scenario UIs and are meant to be committed to git as shared defaults, policies, or authored content. These are tracked scenario-authored assets.

These belong in explicit scenario directories such as:

- `config/`
- `policy/`
- `initiatives/`
- `ideas/`
- `research/`
- `requirements/`
- `docs/`

Do not place these in `.vrooli/` unless they are true repo or manifest metadata.

Do not place these in `package:api-core/storage` unless they are local runtime state rather than shared source.

## Structured Persistence

Filesystem runtime storage is only one part of scenario storage.

Scenarios should also follow these rules:

- resource-backed persistence is declared in `path:scenarios/<name>/.vrooli/service.json`
- schema and seed assets live under `initialization/`
- scenarios should prefer resource-injected environment variables instead of hard-coded connection details
- scenario-private database/file layout details should be documented in scenario-local docs when they matter

Typical examples:

- PostgreSQL schema init in `initialization/storage/postgres/schema.sql`
- scenario dependency declaration in `.vrooli/service.json`
- SQLite database path resolved through `package:api-core/storage` rather than under the scenario folder

## Skills And Authority

The prompt-manager skills are useful implementation steers, not the canonical policy layer.

Relevant skills include:

- `storage-steer`
- `cross-platform-readiness`

Those skills already steer agents toward:

- declaring storage dependencies in `service.json`
- using environment-driven configuration
- using `package:api-core/storage` for mutable filesystem state
- treating deploy directories as disposable

That guidance is aligned with this document, but this page is the canonical cross-scenario documentation layer.

## Anti-Patterns

These are legacy or non-target patterns for scenarios:

- mutable writes to `./data`, `./state`, or similar scenario-local folders
- mutable writes to `../data/...`
- hard-coded absolute paths such as `$HOME/...` or `/tmp/...` for durable state
- hand-rolled `DATA_DIR` traversal logic when `package:api-core/storage` is available
- storing real runtime state under bundle/deploy/app target directories

## Relationship To The Repo Contract

The repo contract defines canonical source-tree layout such as:

- `path:scenarios/<name>/.vrooli/service.json`
- `path:scenarios/<name>/api`
- `path:scenarios/<name>/ui`
- `path:scenarios/<name>/initialization`

It does **not** define scenario-private runtime data layout.

That separation is intentional:

- repo contract: future-state source layout and shared structural semantics
- this document: runtime storage policy for scenarios

## Transitional Reality

Some scenarios still contain repo-local runtime storage patterns today.

Those should be treated as migration targets, not architecture authority.

When migrating a scenario:

- move mutable filesystem state to `package:api-core/storage`
- keep source-tree assets under `initialization/`, `docs/`, `requirements/`, or other canonical source paths
- add or update `docs/internal/STORAGE_AUDIT.md` when storage behavior is important enough to audit explicitly

## Top-Level `data/`

The top-level repo `data/` folder is legacy/transitional from the perspective of scenario runtime storage.

New scenario work should not depend on it.
