# Intent recall log

Measured with `search-hub intents run` (top 5, `--type library,skill,command`).
`device-control` cases require its **skill** to surface; `any leaf` counts any
skill, program or command of the scenario.

## Origin (2026-11-09)

Operator ask: "Show me a screenshot of the home page of my TV". The Codex session
(run cf7b7b66-aa90-498d-964e-8eaeb3d2249a) searched, saw no device-control skill,
and answered that it could not; 28 minutes later the operator said "Using device
control you can" and it succeeded in 2.5 minutes. Operator stance: if the full
ecosystem cannot be used efficiently without being pushed, the scenarios are
disconnected applications rather than one self-improving ecosystem.

Search-hub's share of the charter (the device-control scenario owner owns the
rest, under `scenarios/device-control/`):

1. Make discovery a measured number (this guide, `intents`).
2. `--group <scenario>` must work or fail loudly.
3. Explain the "classifier unavailable" status and the zero-hit skill provider.
4. Find the owner of "CLI rebuilt from current sources; restarting".
5. A generic gap report listing scenarios with the same defect.

## Baseline before any description change (2026-11-09, search-hub corpus of 50 cases)

| Slice | Qualifying leaf in top 5 | Any leaf of the owner in top 5 |
|---|---|---|
| device-control, 12 charter queries (skill required) | **0/12** | not separately recorded |
| device-control, all 16 (skill required) | **0/16** | 15/16 on a clean run (11/16 on an earlier run; the spread is cli-health command results appearing or not, see Findings) |
| other scenarios, 34 cases (any leaf type) | 26/34 | 26/34 |
| everything | 26/50 (52%) | 41/50 |

Cross-scenario misses (every one with `any-leaf-rank=0`, so the scenario is
invisible to a cold agent): ui-health, deployment-manager, secrets-manager,
system-monitor, vrooli-onboarding, money-ledger, music-tools, content-desk. The
set shifts by one or two between runs because ranking depends on live provider
state; treat a case as a defect only when it misses on repeated runs.

Coverage: 14 of 94 scenarios exposing leaves declared an intent before this
work; 80 did not (`search-hub intents coverage`, largest guidance surface first:
swarm-manager, web-console, program-runtime, vrooli-bridge, scenario-to-cloud,
audio-tools, ...).

## After device-control rewrote its skill description (2026-11-09 ~10:00)

| Slice | Before | After |
|---|---|---|
| device-control, 16 cases, skill required, top 5 | 0/16 | **15/16** |
| device-control, 12 charter queries | 0/12 | **11/12** (miss: "press home on the TV") |
| `prompt-manager.skill.primary` provider-direct suite (4 new device cases) | 0/4 met | 4/4 met, rank 1 each |

The skill now ranks 2nd to 5th in the mixed `library,skill,command` list, behind
`command` and program hits. That clears the top-5 bar with little margin
(`google-tv-screenshot` is rank 5). Raise the margin by giving the new programs
(`device-control.observe`, `device-control.open-app`) and cli-health command
summaries the same operator vocabulary, then re-run `search-hub intents run
--scenario device-control`.

## Final (device-control rev 60 skill, observe/open-app programs, rewritten command summaries)

Full corpus: 41/50 (82%), up from 26/50 at baseline; device-control 15/16 skill
(from 0/16), 16/16 any leaf. The only device-control miss is "press home on the
TV". Cross-scenario cases now 26/34; the 8 invisible scenarios listed above are
unchanged and belong to their owners.

## Cross-scenario fixes and corpus growth (2026-11-09, evening)

| Step | Corpus recall (qualifying leaf, top 5) |
|---|---|
| Baseline, 50 cases | 26/50 (52%) |
| After device-control rewrite (dc-owner) | 41/50 (82%) |
| After fixing the eight named misses (50 cases) | 44/50, with 34/34 non-device passing (device-control regressed to 10/16 from the dc-owner's own new programs; fixed by the dc-owner afterwards) |
| Corpus grown to 148 cases (batch 1, 98 new): before fixes | 114/148 (77%) |
| After new skills and description rewrites for the 25 missing scenarios | 146/147 (99%) |
| Corpus 163 cases (batch 2, 8 more scenarios) after fixes | 162/163 (99%); only miss is device-control "press home on the TV", left on purpose |

Coverage: 80 of 94 scenarios had no intent; now 3 do not (the hello-* fixtures).

What was missing was almost always a skill: 24 scenarios had a CLI and no skill
(or a skill with operations-only wording), so the dense skill search had nothing
to match. New skills (`skills/<name>/SKILL.md`, declared in
`.vrooli/service.json` `skills.usage`): system-monitor, money-ledger, music-tools,
network-manager, compute-manager, vrooli-events, scenario-to-ios, switchboard,
signal-inbox, persona, scenario-authenticator, document-manager, nutrition-planner,
music-library, asset-studio, backdrop-studio, api-health, workflow-health,
structure-health, code-facts, go-code-graph, typescript-code-graph, treasury,
device-studio, development-toolchain-validator, meta-optimization-manager.
Descriptions and tags rewritten in user vocabulary: ui-health, deployment-manager,
secrets-manager, vrooli-onboarding, content-desk, workspace-sandbox,
channel-manager, portal, performance-health. Declarations added for ui-health and
workspace-sandbox, whose skill-set phase failed on `role_missing:usage` before.

Lesson from the device-control dip (10/16, then back to 15/16): recall depends on
how many competing leaves one scenario puts in the top 5, not only on wording.
Five device-control leaves (skill, `observe`, `open-app`, `check-app`, commands)
crowded each other out. The dc-owner folded `observe` and `open-app` into
`device-control.check-app` and shortened command descriptions. The skill index
lags a skill or manifest change by a minute or two, so a reading taken in that
window shows the old state.

## Findings

- `prompt-manager.skill` returns zero rows with no error for any request below
  its 0.5 cosine floor; see the guide. That, not routing, is why the skill never
  surfaced.
- `federation status` printed "Classifier: unavailable" because the interactive
  LLM classifier was retired and the wire field is hard-wired false. Status now
  names the active strategy instead.
- `--group <scenario>` returned "Searched 0 corpora": `--group` matched only
  provider groups. It now falls back to scenario scope and explains itself.
- cli-health.commands stalls (4 s, then HTTP 500, then an open circuit) come from
  embedding requests queuing behind model work on the shared Ollama daemon. Only
  22 of its thousands of Search calls failed in three days, in isolated bursts
  (09:50:14-18: 14 in a row; 12:39:07-15: three), each exactly the router's 4 s
  per-provider timeout, with ~100 ms calls on both sides. Reproduced once: a
  direct embedding sent 1 s after a 28 s `qwen3.5:9b` generation (cold load)
  returned after 28.1 s; two later repeats with the model warm did not stall, so
  the trigger is model load, not steady load. The provider asked for MODE_AI
  (dense), which hard-failed with no lexical fallback. Fixed in the shared engine
  (`packages/ai-go/search`): the dense/hybrid leg now has a 2.5 s budget
  (`ServiceOptions.DenseBudget`) and, when it expires with a text leg configured,
  answers from the text leg labelled `Method: "text"`. The caller's own deadline
  still surfaces as an error. Not reproduced live after the fix (the daemon
  would not stall again); covered by unit tests with a blocking embedder.
- The "CLI rebuilt from current sources; restarting" line comes from
  `packages/cli-core/cliutil/stalechecker.go`. Every scenario CLI fingerprints
  the whole `packages/cli-core` tree, including `_test.go` files, so any test
  edit anywhere in cli-core staled every CLI in the fleet. Test files are now
  excluded from every freshness walk (`NonBuildInputSuffixes`). Remaining
  triggers are real: any non-test edit to cli-core or the CLI's own module.
