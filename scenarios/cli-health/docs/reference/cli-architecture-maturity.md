# CLI Architecture Maturity

This document is the design SSOT for the **command architecture maturity**
assessment that CLI Health reports. It defines the maturity ladder, the
finding-code inventory, the exception taxonomy, and the advisory-vs-gating
rollout policy. Later implementation phases (cli-core primitives, the cli-health
classifier, manifest metadata, provider wiring) implement against this contract
and must not invent policy that is not written here.

The problem it solves: a scenario CLI can pass every existing CLI Health check —
manifest present, bindings resolve, help surface aligned — while still running
**separate execution semantics for human and machine output**. The canonical
example is `test-genie execute --json`, whose JSON mode used a blocking legacy
path while its human mode used the durable run path, so the two output formats
had different lifecycle and ownership. Existing checks (`manifest_contract`,
`proto_bindings`, `runtime_surface`, `entrypoint_structure`) cannot see this
because the divergence lives inside a handler's `if ctx.JSON()` branch, not in
the command's declared shape.

The fix is **not** heuristic branch inspection or live behavioral probing (both
are explicitly prohibited — see the plan Constraints). Instead we make the
architecture **machine-recognizable from declared structure**: manifest
metadata, cli-core primitive registration, proto binding coverage, and explicit
exception declarations. A command reaches the top rung by *declaring* how it is
built, and CLI Health verifies the declaration against structural evidence.

Runtime command parity is a separate contract and does use the CLI's bounded
help surface when execution is requested. `cli.command_missing` means the
manifest promises a command that a successfully probed scenario binary does not
expose; it is an ERROR on `runtime_surface` with remediation to rebuild first,
then repair registration or remove the stale declaration. The reverse mismatch,
a runtime command absent from the manifest, remains
`cli.command_undeclared`. Keeping the directions distinct prevents a stale
binary from being misreported as a missing capability and gives fleet repair a
deterministic owner.

## The maturity ladder

CLI Health reports one capability, `command_architecture`, with five rungs. The
rungs are monotone: each implies the one below it.

| Rung | Id | Status label | What it means (machine-recognizable evidence) |
|---|---|---|---|
| L0 | `unclassifiable` | Unavailable | No inspectable cli-core shell or manifest to classify against. The CLI may not exist, may not be runnable here, or owns dispatch in a way cli-health cannot read. Degrade, never hard-fail. |
| L1 | cli-core shell | Foundation | The CLI is built on the cli-core `App`/`ScenarioApp` shell (`cliapp.NewApp` / `ScenarioApp`), so global flags, dispatch, help, and output contracts are framework-owned rather than hand-rolled. |
| L2 | Declarative commands | Ready | Commands parse through the declarative `RunCtx` + `ArgSchema` path (parser and helpgen are schema-driven) rather than legacy `Run(func([]string) error)` handlers with bespoke `flag.FlagSet` parsing. |
| L3 | Declared, not yet verified | Ready | Commands are declared in `cli/manifest.json` and bound to generated proto methods (`binding.kind = connect-rpc`), and may declare an `architecture.primitive` — but that declaration is **not yet verified** against cli-core primitive evidence. A declared primitive is forgeable manifest text; on its own it caps here, not at L4. (`LoadFromManifest` produces `RunCtx` commands, so L3 implies L2.) |
| L4 | Verified renderer-separated primitives | Complete | Every *normal* command's declared cli-core **primitive class** is **verified by matching cli-core evidence** (the primitive the handler was actually built from, stamped onto the command's unexported evidence by a cli-core builder and read via `Command.PrimitiveEvidence()`); every *special-case* command declares a reasoned **exception class** satisfied by the special-case primitive it was built from. Renderer separation is proven, not claimed: no command performs its operation inside an `if ctx.JSON()` / format-selection branch, and no command reaches L4 on manifest text alone. |

The load-bearing invariant at L4: **`--json` is an output contract, never an
operation selector.** A renderer-separated command executes the same operation
and only chooses a renderer at the end; the primitive owns that choice inside
cli-core, so scenario code cannot drift the two formats apart. The verified-vs-declared
split (L4 vs L3) closes the remaining loophole from declaration-only maturity: a
manifest declares *intent*, cli-core primitive evidence proves *implementation*,
and CLI Health only awards L4 when they agree (see `cliapp.ClassifyPrimitiveEvidence`).

### How the conceptual six-rung model collapses

The Phase 1 intent listed six conceptual states (no reliable CLI, runnable CLI,
cli-core shell, declarative commands, manifest/API-bound commands,
renderer-separated primitive commands). "No reliable CLI" and "runnable CLI"
both fold into L0/L1: if cli-health cannot classify a shell at all it is L0; a
runnable cli-core shell is L1. The remaining four map one-to-one onto L1–L4. The
five-rung ladder above is the implementable form.

## Finding-code inventory

All codes belong to the `command_architecture` capability and the `contracts`
dimension. `local_level_impact` is the rung the finding caps the capability at.
Recommended skill: `cli-steer` (add `change-axis-and-evolution-resilience-audit`
for the drift-risk codes).

The classifier emits six codes, and reuses one existing runtime code. Every
code is scored so that only ERROR severity fails the phase; the WARNING codes
are `clean_requirement=required`, which caps the capability rung and counts as
honest debt **without failing the phase** (the scorer only fails on ERROR /
BLOCKER). All are `fix_class: manual` — moving a command onto a primitive, proving
a declaration with evidence, or correcting a declaration requires scenario-specific
judgment.

| Code | Meaning | Level | Default severity | clean_requirement | Fails phase? |
|---|---|---|---|---|---|
| `arch.unclassifiable` | Scenario exposes a CLI/proto surface but has no manifest to classify from, so the capability sits at L0 instead of falsely reporting top maturity by absence of findings. Emitted alongside the existing `manifest.required`. | L0 | WARNING | required | No |
| `arch.primitive_undeclared` | A manifest-bound command declares no `architecture.primitive`/exception at all, so renderer separation cannot be structurally confirmed. Caps the rung at L3. | L4 | WARNING | required | No |
| `arch.primitive_unverified` | A command **declares** a primitive/exception but cli-core reported **no matching primitive evidence**, so renderer separation is declared but not yet proven. The verified-maturity debt signal: a declaration cannot reach L4 on manifest text alone. Caps the rung at L3. | L4 | WARNING | required | No |
| `arch.primitive_mismatch` | A command's declared primitive/exception and the cli-core-observed primitive **disagree** — a contradiction between manifest and implementation. | L4 | ERROR | required | **Yes** |
| `arch.metadata_invalid` | An `exceptions[]` / `architecture` block is stale (its `command` is not exposed at runtime) or malformed (fails `CommandArchitecture.Validate`, surfaced even when the manifest otherwise fails to parse). | L4 | ERROR | required | **Yes** |
| `arch.claimed_maturity_violation` | A top-level `exceptions[]` entry names a command that is actually a normal manifest-bound proto command — a false special-case claim. | L4 | ERROR | required | **Yes** |
| `arch.evidence_malformed` | The committed static primitive-evidence artifact (`.vrooli/generated/cli-primitive-evidence.json`, or the deprecated `cli/primitive-evidence.json` fallback) exists but is unparseable or carries an unrecognized schema, so verified maturity cannot be derived from it. Its evidence is ignored (declared primitives fall back to unverified). Regenerate it from the scenario's evidence generator. | L4 | ERROR | required | **Yes** |
| `arch.evidence_stale` | The evidence artifact parses but its recorded manifest hash no longer matches `cli/manifest.json`, so its evidence describes an older command surface. Its evidence is ignored (declared primitives fall back to unverified) until it is regenerated. | L4 | WARNING | required | No |

The three ERROR codes fire **only after a scenario opts into architecture
metadata and declares it wrong** (a mismatch against observed evidence, a
malformed/stale block, or a false special-case claim), so no un-migrated scenario
is ever failed by them. `arch.primitive_unverified` is the rollout-phase signal:
during rollout, when no cli-core evidence channel is wired for a target scenario,
every declared primitive classifies as advisory not-yet-verified debt — honest,
non-failing, and capped below L4. This satisfies the plan handoff: *if cli-health
cannot observe implementation evidence for a command yet, classify that as
explicit not-yet-verified maturity debt instead of pretending the command is clean.*

**Evidence channel — a committed static artifact (never live execution).** cli-core
primitive evidence reaches CLI Health through the `ArchitectureEvidenceProvider`
seam (see `manifestvalidation.Deps.ArchitectureEvidence`). The production provider
(`manifestvalidation.NewFilesystemArchitectureEvidence`) reads a **committed,
generated JSON artifact** at the canonical scenario-local path
`.vrooli/generated/cli-primitive-evidence.json` (schema `cli-primitive-evidence/v1`),
with a fallback to the deprecated pre-migration path `cli/primitive-evidence.json`
for scenarios mid-migration (plan decision D2 — generated evidence lives under a
generated/ directory, not beside the handwritten CLI code). This is a hard rule
(plan decision D1): **CLI Health must never execute a scenario's commands to learn
what primitive they were built from.** The artifact is generated from the
scenario's own registration metadata — assembling the command tree wires handler
closures but never runs them (`cliapp.BuildPrimitiveEvidence` reads the command's
observed evidence via `Command.PrimitiveEvidence()`, an unexported field only a
cli-core primitive builder can stamp — plan decision D3) — and a scenario keeps it
fresh with a committed golden test (`cliapptest.RequirePrimitiveEvidence`,
regenerated with `UPDATE_CLI_EVIDENCE=1`). The artifact self-describes as generated:
it carries a `do_not_edit` banner and a `source_manifest` provenance field.

The provider classifies the artifact into trust states, and CLI Health compares
the observed primitive per command to the declaration via
`cliapp.ClassifyPrimitiveEvidence`:

- **present & fresh** → agreement is verified (L4, no finding); absence of a
  command entry is `arch.primitive_unverified`; disagreement is
  `arch.primitive_mismatch`.
- **missing** → honest no-evidence state: every declared primitive is
  `arch.primitive_unverified` (never a false L4). No extra artifact finding.
- **malformed** (unparseable / wrong schema) → `arch.evidence_malformed`
  (gating); the evidence is ignored, so declared primitives fall back to unverified.
- **stale** (recorded manifest hash ≠ on-disk `cli/manifest.json`) →
  `arch.evidence_stale` (advisory); evidence ignored, declared primitives unverified.

A nil provider is still legal (empty evidence → everything unverified), but the
production wiring always installs the filesystem provider.

### Project target adoption

The repository root is a first-class CLI Health target with the stable identity
`repo`. Its manifest is `cli/manifest.json`, its proto surface is
`packages/proto/schemas/cli`, and its runtime binary is the control-plane
`vrooli` executable. Project runtime probing intentionally stops at the
immediate help-tree leaves; deeper scenario/resource subtrees are validated by
their owning group rather than being mistaken for root commands.

The first migrated root command is `vrooli scenario list`. It is declared in
the manifest's `scenario-primitives` governance group, assembled with
`cliapp.LoadFromManifestPrimitives`, executed through
`ScenarioControlPlaneService.ListScenarios`, and rendered by `cliapp.ProtoList`.
Its evidence is recorded in the generated root artifact at
`.vrooli/generated/cli-primitive-evidence.json`.

The remainder of the root's declared primitive commands remains deliberately
visible as `arch.primitive_unverified` debt until each handler is migrated.
This is an honest L3 command-architecture result, not a claim that the root
CLI has reached L4 by manifest declaration alone. The current root manifest
contains 29 governance groups and 157 commands; project validation should show
the migrated `scenario list` as verified while retaining the six unverified
legacy scenario handlers as the next migration boundary.

**Why the mature CLI is also the simpler one.** A primitive's operation callbacks
receive a narrow `cliapp.OperationContext` (parsed flags/positionals, bindings,
`Core()`) with **no** `JSON()`, no `Render*`, and no writers. An operation
therefore *cannot* observe the output mode — renderer separation is enforced by
the callback signature at compile time, not by convention. That is why a mature
handler is a plain `call`/`report` pair with no output-format control flow.

**Reused, not re-invented:** the "undeclared custom command" case (a runtime
command that is neither manifest-bound nor a declared exception) is already
covered by the existing runtime check `cli.command_undeclared`. That check is now
exception-aware — a command declared in `exceptions[]` is treated as declared and
no longer flagged — and its remediation text points authors at `exceptions[]`. So
there is no separate `arch.command_unbound` / `arch.exception_undeclared` code.

**Conceptual-only rungs.** L1 (cli-core shell) and L2 (declarative vs legacy
`Run` handlers) are documented rungs of the ladder but are **not** separately
emitted, because distinguishing a legacy `Run` handler from a declarative
`RunCtx` one requires handler AST inspection, which the plan explicitly
prohibits. The structural classifier therefore distinguishes L0 (no manifest),
L3 (manifest-bound; primitive undeclared **or** declared-but-unverified), and L4
(every declared primitive/exception verified by matching cli-core evidence) — the
rungs recognizable from declared structure plus unforgeable primitive evidence.

## Exception taxonomy

Not every command can be a plain proto call. The legitimate special cases are a
**closed vocabulary**; declaring one (with a `reason`) is how a command stays at
L4 without pretending to be a normal proto command. An undeclared special case
(a runtime command that is neither manifest-bound nor a declared exception) is
caught by the existing runtime check `cli.command_undeclared` — there is no
separate `arch.exception_undeclared` code (see the finding inventory above).

| Exception class | For commands that… | Canonical example |
|---|---|---|
| `streaming` | hold a long-lived server stream / follow an event feed to completion. | `test-genie runs follow` |
| `upload` | send multipart/file bodies through the documented REST upload exception (`cliapp.UploadFile`). | image/document upload commands |
| `passthrough` | forward argv to a subprocess/external CLI and stream its output back, adding no proto call of their own. | thin external-tool wrappers |
| `external_delegation` | orchestrate *another* scenario/tool as a step (calling its API/CLI) rather than this scenario's own proto method. | cross-scenario driver commands |
| `durable_run` | own a server-side durable run lifecycle (start → follow/wait → reattach), where the command is a viewer over a run the server owns. | `test-genie execute` |

Rules:

- A declared exception **requires a non-empty `reason`**. Missing reason or an
  unknown class → `arch.metadata_invalid` (gating).
- Each exception class has, or will get, a **matching cli-core primitive** so
  the special lifecycle is framework-owned rather than bespoke (see below). When
  a command uses that primitive, the exception is *satisfied structurally* — it
  is L4, not an undeclared exception.
- Declaring an exception when a primitive already exists is advisory pressure to
  adopt the primitive; it is not an error. The error is running a bespoke
  special-case command with **no** declaration at all.

## Primitive taxonomy (cli-core side)

A **primitive** is a cli-core-owned command class that owns the
parse→call→render (or start→follow→render) lifecycle, so scenario code supplies
only request mapping and human-report mapping — never output-format control
flow. Phase 2 implements these; this section fixes the vocabulary the classifier
recognizes.

| Primitive class | Owns | Renderer |
|---|---|---|
| `proto_list` | one read RPC | `RenderProtoList` — Summary → Results → Retrieval Hints |
| `proto_mutation` | one write/destructive RPC | `RenderProtoMutation` — Result → What Changed → Next Command |
| `operational` | one diagnostic/status RPC | Operational — Status → Triage → Next Steps |
| `action` | a generic single-call action that does not fit list/mutation/operational cleanly | caller-supplied report mapper, cli-core owns `--json` selection |
| `upload` | the multipart upload lifecycle | mutation-shaped result |
| `passthrough` | argv forwarding + output streaming | raw stream, cli-core owns exit-code mapping |
| `streaming` | a server-stream follow lifecycle | event renderer (human) / JSONL (machine) |
| `durable_run` | the durable start → follow/wait → reattach lifecycle | shared across human/`--json`/`--jsonl` — the format is chosen at the end, not at execution |

**A primitive and an exception class are two sides of the same coin.** The four
special-case primitives (`upload`, `passthrough`, `streaming`, `durable_run`)
are exactly the exception classes that have framework support: using the
primitive *is* the declaration, and the observed primitive *is* the verification.
The four normal primitives (`proto_list`, `proto_mutation`, `operational`,
`action`) are how a plain command declares intent; matching cli-core evidence for
that primitive is what carries it the rest of the way to verified L4.

## Rollout policy (advisory vs gating)

**Wave 1 — this plan.** Every code that describes the *un-migrated fleet* is
advisory (WARNING/INFO, `clean_requirement=advisory`, or `uncheckable`). The
capability reports honest current rungs and non-gating debt. The gating codes
(`arch.primitive_mismatch`, `arch.metadata_invalid`, `arch.claimed_maturity_violation`)
require a scenario to have opted into the metadata **and** declared it wrong (a
contradiction against observed evidence, a malformed/stale block, or a false
special-case claim), so no existing scenario is destabilized by adopting the
capability. Templates and reference adopters move to L4 first to prove the docs
are executable: **cli-health itself** is now a verified-L4 adopter — every one of
its manifest commands declares a primitive proven by its committed
`.vrooli/generated/cli-primitive-evidence.json` (`proto_list` / `proto_mutation` / `operational`,
plus `ProtoListOutcome` for `validate scenario`), and the **react-vite template**
generates the same committed artifact as part of scaffolding (still at the
deprecated `cli/primitive-evidence.json` path, which the provider reads via its
migration fallback until the template is migrated). **`test-genie execute`**
is the reference `durable_run` exception: it is built on the cli-core `durable_run`
primitive (`cliapp.RunDurable`) so human/`--json`/`--jsonl` share one server-owned
StartRun path, and it carries matching `durable_run` evidence via
`cliapp.DurableRunLegacy` / `Command.WithLegacyPrimitive`.

**Wave 2 — future, gated on adoption (out of scope here).** Once templates and
reference adopters are migrated and fleet debt is understood, a follow-up plan
may tighten `arch.command_not_declarative` / `arch.command_unbound` to `required`
for scenarios that declare a target maturity, or fleet-wide. Phase 8 records the
migration guidance and hotspots; it does not flip the gate.

Severity still owns phase pass/fail per the shared health contract: only ERROR
and BLOCKER findings fail the `contracts` phase, so the advisory WARNING/INFO
architecture findings never fail a run while the fleet migrates.

## Descriptor design notes (for Phase 3 / Phase 7)

The capability and finding mappings below are the exact shape to add to
`scenarios/cli-health/.vrooli/test-genie.json` under `maturity.capabilities[]`
and `maturity.findings{}`. They are descriptor-compatible with the existing
multi-capability spec (peer to `manifest_contract`, `proto_bindings`, …).

Capability skeleton:

```jsonc
{
  "id": "command_architecture",
  "label": "Command Architecture",
  "description": "Commands converge on cli-core renderer-separated primitives; special cases declare explicit exceptions instead of bespoke output-format control flow.",
  "levels": [
    { "id": "L0", "name": "Command architecture unclassifiable", "status_label": "Unavailable", "capability_summary": "No inspectable cli-core shell to classify.", "next_unlock": "A runnable cli-core CLI shell." },
    { "id": "L1", "name": "cli-core shell", "status_label": "Foundation", "capability_summary": "Commands run on the cli-core app shell.", "next_unlock": "Declarative RunCtx + ArgSchema commands." },
    { "id": "L2", "name": "Declarative commands", "status_label": "Ready", "capability_summary": "Commands parse declaratively via RunCtx.", "next_unlock": "Manifest/API-bound command declarations." },
    { "id": "L3", "name": "Declared, not yet verified", "status_label": "Ready", "capability_summary": "Commands declare architecture but it is not yet verified.", "next_unlock": "Prove each declared primitive with matching cli-core evidence." },
    { "id": "L4", "name": "Verified renderer-separated primitives", "status_label": "Complete", "capability_summary": "Commands are verified renderer-separated primitives; execution is outside any output-format branch." }
  ]
}
```

Each finding mapping carries `capability_id: "command_architecture"`,
`dimension: "contracts"`, the `local_level_impact`, `global_impact`,
`severity_default`, `clean_requirement`, and `fix_class`/`reason` from the
inventory table, plus `recommended_skill_ids: ["cli-steer"]` (drift-risk codes
also list `change-axis-and-evolution-resilience-audit`). The three gating codes
(`arch.primitive_mismatch`, `arch.metadata_invalid`, `arch.claimed_maturity_violation`)
set `severity_default: "SEVERITY_ERROR"` and `clean_requirement: "required"`; the
rest are WARNING with `clean_requirement: "required"` (honest debt, non-failing).

## Manifest metadata contract (Phase 4)

Optional architecture metadata lives in `cli/manifest.json` (schema id
`cli-manifest/v1`) and the cli-core `Manifest` model. Missing metadata defaults
to **legacy/unknown**, so every existing manifest stays valid. The
`architecture.primitive` enum mirrors the **normal declarable** classes cli-core
exports (`DeclarablePrimitiveClasses` — `proto_list` / `proto_mutation` /
`operational` / `action`); special-case classes are declared as exceptions, not
primitives (plan decision D4). The exception enum mirrors `ValidExceptionClasses`.
A Go drift test asserts the schema enums never diverge from these sets.

Two surfaces, because normal and special-case commands live in different places:

- **Per manifest command — `architecture`** (proto-bound commands):
  ```jsonc
  { "name": "query", "binding": { "kind": "connect-rpc", "service": "SearchService", "method": "Search" },
    "governance": { "effect": "read", "run_eligible": true },
    "architecture": { "primitive": "proto_list" } }
  ```
  A command declaring a valid normal `primitive` (`proto_list`, `proto_mutation`,
  `operational`, `action`) is an L4 normal command. The optional nested
  `exception` (`{ class, reason }`) marks a proto-bound command that also carries
  a special lifecycle.

- **Top-level — `exceptions[]`** (custom commands that live *outside* the
  manifest binding path, appended in `register.go`):
  ```jsonc
  "exceptions": [
    { "command": "execute", "class": "durable_run", "reason": "server-owned durable run lifecycle; --json/--jsonl/human share one StartRun path" }
  ]
  ```
  This is how a scenario declares that a custom command (e.g. a durable run or a
  streaming follow) is a *known* special case rather than unknown legacy debt.
  The `command` is the runtime path (`"execute"`, `"runs follow"`).

cli-core validates the vocabulary and the exception-reason rule at parse time
(`CommandArchitecture.Validate`), so a malformed declaration fails fast — this is
what cli-health surfaces as `arch.metadata_invalid`. cli-health cross-checks the
declared `primitive` against structural evidence (a proto primitive on a command
with no `connect-rpc` binding, or a normal primitive on a runtime-divergent
command) to catch `arch.claimed_maturity_violation`.

## Migration guidance

How to move a scenario CLI up the ladder, and how to read the signal.

**Reading partial adoption.** Run the provider CLI without `--json`:
`cli-health validate scenario <target>`. The `command_architecture` capability
line shows the current rung and the blocking/debt counts. A capability that is
**not clean** at L3 means some manifest command either lacks an
`architecture.primitive` (`arch.primitive_undeclared`) or declares one that is
not yet backed by cli-core evidence (`arch.primitive_unverified`); the per-command
findings name exactly which. None of this fails the `contracts` phase (the codes
are WARNING) — it is honest debt to burn down, not a broken build.

**Legacy → L4, one command at a time:**

1. **Legacy `Run` → declarative `RunCtx` + `ArgSchema`.** Replace a
   `Run(func([]string) error)` handler that parses its own `flag.FlagSet` with a
   `RunCtx` handler and an `ArgSchema` (or, better, a manifest command — the
   manifest loader produces `RunCtx` commands for free). This is L1/L2 work and
   is not separately graded (it needs handler inspection cli-health does not do),
   but it is the prerequisite for everything below.
2. **Bind to a proto method (L3).** Add the command to `cli/manifest.json` under
   its group with a `connect-rpc` `binding`. A scenario with a proto surface and
   no manifest is capped low (`manifest.required` + `arch.unclassifiable`).
3. **Declare the primitive (L3, declared).** Add `architecture.primitive`
   matching the command's renderer (`proto_list` / `proto_mutation` /
   `operational` / `action`). On its own this is *declared, not verified* — it
   sits at L3 with `arch.primitive_unverified` until step 4 proves it.
4. **Prove the primitive with evidence (L4, verified).** Build the handler with
   the matching cli-core builder (`cliapp.ProtoList` / `ProtoMutation` /
   `ProtoOperational`; `ProtoListOutcome` for a read whose exit code is derived
   from the payload) and register it through `cliapp.LoadFromManifestPrimitives`,
   so the observed primitive is stamped onto the command's unexported evidence by
   construction and cli-core fails fast if it disagrees with the declaration. Then
   **generate the committed evidence artifact** — add a golden test calling
   `cliapptest.RequirePrimitiveEvidence` (see cli-health's `cli/evidence_test.go`),
   run it once with `UPDATE_CLI_EVIDENCE=1` to write `.vrooli/generated/cli-primitive-evidence.json`,
   and commit it. CLI Health reads that committed artifact statically and, when the
   observed primitive matches the declaration, awards L4. A declaration can never
   reach L4 on manifest text alone, and the artifact is never collected by running
   the command.

**Declaring a legitimate exception** (a command that genuinely cannot be a plain
proto call) — add it to the manifest's top-level `exceptions[]`:

```jsonc
"exceptions": [
  { "command": "execute",      "class": "durable_run",  "reason": "server-owned durable run lifecycle; human/--json/--jsonl share one StartRun path" },
  { "command": "runs follow",  "class": "streaming",    "reason": "long-lived server-stream follow to completion" },
  { "command": "notes attach", "class": "upload",       "reason": "multipart file body; Connect-RPC is not the right transport" }
]
```

An exception whose `command` is not exposed at runtime is a stale declaration
(`arch.metadata_invalid`, gating); an exception naming a normal manifest-bound
command is a false claim (`arch.claimed_maturity_violation`, gating). Both fire
only because the scenario opted into the metadata — un-migrated scenarios never
hit them.

**Fleet migration is out of scope for this plan.** The un-migrated fleet shows
advisory `arch.primitive_undeclared` debt today; tightening those to phase-failing
errors is a deliberate future wave, gated on template + reference-adopter
adoption (already done here) and per-scenario migration. Track it as a follow-up
capture, not an expansion of this plan.

## Cross-references

- [`cli-commands.md`](cli-commands.md) — the CLI command surface and output contracts these rungs formalize.
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — scenario system shape and the thin-CLI invariant.
- [`../../../../docs/reference/health-maturity-assessments.md`](../../../../docs/reference/health-maturity-assessments.md) — the shared provider maturity contract this capability plugs into.
- `packages/cli-core/cliapp/` — the primitives, `RunContext`, and manifest model implementing the rungs.
