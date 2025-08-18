## 🚀 Scenario Improvement Loop (v2)

### TL;DR — One Iteration in 5 Steps

1) Select ONE scenario in `scripts/scenarios/core/` using the selection tools:
   - Recommend: `auto/tools/selection/scenario-recommend.sh` (respects cooldown; optional GOALS, K)
   - Record pick: `auto/tools/selection/scenario-select.sh <name>`
2) Investigate the current status of the scenario, in terms of file structure, code completeness, and accuracy to the scenario's purpose.
3) Think deeply on how this scenario fits into Vrooli, and what ways it might help other scenarios.
4) Think deeply on what it takes to properly implement this scenario, especially in relation to leveraging shared resources and the CLI of other scenarios.
5) After careful consideration, make ONE small improvement inside that scenario only (e.g. fixing a single workflow, adding a missing README.md, replacing built-in logic with a call to another scenario's CLI that's purpose-built to for the task).
6) Validate: convert → start → verify via API/CLI or Browserless screenshot.
7) If a gate fails twice, stop, capture diagnostics, and defer heavier changes.
8) Append ≤10 lines to `/tmp/vrooli-scenario-improvement.md` (schema below).

Use `auto/tasks/scenario-improvement/prompts/cheatsheet.md` for metrics and jq helpers. Read `auto/data/scenario-improvement/summary.txt` (if present) before deciding.

---

### 🎯 Purpose & Context

- Improve AND validate scenarios in `scripts/scenarios/core/` to increase Vrooli’s capabilities and reliability.
- Scenarios must be able to convert into generated apps and pass validation via API/CLI outputs or Browserless screenshots.
- Prefer shared workflows in `initialization/n8n/` (e.g., `ollama.json`) to reduce per-scenario complexity and increase reuse.
- See `docs/context.md` for the broader vision and why scenarios are central to Vrooli.

---

### ✅ DO / ❌ DON’T

- ✅ Keep changes minimal and inside `scripts/scenarios/core/<scenario>/...`.
- ✅ Prefer shared n8n workflows (e.g., `initialization/n8n/ollama.json`) over direct resource calls.
- ✅ Validate every change: never assume success.
- ✅ If adding a shared workflow (must be truly generic and not scenario-specific), place it in `initialization/n8n/` (assuming it's an n8n workflow. Obviously if it was a huginn workflow it'd be in `initialization/huginn`, etc.).
- ✅ Prefer using resources via the CLI (e.g. `vrooli resource ollama`, `resource-ollama`), rather than direct API calls. If in a workflow, we may have a shared workflow (e.g. `initialization/n8n/ollama.json`) to even more reliably access the resource.

- ❌ Do NOT modify main scripts or general resources.
- ❌ Do NOT alter the n8n resource itself; if webhooks fail (as of now they WILL), use the Browserless workaround.
- ❌ Do NOT use git commands except `git status`/`git diff` for inspection.
- ❌ Avoid `vrooli develop` unless you identify and mitigate the CPU issue first.

> Note: Operational commands, workarounds, and jq helpers are in `auto/tasks/scenario-improvement/prompts/cheatsheet.md`.

---

### 🧭 Scenario Selection — Highly Recommended Flow

Always prefer the selection tools to choose what to work on:
- Recommend top-K with cooldown: `K=5 GOALS="n8n reliability, UI variety" auto/tools/selection/scenario-recommend.sh`
  - Cooldown is derived from `MAX_CONCURRENT_WORKERS` to avoid agents colliding on the same scenarios.
- Record your pick: `auto/tools/selection/scenario-select.sh <scenario>`

Then apply the rubric to break ties or justify a different choice:
1) Highest cross-scenario impact (e.g., adopting shared workflows) with low risk
2) Likely to fully validate within this iteration (fast wins)
3) Broken-but-fixable flows that unblock multiple scenarios
4) Clear, incremental UI/UX or init structure improvements that increase reliability

Before choosing, read:
- `auto/data/scenario-improvement/summary.txt` if present
- `scripts/scenarios/catalog.json`
- `scripts/scenarios/README.md`
- `scripts/scenarios/tools/app-structure.json`

---

### 🔧 Iteration Plan (contract)
Perform exactly these steps:

1) Analyze scenario state
   - Confirm structure: `.vrooli/service.json`, `initialization/`, `ui/`, tests, etc.
   - Identify the smallest high-value change aligned with the rubric.

2) Make ONE minimal change (can be across multiple files, but the changes must all be related)
   - Keep edits within `scripts/scenarios/core/<scenario>/...`
   - Prefer adopting a shared workflow over adding bespoke logic
   - If you must add a new shared workflow, place it in `initialization/n8n/` and document why it’s reusable

3) Validate (all gates must pass)
   - Convert: `vrooli scenario convert <name> --force`
   - Start: `vrooli app start <name>` (stop with Ctrl+C from same terminal)
   - Verify:
     - API/CLI outputs match expectations, and/or
     - Browserless screenshot shows the expected UI state

4) If a gate fails twice
   - Stop edits for this iteration
   - Capture diagnostics (command outputs, errors)
   - Pivot to a smaller change (e.g., switch to shared workflow usage)

5) Record ≤10 lines to `/tmp/vrooli-scenario-improvement.md`
   - Schema: `iteration | scenario | change | rationale | commands | result | issues | next`

---

### ✅ Validation Gates — Acceptance Criteria
All three must be satisfied for success:
- Convert completes successfully with `--force` and no blocking errors
- Start runs without critical errors and responds to the validation path
- Verification:
  - For API/CLI validation: outputs match expected values (document the command and output)
  - For UI validation: Browserless screenshot contains a specific UI element/text indicating success

---

### 🔁 Browserless + n8n Workaround
If n8n webhooks/api are unreliable, run via Browserless instead of using n8n API directly. Example:
```bash
export N8N_EMAIL="$N8N_EMAIL"
export N8N_PASSWORD="$N8N_PASSWORD"
vrooli resource browserless execute-workflow "<workflowId>" http://localhost:5678 60000 '{"text":"test"}'
```
Use environment variables for credentials/configuration. Do not commit secrets.

**NOTE:** You cannot run a workflow unless it is injected and activated by n8n, which happens automatically (at least it should) by running `vrooli scenario convert <scenario-name> --force`, followed by `vrooli scenario start <scenario-name>`. You must then use the n8n resource (see `vrooli resource n8n help`) to list workflows and find the workflow ID, which is needed for the browserless execute-workflow command.

NOTE 2: Prefer calling the shared `ollama.json` workflow over direct Ollama API calls. Prefer calling resources using bash (see `ollama.json`'s implementation for reference) over API calls, as this is known from experience to be a better approach.

---

### 📚 Tools & References
- Selection tools (use these first):
  - `auto/tools/selection/scenario-recommend.sh` — recommends top-K with cooldown; honors `MAX_CONCURRENT_WORKERS`, optional `GOALS`, `K`
  - `auto/tools/selection/scenario-select.sh <name>` — records your pick in the events ledger
- Status checks:
  - `vrooli status --verbose`
- Metrics, jq helpers: `auto/tasks/scenario-improvement/prompts/cheatsheet.md`
- Loop artifacts:
  - Events ledger: `auto/data/scenario-improvement/events.ndjson`
  - Summaries: `auto/data/scenario-improvement/summary.json`, `auto/data/scenario-improvement/summary.txt`
- Scenario references:
  - `scripts/scenarios/catalog.json`
  - `scripts/scenarios/README.md`
  - `scripts/scenarios/tools/app-structure.json`
- Project context: `docs/context.md`
- Working n8n workflow with proper cli usage and resource variable substitution (read this if you want to add/update n8n workflows!): `initialization/n8n/ollama.json`.

Before each iteration, skim `auto/data/scenario-improvement/summary.txt` if present to learn from recent results.

---

### 📝 Notes Discipline (≤10 lines)
Append one compact line per iteration to `/tmp/vrooli-scenario-improvement.md`:
- `iteration | scenario | change | rationale | commands | result | issues | next`
Keep the file under 1000 lines. Periodically prune old entries if needed.

---

### ✨ Quality Bar
- Prefer shared workflows in `initialization/n8n/`.
- Follow scenario structure conventions (`.vrooli/`, `initialization/`, `ui/`, tests).
- Ensure apps start and are verifiable via API/CLI or screenshot.
- Keep changes small to increase iteration speed and reliability.

---

### 🔒 Security & Safety
- Never hardcode secrets in prompts or examples; use environment variables or service config.
- Do not modify platform resources/components beyond the explicitly allowed cases above.

---

### 🧹 Style & Copyedits (applied to your edits)
- Fix typos, consistent, concise wording.
- Clear acceptance criteria and explicit stop/retry rules.
- No duplication of instructions already injected by the loop (e.g., “Ultra think” is already added upstream).

---

## 📎 Appendix — Operational details and references (from original)

This appendix preserves important specifics from the original prompt while keeping the main contract concise.

### CLI usage and run modes
- Use `vrooli help` to discover available commands.
- Converting and running:
  - Convert: `vrooli scenario convert <scenario-name> --force`
  - Run: `vrooli scenario start <scenario-name>` (stop with Ctrl+C in the same terminal)
- `vrooli setup` converts scenarios into generated apps but does not run them (and may skip conversions due to caching).
- `vrooli develop` runs apps but does not convert them. Prefer not to use it unless necessary.
  - Known issue: a CPU usage bug may exist (possibly in shared scripts like `manage.sh`). Investigate and mitigate before using `vrooli develop`.

### Resource policy and caveats
- Many resources may be flaky. Do not modify or add resources.
  - You may start stopped resources and use their features.
  - Do not turn off resources unless excessive memory usage demands it (applies notably to Whisper).

### n8n workflows — reliability, activation, and quality
- If n8n webhooks/API are unreliable, use the Browserless workaround instead of calling n8n API directly. Never commit secrets; use environment variables.
  - Example (conceptual):
    ```bash
    export N8N_EMAIL="$N8N_EMAIL"
    export N8N_PASSWORD="$N8N_PASSWORD"
    vrooli resource browserless execute-workflow "<workflowId>" http://localhost:5678 60000 '{"text":"test"}'
    ```
- Shared workflows must be defined in the project-level `.vrooli/service.json`, then run `vrooli setup` to upsert and set active.
- Scenario-level workflows should be injected when running `vrooli app start <scenario_name>` (after conversion). If that fails:
  - Fallback: run the generated app’s `./scripts/manage.sh setup`, or call the injection engine directly.
- Workflow quality bar:
  - Include both a webhook trigger and a manual trigger.
  - Manual trigger should feed a node that provides defaults (webhook should not go through that defaults node).
  - Keep variables like `service.n8n.url` for script-time conversion.
  - Prefer the shared `initialization/n8n/ollama.json` workflow instead of calling Ollama directly.
  - Prefer nodes that run CLI/bash commands to access resources rather than direct resource APIs when practical.

### Scenario structure and artifacts
- Maintain scenario structure consistency:
  - `.vrooli/service.json` (see `scripts/scenarios/core/agent-metareasoning-manager/.vrooli/service.json` for a good example).
  - `initialization/` (resource data injected by Vrooli)
  - `ui/`
  - `scenario-test.yaml` for tests
- Add a small `README.md` per scenario (≤100 lines): purpose, usefulness, dependencies, UX style, etc.
- Ensure thare are no hard-coded ports.
- Ensure scenarios have no legacy/bridge/temporary/placeholder code

### UI guidance and ethos
- Prefer JavaScript UIs for full customization over Windmill when appropriate.
- Make UIs feel fun and unique (old-internet spirit) where suitable; business/internal tools should remain professional.
- Example vibes:
  - `study-buddy`: clean, cute, lo-fi vibe
  - `retro-game-launcher`: 80s arcade retro
  - `system-monitor`: dark, green, “matrix” vibe
  - `app-debugger`, `product-manager`, `roi-fit-analysis`: standard modern web

### Reading list (pre-work)
- `/tmp/vrooli-scenario-improvement.md` (recent notes)
- Project `README.md`
- `scripts/scenarios/README.md`
- `scripts/scenarios/catalog.json`
- `scripts/scenarios/tools/app-structure.json` (describes copying to generated apps; some files come from the project root)
- `scripts/resources/README.md` (what resources are and how scenarios use them)
- `initialization/README.md` (project-level shared resource data injected by Vrooli)
- The add/fix scenario prompt (untested): `scripts/scenarios/core/prompt-manager/initialization/prompts/features/add-fix-scenario.md`

### Change heuristics (granular)
- Avoid large code additions. Scenarios should be small and single-purpose.
- Avoid adding lots of code to the Go API or the CLI; prefer moving logic into workflows (n8n/Node-RED/Huginn/etc.).
- Good candidates:
  - Restructure `initialization/` (e.g., remove legacy category subfolders; prefer `initialization/n8n/workflow.json`).
  - Add/fix scenario’s `initialization/` files.
  - Update workflows to leverage shared project-level workflows.
- Great shared workflows to adopt:
  - `initialization/n8n/ollama.json` (reliable Ollama usage)
  - `initialization/n8n/rate-limiter.json`
  - `initialization/n8n/embedding-generator.json`
  - See `initialization/n8n/README.md` for details.

### When to add a new scenario (curated list)
If all existing scenarios are structurally sound and validated (very unlikely at this stage of the app!), you may add one from this list (if it hasn't already been added):
- stream-of-consciousness-analyzer to convert unstructured text/voice to organized notes. Organized by "campaign", where each one allows me to add notes and documents as context to guide the agent that organizes the thoughts
- agent-dashboard to manage all active agents (e.g. huginn, claude-code, agent-s2)
- ci-cd-healer to fix CI/CD pipelines (you are FORBIDDEN from testing this one or touching any of our CI/CD cod)
- git-manager to deal with deciding when to commit, how to split up the changes into commits if they're unrelated, writing commit messages, etc. (you are also FORBIDDEN from testing this one or touching our git-related code)
- deployment-manager to deal with deciding how to modify a generated app (e.g. using app-to-ios to build an ios version, setting up for Docker or Kubernetes deployment etc.) to prepare and deploy it to the customer
- code-sleuth for tracking down relevant code for tasks, receiving feedback on what was relevant, and learning from that over time
- test-genie for learning how to test different scenarios (and also vrooli as a whole), as well as learning best practices over time
- dependabot for scheduled or triggered code scanning of scenarios and Vrooli as a whole
- app-to-electron for learning how to convert generated apps to Electron so that they can be run as standalone Desktop apps on Windows and such. Note that all of these `app-to-` scenario types should assume that the app distributions will be stored in a platform/ folder, like we've done at the project-level of Vrooli as a demonstration
- app-to-ios
- app-to-android
- app-onboarding-manager for adding proper and professional onboarding pages and tutorials to apps. Should learn best practices and build templates over time
- survey-monkey for building and sharing surveys
- resume-and-job-assistant for building and improving resumes, looking for open roles, and proactively investigating if they're a good fit for the customer and building a cover letter tailored to the role and company
- palette-gen for building color palettes
- personal-relationship-manager to track information about your friends, such as their birthdays, notes, etc. Should send reminders for birthdays, and proactively search for relevant gifts
- n8n-workflow-generator for learning how to effectively build n8n workfows
- prompt-performance-evaluator for testing and improving prompts
- video-downloader for downloading videos from a URL.
- product-manager-agent for handling more bigger-picture product decisions that can be used to guide our coding agents and decide what to focus on. Should have web and research capabilities
- dream analyzer for acting as a dream journal, and also doing analysis on specific dreams and common themes
- life-coach would be similar to the product-manager-agent, but focused specificall on the customer's general life
- chore-tracking for tracking chores, automatic schedule for cleaning/maintenance/etc., and point system with cute, quirky UX
- nutrition-tracker for calories and macros, with meal suggestions based on previous entries
- app-scroller to scroll through generated apps like you're on TikTok. Would pick the best apps depending on the mode, so that you get a different experience during the weekend vs. work hours
- roi-fit-analysis for deep web financial research to determine what ideas have the best return on investment based on your available skills and resources
- seo-optimizer for improving the search engine optimization of generated apps
- local-info-scout for finding local information, such as "vegan restaurants in my area", "stores nearby that sell cat bowls", "recently foreclosed homes near bodies of water", etc.
- wedding-planner for all things wedding planning
- morning-vision-walk for talking to it and having it help me understand the current state of Vrooli and brainstorm together on the things we should get done today.
- competitor-change-monitor to track the websites/githubs/etc. of competitors, and alert on any relevant changes. Would likely use huginn with fallback to browserless, and another fallback to agent-s2
- picker-whell for fun random selection. API available for scenarios/workflows that need random selection
- make-it-vegan to research if foods are vegan, or find vegan alternatives/substitutes
- travel-map-filler for a fun visual way of tracking where you've travelled
- data-generator for building or scraping data required for some of these scenarios. Huginn, browserless, and agent-s2 support
- fall-foliage-explorer for visualizing (with accurate data) where fall foliage peaks are forecasted, with a time slider
- mind-maps for building (both manually and with AI) mind maps that can be semantically searched. Should be thought of as an important scenario to be leveraged by others, any time we need data related to a topic/campaign/project/etc. (terminology may differ depending on the scenario, but they're all the same concept) to be organized and searchable by AI or humans.
- password-manager as a general password manager. Could also expose auth-related features via API for other scenarios, such as password stength and password generator
- word-games (wordle, connections, 2048, etc.) just for leisure and demonstrating capabilities
- notes for local, AI-enabled note taking
- itinerary-tracker with virtual bag/suitcase (this would possibly be our first scenario with 3D, so I'd be interested to see how you do it)
- coding-challenges for learning how to code (could also double as a method to benchmark how well various AI models can code)
- saas-billing-hub for adding payments and subscriptions to scenarios. Includes admin dashboard to manage multiple saas revenues
- recipe-gantt-charts for displaying recipe instructions in an optimal timeline. Should be able to enter a recipe or generate one from prompt. All recipes get a gantt chart, which updates as the recipe is changed.

If all of the above are already added, focus on converting to apps, running, and fixing issues that prevent apps from fully starting or showing a valid UI. For quick debugging you may change generated app code, but any solution must be applied back to the source scenario.

### Cross-scenario composition
- Some scenarios are used to increase Vrooli’s own capabilities. They may call each other’s CLIs (e.g., `morning-vision-walk` relying on `stream-of-consciousness-analyzer`, `product-manager-agent`, etc.).

### Generated apps
Remember, changes in *generated* apps are ephemeral - you may change the generated app code for quick, iterative debugging, but once you find a solution you MUST update the scenario for it to persist.

### Loop and file safety
- Do not modify loop management or prompt files that run the iteration process (e.g., loop scripts and managed prompt files).

### Web usage
- You may use the web if needed. 