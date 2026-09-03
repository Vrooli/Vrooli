# Build And Validation

This page describes the current project-level build and validation surface.

## Current Truth

At the project level, Vrooli does have real build and validation steps.

- `make build` builds the project-level Go binaries
- `make install` installs them into `~/.vrooli/bin`
- `make test` runs the project-level Go test surface
- `vrooli build` is the root CLI build command

This is distinct from older documentation that described the entire platform as having “no build step.”

## Core Commands

```bash
make build
make install
make test
make capability-conformance
make check-platforms
```

Go compile-affecting flags are governed in `.vrooli/repo-contract.json` under
`build.go_flags`. Development and scenario builds use the same `-trimpath`
policy; distribution builds additionally use `-buildvcs=false`. Platform
cross-compilation is intentionally isolated in `make check-platforms` so the
ordinary `make check` path does not fan out into a host-wide compile storm.

## Control-plane structural-debt gate

CI runs a dedicated `tidiness-control-plane` job. It starts `tidiness-manager`
through the control plane and runs:

```bash
go run ./cmd/vrooli hygiene --tidiness-only --fail-on error --json
```

The live provider measures canonical-seam bypasses and the per-target tidiness
metrics for `control-plane:internal`. Seam budgets live in
`.vrooli/canonical-seams.json`; target budgets live in `.vrooli/testing.json`
and the matching nested testing files. A frozen budget equals both its baseline
and live observation, so it records existing debt without reserving progress.
Budget and ratchet failures exit non-zero. Provider unavailability also exits
non-zero, so the gate cannot pass without a live measurement.

To reproduce the job locally:

```bash
vrooli scenario start tidiness-manager --timeout 120
go run ./cmd/vrooli hygiene --tidiness-only --fail-on error --json
```

The developer scenario suites retain their configured skip behavior when the
optional tidiness provider is stopped; only this explicit control-plane gate
requires provider availability.

## Debt metric ownership

`tidiness-manager` owns repository-specific structural seams and cross-file
maintainability budgets. `golangci-lint` owns language-level smells.

| Metric | Owner |
| --- | --- |
| Canonical call/literal bypass populations | tidiness-manager `.vrooli/canonical-seams.json` tier |
| `duplication_line_debt` | tidiness-manager ratchet |
| `long_files` | tidiness-manager ratchet |
| `complexity_over_threshold` | tidiness-manager ratchet |
| `coupling_over_threshold` | tidiness-manager ratchet |
| `debt_markers` | tidiness-manager ratchet |
| `gocyclo` | golangci-lint |
| `goconst` | golangci-lint |
| `mnd` | golangci-lint |

Duplication, complexity, long files, and coupling are owned by the ratcheted
tidiness budgets. `gocyclo`, `goconst`, and `mnd` remain owned by the
repository's `golangci-lint` configuration.

## Platform declaration conformance

`vrooli capability conformance` is a pass/fail gate for authored platform
claims. It discovers tool, safeguard, and scenario claims, maps each to its Go
module, and cross-compiles each module for the six declared host OS/architecture
cells: linux/amd64, linux/arm64, macos/amd64, macos/arm64, windows/amd64, and
windows/arm64. A failure names the manifest, target OS/architecture, module,
and compiler output. It is deliberately
not a percentage or portability coverage cell: a false declaration is a gate
failure.

Run it locally with:

```bash
vrooli capability conformance --json
```

You can also use the root CLI:

```bash
vrooli build
```

## Production UI conformance

Scenario components with a registered builder output are production-serving
artifacts. The `production-ui-artifact` rule requires each such component to
declare a `run.argv`, and the `no-development-server` rule rejects development
servers and watch modes in component commands and authored `develop` steps.
The forbidden command vocabulary is centralized in the deployability rule
registry. A scenario may build with Vite or another toolchain, but it must
serve the resulting production artifact; `vite dev`, `--watch`, `nodemon`,
`webpack serve`, `next dev`, and `run dev` are not production lifecycle
commands.

The same rules run against the live scenario fleet and against dedicated
negative and positive fixtures in `internal/deployability/testdata/`.

## Scenario readiness and build reuse

`startup_grace_period` is a failure ceiling for derived component readiness. It
does not delay the first probe or a successful health verdict. A component's
explicit `run.readiness` wins; otherwise `run.port` derives a port-open probe,
and a component without a port is ready when its tracked process is alive.
Scenario health checks still run after all components are ready.

Restart reuses fresh build artifacts by default. Use
`vrooli scenario restart <name> --force` when an unconditional rebuild is
required; the production build commands and served artifact are unchanged.

## Supervisory module gate

`make verify-supervisory` builds, vets, tests and cross-compiles (darwin/amd64,
darwin/arm64, windows/amd64, linux/arm64) every module listed in
`tools/supervisory-modules.txt`, runs the vrooli-bridge bootstrap script
tests, and runs the invoker registry tests under `internal/cli/rootcli`. The
listed modules are the ones on the boot-recovery path: the autoheal loop and
its recovery floor, the autoheal API and CLI, the bridge agent, and the shared
packages they replace into.

`make install` depends on `verify-supervisory-build`, the build-and-vet half
of that gate. It refuses to install a control-plane CLI while any listed module
does not build. The 2026-09-02 boot outage was an installed CLI whose own
supervisors (the autoheal API, the loop) no longer compiled or parsed against
it; the gate is the check that would have refused that install.

The CI job `supervisory` in `.github/workflows/test.yml` runs the full gate
before the `test` job. The test
`TestEverySpawningRootReplacerIsASupervisoryModule` fails when a scenario
module that replaces the root module and spawns `vrooli` through the invoker
seam is missing from the list.

## The toolchain floor

Every build in the repository runs at a bounded width. The floor is one
rule applied at three seams, and each seam composes with what it inherits
instead of replacing it:

| Seam | Where | What it sets |
| --- | --- | --- |
| Go processes | `envkit.Toolchain` (`packages/envkit-go/toolchain.go`), applied by `internal/shell.Command` for toolchain programs, by the lifecycle step environment, the CLI installer, the stale checker, the agent launcher, and every scenario site that spawns `go`, `pnpm`, `npm`, `cargo`, `uv`, `vite` or `tsc` | appends `-p=<width>` to an inherited `GOFLAGS` only when no `-p` token is present; sets `GOMAXPROCS` (2×width) and pnpm's `npm_config_child_concurrency` / `npm_config_workspace_concurrency` only when absent |
| Makefiles | `mk/toolchain.mk`, included by the root Makefile, every `scenarios/*/Makefile` (through `mk/scenario.mk`) and the template Makefiles | the same variables, exported, with the same composition rule |
| CI | `env:` at the top of `.github/workflows/test.yml` and the react-vite template workflow | `GOFLAGS=-p=4`, `GOMAXPROCS=8` |

The width is the `BuildWidth` tuning lever: `min(4, max(1, NumCPU/4))`,
overridden by `VROOLI_TUNING_BUILD_WIDTH` (see the generated lever table in
`environment-management.md`). Shared packages that cannot import
`internal/tuning` read the same variable from the environment they compose,
with the same default from `envkit.DefaultBuildWidth`.

Two rules keep the floor structural:

- `.ast-grep/rules/no-raw-toolchain-spawn.yml` fails `make lint-portability`
  on any `exec.Command` of a toolchain program whose environment is not
  assigned from `envkit.Toolchain`. It admits no allowlist entries; migrate
  the site.
- Assigning `GOFLAGS` outright (`GOFLAGS=` or `GOFLAGS=-mod=mod`) is a rule
  violation. A site that needs an extra flag passes it through
  `envkit.ToolchainOptions{GoFlags: ...}`, which appends it after the
  inherited tokens.

`mk/scenario.mk` also supplies the shared `build`, `fmt-go`, `fmt-ui`,
`lint-go` and `lint-ui` recipes. A scenario that needs its own body for one
of them lists it in `SCENARIO_CUSTOM_TARGETS` before the include.
`go run ./tools/makefile-include-census` reports drift, and
`TestScenarioMakefilesIncludeToolchainMk` runs it in CI.

The root Makefile refuses `./...` as a build goal at the repository root
(`require-no-root-wildcard` in `mk/toolchain.mk`): one compile per package
across every module at once is what drove the 2026-09-02 host storm. Build a
module list (`make type`, `make verify-supervisory`) instead.

## What `make build` Does

The project Makefile builds:

- `vrooli`
- `vrooli-api`

These are the project-level Go entrypoints under `cmd/`.

## What `make test` Does

The project-level test target is the canonical project test entrypoint.

It covers project-level Go test surfaces. Validation that has its own dedicated policy target should still be run explicitly when your change touches that area.

For scenario-level testing, use:

```bash
vrooli scenario test <name>
```

Or the preferred scenario-local flow:

```bash
cd scenarios/<scenario-name>
make test
```

For domain-specific health maturity findings, run the provider CLI in human mode first:

```bash
proto-health validate scenario <name>
measures-health validate scenario <name>
security-health validate scenario <name>
cli-health validate scenario <name>
ui-health validate scenario <name>
```

Those reports are provider-owned maturity views. Use `--json` only when a programmatic consumer needs the shared `common.v1.MaturityAssessment` structure. See [health-maturity-assessments.md](health-maturity-assessments.md).

## What `vrooli build` Means

`vrooli build` is the root CLI build lifecycle command. Treat the CLI help and current lifecycle definitions as the final authority for its exact behavior.

Use:

```bash
vrooli build --help
```

## CLI Freshness Versus Runtime Freshness

Installed scenario CLI freshness and scenario runtime freshness are intentionally separate.

- Installed scenario CLIs are owned by `path:internal/cliinstall`.
- `vrooli scenario ...` command entrypoints ensure the scenario CLI is installed and current before use.
- Scenario runtime freshness is derived from `components.*.build`; builder inputs, outputs, and digest keys are the sole setup authority for runtime artifacts.

Installed CLI freshness is therefore not a component runtime input during lifecycle start/restart decisions. That boundary prevents dependency restart loops where lifecycle marks a dependency stale because its installed CLI changed, while the component builder only owns runtime artifacts.

## Build Versus Deployment

Do not conflate project builds with deployment portability.

- project builds produce and validate project-level binaries
- deployment portability is governed by the Deployment Hub and target-tier maturity

See [../deployment/README.md](../deployment/README.md).
# Declarative canonical seams

The repository rule file `.vrooli/canonical-seams.json` declares canonical Go implementation seams and the call or literal syntax that bypasses them. Tidiness Manager evaluates these rules over the Go AST and emits `BYPASSED_SEAM` findings only for in-scope matches above each declared budget. Define exclusions for intentional populations; do not raise a budget to hide a mismatch.

The schema is `scenarios/tidiness-manager/schemas/canonical-seams.schema.json`. See `scenarios/tidiness-manager/docs/reference/canonical-seams.md` for the field contract and validation command.
