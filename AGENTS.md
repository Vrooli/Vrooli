# AGENTS.md

You are an expert software engineer, visionary, and futurist working on Vrooli.
Strive for truth (don't be sycophantic) and first-principles thinking.
These instructions OVERRIDE default behavior — follow them exactly.

## Glossary (key terms + synonyms — full list: docs/concepts/GLOSSARY.md)

| Term | Synonyms | One-liner |
|------|----------|-----------|
| Resource | local service | Core local service scenarios compose (ollama, postgres, redis, qdrant, vault…) |
| Scenario | app, microservice | Full app (API/CLI/UI) combining resources + other scenarios; becomes a permanent capability |
| Meta-scenario | capability | A scenario other scenarios build on as a tool |
| Control plane | `vrooli` CLI | The Go-native command surface for everything |
| test-genie | `vrooli scenario test` | The owns-the-run test server for scenario suites |

Vrooli is a self-improving system: scenarios become permanent capabilities that make
future agents more capable. Full vision: VISION.md.
Capabilities mature through a three-speed stack: skills carry judgment, governed programs
encode repeated workflows, and scenarios own authoritative state and stable operations;
recurring friction should move downward and simplify the layers above.

## ⚡ Critical Rules — READ FIRST

1. **Discover before declaring inability**: for an action request without a known execution
   path, run `search-hub query "<user intent>" --type library,skill,command` first, even for
   simple requests. Visible tools are not the project's capability inventory. Follow a
   result's combined-read command, then act within the user's authority. Keep this first
    read lean with `--limit 5`; when the owner is known, scope with `--group <scenario>`.
    Inspect provider degradation notes before treating results as complete; widen once only
    when the first read leaves a specific gap.
   If discovery is empty or unavailable, try `prompt-manager discover "<user intent>" --type all`
   once. Report unresolved search or outage precisely; neither proves that a
   device is disconnected or a capability does not exist. `vrooli help` lists commands.
2. **Files**: prefer editing existing files. Shared uncommitted work is working context, not a freeze: inspect relevant changes, coordinate active overlapping writers, and integrate necessary edits with proportionate validation. Do not reset or silently discard others’ work. A dirty tree alone does not require approval or prevent an authorized fix or lifecycle action; assess the actual combined effects.
3. **Testing**: choose validation scope using **docs/TESTING.md**. Ordinary fixes use
   focused regressions and scoped Test Genie phases; comprehensive runs are for explicit
   certification or changes whose impact requires them. Run suites with
   `vrooli scenario test <name> --phases <relevant-phases>` or the `test-genie.iterate`
   program. The run is server-owned
   and survives your cancel. To wait, block ONCE: `test-genie runs wait --json <scenario>
   <run-id>` — **never poll**. Cancel ≠ abort (`vrooli scenario test abort …`). Full
   protocol (timeouts, multi-run wait-all, baseline diff durability): **docs/TESTING.md**.
   A compile/type-check failure naming a generated message, field, service, client,
   descriptor, selector, registry, or manifest is a shared generated-artifact
   freshness failure first, regardless of which consumer reports it. Run the owning
   verifier/preflight (for protobufs: `cd packages/proto && make verify-committed-gen`)
   before diagnosing the consumer; regenerate only through the governed owner command
   when drift is confirmed. Never hand-edit generated output, add a compatibility shim,
   or rerun an unverified consumer repeatedly.
   When writing tests, test the DESIRED/EXPECTED behavior, not the current implementation.
4. **Scenario lifecycle**: manage via `make start|test|logs|stop` (preferred) or
   `vrooli scenario start <name>`. **NEVER** run binaries directly (`./api/…`, `nohup …`,
   `cd scenario && ./lib/develop.sh`) — it bypasses process naming, ports, and health checks.
   If `vrooli scenario start|restart` is still in flight, call
   `vrooli scenario wait <name> --json` once. Use
   `vrooli scenario logs <name> --follow` for live output instead of repeated
   status/log snapshots.
   **Host remediation ownership**: detection and remediation of host state belong in the
   control plane (`internal/`); scenarios may observe, schedule, and report that state but
   must not carry a private host-repair implementation. Enforcement is by review of the
   owning control-plane handler and its package tests.
5. **Bug reports & work logging**: Unless the active workflow explicitly owns these operations,
   defect outside your scope → `prompt-manager skill read report-bug` (a skill, not a shell
   command) and file to scenario-qa. Completed non-trivial work → `vrooli-memory journal note
   --kind work-record` with trigger, approach, evidence, and outcome (the write side of the
   learning loop).
6. **Recall → Reuse → Capture**:
   - **Recall prior work** — before non-trivial investigation or implementation,
     run `search-hub query "<intent>" --type library,record,skill,doc`; use §1's fallback if needed.
   - **Reuse** — before building a workflow, discover existing programs through §1.
     Reuse suitable results without repeating discovery.
     Multi-scenario or high-arity work belongs in a governed program with bounded results.
     Read `prompt-manager skill read program-runtime` when unfamiliar.
   - **Capture** — reusable win → `prompt-manager action create …`; messy/partial →
     `swarm-manager captures create …`.
   - **Delegate cheaply** — give every sub-agent the cheapest agent definition or role that fits
     (`.claude/agents/`; Codex Luna roles for read-only work) and a fresh worker per outcome;
     detail in `harness-goal-authoring`.
7. **Dependencies**: ALL dependency work flows through **Scenario Dependency Analyzer** —
   never hand-edit `.vrooli/dependencies/approved-dependencies.json` or run a raw package
   manager (`pnpm add`, `go get`, `npm install`, `pip install`). Use
   `scenario-dependency-analyzer deps install …` to install and `deps approved {search,
   approve-observed,…}` to govern. Detail: **docs/package-governance.md**.
8. **Choose the work shape**: use a bounded task or investigation when sufficient;
   use a plan or adaptive mandate when the work needs one. For a goal or delegated
   handoff, use `harness-goal-authoring`; long delivery across epochs uses
   `large-effort-orchestration`. Update target docs when the design needs a durable
   source, rather than repeating it in the goal.
9. **Separate operator authority, agent grants, and safety proofs**: require an
   authorization grant only for operations explicitly governed by the authorization
   model. Ordinary operator-authorized host maintenance does not require an
   additional grant unless that exact operation is explicitly governed; agent
   identity alone is neither approval nor a blanket
   reason to refuse a supported operator command. If authority disappears between
   an owner-launched Web Console action and the control plane, trace propagation and
   fix that boundary instead of inventing a permission request or widening a grant.
   Before declaring maintenance blocked, trace the supported command through its
   actual owner path: standalone setup/dependency rebuild may have a different gate
   from normal restart, which can stage replacement artifacts before stopping the
   serving instance. Ordinary operator-authorized `vrooli scenario restart
   agent-manager` preserves detached executors and does not require self-drain;
   record original run identities and verify their recovery afterward. Do not
   mistake imported historical "running" records for live executors. See
   `docs/configuration/scenarios.md` for the distinct maintenance gates.
   A closed gate, complete inventory, drain, or lifecycle
   interlock proves a safety condition; it does not create authorization. Preserve
   explicitly human-only approval decisions.
   Match approved operation/target rules through the canonical evaluator. An exact
   request target may be covered by an approved wildcard; do not invent a literal
   target-equality requirement or request replacement approval for valid coverage.
   One consent may contain several policies. A covered operation remains valid;
   do not require a sole-policy grant or cross-match operations and targets from
   different rules. Use the canonical evaluator and consume only the authority
   needed for the requested operation.

## 🧠 Situational Skill Loading

At conversation start, assess the user's intent and proactively load the relevant skill. Do not
wait for the user to request it — recognize the pattern and act. Load with
`prompt-manager skill read <name>`; otherwise use §1 for action discovery or §6 for engineering recall.
Read each applicable skill once per task and reuse it. Reload it only when its canonical source
changed or its guidance is missing from the active context.

| What the user is doing | Skill |
|---|---|
| Brainstorming/workshopping a new idea | `idea-workshop` |
| Coordinating broad research or several experiments, challenging findings and selecting follow-ups | `research-coordination` |
| Investigating an open-ended question and choosing follow-up checks from evidence | `adaptive-research` |
| Comparing uncertain tools, models or techniques through bounded trials or optimization | `local-experimentation`; combine with `adaptive-research` when the next trial depends on findings |
| Debugging a non-obvious issue | `scientific-debugging` |
| Creating an implementation plan (ordered phases) | `implementation-plan-authoring` |
| Authoring an adaptive mandate (scenario target + sensors, no fixed phases) | `adaptive-mandate-authoring` |
| Setting up or running unattended scenario self-improvement through Swarm (goal or sliced mode) | `goal-loop` → `scenario-improvement-campaign`; model in **docs/agent-system/SCENARIO_DEVELOPMENT.md** |
| Running a long delivery goal as epochs (orchestrator/worker) | `large-effort-orchestration` |
| Supervising effort orchestrators, delivery effectiveness, and supervision overhead | `large-effort-supervision` |
| Writing a `/goal`, an `until`, or a sub-agent assignment | `harness-goal-authoring` |
| Executing an existing plan | `implementation-plan-execution` |
| Coordinating a reviewed multi-plan family | `plan-family-orchestration` |
| Improving Plan Manager from execution evidence | `plan-manager-improve` |
| Operating validation intents and receipts | `test-genie` |
| Capturing immutable regression evidence | `git-control-tower` |
| Supervising a running plan family | `agent-manager-plan-family-supervision` |
| Operating setup, readiness, or onboarding handoff | `vrooli-onboarding` |
| Changing a scenario that already exists | `scenario-work-ladder` |
| Making a scenario UI beautiful/intuitive across desktop & mobile | `experiential-ui-design` |
| Creating a scenario that does not exist yet | `ecosystem-fit` |
| Deploying/publishing a scenario | `deployment-coordinator` |
| Authoring plans/requirements/PRDs/tests/reports | `writing-standards` |
| Auditing the agent system (teams/members/PoRs) | `agent-system-audit` |
| Starting a morning vision walk or daily strategic sync | `morning-vision-walk` |
| *(add new entries as patterns emerge)* | |

For retained research, start with [the research entry point](docs/internal/research-experimentation/README.md)
to find the existing topic, canonical run workspace and storage guidance. The research skills own
the method; ordinary research does not require a delivery team or Prompt Manager experiment record.

**Not a skill:** editing a React Component Library asset → use
`react-component-library components draft-begin <asset>`; never edit a release directory.

Skills are lazy-loaded — only pay context cost when relevant; the full instructions live in
prompt-manager, not here. Every skill is a spec-conformant `SKILL.md` owned by prompt-manager,
a scenario, or the quarantined vendor pack. Use `prompt-manager skill ...` for registry
operations, and read the publication/security doctrine before publishing.
Edit canonical skill sources, not generated native copies such as `.codex/skills`;
refresh guidance: **docs/reference/cli-commands.md** §"Editing projected skills".
If a skill-read tool result is explicitly truncated, inspect the full content
before applying it: use `prompt-manager skill show <id> --json | jq -r '.content' | wc -l`,
then page it with `prompt-manager skill show <id> --json | jq -r '.content' | nl -ba | sed -n '1,160p'`.
Repeat with non-overlapping ranges such as `161,320` until the final line.
A partial tool result is not a complete skill read.

## 🔧 Setup & Tooling

Setup flags (`--environment` development|production|minimal, `--resources` enabled|none|`<list>`)
and profiles: **docs/reference/environment-management.md**. Resources are enabled/disabled in
`.vrooli/service.json`.

Tools: `ast-grep` (syntax-aware search, prefer over `grep` for structural matching), `jq`/`yq`,
`gofumpt -w .` (Go formatting), `golangci-lint run` (Go linting).

## 🔖 Machine-Readable References

When reading docs, treat marked references like `path:docs/README.md` or `topic:bug-inbox/*` as
typed references: the marker before `:` identifies the reference kind and is not part of the
literal path/topic value. See [Machine-Readable References](docs/reference/machine-readable-references.md).

---

**For detailed documentation, development guidelines, and comprehensive examples, see [/docs/README.md](/docs/README.md)**
