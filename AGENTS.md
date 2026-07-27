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
5. **Bug reports & work logging**: Unless the active workflow explicitly owns these operations,
   defect outside your scope → `prompt-manager skill read report-bug` (a skill, not a shell
   command) and file to scenario-qa. Completed non-trivial work → `swarm-manager records create`
   (the write side of the learning loop).
6. **Recall → Discover → Capture** (reflex, not a checklist):
   - Recall: before non-trivial work, `search-hub query "<intent>" --type record,skill,doc`.
   - Discover: before hand-rolling ops, `prompt-manager discover "<operation-1>" "<operation-2>" --type all`; decompose broad work into generic reusable operations/capabilities, not scenario-specific plan titles.
   - Capture: reusable win → `prompt-manager action create …`; messy/partial → `swarm-manager
     captures create …`.
7. **Dependencies**: ALL dependency work flows through **Scenario Dependency Analyzer** —
   never hand-edit `.vrooli/dependencies/approved-dependencies.json` or run a raw package
   manager (`pnpm add`, `go get`, `npm install`, `pip install`). Use
   `scenario-dependency-analyzer deps install …` to install and `deps approved {search,
   approve-observed,…}` to govern. Detail: **docs/package-governance.md**.

## 🧠 Situational Skill Loading

At conversation start, assess the user's intent and proactively load the relevant skill. Do not wait for the user to request it — recognize the pattern and act.

```
What is the user doing?
├─ Brainstorming/workshopping a new idea  → prompt-manager skill read idea-workshop
├─ Debugging a non-obvious issue          → prompt-manager skill read scientific-debugging
├─ Creating an implementation plan        → prompt-manager skill read implementation-plan-authoring
├─ Changing a scenario that already exists → prompt-manager skill read scenario-work-ladder
├─ Creating a scenario that does not exist yet → prompt-manager skill read ecosystem-fit
├─ Deploying/publishing a scenario        → prompt-manager skill read deployment-coordinator
├─ Authoring plans/requirements/PRDs/tests/reports → prompt-manager skill read writing-standards
├─ Auditing the agent system (teams/members/PoRs) → prompt-manager skill read agent-system-audit
├─ (add new entries as patterns emerge)
└─ None of the above                      → proceed normally, no skill needed
```

Skills are lazy-loaded — only pay context cost when relevant. The full instructions live in prompt-manager, not here.

> **Discover before you hand-roll.** Beyond skills, prompt-manager also indexes executable **actions** (typed wrappers over a single Vrooli CLI command). Before writing deterministic or operational steps yourself, run `prompt-manager discover "<what you need>" --type all` — it returns both skills (judgment) and actions (execution), ranked purely by relevance (best-match, not curated planning packs — that is skill mode). See Critical Rule §6 (Recall → Discover → Capture) for the full reflex.

> **Two skill systems — don't confuse them.** The `prompt-manager` skills above are *internal* — instructions for an agent working on Vrooli right now. The top-level [`skills/`](skills/) folder is *publication source* for external Claude Skills (the open SKILL.md standard) that teach agents in *other* runtimes — Claude Code, Codex CLI, Cursor, etc. — how to use specific Vrooli capabilities standalone. Different audience, different content shape. Internal sessions should keep using `prompt-manager skill ...`; do not load the publication-source folder as if it were a runtime skills directory. See [`skills/README.md`](skills/README.md) for the full distinction.

## 🔧 Setup & Tooling

Setup flags (`--environment` development|production|minimal, `--resources` enabled|none|`<list>`)
and profiles: **docs/reference/environment-management.md**. Resources are enabled/disabled in
`.vrooli/service.json`.

Tools: `ast-grep` (syntax-aware search, prefer over `grep` for structural matching), `jq`/`yq`,
`gofumpt -w .` (Go formatting), `golangci-lint run` (Go linting).

## 🔖 Machine-Readable References

When reading docs, treat marked references like `path:docs/README.md` or `topic:bug-inbox/*` as typed references: the marker before `:` identifies the reference kind and is not part of the literal path/topic value. See [Machine-Readable References](docs/reference/machine-readable-references.md).

---

**For detailed documentation, development guidelines, and comprehensive examples, see [/docs/README.md](/docs/README.md)**
