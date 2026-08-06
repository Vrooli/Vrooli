# Resource Maturity and Migration Playbook

Use this document to assess a resource, rank the resource fleet by migration
need, and produce an implementation-ready plan to move one resource toward the
target architecture in [architecture.md](architecture.md).

The target is not a cosmetic Go wrapper around old scripts. A maximally mature
resource is manifest-authoritative, Go-native in every normal operator path,
template-conformant, and uses the cleanest performant and portable archetype
that is feasible for its actual runtime.

## Scope and Non-Goals

This playbook applies to `resources/<name>/` and the shared resource control
plane.

It does not require every resource to run on every operating system. It does
require an honest portability contract: supported platforms use the same
Go-native operator surface; unsupported platforms are declared and rejected
before an operator reaches a Linux-oriented implementation detail.

Bash is not an acceptable dependency of a mature resource. In particular, a
mature resource does not require Bash for lifecycle, configuration,
diagnostics, tests, or normal installation. A temporarily retained shell shim
is migration debt, not a maturity level.

## Maturity Levels

Assign the lowest level whose conditions describe the resource today. Record
evidence for every level decision; do not infer maturity from the presence of a
`cli/main.go` file alone.

| Level | Name | Observable condition |
|---|---|---|
| M0 | Uncontracted | No valid manifest, no reliable operator surface, or ownership is unclear. |
| M1 | Legacy wrapper | Shell scripts own lifecycle/configuration/tests, even if a manifest or CLI has been added beside them. |
| M2 | Bridged | A valid manifest and Go CLI/control-plane delegation exist, but a normal operator path still depends on Bash or shell-era shared libraries. |
| M3 | Native lifecycle | Lifecycle, logs, status, and diagnostics are Go-native; configuration, tests, or a consumer path still has bounded migration debt. |
| M4 | Template-conformant | The resource follows its selected template, has no normal Bash dependency, and has tested platform gates; remaining limitations are documented runtime constraints. |
| M5 | Maximally mature | M4 plus a clean archetype decision, typed configuration migration, complete capability/test correspondence, consumer validation, and no known avoidable portability or performance debt. |

M5 means *maximal feasible maturity*, not a claim that Docker, an upstream
binary, or a cloud provider is portable by itself. For example, a resource that
genuinely requires Docker can reach M5 only if that host requirement is
explicit, well-tested, and preferable to the feasible alternatives.

For M4/M5, "portable" also includes the resource's claimed deployment targets:
the normal desktop/release path must consume a verified platform artifact or a
declared external runtime, rather than require Go or Bash on the end-user
machine. See [deployment-contract.md](deployment-contract.md) for the target
deployment contract. A resource may be M5 while honestly unsupported for a
given target; M5 means maximal feasible, evidenced support, not universal OS
support.

## Runtime Honesty Requirements

M4/M5 additionally require that the platform's picture of the resource matches
reality. Most hard-to-diagnose reliability failures come from a resource that
reports better state than it has, not from lifecycle mechanics:

- **Pinned runtime.** Every pulled artifact — binary or container image — is
  an immutable reference. See the Pinned Runtime Principle in
  [deployment-contract.md](deployment-contract.md). An engine that can change
  on the next pull is an undeclared regression source.
- **Readiness health semantics.** The manifest health check is a readiness
  probe: it must fail until the resource can serve its primary capability,
  including model/data load. A liveness-only probe (process up, model absent)
  breaks orchestration ordering and defeats consumer auto-recovery, which
  retries against a "healthy" resource that cannot serve. Supplementary checks
  may declare `kind: liveness` explicitly.
- **Declared degradation.** A resource with more than one operating mode
  (GPU/CPU, model sizes, engine fallbacks) must expose the active mode on an
  info/status surface and must surface running below its configured mode as a
  visible status. Degraded is a state, never a secret; silent mode switches
  have historically produced weeks-long invisible failures.
- **Timeout honesty.** `startup_timeout_seconds` budgets the worst normal
  case, including first-run downloads. The human-readable estimate can
  distinguish warm from first-run; the enforced timeout must not kill a
  genuinely progressing first start.
- **Capacity visibility.** A resource that declares a `gpu` block also
  declares a `capacity` block (or records why the broker must not manage it),
  so co-tenant VRAM planning sees every claimant.

## Assessment Dimensions

Score each dimension from 0 to 2. The score supports fleet ordering; the
maturity level above remains the primary classification.

| Dimension | 0 | 1 | 2 |
|---|---|---|---|
| Contract | Missing/invalid or shadowed by scripts | Valid but incomplete or contradicted by implementation | Manifest is the complete declared authority |
| Archetype | Accidental or undocumented runtime shape | Template selected, but not fully followed | Best-fit template/archetype is documented and followed |
| Operator surface | Shell/manual operations are canonical | Go CLI exists but delegates to shell or lacks required operations | Go CLI/control plane owns all normal operations |
| Configuration | Ad-hoc files, substitutions, or shell mutation | Bootstrap exists but is not typed/migration-safe | Typed init/validate/migrate path preserves operator state |
| Runtime and health | Bespoke scripts own runtime/health | Shared driver exists with retained shell exceptions | Shared driver/native runtime owns lifecycle, logs, health, and diagnostics |
| Tests | Shell-only, flaky, or no runtime coverage | Some Go tests or integration coverage | Hermetic Go tests + runtime integration + consumer smoke where applicable |
| Portability | Implicit Linux assumptions | Partial support or incomplete gates | Explicit support matrix; OS-specific code isolated; unsupported systems fail clearly |
| Legacy debt | Normal paths source shared/legacy shell | Isolated, documented migration shim | No normal-path legacy shell dependency |
| Deployment readiness | Target cannot be evaluated | Partial profile or source-build-only delivery | Claimed targets declare delivery, requirements, fallbacks, and evidence |

Use the total (0--18) only to prioritize investigation. A low score with no
consumers can be a cheap cleanup; a low score with many consumers may require a
careful compatibility migration first.

## Evidence Collection

An assessor should collect repository evidence before making recommendations:

1. Read `resources/<name>/resource.json`, its template kind, CLI contract,
   runtime/health/storage declarations, environment exports, and platform
   metadata.
2. Map actual entrypoints and imports: `cli/`, root scripts, `lib/`, `config/`,
   `api/internal/<domain>/`, tests, and any `source`/shared-script references.
3. Find callers outside the resource, including scenario dependencies,
   environment-variable consumers, resource CLI invocations, and direct HTTP
   clients.
4. Trace normal operator flows: fresh setup, configuration bootstrap, start,
   status, logs, diagnostics, upgrade/reset if promised, and stop/uninstall.
5. Inspect test commands and determine whether they exercise the declared
   runtime and at least one real consumer when consumers exist.
6. Compare the observed behavior against the selected template and the shared
   Go control-plane capabilities.
7. For every claimed deployment target, map artifact delivery, OS/architecture,
   host requirements, fallback/degradation behavior, and validation evidence.

Do not count generated binaries, old backups, documentation-only references,
or uncalled helper functions as proof of an active dependency. Label evidence
as runtime, test, operator, consumer, documentation, or historical.

## Archetype Selection

Choose the lightest archetype that accurately owns the resource's real
runtime. Do not start from the existing directory layout or an old Docker
compose file.

```text
Hosted capability owned elsewhere?             → cloud-api
Existing host executable is the runtime?       → external-cli
Vrooli owns an executable/operator binary?     → native-cli
Vrooli owns a local supervised service?        → managed-service (target archetype)
One container is the best supported runtime?   → docker-service
Multiple coordinated containers are required?  → compose-service
Lifecycle cannot yet be owned safely?          → manual-resource
```

`docker-service` and `compose-service` are deliberate host-runtime choices,
not defaults. Before choosing either, record why a cloud API, external CLI,
native CLI, or managed local service is less suitable. This is especially
important for desktop bundles, where a Docker daemon may be too costly or not
available.

The template catalog provides `managed-service` for this case. Start from that
scaffold, declare the signed server artifact and provider policy, and add a
bundled-service profile only for targets that have an explicit readiness
contract. Do not force a resource into Docker merely to fit an older scaffold.

## Required Output From an Assessment Agent

An assessment must produce a concise report with these sections:

1. **Current maturity** — M0--M5, dimension scores, and evidence links.
2. **Active contract and consumers** — distinguish runtime callers from tests,
   docs, and historical artifacts.
3. **Target archetype** — selected template, alternatives considered, and the
   portability/deployment rationale.
4. **Deployment profile** — target-by-target support, artifact/external-runtime
   delivery, requirements, limitations, fallbacks, and current readiness.
5. **Gap list** — exact normal paths that still rely on Bash, shadow the
   manifest, lack validation, or have weak platform behavior.
6. **Migration plan** — sequenced, reversible phases with deletion gates.
7. **Validation matrix** — unit, integration, consumer, and platform-gate
   checks required to call the work complete.
8. **Risks and decisions** — operator-state preservation, secrets, data,
   backwards compatibility, and any decision requiring owner approval.

For fleet ranking, include a table of every assessed resource with maturity,
score, active consumer count, migration size, and recommended next action.
Prioritize the lowest-maturity resources with a clear target archetype and
bounded consumer risk; do not use score alone to schedule destructive work.

## Migration Plan Shape

Every resource migration plan must preserve behavior before deleting legacy
code:

1. **Set the target contract.** Update or validate the manifest, template
   choice, supported platforms, declared capabilities, storage, and health
   contract.
2. **Add the native replacement.** Implement any resource-specific Go
   configuration, diagnostics, or runtime behavior behind the standard CLI and
   control plane.
3. **Protect existing operators.** Import or preserve existing configuration
   and data. Never silently overwrite secrets, generated config, or runtime
   state.
4. **Move callers and tests.** Switch consumer, operator, and test paths to
   the native surface. Keep a temporary shim only when a caller cannot migrate
   in the same change.
5. **Prove equivalence and support.** Run hermetic tests, runtime integration,
   consumer smoke tests, and platform gates appropriate to the target.
6. **Package the deployment artifact.** Build/sign the platform artifacts in
   CI and validate that the selected deployment target does not invoke Go or
   Bash. Keep `cli-installer` as a source/developer path only.
7. **Delete the old path.** Remove scripts, shared-shell imports, stale
   documentation, and manifest claims that no longer correspond to a command.

No migration is complete while the old shell path remains required by a normal
operation, even if the new Go command exists.

## Completion Gate

Mark a resource M4 or M5 only when all applicable statements are true:

- a fresh installation and configuration work through the native surface
- existing configuration/data are preserved or explicitly migrated
- lifecycle, logs, status, health, and diagnostics are Go-native
- runtime references are pinned and health checks have readiness semantics
  (see Runtime Honesty Requirements)
- all claimed resource capabilities have a command/API and focused tests
- capability documentation (endpoints, commands) describes only surfaces that
  exist and are exercised by the resource's tests; stale contract fiction
  fails the gate
- no normal path requires Bash or sources a legacy shell library
- runtime integration tests validate the declared driver/runtime
- at least one consuming scenario smoke test passes when consumers exist
- platform support is exercised or explicitly gated before runtime execution
- each claimed deployment target has a validated readiness result; conditional,
  degraded, and unsupported paths are also tested for clear operator messaging
- the resource documentation states its archetype, host requirements, and
  remaining non-portable constraints honestly

## Suggested Agent Prompt

> Assess `resources/<name>` using `docs/resources/maturity-migration.md` and
> `docs/resources/deployment-contract.md`.
> Do not change files. Establish its M0--M5 maturity and dimension scores from
> repository evidence; map active callers; choose the cleanest feasible target
> archetype; resolve current and target deployment profiles; and return an
> actionable migration plan with phases, deletion gates, validation matrix,
> risks, and owner decisions. Distinguish active runtime behavior from tests,
> docs, backups, and historical artifacts.

For a fleet audit, replace `resources/<name>` with `resources/` and require a
ranked table plus recommended migration candidates.

## Fleet Completion Ledger — 2026-08-05

The following resource READMEs record the current M4 evidence level. The
fleet contract test is the shared gate for shell absence, health semantics,
immutable images, and capability/test correspondence; resource-specific
README notes remain the source for runtime constraints.

| Resource | Level | Evidence |
|---|---:|---|
| adguard-home | M4 | `resource.json`, `cli/main_test.go` |
| antigravity | M4 | `resource.json`, `cli/main_test.go` |
| claude-code | M4 | `resource.json`, `cli/main_test.go` |
| codex | M4 | `resource.json`, `cli/main_test.go` |
| gemini | M4 | `resource.json`, `cli/main_test.go` |
| grok | M4 | `resource.json`, `cli/main_test.go` |
| home-assistant | M4 | `resource.json`, `cli/main_test.go` |
| k6 | M4 | `resource.json`, `cli/main_test.go` |
| kokoro | M4 | `resource.json`, `cli/main_test.go` |
| kopia | M4 | `resource.json`, `cli/main_test.go` |
| kyutai-stt | M4 | `resource.json`, `cli/main_test.go` |
| minio | M4 | `resource.json`, managed artifact checks, `cli/main_test.go` |
| ollama | M4 | `resource.json`, `cli/main_test.go` |
| opencode | M4 | `resource.json`, `cli/main_test.go` |
| openrouter | M4 | `resource.json`, `cli/main_test.go` |
| postgres | M4 | `resource.json`, query readiness, `cli/main_test.go` |
| qdrant | M4 | `resource.json`, managed artifact checks, `cli/main_test.go` |
| redis | M4 | `resource.json`, `cli/main_test.go` |
| reranker | M4 | `resource.json`, `cli/main_test.go` |
| searxng | M4 | `resource.json`, `cli/main_test.go` |
| speaker-verification | M4 | `resource.json`, `cli/main_test.go` |
| twilio | M4 | `resource.json`, `cli/main_test.go` |
| unstructured-io | M4 | `resource.json`, `cli/main_test.go` |
| vault | M4 | `resource.json`, managed artifact checks, `cli/main_test.go` |
| whisper | M4 | `resource.json`, `cli/main_test.go` |
