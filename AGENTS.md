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
   result's combined-read command, then act within the user's authority.
   If discovery is empty or unavailable, try `prompt-manager discover "<user intent>" --type all`
   once. Report unresolved search or outage precisely; neither proves that a
   device is disconnected or a capability does not exist. `vrooli help` lists commands.
2. **Files**: always prefer editing existing files over creating new ones.
3. **Testing**: choose validation scope using **docs/TESTING.md**. Ordinary fixes use
   focused regressions and scoped Test Genie phases; comprehensive runs are for explicit
   certification or changes whose impact requires them. Run suites with
   `vrooli scenario test <name> --phases <relevant-phases>` or the `test-genie.iterate`
   program. The run is server-owned
   and survives your cancel. To wait, block ONCE: `test-genie runs wait --json <scenario>
   <run-id>` — **never poll**. Cancel ≠ abort (`vrooli scenario test abort …`). Full
   protocol (timeouts, multi-run wait-all, baseline diff durability): **docs/TESTING.md**.
   When writing tests, test the DESIRED/EXPECTED behavior, not the current implementation.
4. **Scenario lifecycle**: manage via `make start|test|logs|stop` (preferred) or
   `vrooli scenario start <name>`. **NEVER** run binaries directly (`./api/…`, `nohup …`,
   `cd scenario && ./lib/develop.sh`) — it bypasses process naming, ports, and health checks.
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
7. **Dependencies**: ALL dependency work flows through **Scenario Dependency Analyzer** —
   never hand-edit `.vrooli/dependencies/approved-dependencies.json` or run a raw package
   manager (`pnpm add`, `go get`, `npm install`, `pip install`). Use
   `scenario-dependency-analyzer deps install …` to install and `deps approved {search,
   approve-observed,…}` to govern. Detail: **docs/package-governance.md**.

## 🧠 Situational Skill Loading

At conversation start, assess the user's intent and proactively load the relevant skill. Do not
wait for the user to request it — recognize the pattern and act. Load with
`prompt-manager skill read <name>`; otherwise use §1 for action discovery or §6 for engineering recall.

| What the user is doing | Skill |
|---|---|
| Brainstorming/workshopping a new idea | `idea-workshop` |
| Debugging a non-obvious issue | `scientific-debugging` |
| Creating an implementation plan | `implementation-plan-authoring` |
| Executing an existing plan | `implementation-plan-execution` |
| Coordinating a reviewed multi-plan family | `plan-family-orchestration` |
| Improving Plan Manager from execution evidence | `plan-manager-improve` |
| Operating validation intents and receipts | `test-genie` |
| Capturing immutable regression evidence | `git-control-tower` |
| Supervising a running plan family | `agent-manager-plan-family-supervision` |
| Operating setup, readiness, or onboarding handoff | `vrooli-onboarding` |
| Changing a scenario that already exists | `scenario-work-ladder` |
| Creating a scenario that does not exist yet | `ecosystem-fit` |
| Deploying/publishing a scenario | `deployment-coordinator` |
| Authoring plans/requirements/PRDs/tests/reports | `writing-standards` |
| Auditing the agent system (teams/members/PoRs) | `agent-system-audit` |
| Starting a morning vision walk or daily strategic sync | `morning-vision-walk` |
| *(add new entries as patterns emerge)* | |

**Not a skill:** editing a React Component Library asset → use
`react-component-library components draft-begin <asset>`; never edit a release directory.

Skills are lazy-loaded — only pay context cost when relevant; the full instructions live in
prompt-manager, not here. Every skill is a spec-conformant `SKILL.md` owned by prompt-manager,
a scenario, or the quarantined vendor pack. Use `prompt-manager skill ...` for registry
operations, and read the publication/security doctrine before publishing.
Edit canonical skill sources, not generated native copies such as `.codex/skills`;
refresh guidance: **docs/reference/cli-commands.md** §"Editing projected skills".

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
