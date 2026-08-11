# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario program-runtime`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

**Purpose.** Program Runtime is the governed execution surface where an agent submits a *program* instead of a sequence of tool calls. A program runs in a session-persistent kernel where every Vrooli operation carrying a manifest binding is available as a typed callable, and where results bind to in-kernel handles rather than being copied into the submitter's context.

The problem it removes is arity and materialization. An agent that needs to inspect 128 scenarios today issues 128 tool calls and pays context for 128 full responses, most of which it discards. The same work is one program that issues 128 typed calls in-kernel and returns the six-row answer. Intermediate data stays a variable.

**Primary users.**

- Coding agents in any harness — Claude Code, Codex, OpenCode, and future native runners. Access is an ordinary CLI command, so no harness-specific integration exists or is needed.
- The meta-optimization team, which reads the Act projection this scenario owns.
- Human operators who want the same programmatic surface at a terminal.
- Other scenarios — agent-manager workflow steps, swarm-manager work execution — that need governed multi-operation composition.

**Deployment surfaces.** A Tier 1 local scenario: Go API, thin CLI over that API, operator UI, and a Python kernel sidecar. A kernel is a per-session child process, not a long-lived singleton.

**Value promise.**

1. Operations compose without round-trips, so an agent's context cost stops scaling with the volume of data it inspects.
2. Every operation crosses one boundary that already knows its declared effect and permissions, so governance is enforced rather than described.
3. A failed program is an analyzable artifact with a deterministic error, which is a higher-quality friction signal than intent inferred from tool-call transcripts.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Runtime binding generation | The Program Runtime MUST generate callable bindings at session start from the proto descriptor image and the scenario CLI manifests, and MUST NOT require per-scenario integration code for a scenario that ships a manifest.
- [x] OT-P0-002 | Pre-flight argument validation | When a program calls a binding with arguments that do not satisfy the bound proto method's field contract, the Program Runtime MUST reject the call before issuing any request and MUST name the offending field.
- [x] OT-P0-003 | Handle-based results | The Program Runtime MUST return operation results as in-kernel handles whose default representation is bounded, and MUST require an explicit materialization call before result data enters the submitter's output.
- [x] OT-P0-004 | Session-persistent kernel state | While a session is live, the Program Runtime MUST preserve kernel variable state across separate program submissions so a later program can reference an earlier program's handles.
- [x] OT-P0-005 | Governance enforced at the binding boundary | The Program Runtime MUST NOT generate a binding for a command whose CLI manifest governance declares run_eligible false, and MUST refuse a binding whose declared effect is destructive unless the session holds an explicit matching grant.
- [x] OT-P0-006 | Typed program telemetry | The Program Runtime MUST emit a typed platform event for every program submission, binding invocation, and program failure through the shared event bus, so friction analysis reads program evidence without a scenario-local analysis stack.
- [x] OT-P0-007 | Act projection ownership | The Program Runtime MUST serve the Act denominator through the shared space projection verb.
- [x] OT-P0-008 | Act numerator registry RPC | The Program Runtime MUST expose a binding-registry RPC that reports, per Act operation class, whether every operation it names resolves to a manifest-bound governed binding, so meta-optimization-manager computes the Act numerator live and never stores it.
- [x] OT-P0-009 | Domain measure coverage | Every stateful domain the Program Runtime owns MUST carry at least one declared measure in the CLI manifest, or a `measures.omitted` waiver naming the domain and its reason, so measures-health grades the scenario on a stated position rather than an absence.

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | In-kernel capability discovery | The Program Runtime SHOULD expose capability discovery as an in-kernel call so a program resolves an operation by intent without the callable surface being preloaded into the submitting agent's context.
- [x] OT-P1-002 | Typed inference bindings | The Program Runtime SHOULD expose classify, extract, and judge operations resolved through ai-gateway, so a program performs bounded typed inference without spawning a delegated agent run.
- [x] OT-P1-003 | Delegated agent runs from a program | The Program Runtime SHOULD let a program spawn an agent-manager run and collect its evidence, so unbounded agentic work stays distinguishable from bounded inference.
- [x] OT-P1-004 | Sandbox composition | The Program Runtime SHOULD bind a session to a workspace-sandbox workspace so filesystem effects are isolated and reviewable.
- [x] OT-P1-005 | Bounded session lifecycle | The Program Runtime SHOULD enforce idle reclamation and wall-clock and memory ceilings per session, and SHOULD report the reason when it reclaims a session.
- [x] OT-P1-010 | Per-session inference spend ceiling | The Program Runtime SHOULD enforce a configurable per-session ceiling over ai-gateway `Usage.cost_micros`, input tokens, and output tokens, and SHOULD report `inference_spend_exceeded` when the ceiling reclaims or refuses work.
- [ ] OT-P1-011 | Per-session delegated-run spend ceiling | The Program Runtime SHOULD enforce a separate configurable per-session ceiling over agent-manager delegated-run spend, and SHOULD report `delegated_run_spend_exceeded` when that ceiling reclaims or refuses work.
- [x] OT-P1-006 | Queryable program corpus | The Program Runtime SHOULD retain submitted program source and failure detail as a queryable corpus so recurring failure shapes are derivable mechanically.
- [x] OT-P1-007 | Binding registry inspection surface | The Program Runtime SHOULD provide an operator surface that browses the resolved callable namespace and, for every fleet capability that is unbound, states which of the declared unbound reasons applies. This is promoted out of OT-P2-001 because it is the only operator surface that carries information before any program has run, and because it renders the same registry state the Act numerator computes.
- [x] OT-P1-008 | Program provenance | The Program Runtime SHOULD record whether a program was submitted by an agent or by a human operator, and SHOULD exclude operator-submitted programs from corpus mining by default, so the corpus keeps measuring what agents attempt rather than what operators experiment with.
- [x] OT-P1-009 | Act denominator audit | Once the binding-registry numerator is live, the Program Runtime SHOULD audit every Act denominator cell against the resolved registry and raise the stated denominator confidence above `SKETCH`, so Act coverage on the readiness board is measured rather than authored. The denominator was written before this scenario existed and 12 of its 28 cells are marked unaudited; without this target the board reports an Act percentage at `SKETCH` confidence indefinitely.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Operator inspection surface | Beyond the binding registry promoted to OT-P1-007, the Program Runtime MAY provide operator surfaces for sessions, kernel variables, and program history, including replaying a historical program and forking it into a new session.
- [ ] OT-P2-002 | Program corpus mining | The Program Runtime MAY analyze recurring program shapes in the corpus and propose them as skill or action candidates with their call sites as evidence.
- [x] OT-P2-003 | Named durable workspaces | The Program Runtime MAY support named sessions that survive across agent runs so a long investigation reuses accumulated state.
- [ ] OT-P2-004 | Alternate kernel adapters | The Program Runtime MAY support kernel languages beyond Python where a measured capability argument justifies the added surface.

## 🧱 Tech Direction Snapshot

**Preferred stacks.** Go for the API, CLI, and proto contracts, matching the contract every scenario follows: proto-defined types, and a CLI that is a thin wrapper over the API. The kernel is a Python sidecar under `kernel/`, following the `browser-automation-studio/playwright-driver` precedent — a sidecar inside the scenario rather than a shared package, because it has exactly one consumer.

Python is selected for the kernel on capability grounds, not consistency grounds. The value of this scenario depends on a model writing a correct program on the first attempt; model competence at Python in a REPL is materially higher than at the alternatives, and IPython supplies session persistence, async execution, and rich representations — the handle mechanism — without new code. The kernel is an execution substrate rather than a scenario, the same category as postgres or ollama, which are also not Go.

Scenario-language independence is structural, not a convention: transport is Connect over HTTP and JSON, so the kernel language is decoupled from the language of any scenario it calls. A scenario written in any language is callable as long as it ships a manifest.

**Data and storage.** SQLite for sessions, program history, binding-resolution state, and telemetry. No shared resource is required. Kernel variable state is process memory and is deliberately not persisted: a handle is a live reference, and promoting it to durable storage would recreate the materialization cost this scenario exists to remove.

**Integrations.** Bindings are generated at runtime from the generated proto descriptor image and the scenario CLI manifests. Outbound calls use ConnectRPC. Typed inference resolves through ai-gateway. Delegated agent work resolves through agent-manager. Telemetry emits through the shared api-core event bus.

**Non-goals and guardrails.**

- Not a security sandbox. Filesystem isolation composes from workspace-sandbox and the governance boundary covers the API dimension; neither claim extends to adversarial code.
- Not a code-synthesis engine. The submitting agent writes the program.
- Not a replacement for agent-manager runs. A program delegates to a run; it does not become one.
- Not a general notebook server. A session exists to serve governed programs, not arbitrary compute.
- Not a second inference front door. Typed inference resolves through ai-gateway only.

## 🤝 Dependencies & Launch Plan

**Required resources.** A CPython runtime with IPython for the kernel sidecar, resolved through the existing host-requirement path. No new Vrooli resource is expected. Every third-party package is found and installed through Scenario Dependency Analyzer; no raw package-manager call and no hand-edited governance JSON.

**Scenario dependencies.**

| Scenario | Why |
|---|---|
| `ai-gateway` | Typed inference inside programs. The typed Connect/CLI surface, schema gate, role catalog, batch operation, and usage accounting are now available; program-runtime bindings remain owned by OT-P1-002 implementation work. |
| `agent-manager` | Delegated agent runs spawned from a program. Separately, agent-manager must **subscribe** to this scenario's events — see the external obligations below. |
| `vrooli-events` | Telemetry through the shared event bus. This scenario can prove emission and delivery; it cannot make anything consume them. |
| `search-hub` | In-kernel capability discovery. |
| `workspace-sandbox` | Optional filesystem isolation for a session. |

`meta-optimization-manager` is a consumer, not a dependency: it reads the Act projection this scenario owns.

**External obligations.** Three pieces of required work sit outside this scenario's boundary. Each is stated here with a named owner so it is a scheduled hand-off rather than an unowned risk, because in every case this scenario can reach 100% green while the obligation stays unmet and the value stays unrealised.

| Obligation | Owner | What it unblocks | Consequence if it stalls |
|---|---|---|---|
| Subscribe to this scenario's program events | `agent-manager` | Friction analysis reading program evidence | OT-P0-006 is green, events are delivered to the bus, and nothing reads them |
| Raise `cli/manifest.json` coverage beyond 58 of 128 scenarios | fleet-wide; surfaced by `cli-health` and ranked by `meta-optimization-manager` `focus next` | The ceiling on the entire Act surface | Act coverage is capped at roughly 45% of the fleet no matter how complete this scenario is |

**Operational risks.**

1. *Identity propagation is unsolved and deliberately deferred.* Agent identity reaches event receipts today through Go shared packages. The kernel is a non-Go sidecar, so identity across agent to program-runtime to program to ai-gateway does not inherit and must be carried explicitly. This blocks trustworthy attribution of in-program inference; it does not block the runtime.
2. *Manifest coverage bounds the Act surface.* 58 of 128 scenarios ship a CLI manifest. The callable surface cannot exceed manifest coverage, and the Act numerator reports that honestly rather than concealing it. Raising the ceiling is fleet work, not this scenario's — it is listed under External obligations with a named owner so it is scheduled rather than merely acknowledged.
3. *Execution is not adversarially contained.* Recorded as a stated boundary, not a defect.
4. *Handle discipline carries the whole value.* Bindings without bounded materialization produce the same context cost with different syntax. This is a design risk, not an implementation detail, and it is why context bytes per query is an acceptance signal rather than a nice-to-have.

**Launch sequencing.**

Ordered by the domain read graph in `docs/concepts/DOMAINS.md`, not by surface.

1. Binding registry — generation, typed pre-flight validation, and bind-time governance refusal, proven against the live descriptor with no kernel present.
2. Act projection — the space verb, the numerator RPC, then the denominator audit. Deliberately ahead of the kernel: it is a pure read over the step-1 registry, `meta-optimization-manager` already waits on it with a placeholder branch, and it converts manifest coverage from an anecdote into a tracked board number before any program has run. Steps 1–2 are a coherent shippable slice with no Python in it.
3. Kernel sidecar and session lifecycle, including the grants a session holds.
4. Programs, handles, and bounded materialization. The acceptance signal is that context bytes do **not** scale with result size, not an absolute byte budget — a budget proves handle discipline but cannot falsify the scaling claim this scenario is built on.
5. Runtime governance refusal, then telemetry and the event contract, with delivery to the bus proven rather than emission alone.
6. Domain measures — one declared measure per stateful domain, or a waiver with a reason. The binding-coverage measure and the context-avoided measure land with the domains that own them, not as a separate analytics pass.
7. Detemplate the scaffold's worked example and close requirement traceability.
8. Inference and delegation bindings. Inference is gated on the ai-gateway promotion named in External obligations; delegation is not, and can land first.
9. The binding registry inspection surface. Scheduled last but not dependent on anything after step 2, so it can move earlier whenever an operator needs it.

## 🎨 UX & Branding

**Look and feel.** The `vrooli-default` Vrooli Operational Console kit, adopted unmodified. This is an operator console: dense, evidence-first, and read-mostly.

The organizing metaphor is an **IDE for programs an agent wrote**. Because the primary user is an agent that never opens the interface, the console's job is forensic — what was submitted, what it called, what was refused, what came back, what stayed in the kernel. The IDE mapping is deliberate and near-complete: the binding registry is the symbol explorer, the corpus is the file history, program detail is the editor, sessions are debug sessions with a watch window, and live execution is the output pane. Five surfaces become panels around one center rather than five unrelated screens.

Program source renders in a real code editor rather than a formatted block. The editor is the anchor of the design language: monospace, gutter-first, dense. It earns that place by carrying scenario-specific meaning that a viewer cannot — hover resolves a binding's typed signature and declared effect from the proto descriptor, gutter markers distinguish the three failure classes, and inline decorations report per-line what each call fetched versus what it materialized.

**Voice and messaging.** Precise and non-reassuring. A failed program reports the failing line and the offending argument; it never summarizes a failure as a warning. Every number is reported with its basis, matching the honesty contract the readiness board already holds — an unmeasured surface reports unavailable with a stated reason rather than zero.

**Accessibility.** The template accessibility floors are binding: `role` and `aria-*` primitives, `data-testid` selectors, keyboard reachability for every control, and status meaning never carried by color alone. Program output and error text must stay legible to a screen reader, which constrains how failures render — a traceback is content, not decoration, and must not be collapsed into an icon or a color swatch. A code editor is the hardest surface to keep accessible, so the editor's meaning must survive without it: every marker the gutter shows also exists as text in an accessible list, and no failure is conveyed only by a squiggle.

**Branding hooks.** Generic template icons and the seeded install assets stand until product branding exists.

**Experience contract.** Authored in `experience/`: six product pages, two journeys, and per-state behavior for the three failure classes. Five of the six are the IDE panels above — binding registry, corpus, program detail, sessions, and live execution — arranged around one center rather than as unrelated screens. The sixth is the declared-measures statistics surface, which stands apart because it reports the scenario's own numbers rather than any one program's. The triage journey — a failed program read, understood, forked, and resubmitted — is the spine that the other surfaces branch from.

## 📎 Appendix

**Design lineage.** The programmatic-tool-calling model, the treatment of context as a variable, and the harness-state refinement loop are adapted from Prime Intellect's Prime Agent (MIT) and the Recursive Language Model and Continual Harness papers behind it. Vrooli implements the techniques natively rather than adopting that harness, because the proto, Connect, and CLI-manifest substrate already present here supports a governed version those designs cannot offer — their kernel is documented as explicitly not a security sandbox, whereas bindings here are refused or granted from declared manifest governance.

**Related canon.**

- `meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md` — the projection model this scenario's Act denominator belongs to.
- `docs/spaces/act-space.md` — the Act denominator, relocated here from meta-optimization-manager on creation of this scenario.
- `prompt-manager/docs/concepts/ACTIONS.md` — the curation layer above these bindings; actions wrap argv, bindings wrap typed methods.
- `.vrooli/schemas/cli-manifest.schema.json` — the governance vocabulary this scenario enforces rather than redefines.
