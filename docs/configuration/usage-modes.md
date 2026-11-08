# Usage Modes

A usage mode is an expiring operator policy that lowers what **new** coding-agent work runs on, so quota lasts longer. It can also lower effort and set a context limit. Active runs do not change unless the operator explicitly asks.

## Behaviour

| Rule | Detail |
|---|---|
| Duration | Until the 5-hour reset, until the weekly reset, for a set time, or until turned off |
| Scope | Global, runner, role or profile. The most specific matching mode wins |
| Explicit choices | An explicit model always wins: `--model`, `-m`, or a run request's `model` |
| Resumed sessions | Keep their current settings. Switching a live session forces a full context re-read |
| Exempt roles | `code.fallback`, `code.review`, `judgment.supervision`, `code.supervision` and acceptance roles keep their model unless the mode lists them |
| Lower only | A mode may only map a model to a lower-cost tier in the same billing class. Restricted, excluded and metered models are rejected. A map entry that does not go down the tier order, and a `--model` override that is not provably below every model it replaces, are rejected too. A mode never raises a model a harness is already configured to use (see below) |
| Fail open | A missing, corrupt or expired file means no mode. Reading it can never block or fail a launch |

## Policy file

Location: `~/.vrooli/state/vrooli/usage-mode/mode.json`, resolved through `VROOLI_HOME` (the `~/.vrooli` directory) when set. Writers replace it atomically, and every writer (Agent Manager, Web Console) changes it only through `usagemode.Update`, which holds a file lock across read, change and write so concurrent writers cannot lose each other's change. Fields a writer's struct version does not know (for example `learn`) are kept on rewrite. Readers ignore a file that is not a regular file or is over 1 MiB, so a FIFO or huge file cannot block a launch. Readers import `github.com/vrooli/cli-core/usagemode` and read the file directly, so Web Console and the launcher need no Agent Manager connection. Expiry is checked on read, so nothing has to clean up.

```json
{
  "schema_version": 1,
  "modes": [
    {
      "id": "um-20261030-a1b2c3",
      "label": "Stretch until weekly reset",
      "kind": "save",
      "scope": { "runner": "codex", "role": "", "profile": "" },
      "started_at": "2026-10-30T18:00:00Z",
      "started_by": "operator@example.com",
      "source": "agent-manager",
      "until": { "kind": "window_reset", "provider": "openai", "window_minutes": 10080 },
      "expires_at": "2026-11-04T06:57:00Z",
      "exempt_roles": ["code.fallback", "code.review"],
      "runners": {
        "codex": {
          "model_map": { "gpt-6.1-terra": "gpt-6-luna" },
          "session_model": "gpt-6-luna",
          "effort": "",
          "context_limit_tokens": 0
        }
      }
    }
  ]
}
```

`until.kind` is `window_reset`, `time` or `off`. `off` modes have no `expires_at`. A `window_reset` mode stores the known reset time as `expires_at`. Agent Manager ends it sooner when a window resets early. If Agent Manager is down, the mode lasts until the stored time.

`runners` holds concrete values computed when the mode starts, so readers never parse resource policies.

## Tier mapping

Each coding-agent resource declares its tiers in `model-policy.json` under `usage_tiers`, because resources own concrete models:

```json
"usage_tiers": {
  "model_map": { "gpt-6.1-terra": "gpt-6-luna" },
  "session_model": "gpt-6-luna",
  "exempt_roles": ["code.fallback", "code.review", "judgment.supervision"]
}
```

The tier order is derived from `model_map` with no extra field: a model the map lowers sits one above its target, and a target that is never lowered is the floor (`fable` > `opus` > `sonnet`). A model that appears nowhere in the map, such as a Haiku, has no provable rank, so it is never accepted as an override or challenger.

`resource-<runner> policy refresh` updates a catalog from the installed runner's own evidence: Claude Code's built-in model catalog, or `codex debug models`. It moves aliases, role canonical models, role fallbacks and concrete tier entries to what the aliases now select, and records the source and date. It never changes which alias a role uses. It writes nothing and fails when the runner can't prove a model a role names, and for runners with no evidence reader, whose policies are edited by hand.

## How each path applies a mode

| Path | How | Reach |
|---|---|---|
| Harness shim, `vrooli agent launch` | Launcher adds settings when it starts the harness, and sets `VROOLI_USAGE_MODE=<id>` | Terminal and Web Console sessions |
| Agent Manager runs | Overlay after role resolution, recorded as `usage_mode:<id>` in the run's selection reason | Headless and interactive runs with a role |
| Web Console managed Codex | At create only: the descriptor model is filled when empty (under the same configured-default rule) and stored, and the mode's context limit is added as `model_auto_compact_token_limit`. Recovery after a restart replays the stored descriptor and never applies the current mode; the recovered thread uses the harness's own compaction limit | New managed Codex sessions |
| Claude sub-agents | Inherit from the parent, or follow the alias remap. `CLAUDE_CODE_SUBAGENT_MODEL` is never set, because it would force a Haiku sub-agent up | Sub-agents of launched sessions |
| Desktop apps and harnesses started outside the shim | Not reachable | Reported in reach; their usage is still imported and measured |

### Claude Code settings at launch

Environment variables are used because they don't persist and an explicit `--model` still beats them. A caller-set `ANTHROPIC_MODEL` is also an explicit choice, so the whole Claude overlay (alias remaps included) is skipped for that launch. The session model is set only when Claude's `settings.json` / `settings.local.json` (under `CLAUDE_CONFIG_DIR` or `~/.claude`) names no `model`, or names one the mode lowers (a key of `model_map`); a lower default such as Haiku is left alone, and an unreadable settings file means no change. Typed `/model` and `/autocompact` are not used, because they save themselves as new defaults.

| Setting | Variable |
|---|---|
| Session model | `ANTHROPIC_MODEL` |
| Alias remap | `ANTHROPIC_DEFAULT_OPUS_MODEL`, `ANTHROPIC_DEFAULT_SONNET_MODEL` |
| Effort | `CLAUDE_CODE_EFFORT_LEVEL` |
| Context limit | `CLAUDE_CODE_AUTO_COMPACT_WINDOW` (100k minimum; a smaller limit is raised to 100k) |

### Codex settings at launch

The launcher adds `-c model=`, `-c model_reasoning_effort=` and `-c model_auto_compact_token_limit=` for interactive sessions and `exec`. `-c model=` is added only when `config.toml` (under `CODEX_HOME` or `~/.codex`) sets no top-level `model` or sets one the mode lowers; an unreadable file means no change. An explicit model choice (`-m`, `-c model=`, `-c model_provider=`, `-p`/`--profile`, `--oss`) leaves the whole launch unchanged, effort and context limit included: Agent Manager runs always pass `-m`, so the shim never adds settings on top of a managed run, exempt roles included. `codex exec -o <file> resume` is a resume like `codex exec resume`. Other subcommands, including `resume` and `exec resume`, are unchanged. An explicit `-c` for the same key also wins. The shim carries no role or profile, so only global and runner-scoped modes reach it; role and profile scopes apply through Agent Manager runs. A mode's context limit is a number of tokens per runner. Claude ignores anything under 100k, so a smaller limit is raised to 100k (`ClaudeContextLimit` in the package) and the raise is stated wherever the limit is recorded. Claude launches with `--resume`, `-r`, `--continue` or `-c` are left alone, as is any variable the caller already set.

## Context limits and compaction

A context limit makes the harness compact a conversation sooner, so each request re-reads less. It is separate from the model and can be used alone.

| Where | How | Reach |
|---|---|---|
| Mode | `context_limit_tokens` per runner, applied by the shim at launch (above) | New terminal and Web Console sessions |
| Agent Manager run | Applied at run creation from the mode and recorded on the run's resolved config (`contextLimitTokens`) and its selection reason (`usage_mode_context_limit:<n>`; a Claude limit raised to the minimum also records `usage_mode_context_limit_raised:<asked>_to_claude_minimum_100000`). Codex gets `-c model_auto_compact_token_limit=<n>`, Claude gets `CLAUDE_CODE_AUTO_COMPACT_WINDOW` in the run's environment. Resumes and wakes carry the same value | Managed runs, headless and Codex interactive. A Claude interactive run is launched by Web Console and is not covered |
| New Web Console launch | The launcher's compaction limit. The options come from the model choice: 1M, 300k and 150k, kept only when they fit the model's window and meet the harness's minimum (see below). Choosing none adds nothing, so a mode or the harness default applies. It becomes an environment prefix (Claude) or a launch argument (Codex) on that launch's command and is never stored. A command that already states a limit, a wrapper or pipeline, and harnesses that declare no context limit are left unchanged | That one session |
| Running session | "Context and compaction" in a tab's menu: **Compact when idle**, and an optional threshold (150k, 200k, 300k; off by default) that compacts the session when its context passes the size and it is idle | That one session |
| Many sessions | "Compact idle sessions" in the launcher lists every open Claude and Codex session with its size, state and whether it can be compacted, then compacts only the ones ticked and returns one result per session, up to 16 per request, run together so a request takes about one verification wait | Operator only |

A model's window comes from the harness, never from its name: Claude's status line reports it for the running model and `resource-claude-code statusline run` keeps it per model in `~/.vrooli/state/vrooli/claude-code/model-context-windows.json`, and Codex lists it in `models_cache.json` under `CODEX_HOME`. A model no session has reported yet has an unknown window, which the launcher states. Its options are then only those the harness accepts.

Compaction is done by typing `/compact` into the session. Web Console never types `/model` or `/autocompact`, because those save themselves as the operator's defaults, so nothing here changes a harness setting. A session is typed into only when all of these hold at that moment, read fresh from the activity signal and its transcript:

- the activity state is idle with confidence of at least 0.6, no prompt is showing, and the terminal has been quiet for 15 s;
- the composer is on screen and holds no unsent text. Any text after the prompt marker is a draft unless every character is drawn dimmed (faint), which both harnesses use for placeholder hints; a hint-looking draft such as `explain ...` is a draft. A composer that cannot be found is skipped with that reason;
- the pane's foreground process now (read from tmux, not the stored agent type) is not a shell, so a session whose agent exited to a shell is never typed into;
- no goal is active. A Codex `thread_goal_updated` or a Claude `goal_status` that is not met counts as active, and so does a Codex `create_goal` call (which writes no goal event), a goal record that cannot be decoded, and a Codex goal status other than exactly `paused` or `complete`. Only a paused or finished goal permits compaction, because compacting a paused goal is verified safe and compacting an active one is not. The finished record must name the tracked goal: a late status for another objective or thread changes nothing. A goal finished by Codex's `update_goal` tool writes no event, so that session stays refused;
- the transcript shows a context of at least 20k tokens (a smaller summary saves nothing) and its size is known.

A busy session or one with an active goal is skipped, never waited for, and the skip states its reason. A pending "Compact when idle" waits only for transient reasons (busy, waiting for input, recent activity, an unsent draft); a goal refuses it outright. The outcome is confirmed from the transcript (Claude `compact_boundary`, Codex `compacted`), not from the keystroke, and every attempt, including each skip, is listed under "What happened" with the size before and, once the session has answered again, after. Watcher settings live only in the Web Console process and are lost when it restarts. Agents are refused: every control needs a verified human on a same-origin request.

## Operator surfaces

```bash
agent-manager usage mode start --preset one-tier-down --until weekly-reset --runner codex
agent-manager usage mode status
agent-manager usage mode end <id>
```

Starting and ending a mode requires verified operator authority through the Agent Manager API, which rejects agent identities. Web Console's start, end and compaction controls require the same verified independent human on a same-origin request: a human credential forwarded by an agent (for example the owner delegation inside a terminal session) carries agent provenance and is refused even on loopback. Agents can read status. The file itself is ordinary local state, so this protects the supported path rather than the file.

## Agent Manager behaviour

- **Authority.** `StartUsageMode` and `EndUsageMode` require a verified independent human with `agent-manager:write`; an agent or service identity gets 403 and the request never reaches the file. `started_by` is that verified subject. `ListUsageModes` is readable by agents. The CLI uses the local operator exchange automatically outside an identified agent run.
- **Starting.** `--preset one-tier-down` (or `--model X`, which needs `--runner`) builds the mode from the resources' `usage_tiers`, found through each resource's own policy path. Without `--runner` the mode covers Claude and Codex, which is why `--until 5h-reset|weekly-reset` (a provider window) needs `--runner`. Claude has a 5-hour and a weekly window; Codex has only the weekly one, so `5h-reset` is refused for it. A reset keyword with no quota reading for that window is refused rather than guessed. Starting replaces an active mode with the identical scope. `--model X` is accepted only when X is a lower tier than the model it replaces; the same rule bounds a Learn challenger, which may never exceed the incumbent.
- **Overlay.** After role resolution and before an explicit model, a new run is lowered when a mode matches its runner, resource role or portable role, and profile key. The selection reason records `usage_mode:<id>`. If the target is excluded, the candidate is metered, a delivery role pins its effort, or the runner's model probe rejects it, the original candidate stays and the reason records `usage_mode_skipped:<why>`; fallbacks are never walked. Effort only moves lower than the run's own. A context limit applies to shim launches only.
- **Rebase.** A rebase applies the mode's own scope: a role- or profile-scoped mode moves only runs of that role or profile, and it refuses, with a per-run reason, runs created with an explicit model and runs in a Learn comparison (any canary arm).
- **Early end.** The usage alerter's five-minute evaluation ends a `window_reset` mode when a newer reading shows its window reset before `expires_at`.
- **Status.** Shows each mode's scope, settings, expiry, time left and the observed quota increase since it started (`usage burners --since`), plus a reach table per current window: operator sessions, profile runs and unattributed, in estimated points. The burn is observed use, not a saving.

## Usage page and notifications

The Usage page shows a banner for each active mode (what, until when, burn since it started, extend, end) and a mode sheet (preset, duration, scope under Advanced, an unmeasured effect range from the recommendation numbers, reach). Quota-lever recommendation cards start a mode for their provider. A start on the sheet affects new work only. Extending from the banner calls `ExtendUsageMode`, which moves the end of the same mode and keeps its id, start, starter and settings, so the burn since start stays continuous.

## Switching running work

"Also switch running sessions" on the sheet, and "Switch running work" on the banner, open a preview that changes nothing: one row per non-terminal run with what would happen and, for each, the estimated re-read cost (the run's last context size; a changed model cannot reuse the prompt cache). Only after the operator confirms does `RebaseUsageModeRuns` act, and it returns one result per run:

| Result | Meaning |
|---|---|
| `switched_at_next_resume` | A managed (codec pipe) run that is parked or finished and has a session. A pending change is recorded on the run; its next continue or wake moves the retained model to the mode's lower model, writes `usage_mode_rebase:<id>` to the selection reason and appends to the run's `modelRebases` history. The creation-time policy snapshot is not rewritten, so it still says what the run was created with. The model is checked again at resume: an ended mode or a refused target leaves the model alone and says so in the run log |
| `applies_on_next_launch` | An interactive session. It is never changed in place: switching forces a full re-read and a typed `/model` would be saved as a default |
| `unable` | With the reason: mid-turn, ended, no session, not managed by Agent Manager (imported or attached), exempt role, the mode does not lower this model, or the lower model is refused (metered, excluded, delivery effort pinned, probe failed) |

Both calls require a verified operator; agents are refused. Nothing changes for any run unless it was asked for.

Agent Manager asks the operator through notification-hub when a window will run out early, 15 minutes before a mode ends (once per expiry), and daily while an until-off mode runs. A decision that was answered or expired is not raised again. The flood cap is per kind and runner, and a requested timed start (`/api/v1/usage/modes/ask`) rounds its length to 15 minutes and allows two undecided asks per runner (429 beyond that). What an answer starts or ends is read from the ask Agent Manager stored when it raised it, never from the correlation echoed back, and an answer marked late does nothing. Answers start, extend or end a mode only when the answer carries a verified human, who is recorded as `started_by`. The categories, decisions and merge rule are in `scenarios/agent-manager/docs/concepts/USAGE-INTELLIGENCE.md`.

## Learn mode

A learn-kind mode (`"kind": "learn"`) is a controlled comparison, not a saving. It tests whether a lower model earns its keep for one role by splitting that role's new managed runs at random between the model it runs on today (incumbent) and a lower one (challenger).

```json
{
  "id": "um-20261101-d4e5f6",
  "kind": "learn",
  "scope": { "runner": "codex", "role": "code.default" },
  "runners": {},
  "learn": {
    "split_percent": 50,
    "runners": { "codex": { "model_map": { "gpt-6.1-terra": "gpt-6-luna" }, "session_model": "gpt-6-luna" } },
    "sessions": { "opt_in": true, "block_days": 1, "opted_in_by": "operator@example.com" },
    "paused": false
  }
}
```

The challenger's settings sit in `learn.runners`, and `runners` stays empty, so a reader that does not know Learn finds no settings and changes nothing. Readers still fail open: `usagemode.Resolve` never returns a learn mode, and `ResolveLearn` and `ResolveLearnSession` return one only for the role (or opted-in sessions) it covers.

| Rule | Detail |
|---|---|
| Who joins | Managed runs of the covered role. Exempt roles never join, even when a mode is scoped to one. An explicit model, a role that declares its own challenger, and resumed or recovered runs stay out |
| Assignment | Agent Manager's overlay declares the challenger and the sample rate on the selected candidate; the existing `SelectCanaryArm` assigns the arm once from the run id and records `canary_arm` in the policy snapshot (`usage_mode_learn:<id>` in the selection reason). Without a learn mode the canary stays inert |
| Operator sessions | Only with `--sessions`. Randomized day blocks: the arm comes from the mode id and the block's first UTC day (`usagemode.SessionArm`), so the launcher needs no Agent Manager connection. The launcher prints `Learn mode <id>: <arm> day`, exports `VROOLI_USAGE_ARM` and `VROOLI_USAGE_MODE`, and appends a marker to `arms.jsonl` beside the mode file. `VROOLI_FULL_POWER=1` or an explicit model on a challenger day runs as asked, is recorded as an exception and stays in its assigned arm |
| Save wins | A save mode that applies to a run or launch takes priority and leaves it out of the comparison |
| Pause | While a quota window for the runner's provider is forecast to run out, Agent Manager sets `learn.paused` (reason starts `forecast:`) and resumes when it no longer is. Assigned work keeps its arm |
| One per role | A second comparison for the same runner and role is refused; a save start never replaces a comparison |

```bash
agent-manager usage experiment plan --runner codex --role code.default           # expected time to a verdict, nothing started
agent-manager usage experiment start --runner codex --role code.default --until off --confirm [--sessions] [--split 50]
agent-manager usage experiment status [<id>]
agent-manager usage experiment end <id>
agent-manager usage experiment propose <id>                                       # ask a verified human to approve the role-default change
```

Start, end and propose need verified operator authority and refuse agents (403); plan and status are readable. `start` without `--confirm` only prints the plan. Analysis, verdicts, the power simulation and the time to a verdict are in `scenarios/agent-manager/docs/reference/operational-stats.md` ("Learn mode"). The Usage page has a full-page setup at `/usage/learn` and shows each comparison on the Evidence tab with design **Randomized**.

**Role recommendations.** When a verdict reads cheaper looks better or about the same, the evidence view adds a role-default recommendation with its evidence and the exact policy change (`replace /roles/<role>/model ...` in the resource's `model-policy.json`). It is an ask: `propose` raises an operator decision that only a verified human can approve, and approval records that person and keeps the change. Agent Manager does not edit resource files, and there is no governed apply path from its API, so the operator applies the change. No `model-policy.json` schema changes; no resource CLI rebuild is needed.

## Limits

- Savings estimates use unfitted price ratios until mode periods provide measurements.
- Compaction needs the session's transcript to be found; a session whose transcript Web Console cannot locate is skipped as "context size unknown".
- Agent Manager compacts a parked managed Codex session under the same goal rule, and skips one whose rollout cannot be read or whose context size is unknown. The transcript rules for both live in `packages/cli-core/agenttranscript`.
- Web Console recognises an empty composer by blank text or dimmed (faint) text only. A placeholder drawn in a theme colour instead of faint looks like a draft and the session is skipped, never typed into.
- A model's lower price does not prove a lower cost per outcome. See the evidence view in Agent Manager.
- Modes apply per machine.
