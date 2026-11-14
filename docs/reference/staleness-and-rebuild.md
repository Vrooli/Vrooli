# Scenario freshness and rebuilds

Vrooli uses the manifest freshness engine in `packages/cli-core/cliutil` for
scenario artifacts. The engine evaluates declared inputs by file content and
the stat cache; it does not use source-versus-artifact mtime as a freshness
verdict and it does not expose a stale-check bypass flag.

Freshness sidecars (`<artifact>.freshness.json`) and the atomic writer's numeric
`.freshness.json.tmp-*` files are generated metadata. Input enumeration and
legacy manifest evaluation exclude them, so concurrent component publication
cannot hash another writer's transient file. Similarly named authored source
files remain inputs.

The retired `--no-stale-check` global is tolerated until 2026-12-01: the root
parser consumes it, prints one warning to stderr, and dispatches the command
unchanged. The tolerance table lives in `internal/cli/rootcli/rootcli.go`
(`retiredGlobals`). Processes that spawn `vrooli` must not rely on it; the
invoker registry test in `internal/cli/rootcli/invokers` fails on any
registered argv that still carries a retired global.

## Freshness contract

The installed root CLI and emergency watchdog use the root build stamp owned by
`internal/buildinfo`. Their conservative portable input roots include their
command tree, `internal`, shared `packages`, and root module files. The root CLI
also includes its compiled `cli` package and embedded `cli/manifest.json` command
contract. Changing that contract invalidates the same canonical stamp used by
installation and distribution builds. The stamp
includes Go and native compiler sources, headers and assembly, plus the actual
assets selected by `//go:embed` comments in those sources. Embedded directory
membership participates, so adding or removing an asset also changes freshness.
Quoted and `all:` patterns are supported. An unresolved embedded pattern or
explicitly selected symlink is an input error; compiler-excluded children of a
recursive directory match are ignored. Unembedded prose
does not participate. The watchdog consumes the same canonical input-root
selector as the root build tools rather than maintaining its own shared list.
The parsed compile-flag policy from the canonical repository contract also
participates. Changing `build.go_flags` invalidates the stamp even when source
files and the runtime-home layout stay unchanged. Unsupported policy flags are
an unresolved-input error rather than an implicit default build.

Proto owns the classification of its default generation and verification work
trees, compatibility publication staging, and `packages/proto/gen.previous`
recovery backup. Root stamps omit those uncompiled copies while tracking the
active `packages/proto/gen` projection. Authored hidden directories elsewhere
and explicitly selected embedded assets remain inputs. Concurrent verification
and its cleanup therefore cannot change the root source identity.

These root stamps identify source content. Scenario components continue to use
their precise import closure and keyed compiler environment described below;
root source stamps alone do not certify deployment, toolchain equivalence or
adoption by a long-lived running process.

A fresh root build is not necessarily installable. While root updates are
enrolled, `make install` and self-rebuild hand the candidate to the independent
coordinator (`vrooli runtime update apply`), which activates it with automatic
rollback only when its `persistent_state_contract_sha256` (see
`vrooli version --json`) equals the serving one. For the frequently edited
persistence owners that contract covers what they store (compiled envelope
shapes, enum constants, custom codecs and declared format facts), not their
function bodies, so a behaviour-only change activates normally; the
activation machinery and native identity owners are hashed whole. A real
stored-format change is refused, and `make install` prints each change by
owner and stored type, field or fact, followed by the remedy. The installed
CLI stays on its prior build; that is a rollback-safety refusal, not a build
failure. The companion binaries (`vrooli-api`, `vrooli-agent-launcher`,
`vrooli-policy-runner`) install only after the root activates, so they are
left unchanged too. When no receipt comes back and the images do not differ
in contract, the outcome is reported as unresolved, with the coordinator's
diagnostics, never as a refusal. Crossing a stored-format change is the
operator's decision, `make rebaseline`, described in
[the deployment contract](../resources/deployment-contract.md#root-control-plane-activation-target);
an agent must not run it to push its own build, and it refuses to run inside
an agent session. Verify root CLI changes with
their focused tests until the build activates.

An artifact is stale when it is missing, has no manifest, or its manifest no
longer matches the declared inputs. A missing manifest is therefore a
one-time build condition. Successful builds stamp only the component that was
built, allowing API, UI, and other components to rebuild independently.

Input enumeration uses `git ls-files --cached --others --exclude-standard`
inside a work tree and a `WalkDir` fallback outside one. Git supplies the file
set only; the verdict remains content-based, so committing a change cannot by
itself make an artifact fresh or stale.

The same rule applies to generated package outputs. Source schemas, manifests,
and generator inputs are authoritative; generated code, selectors, registries,
descriptors, and bundles are derived outputs. A consumer must either enter its
owner's lifecycle (which regenerates declared outputs) or run the owner's
read-only verifier before a focused test. A direct compiler invocation is not a
freshness check. This prevents a stale output from being misdiagnosed as a
consumer defect and keeps the repair at the owner boundary instead of adding
per-consumer rebuild hooks.

Every package that declares `generated_outputs` must also declare a
`lifecycle.generate` or `lifecycle.build` owner with non-empty output paths.
The package registry reports this as an error, and lifecycle preflight rejects
it before any consumer is compiled. Generated-output metadata without an owner
is never treated as a source-only package.
Repository preflight includes both dedicated generators and build-owned
projections; ordinary build-only packages remain scenario-scoped. Canonical
special handling is selected by the command's declared `artifact_selection`,
not by a package name, so fixtures and future owners cannot accidentally
inherit Proto-only behavior.
When generated owners depend on one another, `lifecycle.requires` is resolved
before repository-level preflight so owner order is deterministic rather than
an accident of directory enumeration.

Go-native scenario and resource CLIs always include `packages/cli-core` in
their freshness identity, even when a manifest declares custom input globs.
`cli-core` supplies the shared runner, stale-check and control-plane behavior;
omitting it lets an installed CLI continue serving an older implementation
after a shared fix. The canonical contract is enforced by
`CanonicalScenarioGoModuleFreshnessSpec` and
`CanonicalResourceGoModuleFreshnessSpec`; after changing `cli-core`, run the
owning lifecycle/setup so the affected installed CLI is rebuilt.

The same specs add the local `replace` targets the CLI imports
(`WithGoReplaceFreshnessInputs`), so no hand-maintained input list is needed.
The import closure is followed package by package through the main module's
replace table. Each imported package directory becomes an input, plus the
replaced module's `go.mod` and the main `go.mod`. A repo-root replace
(`=> ../../..`) contributes only the packages the CLI imports, never the repo
root. A module that already holds a declared input is left to it: the proto
artifact stamp stands for the generated proto tree, and a declared
`../../packages/cli-core` covers all of cli-core. The installer, the installed
manifest check and the runtime StaleChecker all derive the same list, so a
caller-supplied input list cannot cause a rebuild loop. Explicit
`resource.json` `cli.freshness.inputs` still work and are kept first.

## Lifecycle behavior

Builder specifications in `internal/lifecycle/components.go` declare inputs,
skip suffixes, digest keys, closure resolution, outputs, and commands. The
generic lifecycle evaluator uses those fields for Go, Node, and Python/uv
components. Fresh components are skipped unless setup is explicitly forced;
independent stale components may build concurrently within the memory budget.

Proto generation is invoked through the direct `protogen` command and receives
the changed schema scope. `make` is not required on the scenario start path.
Every lifecycle `setup` phase first preflights repository-level generated
owners through their declared generation contract, before scenario-specific
shared-package selection and component builds. The package-test path and
`vrooli scenario test` entrypoint use the same boundary before delegating to
Test Genie. This applies to protobufs and future governed selectors, registries,
descriptors, and runtime bundles; no consumer owns a private regeneration hook.
Scenario setup continues to use lifecycle-owned artifact selection for
scenario-scoped outputs and does not run an ad hoc generator.

## Diagnostics

Use lifecycle logs and component freshness verdicts to identify the content
cause. The rebuild-ledger evidence folder contains the retained baseline,
focused regression proofs, and validation commands:
`docs/architecture/evidence/rebuild-ledger/README.md`.

### Pre-deploy report

The report is read-only and identifies both the candidate working tree and the
source selected for the instance. Without an active source split, those sources
are the same. During an engagement, the live instance uses the frozen serving
generation while a shadow instance uses the working tree. For an engaged live
instance, the report compares the frozen generation's recorded build manifest
(`<artifact>.freshness.json`) with current candidate inputs. Its candidate
delta does not mean a restart will load those candidate files: the live
instance remains on its frozen generation until the engagement is resolved
through the normal gated baseline transition and a later lifecycle cycle. The
report prints the candidate and serving paths plus available engagement and
capture provenance; JSON exposes the same fields. Before restarting a shared
service, run `vrooli scenario deploy-report <name> [--json]`, or
`vrooli scenario restart <name> --deploy-report` to print the report to stderr
first (the restart itself is unchanged). For each build artifact it lists
added, modified and removed files grouped by directory, with directories
outside the scenario marked `[shared]`. It also names build-key changes and
whether a process runs a binary that was since replaced on disk (Linux
`/proc`). For Go artifacts it lists environment switches the current sources
add: names passed literally to `Getenv`/`LookupEnv` plus every `VROOLI_*` and
`*_ENFORCE*` literal. Builds record that set in the manifest (`env_switches`);
a build stamped before the recording reports "unknown for the current build;
known after the next build" and lists switch names found in changed files
instead.

For the next investigation, `vrooli scenario timings --since <date>` reports
the setup and build timing window without reintroducing a second freshness
implementation.


### Coherent replacement sources

A replacement build must use one app source generation and its captured shared
source/generated-contract closure. An engaged serving baseline with no frozen
shared-source record cannot establish that closure after the repository changes.
Keep its running artifacts available. Refuse a requested rebuild before package
provisioning or component mutation, and name the engagement and supported
promotion/cycle remedy. Do not rebuild an old app against today's generated
contracts or edit its serving snapshot to add compatibility methods. Resolve the
engagement through the normal gated baseline transition, then verify the new
serving build and a representative native capture.

A new capture with explicit `--allow-unfrozen` records a degraded operator opt-in;
this remains distinct from a legacy capture with no source-closure record. It
must never be reported as frozen qualification. Self-contained legacy component
builds with an empty resolved shared-source closure remain supported.

Frozen shared-source captures preserve each package owner's declared generate/build
output prefixes even when Git ignores those paths. These prefixes travel with the
captured root and survive parent-directory ignores. Copy exclusions for installed
dependencies/caches and the floor's private runtime-state rules still apply.
Ignoring `dist/` for version control must not remove a consumer's runtime exports
from an immutable capture. Use a new capture generation after changing this
policy; do not patch a materialized serving tree or mutate an older snapshot.
