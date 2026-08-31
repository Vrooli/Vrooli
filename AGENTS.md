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

## ⚡ Critical Rules — READ FIRST

1. **Help**: `vrooli help` lists commands.
2. **Files**: always prefer editing existing files over creating new ones.
3. **Testing**: run suites with `vrooli scenario test <name>`. The run is server-owned
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
6. **Recall → Reuse → Capture** (reflex, not a checklist):
   - **Recall** — before non-trivial work: `search-hub query "<intent>" --type record,skill,doc`.
     Falls back to `prompt-manager discover "<operation>" --type all` when search-hub returns
     nothing or is unavailable; discover returns both skills (judgment) and actions (typed
     wrappers over a single CLI command), ranked by relevance. Decompose broad work into
     generic reusable operations, not scenario-specific plan titles.
   - **Reuse** — for any *recurring* task (heartbeat, walk, audit, sweep), look for an existing
     program before hand-rolling: `search-hub query "<task>" --type library`. Multi-scenario or
     high-arity work belongs in one governed program rather than a long tool-call loop: it gives
     governed sessions, flat scenario namespaces, bounded Handles, and typed failure evidence.
     How-to: `prompt-manager skill read program-runtime`.
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
`prompt-manager skill read <name>`; if nothing below matches, fall back to §6 Recall.

| What the user is doing | Skill |
|---|---|
| Brainstorming/workshopping a new idea | `idea-workshop` |
| Debugging a non-obvious issue | `scientific-debugging` |
| Creating an implementation plan | `implementation-plan-authoring` |
| Executing an existing plan | `implementation-plan-execution` |
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
