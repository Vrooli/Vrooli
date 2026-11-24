# Skill authoring

Universal quality bars for every skill. Per-category specifics live in the matching `skill-authoring-<category>` skill; the Steer guide is the bare `skill-authoring` skill. A skill carries judgment an agent needs and a tool cannot hold ([RULES.md](RULES.md), construct table); it exists to make an agent faster at its outcome, and is deleted when it does not (R10). Skills cite rules and canon, never restate them (R7).

**Source of truth.** Every skill is a spec-conformant `SKILL.md` in its owning scenario (`scenarios/<scenario>/skills/<id>/`) or the Prompt Manager pack tree (`scenarios/prompt-manager/store/skills/packs/core/<id>/`). YAML frontmatter is the source for `name`, `description`, license and Vrooli `metadata`; Prompt Manager generates `skill.json` as an index artifact, so never edit a sidecar without changing the frontmatter.

## Skills are conditioning signals

A skill is not documentation that a model happens to read; it shapes the distribution of behavior an executor produces. A prompt token points into the model's prior, not at its dictionary meaning. Judge any skill through four lenses:

- **Focality.** A named, real, widely documented standard ("EARS requirements", "ASD-STE100", "Given/When/Then") invokes coherent behavior in one phrase, and independent agents converge on it. Hand-rolled rule lists are low-precision pointers where drift lives. Verify a standard exists before citing it. `writing-standards` holds the placement map.
- **Interpretive entropy.** How many divergent-but-compliant readings does the text admit? Decision tables and one meaning per term shrink the space. Test it behaviorally: two agents reading the skill against one situation must make the same decision.
- **Verifiability.** Can compliance be checked by lint, CLI assertion or a named artifact rather than judgment?
- **Attention economy.** Twenty weak rules dilute each other; one named standard plus three sharp exceptions does not. A skill that reads as one argument conditions better than an equal-length pile.

Name-and-delete, never name-and-keep: a named standard earns its place only by letting you delete the hand-rolled rules it replaces.

### Conditioning defect patterns

The single source for audits (`skill-validation`), which cite rows by ID.

| # | Defect | Signal | Fix |
|---|---|---|---|
| C1 | Hand-rolled rule cluster | 5+ style or format rules a real named standard already encodes | Adopt the standard by name; delete the rules it replaces |
| C2 | Name-and-keep decoration | A standard cited beside the rules it describes | Delete the name or the rules |
| C3 | Attention-splitting rule pile | Many weak, orthogonal or contradictory rules aimed at one behavior | Consolidate into one argument or one work table |
| C4 | High interpretive entropy | Two compliant readings of a key instruction produce materially different executions | Make a decision, not more prose; prefer work tables |
| C5 | Unverifiable rules | Compliance judged by adjectives ("clear", "appropriate") | Restate as a mechanically checkable condition, or apply "Destination over direction" |

A confirmed C4 (the `skill-validation` divergence probe produced two different compliant executions) is an executability defect and stays a validation finding. C1, C2, C3 and C5 are conditioning-cost defects and belong to `skill-validation`.

## Universal quality bars

Every skill includes:

- **Intent statement** at the top (1-2 sentences): "If I only read this sentence, what would I prioritize?" Format: `## <Category> focus: <Skill name>`. Categories (`modes[0]`): **Steer** (change scenario behavior), **Platform** (shared platform or packages), **Search** (find information), **Tools** (use a tool, resource or scenario), **Practice** (an engineering methodology), **Meta** (govern the skill system), **Contract** (a workflow node's rendered prompt; exempt from this format).
- **Boundary**: what is in scope and out of scope, naming adjacent skills. Session constraints (preserve behavior, add no features) belong to scope skills (`defaultScope` with `--with-scope`), not inline sections.
- **Convergence patterns** when choices must be consistent: a decision table with explicit criteria admits one reading where prose conditionals admit many.
- **Output expectations**: what can, cannot and must change, and the result format.
- **Human-first CLI use**: prefer direct CLI output over parser pipelines (`--json`, `jq`) and stable selectors over opaque IDs.
- **STE-100 procedural prose** for procedure and rule text: one instruction per sentence, active voice, imperative mood, one meaning per term, concrete commands and paths. Avoid decision-hiding words (canonical list, cited rather than copied): "robust", "comprehensive", "appropriately", "properly", "seamless", "handle", "improve", "enhance", "leverage", "as needed". Rationale passages stay in normal prose.
- **`Troubleshooting & Edge Cases`** for skills with operational CLI complexity (multi-step, mutable state, external dependencies); failure matrices and recovery live there, not in the core workflow. Repeated troubleshooting clarifications are a tooling signal: promote them to CLI output before adding prose. A simple skill states `No known operational edge cases for standard usage.`

**Skill structure** (the shape; sections vary): focus statement, required reading (`prompt-manager skill read <id>` and doc paths), tooling prerequisites, numbered core principles, an optional audit section (discovery commands, red-flag checklist, findings template, for brownfield assessment), memory management, output expectations, troubleshooting.

**Agent memory.** Sessions are stateless. A skill for investigation or audit states what to read at start, what to write at end and in what shape, usually `docs/internal/` findings in the target scenario (`SEAMS.md`, `PROBLEMS.md`); use `visited-tracker` only for multi-pass file coverage. Write to the owning home (R4).

**Referencing.** Required reading: ``- `prompt-manager skill read <skill-id>` ``; doc citations as ``- `path:docs/agent-system/<file>.md` ``. Require only what is essential.

**Never.** Do not create skills for one-off tasks, duplicate `CLAUDE.md`, scenario docs or canon, or restate framework canon. Do not use time quotas or output quotas ("add 5 tests per file"). In Steer skills, do not hardcode paths without `{{TARGET}}`. Search for an existing skill and extend it before creating a new one.

## Contract skills: machine-invoked workflow prompts

A contract skill is the prompt contract of a declared Agent Manager workflow node, referenced by `promptRef`. The reader is a headless run with no human and no history. The node's `resultSpec.schema` is the single source of output shape and the engine renders it into the prompt, so a contract skill never restates field lists or enums. It owns the outcome decision rules. Required content (this is the complete structure; the generic template does not apply):

1. **Task statement**: 1-3 STE-100 sentences.
2. **Outcome work table**: one row per schema outcome value, keyed to observable end states (commands run, artifacts produced, checks passed). Two agents at the same end state must select the same value.
3. **Variable legend**: one line per template binding (`{{.snapshot}}`, …) with what it contains and what is authoritative.
4. **Authority boundary**: what the run must not mutate, stated positively where possible.
5. **Method by citation**: `prompt-manager skill read <skill-id>`; do not inline another owner's doctrine.

**Conservative-branch default.** When an affirmative row's predicate is not proven by evidence in hand, select the conservative outcome and name the unproven predicate. This applies where the conservative outcome routes to review; a workflow whose conservative branch itself acts (rollback, deletion) states its own calculus. Exemplar: `swarm-manager-workflow-plan-author`. Contract skills are exempt from the audit section, memory loop, troubleshooting section and `{{TARGET}}` rules. Validation is the divergence probe against the outcome table, plus a cross-check that every template variable has a node binding and no schema is restated.

## Destination over direction: maturity sources for audit-shaped skills

Skills converge when they are less prescriptive about the path and more prescriptive about the destination: a verifiable end state lets agents arrive from any start. An **audit-shaped skill** examines an existing scenario against a target (the ~30 steer skills: architecture, temporal flow, API, CLI, testing seams and so on). Tools, Search, Practice and greenfield skills may borrow the pattern.

Four ingredients, mandatory for audit-shaped skills:

1. **Scope boundaries up front**, anchored to `path:scenarios/{{TARGET}}/`: in, out, and handoffs to sibling skills.
2. **A named maturity source.** Either a skill-owned ladder where every level is gated by a verifiable artifact (`ls`, `grep`, a CLI assertion) with "What exists" and "When to stop here", or a provider-owned report via `.vrooli/maturity.json`, in which case the skill names the default human CLI command (for example `proto-health validate scenario {{TARGET}}`) as the single source and never copies the provider's L0-L5 ladder.
3. **A decision or archetype table** the agent walks row by row, so two agents land in the same row.
4. **Durable-doc output anchors.** Findings go to `ARCHITECTURE.md`, `SEAMS.md` or `PROBLEMS.md` in the target scenario, never a standalone `*_AUDIT.md`, which freezes one session's view.

Exemplars: `resilience-lenses` §2 and `architecture-lenses` §2. A destination-clear skill is the precondition for climbing the promotion ladder below: a tool can only check what the skill named.

## Registration and metadata

1. Create the skill directory (scenario-owned roles under `scenarios/<scenario>/skills/<id>/`; cross-scenario skills in the core pack).
2. Add `SKILL.md`. Pack skills also carry a generated `skill.json` (`skill-sidecar-gen`); scenario skills need frontmatter only, and it must carry the full Vrooli block (`kind`, `modes`, `status`, `revision`, `requires`, `origin`), not the bare plugin dialect.
3. Run `prompt-manager skill sync`, then `prompt-manager skill show <id>`.

`status` is `active` or `draft`. There is no retired state: retirement is deletion (`prompt-manager skill delete <id>`) and git history is the archive, because no surface consumes a `retired` label and superseded skills pollute list, discover and search.

**`targetDimensions`** (steer skills that Swarm Manager may list as candidate work): the improvement dimensions the skill closes, as a subset of `packages/maturity-go/dimensions/dimensions.json`, the one vocabulary. Non-steer skills omit the field.

## Promotion ladder

As a workflow stabilizes, its prose compresses into structure, and a rule a tool can enforce becomes the tool (R7). The trigger is stability of the routing rule, not age. Many skills settle at S1 or S2 because judgment is genuinely needed.

| Rung | The step is | Promotes when |
|---|---|---|
| S0 | Prose the agent reads and judges | The judgment settles into a fixed table |
| S1 | One CLI command with a deterministic output contract | The same call recurs with one shape across agents |
| S2 | A Prompt Manager Action wrapping that command | Several S1/S2 steps always run together with stable joins |
| S3 | A contract program (`scenarios/program-runtime/docs/guides/program-contracts.md`) as one step | The whole happy path is one program and the tree branches only on its envelope; a program that keeps compensating for a missing invariant promotes that primitive to the scenario CLI |
| S4 | A thin tree over one library program; superseded prose and workarounds deleted | Authorized retirement; replacement verified |

A usage skill is a decision tree and every leaf carries its rung label (`[S0]` to `[S4]`), so a reader knows its cost and what promotes it. This table is the only ladder; "Step rungs" below adds only the rules for a leaf that names a program.

Programs and scenario commands are not competing destinations. Programs are cheap governed composition and may stay permanently; they must not stay the owner of validation, storage or recovery. A `learn.act` fragment is the step-level path: keep the hole in the program, keep traces in the fragment cache, and promote through a reviewable improve cycle, never from a cache hit alone.

**Delete a prose section when** the CLI or tool returns a deterministic status, its output names next steps for common failures, the Action contract is discoverable and validated (for a single operation), and keeping both would duplicate volatile logic. **Keep in prose, permanently:** safety constraints, scope and ownership boundaries, human handoff rules. **Convert to an Action only when** a Vrooli CLI command already covers the behavior (find it with `cli-health search "<operation>"`; otherwise repair the owner first), the behavior is deterministic, and discovery would cut future cost. An Action wraps exactly one command; branching belongs in skills, composition in programs, primitives in scenarios. Baseline the prose before converting and measure the delta after (`prompt-manager discover`, `action validate`, `action run --dry-run`).

Retiring a skill, agent or team (windows, roadmap check, archive path) is in [SCENARIO_DEVELOPMENT.md](SCENARIO_DEVELOPMENT.md#retiring-a-scenario-or-capability).

Skills that audit other skills (`skill-validation`, `conversation-friction-analysis`) classify each major instruction in a table named **Prose Retirement Map** with columns `Instruction / Gate | Disposition (Keep/Collapse/Delete) | Rationale | Prerequisite contract | Risk`. `Keep` rows cite the retention rule above.

## Scenario skill sets: roles, step rungs, and the learning spine

Read [Scenario Development Under a Grant](SCENARIO_DEVELOPMENT.md) for the target artifact contract and execution authority; an improve skill supplies judgment inside an authorized engagement. A scenario owes a small, derivable skill set, declared once in `.vrooli/service.json` under `skills`, living in `scenarios/<scenario>/skills/<id>/SKILL.md`. `prompt-manager skill-set validate <scenario>` (and Test Genie's `skill-set` phase) checks structure; `skill-validation` judges quality; program fixtures establish behavior.

### The three-speed capability stack

| Layer | Optimizes for | Owns | Must not become |
|---|---|---|---|
| Skill | Cheapest change, least rediscovery | Applicability, judgment, safety, routing, learning | A copied CLI manual |
| Governed program | Cheap repeatable composition | Typed workflows, budgets, effects, bounded envelopes, failure evidence | An ungoverned script or a permanent substitute for a missing invariant |
| Scenario | Robustness and reuse | State, domain rules, stable operations, recovery | Agent-specific orchestration |

A repeated path graduates from skill to program; a program that keeps compensating for a missing primitive routes work to the scenario, and once it grows the program and skill are simplified (R8). One-off judgment stays in the skill, one deterministic operation in the CLI or an Action, and only recurring multi-operation workflows earn a program contract. Programs may consume other scenarios' declared programs: search the library first, keep the caller's judgment with the caller and state with the owner, and test the nested path including child failure. Editing an artifact grants its future runs no extra permission.

### The three roles

| Role | Category | Owed when | Content |
|---|---|---|---|
| **Usage** | `tools` | The scenario has a CLI manifest | The judgment that beats `--help`, as a decision tree whose leaves carry a rung label; the learning spine when a memory scope is declared; settings the agent may change without a diff; debug order; safety |
| **Feature** | `tools` or `practice` | Optional, after the pattern appears in two or more agents | One decision table for one use case |
| **Improve** | `practice` | The scenario owns a projection (Answer, Validate, Guide, Act), or two or more scenarios or teams depend on it, or a `team.json::instrument.scenario` names it | The control-loop sections of `scenario-skill-setup`: setpoint, sensors, golden corpora, curation moves, ladder rungs, evidence, stop rules |

**The instrument trigger.** A scenario a standing team reads as its instrument owes both roles however few scenarios import it, because every member reads it on every heartbeat. Resolve it from `team.json::instrument.scenario` across `scenarios/prompt-manager/store/teams/`, since `dependencies.scenarios` does not record teams. Without the roles the team's procedure sits in member files ([STANDING_TEAMS.md](STANDING_TEAMS.md)). An owed role the scenario does not ship needs a dated waiver; a waiver that hides an owed role earns zero credit. There is no "light improve" role: learning during use belongs to the usage skill, and curation of the scenario's own instruments to the improve skill.

Skill ids share one namespace: `<scenario>` (usage), `<scenario>-improve`, `<scenario>-<feature>`; frontmatter `name` equals the folder name.

### Step rungs

The rungs are the table under "Promotion ladder". A leaf that names a program writes `run <scenario>.<program>` and branches on the envelope's `status` and `errors[0].class`; the contract, not the skill, declares them. A program must exist as `scenarios/<scenario>/.vrooli/program-runtime/<name>.py` with a sibling `<name>.json`, or as a promoted library entry. Promotion needs observed repetition showing stable inputs, joins, effects and stop conditions. An improve skill flags a program that contains compatibility repair, retry policy or recovery logic belonging to the scenario. A scenario with no programs is not failing: its leaves are S0 to S2.

### The learning spine

A usage skill learns in use when the scenario declares a memory scope, in frontmatter so the validator sees it:

```yaml
metadata:
  learning:
    scope: "<scenario>-usage"      # a vrooli-memory scope with a facet vocabulary
    capture: "every attempt"       # or: "on failure", "on novel outcome"
```

A program that declares `learning_task` gets Memory preparation and capture from Program Runtime, and the caller supplies only domain decisions and observed evidence. A direct operation follows `prompt-manager skill read vrooli-memory`. Usage skills name their scope, entry points and domain verification; they do not copy attempt schemas, invocation syntax or retry rules, never wrap automatically learned calls in duplicate recall or capture, and never replay the domain action to recover capture. Retrieval is not adoption or success: the scenario decides whether advice applied and whether the outcome was verified. A scenario whose own ledger is the memory (web-search's findings) declares that ledger as the scope's substitute.
