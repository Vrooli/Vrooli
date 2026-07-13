# Interactive runner design — plug-in seam + per-agent transcript contract

Status: Phase 1 design, finalized 2026-07-13.
Plan: `agent-manager-interactive-runner-coding-agent-runs-inside`
(plan-manager exec `ab16811e-1fc8-4dce-98d8-59fd07d421aa`).

This document settles, before any build work, (a) **where** interactive mode
plugs into the runner architecture, (b) the **per-agent transcript-locating
contract** with empirical evidence, (c) the **opencode include/descope** call,
and (d) the **durable run-state additions**. It is written against the seven
locked operator decisions (2026-07-12) and does not relitigate them.

Interactive mode = agent-manager creates a web-console session (persistent /
tmux) and launches the **real interactive agent CLI** inside it. Events do not
come from the codec stdout pipe (`DecodeStreamLine`) — they come from
agent-manager **tailing the agent-owned transcript file** the CLI writes, using
the per-agent transcript parser (`ParseTranscriptLine` / `NewTranscriptParser`).
Completion is a transcript terminal marker, the same mechanism crash recovery
already uses. Stdin is a free-for-all (human + agent-manager both type; no
lease). Interactive mode is allowed only for non-protected (in-place) runs.

---

## 1. Plug-in seam decision: parallel execution path, NOT a `runner.Launcher`

**Decision: implement interactive mode as a parallel execution path that reuses
the transcript-tail machinery (`runner.Consume` + the codec transcript parsers),
NOT as a new `runner.Launcher` implementation.**

### Why not a `runner.Launcher`

`SandboxLauncher` is the tempting precedent — it already launches a "remote"
process (over workspace-sandbox HTTP+SSE) and returns a `LaunchedProcess`. But
the `Launcher`/`LaunchedProcess` contract is the wrong abstraction for
interactive mode, on three concrete counts against the real `core/runner.go`
code:

1. **`LaunchedProcess.Stdout()` is fed to `codec.DecodeStreamLine`, not
   `ParseTranscriptLine`.** In `core.Runner.scanStream` every stdout line is
   dispatched to `DecodeStreamLine` (the `--print` / `exec --json` /
   `streaming-json` **stdout** decoder). Interactive mode has no
   agent-manager-owned stdout pipe at all — the CLI's stdout goes to the
   web-console PTY. Its events live in the agent-owned **transcript file**,
   which is a *different dialect* parsed by `ParseTranscriptLine` (see §3). A
   `Launcher` impl would have to hand `scanStream` transcript bytes that then
   get routed into the wrong decoder.

2. **`LaunchedProcess.Wait()` means "process exited"; completion here is a
   transcript marker.** `core.Runner.Execute` blocks on `proc.Wait()` and
   classifies on exit code. In interactive mode the web-console session (a
   long-lived shell/tmux pane) outlives the agent turn, so there is no local
   child whose exit equals run completion. Completion is a terminal marker in
   the transcript (locked decision 3), exactly as `orchestration/recovery.go`
   finalizes recovered runs from `TranscriptTerminal` without any `Wait()`.

3. **`Stop` is SIGTERM-to-pgid; here it is an interrupt key sequence + session
   delete.** `core.Runner.Stop` calls `proc.Signal(grace)` / `proc.Kill()` on a
   process group. Interactive Stop (locked decision 6) is a web-console
   `TerminalService.SendInput` interrupt sequence, with session `Delete` as the
   hard-kill fallback — not a signal to a local pgid.

Forcing interactive mode through `Launcher` would require a fake
`LaunchedProcess` whose `Stdout()` secretly reads a tailed file and whose
`Wait()` secretly blocks on a marker — an abstraction lie that is then
mis-wired by `scanStream` and `Stop`. Rejected.

### What the parallel path reuses (the machinery that IS right)

The valuable, already-proven machinery is **`runner.Consume`** (the live
transcript tailer in `transcript_consumer.go`) plus the per-agent
**`codec.NewTranscriptParser()/ParseTranscriptLine`** seam. The durable-transcript
path (`core.runDurable`) and crash recovery (`recovery.drainTranscript`) already
consume transcripts exactly this way:

```
runner.Consume(ctx, ConsumeArgs{
    Transcript:  <agent-owned path>,
    Live:        true,
    ParseFn:     codecTranscriptParser.ParseTranscriptLine,
    EventSink:   <run event sink>,
    OnAdvance:   persist TranscriptCursor/LastSeq,
    OnSessionID: persist SessionID,
})  // returns (cursor, *TranscriptTerminal, err)
```

The interactive path is a sibling to `runDurable` that:
1. resolves a web-console session (Connect `SessionsService.Create`, §2/Phase 2),
2. builds an env-scoped launch command (§4) and lets web-console paste+execute it,
3. **discovers** the agent-owned transcript path post-launch (§3),
4. tails it with `runner.Consume(Live=true, …)` — the identical tailer,
5. treats the terminal marker as completion (locked decision 3),
6. persists `ExecutionMode` + `WebConsoleSessionID` + `TranscriptPath` (§5).

It does **not** call `launcher.Launch`, `scanStream`, `proc.Wait`, the stderr
accumulator, or SIGTERM `Stop`.

### Where it physically lives

An orchestration-level execution strategy branch (in `run_executor` /
`phases/execute`), selected by the run's `ExecutionMode`, is preferable to a new
method on `core.Runner`, because it must call the web-console Connect client and
own the session lifecycle — concerns that live at the orchestration layer, above
the codec-parameterised `core.Runner`. `core.Runner` stays the codec-pipe engine;
the interactive strategy borrows only `runner.Consume` and the codec's transcript
parser (both already exported and used cross-package by recovery).

---

## 2. web-console consumption (read-only, via generated Connect client)

agent-manager consumes web-console's Connect API read-only through
`packages/proto/gen/go/web-console/v1/...` (never edits web-console). The
foundation plan (shipped 07-13) provides everything needed:

- `SessionsService.Create` (`sessions.CreateRequest`): `Owner` (set
  `"agent-manager"`), `DisplayLabel` (human sidebar label, e.g. the run tag),
  `Origin` = `SESSION_ORIGIN_PROGRAMMATIC`, `LaunchCommand`, and
  `ExecuteLaunchCommand=true` (server pastes+runs the launch command via the
  recovery paste seam, no readiness gate).
- `SessionsService.Get` / `Delete` — poll/inspect and hard-kill.
- `TerminalService.SendInput` — free-for-all stdin: agent-manager types the
  prompt, Continue follow-ups, and the Stop interrupt sequence. Source
  attribution (`"agent-manager:run-<id>"`) is diagnostic only (locked
  decision 4).

The run-detail UI deep-links to the live session (locked decision 7) using the
stored `WebConsoleSessionID` and the web-console UI base.

---

## 3. Per-agent transcript-locating contract (empirically verified)

**Critical empirical finding.** The existing codec transcript parsers were
written for the CLI **stdout** dialects (`claude --print --output-format
stream-json`, `codex exec --json`, `grok --output-format streaming-json`). The
**on-disk session transcripts** an *interactive* CLI writes are a **different
dialect**. Measured on this host by feeding real on-disk files through each
codec's `ParseTranscriptLine` (throwaway harness
`codecs/interactive_transcript_probe_test.go`, run output in §7):

| Agent | On-disk path rule (verified) | Existing `ParseTranscriptLine` on the on-disk file | Terminal marker in on-disk file |
|-------|------------------------------|----------------------------------------------------|---------------------------------|
| claude | `~/.claude/projects/<cwd-slug>/<session>.jsonl` | **partial**: 273 lines → 116 real events (32 message, 43 tool_call, 41 tool_result) **+ 98 "unhandled type" debug + 59 debug**; **0 terminal, session-id NOT captured** | none as `result`; per-turn `assistant.message.stop_reason=end_turn` |
| codex | `$CODEX_HOME/sessions/YYYY/MM/DD/rollout-<ts>-<uuid>.jsonl` | **zero events** (112 lines → 0) — rollout wraps everything as `{"type":"event_msg"\|"response_item","payload":{…}}`, foreign to the `exec --json` decoder | `event_msg` `turn_completed` / `turn_aborted` |
| grok | `$GROK_HOME/sessions/<url-encoded-cwd>/<session>/updates.jsonl` | **zero events** (10 lines → 0) — ACP JSON-RPC `{"method":"session/update","params":{"update":{"sessionUpdate":…}}}`, foreign to the flat `{"type":…}` decoder | `params.update.sessionUpdate=turn_completed` |
| opencode | **no file** — SQLite `~/.local/share/opencode/opencode.db` (`message`/`part` tables) | n/a (nothing to tail) | n/a |

### Consequence for the locked "existing codec parsers" decision (decision 2)

The **seam** the operator locked — per-agent `codec.ParseTranscriptLine`
producing `domain.RunEvent`, tailed by `runner.Consume` — is retained exactly.
What the evidence shows is that each codec's transcript parser must be **taught
the on-disk session dialect** in Phase 3; it is not a new seam, it is a format
branch inside the existing `NewTranscriptParser()` (the parser already
distinguishes a transcript-replay path from the live stdout path). This is a
Phase-3 scope item, not a seam change. Specifically:

- **claude** — extend the transcript parser to (a) read `sessionId` (camelCase;
  the stdout dialect uses `session_id`), (b) ignore the interactive-only record
  types (`mode`, `permission-mode`, `ai-title`, `last-prompt`, `attachment`,
  `file-history-snapshot`) instead of emitting "unhandled" debug noise, and
  (c) synthesize the terminal marker from the final `assistant` message's
  `stop_reason` (there is no `result` line on disk). Message/tool_use/tool_result
  extraction already works (the message-content shape matches).
- **codex** — add a rollout-dialect branch: unwrap `payload`, map
  `event_msg.agent_message`→assistant message, `response_item.function_call`/
  `function_call_output`→tool call/result, and `event_msg.turn_completed`/
  `turn_aborted`→terminal. Session id is the `session_meta.payload.id` (also the
  rollout filename uuid).
- **grok** — add an ACP-dialect branch: read `params.update.sessionUpdate`
  (`agent_message_chunk`→assistant text, `tool_call`/`tool_call_update`→tool
  events, `turn_completed`→terminal), session id from `params.sessionId`.

### Transcript discovery rules (per agent)

- **cwd→slug (claude)**: replace every non-alphanumeric character in the
  absolute cwd with `-`. Verified against real dirs: `/home/matthalloran8/Vrooli`
  → `-home-matthalloran8-Vrooli`.
- **which file / which session**: because agent-manager owns the per-run
  `CODEX_HOME`/`GROK_HOME` (§4), the run's rollout/`updates.jsonl` is the
  **only** transcript under that run-scoped home — unambiguous, no
  "newest-after-launch" race. For **claude** the projects tree is shared
  (see §4), so discovery is "the `<slug>` subdir's `*.jsonl` with the newest
  ctime created after launch time"; capture that path once, then tail it by
  fixed path (the existing `recovery.findClaudeNativeTranscript` uses the same
  `~/.claude/projects/*/<sessionID>.jsonl` glob).

---

## 4. Launch-env control (resolves the "agent-manager controls env" tension)

Locked decision 2 says agent-manager controls the launch env (its own
`CODEX_HOME`/`GROK_HOME`). But the agent runs **inside** a web-console PTY, and
web-console's `defaultSessionEnv` already injects its **own** per-session
`CODEX_HOME`/`GROK_HOME` (verified in `web-console/api/session_factory.go`,
read-only). Reconciliation: **agent-manager puts an explicit env prefix in the
launch command string**, which wins over the PTY's exported env for that
process, and points at an agent-manager-owned run-scoped directory. Example
launch command agent-manager sends as `CreateRequest.LaunchCommand`:

```
cd <workdir> && CODEX_HOME=<am-run-dir>/codex codex          # codex
cd <workdir> && GROK_HOME=<am-run-dir>/grok grok             # grok
cd <workdir> && claude                                        # claude (see below)
```

This satisfies decision 2, decouples agent-manager from web-console's internal
path layout, and makes the transcript path deterministic (agent-manager chose
the home, so the run's rollout/updates file is unambiguous, §3).

**claude exception (empirically discovered).** `CLAUDE_CONFIG_DIR` relocates the
whole claude config dir, **including auth** — pointing it at a fresh dir triggers
OAuth re-onboarding (verified: a fresh `CLAUDE_CONFIG_DIR` claude launch dropped
to the `https://claude.com/…/oauth/authorize` login screen). web-console
deliberately does **not** set `CLAUDE_CONFIG_DIR` (its `pty_test` asserts no
override) so claude uses the shared, authenticated `~/.claude`. Therefore for
claude, agent-manager does **not** relocate the config dir; it uses the shared
`~/.claude/projects/<cwd-slug>/` and discovers the session file by slug + newest
ctime after launch (§3). Seeding a private `CLAUDE_CONFIG_DIR` with symlinked
auth (as web-console does for grok) is a possible future hardening but is out of
scope for v1.

---

## 5. Durable run-state additions

Interactive runs need three durable facts on the `Run` record. Field names,
types, and where they live:

| Concern | Domain field (`internal/domain/types.go` `Run`) | DB column | Proto (`v1/domain/run.proto`) | Notes |
|---------|--------------------------------------------------|-----------|-------------------------------|-------|
| Execution mode | `ExecutionMode ExecutionMode` (new string enum: `codec_pipe` \| `interactive`; default `codec_pipe`) | `execution_mode` (text, default `'codec_pipe'`) | new field `execution_mode` (next free tag `= 37`), backed by a proto enum `ExecutionMode` in `run.proto`/`types.proto` | orthogonal to `RunMode` (sandboxed/in_place). UI shows it; drives the strategy branch in run_executor. |
| web-console session id | `WebConsoleSessionID string` | `web_console_session_id` (text) | new field `web_console_session_id` (`= 38`) | used to build the run-detail deep link and to route Continue/Stop `SendInput` + `Delete`. |
| Resolved transcript path | **reuse** existing `TranscriptPath string` (`transcript_path`) | existing | existing (db-only recovery metadata; not proto-exposed) | for interactive runs this holds the **discovered agent-owned** path; for codec-pipe it holds the agent-manager-written stdout file. Same field, provenance differs. `TranscriptCursor`/`TranscriptLastSeq` reused as-is for tail resume. |

No new session-id field: the existing `Run.SessionID` (claude session / codex
thread / grok session) is still populated — by `Consume`'s `OnSessionID`
callback from the transcript, exactly as today.

Migration: additive columns only (`execution_mode` with default, and
`web_console_session_id`) — follows the SQLite migrate-never-recreate rule. The
enum default keeps every existing run `codec_pipe`, so behavior is unchanged for
non-interactive runs.

Policy gate (locked decision 5): interactive mode is rejected at run-validation
time for protected runs (`RunMode == sandboxed`), with a clear error; protected
runs keep the sandbox launcher / codec-pipe path.

---

## 6. OpenCode: DESCOPED from v1 (explicit call)

**opencode interactive mode is descoped from v1.** Rationale (empirically
grounded):

1. **No tailable transcript file.** opencode persists conversation state to
   SQLite (`~/.local/share/opencode/opencode.db`, tables `message` / `part` /
   `session`) — verified on this host. There is no per-session JSONL to tail,
   so the file-tail contract the other three share does not exist for opencode.
2. **Server + SSE model.** opencode runs a local server (`opencode serve`) and
   streams over SSE; the only "live event" route is that server's stream, which
   web-console already watches. Consuming it would make web-console conversation
   events the event source — explicitly rejected by locked decision 2.
3. **A SQLite-WAL tailer would be a distinct, fragile mechanism** (poll `message`/
   `part` inserts under an actively-written WAL), not the shared `runner.Consume`
   file tailer — disproportionate for v1.

v1 interactive support is **claude, codex, grok**. opencode continues to run
fine in the existing **codec-pipe** path (unaffected). A future opencode
interactive contract would define either a SQLite change-feed adapter or an
opencode-server SSE adapter; noted for a follow-up, not built now.

---

## 7. Validation evidence

### 7a. Real on-disk transcripts replayed through each codec parser

Harness `codecs/interactive_transcript_probe_test.go` (throwaway; delete after
Phase 1) fed the newest real on-disk file for each agent through
`NewTranscriptParser().ParseTranscriptLine`:

```
[claude-native] lines=273 linesWithEvents=273 totalEvents=273 terminals=0 sessionIDcaptured="" parseErrs=0
[claude-native] events by type: map[log:debug:59 log:unhandled:98 message:32 tool_call:43 tool_result:41]
[codex-rollout] lines=112 linesWithEvents=0 totalEvents=0 terminals=0 sessionIDcaptured="" parseErrs=0
[codex-rollout] events by type: map[]
[grok-updates]  lines=10  linesWithEvents=0 totalEvents=0 terminals=0 sessionIDcaptured="" parseErrs=0
[grok-updates]  events by type: map[]
```

Interpretation: claude's on-disk transcript is **partially** parseable today
(messages + tool calls/results survive; terminal + session-id do not, and 98
interactive-only records become debug noise); codex and grok on-disk transcripts
are **not** parseable by the current parsers at all. This is the evidence behind
the §3 Phase-3 scope items. It does **not** change the §1 seam decision.

### 7b. Turn-completion markers exist on disk (locked decision 3 is achievable)

- claude: `assistant.message.stop_reason` present per turn (`end_turn` on a
  cleanly finished turn; the sampled active session showed `tool_use`, i.e. mid-work).
- codex rollout: `event_msg` `turn_completed` / `turn_aborted` observed at tail.
- grok updates: final `params.update.sessionUpdate == "turn_completed"` observed.

So "completion = transcript terminal marker" holds for all three v1 agents; the
markers are the on-disk turn-boundary records (Phase 4 maps them), not the
stdout `result`/`turn.completed`/`end` events.

### 7c. Path rules verified

- CLIs installed: `claude` (~/.local/bin), `codex` (/usr/bin), `grok`
  (~/.local/bin), `opencode` (~/.opencode/bin), `tmux` present.
- claude slug: `/home/matthalloran8/Vrooli` → dir
  `~/.claude/projects/-home-matthalloran8-Vrooli/` (real).
- codex: `~/.codex/sessions/2025/11/30/rollout-2025-11-30T23-59-17-<uuid>.jsonl` (real).
- grok: `~/.grok/sessions/%2Fhome%2Fmatthalloran8%2FVrooli.../<session>/updates.jsonl` (real).
- claude `CLAUDE_CONFIG_DIR` relocation → OAuth re-onboarding (real, §4).

### 7d. Review against the seven locked decisions — no contradictions

1. web-console session + real interactive CLI, codec pipe unused → §1, §2. ✔
2. events from tailing agent-owned transcripts with the codec parser seam → §1, §3.
   The parsers need on-disk-dialect branches (§3), which *extends* the seam,
   does not replace it. ✔ (surfaced, not contradicted)
3. completion = transcript terminal markers, same as recovery → §1, §7b. ✔
4. stdin free-for-all, attribution diagnostic only, no lease → §2. ✔
5. interactive only for non-protected runs, enforced at validation → §5. ✔
6. Continue via `SendInput`, Stop = interrupt seq + session delete fallback → §1, §2. ✔
7. run records mark execution mode; detail UI links to live session → §5, §2. ✔

---

## 8. Risks carried to later phases

- **R1 (Phase 3, high):** each codec's transcript parser needs an on-disk-dialect
  branch. codex/grok extract **zero** today; claude misses terminal + session-id
  and emits noise. Mitigation: the parse seam is unchanged; add golden fixtures
  from the real files captured here.
- **R2 (Phase 4):** claude has no on-disk `result` line — completion must be
  synthesized from `stop_reason=end_turn` plus an idle-debounce (interactive
  sessions stay open awaiting input). Define the debounce so a tool_use pause is
  not mistaken for completion.
- **R3 (Phase 2/4):** claude transcript file discovery on the **shared**
  `~/.claude/projects` tree is a newest-ctime-after-launch heuristic; concurrent
  claude sessions in the same cwd could race. Mitigation: bound the search to
  files created after the recorded launch timestamp; prefer the session whose
  first line's cwd matches; capture the path once and pin it.
- **R4 (Phase 4):** codex rollout files are date-pathed
  (`YYYY/MM/DD`); a run that crosses midnight or a long idle could confuse a
  naive glob. Mitigation: the run-scoped `CODEX_HOME` isolates to one run, so
  glob `**/rollout-*.jsonl` newest under that home.
- **R5 (Phase 5):** Stop via interrupt sequence is best-effort; the session
  `Delete` fallback must be idempotent and must finalize the run even if the
  interrupt left the CLI mid-turn.
- **R6 (later):** opencode interactive requires a wholly different adapter
  (SQLite change-feed or opencode-server SSE); descoped (§6).

---

## 9. Phase 4 addendum — completion, turn-boundary debounce, and recovery

Phase 4 built the interactive execution path (`orchestration/interactive.Coordinator`)
and its restart recovery. This section documents the completion contract and the
per-agent supplement (design R2), settled with real evidence, and logged as
plan-manager decisions.

### 9.1 Completion = terminal marker, every success terminal is a turn boundary

Completion is a `TranscriptTerminal` marker (locked decision 3), the same trust
model `orchestration/recovery.go` uses — **never** process exit or byte-silence,
because interactive CLIs stay alive between turns. Per-agent markers (Phase 3):

| Agent | On-disk terminal | Meaning |
|-------|------------------|---------|
| claude | synthesized from the final assistant `stop_reason=end_turn` | a **turn** ended (there is no on-disk `result` line) |
| codex | `event_msg.task_complete` / `turn_completed` (success), `turn_aborted` (failure) | a **turn** ended |
| grok | `params.update.sessionUpdate == turn_completed` | a **turn** ended |

Because each marker is a **turn boundary**, not a run boundary, the coordinator
applies an **idle-debounce** uniformly to all three agents: after a *success*
terminal it waits a debounce window (default **5 s**) for the transcript file to
grow beyond the consumed cursor. Growth ⇒ a new turn began (keep tailing from the
cursor); no growth ⇒ the run is complete. A *failure* terminal (`turn_aborted`)
finalizes the run Failed immediately, no debounce.

claude is the load-bearing case (its `end_turn` is the *only* signal). File-byte
growth is a faithful new-turn proxy because claude writes its interactive-only
records (`file-history-snapshot` / `ai-title` / `last-prompt`) at the **start of
the next turn**, not after `end_turn` on idle. Uniform application gives Phase 5
multi-turn Continue a single reusable turn-boundary notion: a typed follow-up
grows the transcript, the debounce sees it, the run keeps going.

### 9.2 Where it lives (Phase 5 seams)

`interactive.Coordinator` is the parallel execution path (design §1), selected by
`Run.ExecutionMode` in `orchestrator.executeRun` → `executeInteractiveRun`. Its
reusable seams:

- `Coordinator.Execute(ctx, run, LaunchParams, onRunning)` — live path: Launch →
  persist → Running → `tailToCompletion` → `Finalize`.
- `Coordinator.TailToCompletion(ctx, run)` — drain-from-cursor + turn-boundary
  debounce + mid-tail session watch. **Phase 5 Continue** types into the session
  (via `Substrate`/`SessionController.SendText`) between turns; the debounce
  already treats the resulting growth as a new turn, so Continue needs no change
  here. **Phase 5 Stop** calls `Substrate.Stop` (interrupt + delete); Finalize's
  session-gone branch already fails the run cleanly if Stop deletes the session.
- `Coordinator.Finalize(ctx, run, terminal, tailErr)` — the single completion
  seam (Complete / Failed / leave-Running-on-graceful-shutdown).

### 9.3 Recovery + session-gone (resolves R3/R4 liveness)

Interactive-run liveness is the **web-console session** (`SessionsService.GetSession`),
not the reconciler pgid scan — the CLI lives in web-console tmux. On restart,
`RecoverInFlightRuns` / `handleStaleRun` route `ExecutionMode=interactive` runs to
`recoverInteractiveRun`, which:

1. drains the transcript from the persisted cursor (reusing the codec-pipe drain);
2. a **failure terminal** ⇒ finalize Failed;
3. else `GetSession`: **gone** ⇒ (success terminal ? Complete : Failed with
   `"web-console session <id> no longer exists; interactive run cannot be
   recovered"`); **alive** ⇒ reattach the tailer from the cursor (no duplicate
   events) and let its debounce drive true completion.

`handleStaleRun` returns early for interactive runs so the pgid/MaxRecoveryAge
kill path never fires. A vanished session **mid-tail** is caught by a background
`GetSession` watcher that cancels the tail. When the reconciler has no web-console
client wired, interactive recovery is a logged idempotent no-op (never falsely
completes/fails). Known limitation: codex rollout **rotation** is not followed
across a restart (the run dir is not persisted, per §5); the pinned rollout path
still tails correctly within a session.

---

## 10. Launch-flow addendum — initial-prompt delivery + launch gates (load-bearing)

Sections §1–§4 described the launch as "create session (execute launch command)
→ discover the transcript". Live end-to-end testing (Phase 6 follow-up, finding
`54c9b5b5`) showed that is incomplete on **three** counts, each of which alone
makes a real interactive run time out with `transcript did not appear …`. All
three are handled in `Substrate.Launch` (`orchestration/interactive/substrate.go`),
which owns the create→seed→deliver→discover ordering.

### 10.1 The initial prompt must be typed in (the primary defect)

The launch command only starts the **CLI**, which then sits at an idle TUI
prompt. A freshly launched claude/codex/grok TUI writes **no transcript until its
first turn begins**, so a discover-immediately-after-create launch never sees a
transcript — nothing delivered the run's task prompt. The prompt is *typed into
the session* (locked decision 4 — a typed input, not a command-line argument):
`LaunchParams.Prompt` carries it, `orchestrator.executeInteractiveRun` fills it
via `interactiveInitialPrompt` (which recombines the codec-pipe system-prompt +
user-message split into the single channel a raw CLI has), and delivery reuses
the Phase-5 `SessionController.SendPrompt` (bracketed-paste + Enter submit) — the
same path Continue uses. Source attribution is `agent-manager:run-<id>`.

Prompt delivery MUST precede discovery: the paste is what creates the transcript.

### 10.2 The relocated home must be seeded (auth + directory trust)

codex/grok relocate their session home to a fresh run-scoped dir (§4) so the
rollout is unambiguous. But a fresh home has **no credentials and no trusted
projects**, so the CLI drops to a **sign-in wall** and then a **"trust this
directory?"** gate — never reaching input, so the typed prompt lands on a menu
and no turn starts. `seedRelocatedHome` fixes both before launch:

- **copy** `auth.json` + `config.toml` from the shared home (`~/.codex`) into the
  run-scoped home (kept logged in). Copy, not symlink: the CLI rewrites these at
  launch, and a symlink would route those writes into the shared file that dozens
  of concurrent CLI processes on a busy host contend on — an observed flaky-launch
  source. A copy fully isolates the run.
- **append** the exact working dir to the private `config.toml` in codex's own
  `[projects."<dir>"] trust_level = "trusted"` shape, so codex boots straight to
  its input. (A `-c` command-line override was tried and rejected — it does not
  reliably suppress the on-disk trust check.)

claude is the exception (§4): it keeps the shared, authenticated `~/.claude`, so
no seeding. It still shows a first-run "trust this folder?" dialog for a fresh
dir; that is handled in-band by re-delivery (below) — the first paste's Enter
accepts the dialog's default, a later delivery lands the prompt.

### 10.3 Bounded readiness — re-deliver until the transcript appears

A paste into a not-yet-rendered TUI can be dropped, and claude's trust dialog
consumes one delivery, so a single paste is unreliable. Bounded guards, no
heavyweight handshake:

1. a short **boot delay** (`promptBootDelay`, default 2 s) before the first paste;
2. **transcript appearance is the ack** — while none has appeared, discovery
   **re-delivers the prompt every `promptResendAfter` (default 5 s)**, bounded by
   `discoveryTimeout` (default **60 s** — a real cold boot + first turn was
   measured at up to ~30 s on a busy host). A delivered prompt starts a turn whose
   transcript stops the loop within a poll, well before the next re-delivery.

An empty `Prompt` skips delivery entirely (the fake-CLI integration harnesses
self-start their first turn), preserving their contract. A hard `SendPrompt` RPC
error on the first delivery is surfaced immediately (session id still returned so
the caller tears the session down). `Continue`/`Stop` are unchanged.

### 10.4 Paste/submit race (the load-bearing reliability fix)

`SendPrompt` delivers a prompt as **two** `SendInput` calls — a bracketed paste,
then an Enter keypress. Firing them back-to-back is unreliable: under load the
Enter races ahead before the TUI has finished ingesting the paste, so it does
**not** submit, and the pasted text accumulates on screen unsent (reproduced with
codex: five deliveries left five unsubmitted "Reply with exactly: pong" lines and
no turn). A short `pasteSubmitDelay` (400 ms, in `webconsole.Client.SendPrompt`)
between the paste and the Enter makes the submit reliable across claude/codex/grok.
This is why manual replays that shelled out per keystroke (natural inter-process
gaps) always submitted while the back-to-back client path intermittently did not.
The fix applies to both the initial-turn delivery and Continue, which share
`SendPrompt`.
