# AGENTS.md

You are an expert software engineer, visionary, and futurist. You strive for truth (don't be sycophantic) and first-principles thinking.

This file provides essential guidance to Claude Code (claude.ai/code) when working with this repository.

## ⚡ Critical Rules - READ FIRST
1. **Commands**: 
   - Run `vrooli help` to see available.
1. **Testing**: 
   - Use `vrooli scenario test <name>` (or `test-genie execute <name>`) to run scenario tests.
   - **The run is owned by the test-genie server, so it survives your command being cancelled.** Just run it. The run id + a re-attach command are printed up front, and a known-long run auto-backgrounds so your shell returns immediately.
   - **Do NOT poll with repeated "still waiting" checks. To wait, block ONCE with the quiet wait verb:**
     `test-genie runs wait --json <scenario> <run-id>` (also `vrooli scenario test wait <scenario> <run-id>`). It blocks server-side and returns exactly once with the verdict + the run's real exit code (0 passed, 1 failed/aborted, 124 if you pass `--timeout` and it elapses first). It does NOT stream — one call, one return. This is the verb the start banner and re-attach commands print; copy it verbatim.
     - **If you must bound the wait, use `--timeout=<seconds>`.** On timeout it returns `124`, the JSON snapshot still carries `recommended_next_check_seconds`, and stderr prints the exact re-invoke line. **Re-call only after that many seconds — never poll faster, never re-run immediately.**
     - `test-genie runs follow <scenario> <run-id>` is the **human** live-watch verb (a continuous, heartbeating stream). Do not use it to "wait" as an agent — a backgrounded stream re-wakes you on every heartbeat. Use `runs wait --json`.
     - Cancel ≠ abort — to actually stop a run use `vrooli scenario test abort <scenario> <run-id>`.
   - **One run per scenario at a time.** The test-genie server allows at most one in-progress run per scenario (different scenarios run concurrently). An identical re-request coalesces onto the running run (no second suite); a *different* request for a busy scenario is rejected with the in-flight run id + `runs wait --json`/`runs abort` guidance — wait or abort, don't retry-spam.
   - **Waiting on several runs at once?** Use one `test-genie runs wait-all --run <scenario>:<run-id> --run …` call (repeatable; add `--json`). It blocks until every named run is terminal and returns one aggregate exit code (0 all passed, 1 any failed, 124 any still in-flight at `--timeout`, 2 any not-comparable) — so two parallel suites/diffs resolve in a single call instead of two backgrounded streams.
   - **Baseline diff is durable too — don't poll it.** `git-control-tower baseline diff --scenario S --name N` returns immediately with a run id + re-attach command (it reuses a clean-tree run when one exists, so it usually doesn't even re-run the suite). Resolve the verdict with `git-control-tower baseline diff status --scenario S --name N --run <run-id>` (exit `0` clean, `1` regression, `2` not-comparable, `3` not-ready), or add `--wait` to block server-side and print it inline.
2. **Files**: Always prefer editing existing files over creating new ones
3. **Bug reporting & work logging**:
   - If you spot a defect outside your current scope, load the writer instructions with `prompt-manager skill read report-bug`, then file to scenario-qa with `prompt-manager team knowledge-add scenario-qa --topic="bug-inbox/<signal-type>/<slug>" --content "<front-matter + body>"`. `report-bug` is a prompt-manager skill, not a shell executable. If the knowledge writer is unavailable, fall back to `swarm-manager captures create --text "<bug report + attempted command>"` and say the bug writer was unavailable.
   - If you complete work worth preserving (non-trivial fix, feature, refactor, investigation), write a record via `swarm-manager records create`. Records are the recursive-learning loop's write side; without them, future agents lose your work.
4. **Recall → Discover → Use → Capture** (continuous reflex, not an end-of-task checklist):
   - **Recall prior work first.** Before non-trivial work, search what the system has already learned: `search-hub query "<one-sentence intent>" --type record,skill,doc` (widen to `record,backlog,initiative,skill,doc` when planning). Read the top hits — a `record` hit carries *how* a prior agent solved it (its trigger + approach), not just a link. Three exits: nothing relevant → proceed; related prior work → build on it (cite the record/skill); a near-duplicate of an already-solved problem → stop and reconcile before redoing it. If search-hub is unavailable, fall back to `swarm-manager records search "<intent>"`. This is the *read* side of the recursive-learning loop; the record you write at the end (Rule 3) is the *write* side — recall makes those writes pay off.
   - **Discover first.** Before hand-rolling deterministic or operational work, run `prompt-manager discover "<what you need>" --type all` to find an existing skill or action. `--type all` is best-match relevance (skills and actions ranked purely by score, no curated topic packs), so phrase the query as the operation / what you need. (A miss is logged automatically — no action needed from you.)
   - **Capture as you go.** When you accomplish something reusable:
     - One Vrooli CLI command cleanly does it and no action exists → `prompt-manager action create --name "…" --command '<argv with {{placeholders}}>'` (previews by default; add `--apply` to register). Creating an action is free — no decision required.
     - It took several commands, only partly worked, or you improvised → `swarm-manager captures create --text "<intent / what you did / the friction>"`. Don't stitch multiple actions together; the capture becomes an enhancement or capability-gap the system triages into a single clean command later.
     - Something is broken → use the `report-bug` skill workflow above; do not run `report-bug` as a shell command.
5. **Dependencies**: All third-party dependency work — finding, approving, changing, and installing packages — flows through **Scenario Dependency Analyzer (SDA)**, the dependency-intelligence authority. Never hand-edit `.vrooli/dependencies/approved-dependencies.json` and never run a raw package manager (`pnpm add`, `go get`, `npm install`, `pip install`).
   - **Find** a package: `scenario-dependency-analyzer deps approved search "<purpose>" [--ecosystem npm] [--framework react] [--surface ui]` (AI-ranked approved/denied records + ranges + security state). Also reachable via `search-hub query "<purpose>" --type dependency`.
   - **Install** into a surface: `scenario-dependency-analyzer deps install <ecosystem>/<package>[@ver] --scenario <name> --surface <ui|api|cli>` (resolves the package manager + manifest, enforces governance, re-scans for CVEs; dry-run by default, add `--apply`).
   - **Approve / change / deny**: `scenario-dependency-analyzer deps approved {approve-observed,widen-range,deny-vulnerable,explain,list}` (validated, dry-run-by-default, captures evidence + security scan).
   - These commands never need raw-install permission and are the only sanctioned path in autonomous (no-human) runs.
6. **Managing Scenarios**:
   - **ALWAYS use**: Scenario Makefiles for comprehensive management: `make start`, `make test`, `make logs`, `make stop`
   - **Alternative**: `vrooli scenario start <name>` for direct CLI management
   - **NEVER use**: Direct execution like `./api/scenario-api` or `cd scenario && ./lib/develop.sh`
   - The lifecycle system ensures proper process naming, port allocation, and logging
   - Direct execution bypasses critical infrastructure and causes detection issues

## 🎯 Understanding Vrooli's True Nature

### Key Definitions
- **Resources**: Core local services (AI/ML like claude-code, ollama; storage like postgres, redis, qdrant; development helpers like judge0, browserless, vault) that scenarios can compose.
- **Scenarios**: Full applications or microservices - with APIs, CLIs, and UIs - that combine resources and other scenarios to deliver reusable business capabilities.

**CRITICAL CONTEXT:** Vrooli is not just an automation platform - it's a **self-improving intelligence system** where:

### The Core Vision
- **Shared Local Resources:** Apps share local resources like Ollama, Redis, Qdrant, and PostgreSQL so they can work together and build off each other.
- **Scenarios Become Capabilities:** Every app (which is generated from a scenario) built becomes a permanent tool the system can use forever
- **Recursive Improvement:** Agents build tools → Tools make agents smarter → Smarter agents build better tools → ∞
- **Compound Intelligence:** The system literally cannot forget how to solve problems, only get better at solving them
- **Scenario-Based Business Model**: Scenarios target measurable value; deliverables can deploy directly, ship as SaaS, serve enterprise installs, or simply act as internal tools or microservices for other scenarios to leverage. Each scenario we complete should increase Vrooli's capabilities and/or be a new monetizable service

### The Evolution That Changed Everything
- **Phase 1 (Past):** Web platform where agents could only interact through APIs (limited but proved the concept)
- **Phase 2 (Current):** Physical server with local resource access - agents can now build complete applications by building off of existing resources and scenarios
- **Phase 3 (Future):** Specialized servers for engineering, science, finance. Hardware line where businesses and households can run their own specialized Vrooli server

### Understanding Scenarios
Scenarios are NOT just test cases or demos. They serve triple duty:
1. **Products:** Generate real revenue when deployed
2. **Validation:** Serve as implementation references for building future scenarios

3. **Capabilities:** Become new tools that enhance Vrooli itself or solve future problems

When working with scenarios, remember: **You're building businesses and expanding intelligence.**

### Deployment Vision
- Current deployments run via the Tier 1 local stack (full Vrooli installation + app-monitor Cloudflare tunnel).
- Future tiers (desktop, mobile, SaaS, enterprise) are documented in the [Deployment Hub](docs/deployment/README.md); consult it whenever considering packaging or delivery tasks.

### Working with Resources
Local resources (Ollama, PostgreSQL, etc.) aren't just "integrations" - they're the building blocks of emergent capability:
- Each resource multiplies what agents can accomplish
- Agents discover novel combinations we haven't imagined
- Resources enable the shift from "calling APIs" to "building the APIs"

### The Recursive Learning Loop in Practice
1. Agent solves problem using available resources
2. Solution gets crystallized as reusable scenario
3. Future agents use that solution as a building block
4. More complex problems become solvable
5. Each iteration makes ALL future iterations more powerful

**Remember:** Every line of code you write, every routine you create, every scenario you build - it all becomes permanent intelligence that the system uses to improve itself forever.

## 🚀 Quick Start Commands
```bash
# Setup project (includes CLI installation and system configuration)
# NOTE: First run requires sudo for kernel parameter configuration when using certain resources
make setup

# Start development environment
vrooli develop

# Run tests
vrooli scenario test <name>  # Run scenario test suite

# Manage scenarios (PREFERRED method)
cd scenarios/<scenario-name> && make start   # ✅ BEST - comprehensive management
cd scenarios/<scenario-name> && make test    # ✅ Run scenario tests
cd scenarios/<scenario-name> && make logs    # ✅ View scenario logs
cd scenarios/<scenario-name> && make stop    # ✅ Stop scenario

# Alternative: Direct CLI management
vrooli scenario start <scenario-name>        # ✅ ALTERNATIVE - CLI management

# NEVER: Direct execution bypasses lifecycle
# NEVER: ./scenarios/name/api/binary         # ❌ WRONG - bypasses lifecycle
# NEVER: nohup ./api/scenario-api &          # ❌ WRONG - no process tracking
# NEVER: cd scenario && ./lib/develop.sh     # ❌ WRONG - old pattern
```

> **Note**: When writing tests, make sure you're writing them to test against the DESIRED/EXPECTED behavior, not the actual implementation. This is important for the test to be useful and not just a checkmark.

## ❌ Common Pitfalls
- DON'T skip reading memory files at session start
- DON'T use mass-update scripts or automated tools to modify multiple files - check and update each file individually
- DON'T start scenarios with direct execution (`./api/scenario-api`, `nohup ./api/binary &`, etc.)
- DON'T bypass the lifecycle system - it manages process naming, ports, and health checks

## 🔍 Available Tools
- **ast-grep (sg)**: For syntax-aware code search - default to `ast-grep --lang <language> --pattern '<pattern>'` over `grep` for structural matching
- **jq/yq**: For JSON/YAML processing
- **gofumpt**: Stricter Go formatting (superset of gofmt) - use `gofumpt -w .` to format Go code
- **golangci-lint**: Comprehensive Go linting - use `golangci-lint run` to check Go code quality and catch issues

## 🔖 Machine-Readable References

When reading docs, treat marked references like `path:docs/README.md` or `topic:bug-inbox/*` as typed references: the marker before `:` identifies the reference kind and is not part of the literal path/topic value. See [Machine-Readable References](docs/reference/machine-readable-references.md).

## 🧠 Situational Skill Loading

At conversation start, assess the user's intent and proactively load the relevant skill. Do not wait for the user to request it — recognize the pattern and act.

```
What is the user doing?
├─ Brainstorming/workshopping a new idea  → prompt-manager skill read idea-workshop
├─ Debugging a non-obvious issue          → prompt-manager skill read scientific-debugging
├─ Creating an implementation plan        → prompt-manager skill read plan-skill-discovery
├─ Building/repurposing a scenario        → prompt-manager skill read ecosystem-fit
├─ Deploying/publishing a scenario        → prompt-manager skill read deployment-coordinator
├─ (add new entries as patterns emerge)
└─ None of the above                      → proceed normally, no skill needed
```

Skills are lazy-loaded — only pay context cost when relevant. The full instructions live in prompt-manager, not here.

> **Discover before you hand-roll.** Beyond skills, prompt-manager also indexes executable **actions** (typed wrappers over a single Vrooli CLI command). Before writing deterministic or operational steps yourself, run `prompt-manager discover "<what you need>" --type all` — it returns both skills (judgment) and actions (execution), ranked purely by relevance (best-match, not curated planning packs — that is skill mode). See Critical Rule §4 (Discover → Use → Capture) for the full reflex.

> **Two skill systems — don't confuse them.** The `prompt-manager` skills above are *internal* — instructions for an agent working on Vrooli right now. The top-level [`skills/`](skills/) folder is *publication source* for external Claude Skills (the open SKILL.md standard) that teach agents in *other* runtimes — Claude Code, Codex CLI, Cursor, etc. — how to use specific Vrooli capabilities standalone. Different audience, different content shape. Internal sessions should keep using `prompt-manager skill ...`; do not load the publication-source folder as if it were a runtime skills directory. See [`skills/README.md`](skills/README.md) for the full distinction.

## 🔧 Setup Configuration

**Environment Profiles** (`--environment`):
- `development` (default): Full setup with all dev tools (bats, shellcheck, ast-grep, Go dev tools, Helm, etc.)
- `production`: Production runtimes only, skips dev tools - ideal for VPS deployments
- `minimal`: Only Docker + essential system deps - fastest possible setup

**Resource Installation** (`--resources`):
- `enabled` (default): Install resources marked as enabled in `.vrooli/service.json`
- `none`: Skip all resource installation
- `<list>`: Install only specified resources (comma-separated, e.g., `postgres,redis`)

**Examples**:
```bash
make setup                                                   # Full dev setup
vrooli setup --environment production          # Production (no dev tools)
vrooli setup --environment minimal --resources none  # Fastest possible
vrooli setup --resources postgres,redis        # Only specific resources
```

**Resource Management**:
- Enable/disable resources by editing `.vrooli/service.json`
- Resources marked as enabled will be installed on next setup run

## ⏱️ Timeout Guidelines for Long-Running Commands
**Remember to set appropriate timeouts when running:**
- Test suites: Can take 15+ minutes in worst case scenarios (better to be safe than sorry)
- Type checking full packages: Can take 15+ minutes
- Building/compiling: Can take 10+ minutes (UI build alone takes 5-10 minutes due to 4444+ modules)
- Database migrations: Can take 3+ minutes
- Docker builds: Can take 20+ minutes
- UI build performance issue: vite build processes 4400+ modules, causing 5-10 minute build times

The default timeout is 2 minutes, which is often insufficient for these operations.


---

**For detailed documentation, development guidelines, and comprehensive examples, see [/docs/README.md](/docs/README.md)**
