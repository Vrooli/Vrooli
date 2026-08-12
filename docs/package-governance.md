# Package Governance

Structure Health is the `claim:package.manifest` sole authority for package structural validation. Its
generated catalog and coverage matrix define package manifests, module roots,
private-import boundaries, and local Go replace requirements:

- [Structure Health rule catalog](../scenarios/structure-health/docs/reference/structure-rules.md)
- [Structure Health coverage matrix](../scenarios/structure-health/docs/reference/structure-rule-coverage.md)

This document owns only package registry and lifecycle operations. The control
plane keeps these read/build/generate/test/refresh verbs because they operate on
the package registry; it does not emit a competing structural verdict.

## Lifecycle

```bash
vrooli package list
vrooli package info <name>
vrooli package dependents <name>
vrooli package build <name>
vrooli package generate <name>
vrooli package test <name>
vrooli package refresh <name> all
```

Non-interactive refreshes do not restart running consumers by default. They
reconcile the package outputs and leave lifecycle ownership with the operator;
use the explicit interactive restart option when a restart is deliberately
required.

Dependency changes and local Go-module reconciliation go through Scenario
Dependency Analyzer:

```bash
scenario-dependency-analyzer deps reconcile --all
scenario-dependency-analyzer deps reconcile --scenario <name> --apply
```

Do not hand-edit the approved dependency registry or run a raw package manager.
Scenario UIs remain isolated projects and shared package adoption remains
explicit through their package manifests and local dependency declarations.

## Shared TypeScript packages

Scenario setup provisions every governed `file:` dependency under `packages/`
before `install-ui-deps`. Each TypeScript package's build lifecycle first
installs its frozen lockfile outside the repository root workspace, then
produces compiled JavaScript and declaration
files from their declared output directory; consumers must not alias an
`@vrooli/*` package into `packages/*/src`. The clean-environment proof lives
in the `Clean shared-package provisioning` job in `.github/workflows/test.yml`.

The build-output digest is part of UI freshness, so rebuilding a shared package
invalidates the consumer's installed copy on its next setup. Generated outputs
remain lifecycle-owned artifacts and are not hand-built by scenario operators.

Provisioning is single-flight at two levels: lifecycle phase entrypoints hold
the scenario lock, and each governed package is protected by a shared-package
lock under `$HOME/.vrooli/state/locks`. This is required because different
scenario consumers can provision the same `file:` dependency concurrently.
Lifecycle-owned package commands receive the setup environment and EOF stdin,
so package-manager prompts fail deterministically and cannot wait on an
operator terminal. Lock waits and command durations are recorded in the
scenario lifecycle log with the package root, process ID, and wait duration.
