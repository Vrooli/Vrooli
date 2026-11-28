# Messages View: Projection UX

Design record for the Messages view of Web Console: what it shows, how it scrolls, how sends stay visible, how search and playback work, and which decisions are pinned. Requirement module: `requirements/17-messages-view-projection-and-playback/module.json` (`MOD-P0-017`).

## Purpose

Web Console is the operator's daily surface for supervising coding agents. The Messages view makes a long agent session readable on a phone. This record defines the view as an honest projection of the agent session: calm to scroll, safe to send from, and fronted by one playback pill.

## The projection principle

Messages shows only what capture understands, and says so when it does not. The terminal stays the ground truth; the view toggle between Terminal and Messages is unchanged. A state the projection cannot render (a tool call, a prompt a harness does not expose) is shown as a card that hands the user to the terminal, never as silence and never as a guess.

Every captured row is stored, classified at ingestion (see [What a row is](#what-a-row-is)); the view lists only the conversation — what the operator and the agent said to each other — and folds everything else into the lines between it. Session activity (below) explains the time between turns.

## What a row is

A transcript row arrives on a channel (`role`: user or assistant), but the channel does not say what the row is or who wrote it: harnesses put tool calls on the assistant channel and inject instructions, task updates, other agents' messages and launch prompts on the user channel. Before classification about 400,000 of 650,000 stored rows were tool calls and results, and about 30,000 of 43,000 user-channel rows were not typed by the operator; all of them rendered as messages, and the injected ones as "You".

Every row now carries three things, set when it is captured (`agenttranscript.Message`, shared by Web Console and Agent Manager):

| Field | Values | Meaning |
|---|---|---|
| `kind` | `message`, `attachment`, `notice` | The conversation: listed as rows |
| | `tool_call`, `tool_result`, `reasoning`, `context` | Work: folded into the line between messages |
| `author` | `human`, `agent`, `peer`, `automation`, `harness` | Who wrote the text; `role` stays the channel |
| `thread_id` | empty, or an agent thread | Empty is the session's own conversation; otherwise a subagent's, a teammate's or the approval reviewer's |

Classification starts from native structure where the transcript has it — Claude's `origin` (peer, task notification, auto-continuation), `isMeta`, `isCompactSummary`, `isSidechain`/`agentId` and its subagent files with their `meta.json`; Codex's `session_meta` (`source.subagent.thread_spawn`, the guardian reviewer) and its function-call records — and falls back to the recognisers in `packages/cli-core/agenttranscript/message-classes.json` for rows that only carry text (hooks, archives, history). The recognisers are data: each rule is a prefix or pattern, a harness list, the kind and author it means, how to read the author's name, and whether it opens a thread. A lead's messages (`lead_agent_names`, `team-lead`) are the prompts of the agent it launched, so in that agent's own transcript they stay on its main thread. The first main-thread prompt of a programmatic session is attributed to the program that opened it (`author=automation`, `label=launch_prompt`), never to the operator.

History is classified by a background pass (`conversation_event_classification_v1` in `web_console_migrations`): rowid-cursor batches of 500, each committed with its cursor, after the projection index (`idx_conversation_events_projection`) is built, so startup never waits on it, an interrupted pass resumes, it is idempotent, and nothing is deleted. On a copy of the live database (651,854 rows, 4 GB) the index took 48 s and the pass 2 min 11 s. Bump the migration name when the recognisers change in a way history should see.

## Work between messages

The server, not the client, projects the list (`Get` with `projection=MESSAGES`, `ListProjectionPage`): a page holds a thread's conversation rows, and before each one a synthesised `work` row (never stored) whose `ConversationWorkSpan` covers every sequence since the previous conversation row and counts that thread's tool calls, results, reasoning and context with their times. The span's range includes other threads' rows, so pages stay contiguous by sequence and the store's gap detection works unchanged; `limit` counts conversation rows, so a page is fifty messages, not fifty tool calls. The newest page also carries the trailing span. Tool text never travels with the list: a span's rows are fetched only when it opens (`GetRange` with `projection=WORK`), as 1,200-character previews.

The line reads "Worked for 2m 12s · 14 tool calls", "Thought for 15s" when the span is reasoning only, and adds "· 2 context updates" for injected context; the duration runs from the conversation row before to the one after, so it is how long the agent took to answer. A span with nothing of this thread draws no line. Tapping the line opens it in place (`CollapsibleRegion`) to one line per row — the tool and its first argument, the result, the reasoning — and each of those opens to its preview. Live rows fold the same way in the store (`foldLiveRow`): the SSE hub sends work rows without their text, and they grow the tail's line.

## Notices

Text a harness or a program put into the conversation that a person should still see — a background task update, an interruption, a slash command, a usage-limit reset, a subagent's report, the auto-reviewer's verdict — is a `notice`: one quiet line in secondary text with an icon, a title from its recogniser label and a one-line summary read out of its markup (the task summary, the command and its arguments, the risk level). Tapping shows the full text. An image the operator attached is an `attachment` line. Neither is ever shown as a message from "You"; injected instructions, environment blocks, skill bodies and compaction summaries are `context` and live in the work line.

## Agent threads

A subagent (Claude's Task/Agent workers and in-process teammates, Codex `thread_spawn` rollouts), a teammate that messages the session, and the approval reviewer each get a thread (`conversation_threads`: id, kind, name, what it was asked, parent). The session's own conversation shows each as one group at the point it began — the work line that covers its first row — with its name, kind, message count and how long it ran, and a line of what it was asked. Tapping the group opens the thread's own conversation in the pane in place of the list (as the reader does), with its work collapsed the same way; Back returns to the list where it was. A teammate's message to the lead files under that teammate's thread; a subagent's brief is filed in its thread as written by the parent (`author=peer`).

## Session activity contract

Every session carries one `SessionActivity`, pushed over the existing SSE hub as kind `session_activity` and included in the initial session list.

```proto
enum SessionActivityState {
  SESSION_ACTIVITY_STATE_UNSPECIFIED = 0;
  SESSION_ACTIVITY_STATE_UNKNOWN = 1;
  SESSION_ACTIVITY_STATE_WORKING = 2;
  SESSION_ACTIVITY_STATE_IDLE = 3;
  SESSION_ACTIVITY_STATE_WAITING = 4;
}
enum SessionActivitySource {
  SESSION_ACTIVITY_SOURCE_UNSPECIFIED = 0;
  SESSION_ACTIVITY_SOURCE_SCREEN = 1;        // emulator screen + PromptDetector
  SESSION_ACTIVITY_SOURCE_OUTPUT_CLOCK = 2;  // LastFrameAt quiet window
  SESSION_ACTIVITY_SOURCE_HOOK = 3;          // Claude Notification/Stop hooks
  SESSION_ACTIVITY_SOURCE_HARNESS_EVENT = 4; // OpenCode SSE
}
message PromptOption { string key = 1; string label = 2; bool selected = 3; }
message PendingPrompt {
  string kind = 1;            // "question" | "permission" | "unknown"
  string text = 2;            // prompt text, empty at level 1
  repeated PromptOption options = 3;
  bool answerable = 4;        // level 3 available for this harness+version
  string free_text_hint = 5;
}
message SessionActivity {
  string session_id = 1;
  SessionActivityState state = 2;
  SessionActivitySource source = 3;
  float confidence = 4;       // 0..1; < 0.6 is reported as UNKNOWN
  google.protobuf.Timestamp since = 5;
  google.protobuf.Timestamp last_output_at = 6;
  PendingPrompt prompt = 7;   // set only when state == WAITING
  string harness = 8;
  string harness_version = 9;
}
```

The detector runs on the server (`api/session/activity.go`) against the server's VT emulator, so every device sees the same state and it works while no browser is open. Evidence priority when inputs disagree: harness event (0.95) > hook (0.9) > screen prompt box (0.85) > screen prompt glyph (0.75) > output clock alone (0.5, reported as `unknown`). The detector evaluates on a 150 ms trailing debounce after output frames and publishes only when the state or prompt changes, plus a 15 s heartbeat while working.

## Maturity levels per harness

| Level | Meaning |
|---|---|
| 0 | Terminal only |
| 1 | Detected: the state is known, the content is not |
| 2 | Rendered: prompt text and options shown read-only |
| 3 | Answerable: options are buttons that send verified keystrokes or API calls |

| Harness | Level |
|---|---|
| Claude Code | 2; 3 on verified versions when the host switch names `claude` (see Answering) |
| OpenCode | 2; 3 for permissions through the OpenCode permission API when the host switch names `opencode` |
| Codex | 1 |
| Grok | 1 |

### Answering (level 3)

Answering is off unless the host turns it on: `WEB_CONSOLE_PROMPT_ANSWERING` lists the harnesses (`claude`, `opencode`), or a `prompt-answering` file in the scenario's data directory holds the same list; the variable wins. The operator sets it per host; nothing ships it on.

- Claude Code prompts are answerable only on a verified version (`api/backends/claude/answerable_versions.go`, read against the host's `claude --version`): versions whose prompt boxes are captured in `testdata/prompts` and whose answers were checked live. An answer types the option's key; Enter follows only if the same prompt is still up about 600 ms later, so a box that resolves on the digit never receives a stray Enter. Cancel (Escape) is offered only when the box's hint says "Esc to cancel".
- OpenCode permissions are answered through its reply API (`POST /permission/{requestID}/reply` with `once`, `always`, or `reject`); OpenCode questions stay at level 2.
- Every answer carries the hash of the prompt it answers (kind, text, and options; not the selection). The server refuses, and sends nothing, when the session is no longer waiting, the prompt changed, answering is off, or the option is not on it.
- The chosen option's label shows as an echo row. The card's buttons stay disabled after an answer until the next activity arrives; a refusal shows its reason and leaves the buttons usable.

## The state slot

Messages renders exactly one presentation under the last row (`data-testid="messages-state-slot"`), outside the virtual list and with `overflow-anchor: none`:

| Activity | Slot |
|---|---|
| `working` | Strip: pulse, "Working · m s" from `since`, "last output N s ago" |
| `waiting`, no prompt text | Level-1 card: "<Speaker> is asking you something", age and source, "Open terminal" |
| `waiting`, prompt text, not answerable | Level-2 card: prompt text, read-only options, "Answer in terminal" |
| `waiting`, answerable | Level-3 card: option buttons that send, Cancel when the prompt allows Escape, plus "Open terminal" |
| `idle`, `unknown`, confidence < 0.6 | Nothing |

## The row and action reveal

The row header is `speaker · time`; the sequence number appears only in the time's tooltip. Speaker comes from the row's author (`speakerKey`/`speakerParams`): the operator is "You" and only the operator's messages sit in the "You" bubble; the agent maps the capture source (`claude_hook` Claude, `codex_tailer` Codex, `grok_tailer` Grok, `opencode_api` OpenCode, otherwise Agent); another agent is named ("serving-fabric"); a launch prompt is "agent-manager · automated". A row from a server that predates authors falls back to its role.

Actions stay declared in `ui/src/components/messages/messageActions.ts`. They are hidden at rest. Revealed, they are the library's ghost icon buttons (no fill or border at rest) sitting right after the timestamp in the row's fixed-height header line, so revealing them never changes the row's height; their tap targets overhang the line. Copy is the library's `CopyIconButton`: the icon morphs to a check in the success colour. On a fine pointer, hover or keyboard focus reveals at most three inline actions; right-click opens the full list. On a coarse pointer a tap reveals the same inline actions and the More button, on one row at a time; tapping the row again hides them, and a tap on a link or control inside the row is that control's. A long-press opens a bottom sheet with every applicable action, primary first. `j`/`k` move focus between rows, `Enter` opens the list, `Escape` closes it. The `audio-settings` action is removed; `read-from-here` and `open-in-reader` exist.

Handoff suggestions render inside the row's measured block. One rule matching several times in one message is one chip: it names the rule and the count, lists the matched file names, hands them all off together, and on request lists each match so one can be handed off or dismissed alone.

## The reader

Rows taller than 400 px end in a gradient and an outline footer ("Open in reader · N words · N headings · N code blocks"). The footer opens the reader, which takes the list's place inside the pane rather than covering the page, so the composer bar below stays usable: the operator drafts a reply while reading. Its header is a back arrow, the find-in-message field, copy, and More, which opens the message's full action list (the same one the row offers, less "Open in reader", plus "Play this message"). The match count and the previous and next match controls appear only once there is a query. The speaker, time, and sequence head the text, which has its own scroll and padding. The footer is previous reply, a wide **Return**, and next reply: a reply is the agent's own message on the main thread, so the operator's messages, work lines, notices and thread groups are all stepped over. Return closes the reader onto the list exactly where it was left, however far the reader stepped. A tap on an arrow steps one reply; a hold past the long-press delay scrubs like a video player's fast-forward — 2×, 4×, 8×, then 16× over about four and a half seconds (`SCRUB_TIERS` in `hooks/useHoldToScrub.ts`). The rate is per second, not per frame, so a slow render skips replies rather than slowing the scrub. While the press is pending a ring fills around the arrow; once scrubbing, the arrow fills and doubles (⏪/⏩), and a readout above the footer shows the speed, its tier, and the place (`12 / 48`) with a position bar. At the first or latest reply the readout says so with a small bump, and the scrub holds there until release; the arrow is only disabled once the finger lifts. Android gets a short vibration on each tier and at an end. Enter or Space held on a focused arrow does the same; the settled place is announced to screen readers once, never per step. Text size is a two-finger pinch on the text or Ctrl/⌘ `+`, `−` and `0` (2 px steps, 12–32 px, 16 px reset) while the reader has focus — these size the reader instead of zooming the page — and the size is kept across readers and reloads. A reply that arrives while the reader is open marks the next-reply button with a count until the reader steps to it; a send never closes the reader. While a phone keyboard is up the footer is hidden, so the text keeps its room.

On touch the reader takes focus itself, so opening it never raises the keyboard; on a keyboard the find field takes it. Back and `Escape` return to the list. The list stays laid out and live underneath (hidden, it would measure every row at zero) and is inert while covered, so `j`/`k` and search never reach it; closing returns focus to the row the reader showed last. A live pane's open reader survives a tab switch for as long as the page is open, never across a reload; the archive's reader is its own. File and diagram viewers opened from the reader stay overlays. There is no inline expand.

## Echo rows and history

Send types the draft verbatim and never presses Enter. For a coding agent that leaves two stages — the draft lands in the agent's input, and a second Enter submits it — so an acknowledged send becomes an **outbox entry** whose state is read from authoritative signals, not guessed from the terminal screen:

| State | Meaning | Label |
|---|---|---|
| `staged` | Acknowledged and sitting in the agent's input, unsubmitted | "typed into terminal, not submitted", with "Press Enter ↵" and "Open terminal" |
| `submitted` | The user pressed Enter, or the agent was observed to start working after the send was staged | "submitted · waiting for the agent to record it" |
| `queued` | Submitted while the agent was already working, so it waits behind the current turn | "queued behind the current turn" |

Transitions are driven by the send acknowledgement (→ `staged`), the user pressing Enter (→ `submitted`/`queued`, decided by the current `useSessionActivityStore` state), the activity stream reporting the agent began working after the send (→ `submitted`, so a submit made outside the console still clears "Press Enter"), and the harness recording the matching user turn (the entry is removed). An entry that is never recorded expires after five minutes.

**Coalescing:** one input box holds one unsubmitted message, so consecutive `staged` sends fold into a single entry (joined by newline), never two rows for two line-sends.

**Attachments:** the entry stores the user's text and the full payload (resolved paths) separately, and the harness match aligns on either — so a send the agent renders as a chip ("[Image #1]", "+1 attachment") still clears instead of stranding a phantom "Press Enter". The entry also shows a "+N attachment" hint.

Echo rows are transient and local; they are never persisted or merged into events. The pane no longer flips to the terminal after a send. The Send button reflects what a press will do — **send** when there is text, **submit** (an Enter glyph) when the box is empty and a staged send is waiting, a muted **enter** otherwise — and a long-press (toolbar) or a dedicated button (full-screen composer) offers "Send and press Enter". The expanded reader carries the same pending pill so a staged send stays submittable without leaving the reply. The composer shows a session state chip that never disables Send. History keeps the last 50 sends per device (localStorage) with Insert and Resend, reachable from the toolbar and the expanded composer.

In Messages the composer bar shows on every device, not only on touch: it is the one composer, with the same draft as the expanded composer. On a desktop it takes the focus the terminal would take, dictation lands in it for review, and a send keeps focus in it. "Send to composer" inserts the message into the bar and focuses it (inside the tap, so a phone's keyboard rises); the expanded composer opens only where the bar is not on screen.

## One search

Session, navigator, and archived search share one repository function (`searchEvents` in `api/conversation_search.go`) over the FTS5 mirror `conversation_events_fts`, with modes `text`, `regex`, and `fuzzy` and flags case, whole word, role and kinds. The Messages view searches the conversation it lists (`kinds` message, attachment, notice); the **Tool calls & context** chip in the filter popover drops the filter. A hit carries its kind and thread: a hit inside work opens the work line that holds it with the hit outlined, and a hit in a subagent's conversation opens that thread. The index narrows; an exact Go matcher decides:

| Mode | Candidates | Match |
|---|---|---|
| `text` | the query's words as prefix terms (`"loop"*`) | the literal query, punctuation included (`%` and `_` are literal); case-folded unless case-sensitive |
| `regex` | words from the pattern's literal runs that begin at a word boundary, as prefix terms | Go RE2; ranges from `FindAllStringIndex` |
| `fuzzy` | word prefixes, `NEAR` for two words | every word begins a word in the text |

A query the index cannot narrow (a regex with no usable literal, punctuation only) scans the newest 20,000 rows and reports `truncated`. Text mode matches word prefixes, so a match inside a word (`settle` in `unsettled`) needs regex. Whole word keeps only matches bounded by non-word characters. An invalid or oversized (> 512 bytes) pattern returns a readable `error` in the response, never a transport error. Each hit carries its role, time, an excerpt, and the match ranges inside it in UTF-16 units.

The navigator lists server hits in sequence order with those ranges for highlighting; a hit outside the loaded window keeps the server's role and time, and status and content filters (which need the event) keep loaded hits only. Its header is the search field, focused when it opens (on a phone a stand-in field focused inside the tap keeps the keyboard up until the sheet's field mounts), with an Export icon beside it; the row below has the `Text`/`Regex`/`Fuzzy` control, `Aa`, and `Whole word`. Nothing is matched client-side: the playback picker, which has no server search behind it, shows no search field. The navigator measures its rows, caps the mobile sheet to the viewport, and keeps its footer visible.

## The pill

Playback is one floating pill over the list that takes no layout height. It reads messages only: "Read from here" queues the messages after the row and skips work and notices, auto-play and auto-summarise take only the agent's main-thread replies, and the unread count is the agent's replies. Collapsed: play/pause, what is playing, elapsed and total, close, and a progress hairline. Expanded: scrub, previous and next, speed, summarize mode, voice, jump to message, and settings. Tap toggles; swipe down dismisses and stops playback; swipe left or right steps messages; drag on the progress zone scrubs. Transport values come from provider events, not polling. Media Session and lock-screen controls keep working. A dismissed pill is gone, not minimized; the next message to speak brings it back. While a reader is open the pill rises by the reader footer's height (`--wc-reader-footer-h`), so it never covers the footer's controls.

## Scroll invariants

- Only the scroll handler writes `follow`, and only for user scrolls (never during a programmatic scroll).
- New content moves the viewport only while `follow` is true.
- The only anchors are the virtualizer's resize compensation and the prepend anchor.
- Neither anchor writes scrollTop while the user scrolls; the rows carry the correction until the scroll ends (a write would stop an iOS fling).
- The jump-to-bottom button and the new-messages pill are centred by a full-width row, never by a transform.
- Position restore loads the page containing the saved message before it scrolls, once, with no retries.

## Decisions

- **D1 — Two views, one toggle:** Terminal and Messages stay separate; the single icon toggle stays where it is.
- **D2 — Messages is a projection:** The view shows what capture understands and offers the terminal for everything else; `unknown` renders nothing.
- **D3 — Detect before parse before answer:** Each harness signal ships at level 1 before level 2 before level 3. A wrong answer sent to an agent is worse than a tap.
- **D4 — One follow boolean:** The scroll model is `follow` plus the prepend anchor and the virtualizer's resize compensation.
- **D5 — Restore pages before it scrolls:** Position restore loads the page containing the saved event and never falls back to the bottom silently.
- **D6 — Persist view mode and position in a dedicated store:** `useMessagesViewStore` (`wc-messages-view`) with its own migration ladder.
- **D7 — Reader instead of inline expand, in the pane:** Long replies open a reader that replaces the list inside the pane on every platform, never a page overlay, so a reply can be drafted while reading.
- **D8 — Actions hidden at rest:** Hover or focus reveals up to three inline actions on fine pointers; long-press opens a sheet on coarse pointers; hit targets stay 44 px.
- **D9 — Send never presses Enter:** Enter stays a separate, visible action; no setting changes the default.
- **D10 — Sends are never blocked and never routed:** The composer labels state and outcome; it does not gate or redirect bytes.
- **D11 — Echo rows are an authoritative, transient outbox:** State comes from the send acknowledgement, the session activity stream, and the user's own Enter — not from matching the terminal screen. They resolve against the harness's own user event (text or payload, same session, within the match window) or expire. They are never persisted or merged into events.
- **D12 — No post-send view flip.**
- **D13 — History is per device:** 50 entries in localStorage, no server sync.
- **D14 — One search path on FTS5:** Bounded server-side regex over FTS candidates.
- **D15 — Playback is a floating pill that stops when dismissed.**
- **D16 — Transport by events, not polling.**
- **D17 — Prompt and activity detection are server-side.** The console consumes the session activity stream (`working`/`idle`/`waiting`) rather than reading the client's xterm buffer; the old screen-scrape "still on screen" check for echo rows has been removed.
- **D18 — Activity is pushed on the existing SSE hub** as kind `session_activity`.
- **D19 — Pill, reader, and state slot are scenario-local;** recorded in `docs/reference/component-library-gaps.md`.
- **D20 — The prior plan `web-console-messages-view-open-reliably-at-thousands-of` is superseded.**
- **D21 — One composer in Messages:** The composer bar shows in Messages on every device; there is no composer inside the reader, and "Send to composer" stages into the bar.
- **D22 — Classify at ingestion, from native structure first:** Kind, author and thread are stored on the row by the shared parser; text recognisers are a data table per harness, and history is reclassified by an idempotent background pass rather than at read time. Switch cost: a new recogniser needs a migration-name bump to reach history.
- **D23 — The server projects; the client never holds raw tool rows:** Paging counts conversation rows and work rows are synthesised per page, because 58% of rows are tool rows averaging 6 KB. Work rows cover whole sequence ranges so the store's contiguity rules hold. Alternative considered: client-side folding over full pages (rejected: pages of tool output on a phone).
- **D24 — Work is one secondary line between messages, opened in place:** Not a separate tab or a toggle that re-lists everything; its rows load on demand as previews.
- **D25 — Injected content is a notice or context, never "You":** Only `author=human` renders as the operator. A launch prompt is the launching program's.
- **D26 — Agent threads are groups that open their own conversation:** Placed where the thread began, opened in place of the list like the reader. Codex and Claude subagent history captured before threads existed stays interleaved; teammate messages found in history text are threaded by the backfill.
- **D27 — Search defaults to the conversation:** The kinds filter is server-side; including work is one chip.

## Native control and projection boundary

Messages is a semantic projection, not a PTY transcript. Conversation rows come
from the Claude, Codex, Grok, and OpenCode adapters when those adapters can
identify a natural-language turn. Terminal activity detection reads the live
server-side emulator only to classify working, idle, or waiting state; it does
not manufacture transcript rows. Generic event logs, redraws, tool UI, and
unsupported prompts remain terminal-only.

Each projected event may carry optional harness-owned native provenance
(`provider`, `session_id`, `turn_id`, `message_id`, `boundary_id`, and
`compaction_lineage`). Web Console ids and sequences remain display and paging
coordinates. Missing or malformed provenance makes native control unavailable;
the UI must offer the terminal instead of inferring a command from text.

The conversation-control preflight endpoint is read-only and returns stable
reason codes (`unsupported`, `unavailable`, `stale_target`, and
`confirmation_required`) plus consequences and an expiry. Execution is
fail-closed until a harness adapter is verified to stop safely, invoke its own
rewind operation, and prove the native postcondition. A timeout or transport
loss is never rendered as success, and projected history is preserved unless
native verification completes.

OpenCode is the first verified native adapter. The managed `opencode serve`
owner exposes `POST /session/:id/revert` with persisted native `messageID` and
`partID`. Web Console checks session status, records a durable operation
receipt, aborts a busy turn through the owner API with a bounded wait, invokes
native revert, and verifies the returned `revert.messageID` before projection
reconciliation. A compaction lineage, changed ownership, timeout, or uncertain
postcondition preserves projected history and prevents blind replay; an
identical request replays the durable result without repeating the side effect.
The installed owner was OpenCode 1.18.30 during validation. Claude, Codex, and
Grok remain explicitly unsupported until their owners expose an equivalent
verified operation.

OpenCode rows reach Messages only after the pane is bound to its native
session, and that binding is by identity, not inference. For OpenCode 2 the
launcher preselects the session id (`--session <id>`) and records it in the
pane's state (`opencode/<pane id>/binding.json`); the capture owner binds the
pane when that session appears. Directory + time attribution remains only as a
fallback for OpenCode 1 and `--continue`, and it ignores subagent sessions. On
2026-11-24 that fallback, fed by subagent sessions in the shared repository
root, left one pane ambiguous and then every later pane, so no OpenCode pane
showed messages until bindings replaced it (validated against OpenCode 2.3.2;
see `docs/guides/CONVERSATION_TRACKING.md`). The reproducible served-browser safety-dialog case is
`bas/cases/02-messages/05-rewind/message-rewind-safety-dialog.json`.

## Mockups

- Round-two mockup page: `/home/matthalloran8/.vrooli/plan-artifacts/web-console-messages-view-honest-projection-calm-scrolling/messages-view-redesign-round2.html` (published at https://claude.ai/code/artifact/5be808f9-9665-47e0-8d64-338e3bb7e468).
- Operator decisions: `/home/matthalloran8/.vrooli/plan-artifacts/web-console-messages-view-honest-projection-calm-scrolling/operator-decisions-2026-09-11.md`.
