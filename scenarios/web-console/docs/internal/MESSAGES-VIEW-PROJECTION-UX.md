# Messages View: Projection UX

Design record for the Messages view of Web Console: what it shows, how it scrolls, how sends stay visible, how search and playback work, and which decisions are pinned. Requirement module: `requirements/17-messages-view-projection-and-playback/module.json` (`MOD-P0-017`).

## Purpose

Web Console is the operator's daily surface for supervising coding agents. The Messages view makes a long agent session readable on a phone. This record defines the view as an honest projection of the agent session: calm to scroll, safe to send from, and fronted by one playback pill.

## The projection principle

Messages shows only what capture understands, and says so when it does not. The terminal stays the ground truth; the view toggle between Terminal and Messages is unchanged. A state the projection cannot render (a tool call, a prompt a harness does not expose) is shown as a card that hands the user to the terminal, never as silence and never as a guess.

Only natural-language turns are stored as conversation events. Session activity (below) explains the time between turns.

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

The row header is `speaker · time`; the sequence number appears only in the time's tooltip. Speaker comes from `speakerLabel(event)`: user rows are "You"; assistant rows map the capture source (`claude_hook` Claude, `codex_tailer` Codex, `grok_tailer` Grok, `opencode_api` OpenCode, otherwise Agent).

Actions stay declared in `ui/src/components/messages/messageActions.ts`. They are hidden at rest. On a fine pointer, hover or keyboard focus reveals at most three inline actions with 44 px hit areas; right-click opens the full list. On a coarse pointer a tap reveals the same inline actions and the More button, on one row at a time; tapping the row again hides them, and a tap on a link or control inside the row is that control's. A long-press opens a bottom sheet with every applicable action, primary first. `j`/`k` move focus between rows, `Enter` opens the list, `Escape` closes it. The `audio-settings` action is removed; `read-from-here` and `open-in-reader` exist.

## The reader

Rows taller than 400 px end in a gradient and an outline footer ("Open in reader · N words · N headings · N code blocks"). The footer opens a full-height reader with its own scroll and padded text. Its header has copy, play, and More, which opens the message's full action list (the same one the row offers, less "Open in reader"). Find-in-message is one field; the match count and the previous and next match controls appear only once there is a query. The reader's footer steps to the previous and next reply (the operator's own messages are skipped) and changes the text size in 2 px steps from 12 to 32 px; a two-finger pinch on the text does the same, and the size is kept across readers and reloads. The list stays mounted underneath and does not move; closing returns focus to the row the reader showed last. There is no inline expand.

## Echo rows and history

Send types the draft verbatim and never presses Enter. After a send is acknowledged, Messages shows a dimmed echo row for it:

| Condition | Label |
|---|---|
| Just acknowledged | "sending…" |
| Text still visible on the terminal screen (a line the terminal soft-wrapped counts), no matching user event | "typed into terminal, not submitted", with "Press Enter ↵" and "Open terminal" |
| Matching user event arrives (normalised text, same session, 60 s) | The echo is replaced by the real row |
| Neither within 60 s | "sent · not seen on screen" |

Echo rows are transient and local; they are never persisted or merged into events. The pane no longer flips to the terminal after a send. A long-press on Send offers "Send and press Enter" as an explicit secondary action. The composer shows a session state chip that never disables Send. History keeps the last 50 sends per device (localStorage) with Insert and Resend, reachable from the toolbar and the expanded composer.

## One search

Session, navigator, and archived search share one repository function (`searchEvents` in `api/conversation_search.go`) over the FTS5 mirror `conversation_events_fts`, with modes `text`, `regex`, and `fuzzy` and flags case, whole word, and role. The index narrows; an exact Go matcher decides:

| Mode | Candidates | Match |
|---|---|---|
| `text` | the query's words as prefix terms (`"loop"*`) | the literal query, punctuation included (`%` and `_` are literal); case-folded unless case-sensitive |
| `regex` | words from the pattern's literal runs that begin at a word boundary, as prefix terms | Go RE2; ranges from `FindAllStringIndex` |
| `fuzzy` | word prefixes, `NEAR` for two words | every word begins a word in the text |

A query the index cannot narrow (a regex with no usable literal, punctuation only) scans the newest 20,000 rows and reports `truncated`. Text mode matches word prefixes, so a match inside a word (`settle` in `unsettled`) needs regex. Whole word keeps only matches bounded by non-word characters. An invalid or oversized (> 512 bytes) pattern returns a readable `error` in the response, never a transport error. Each hit carries its role, time, an excerpt, and the match ranges inside it in UTF-16 units.

The navigator lists server hits in sequence order with those ranges for highlighting; a hit outside the loaded window keeps the server's role and time, and status and content filters (which need the event) keep loaded hits only. Its header is the search field, focused when it opens (on a phone a stand-in field focused inside the tap keeps the keyboard up until the sheet's field mounts), with an Export icon beside it; the row below has the `Text`/`Regex`/`Fuzzy` control, `Aa`, and `Whole word`. Nothing is matched client-side: the playback picker, which has no server search behind it, shows no search field. The navigator measures its rows, caps the mobile sheet to the viewport, and keeps its footer visible.

## The pill

Playback is one floating pill over the list that takes no layout height. Collapsed: play/pause, what is playing, elapsed and total, close, and a progress hairline. Expanded: scrub, previous and next, speed, summarize mode, voice, jump to message, and settings. Tap toggles; swipe down dismisses and stops playback; swipe left or right steps messages; drag on the progress zone scrubs. Transport values come from provider events, not polling. Media Session and lock-screen controls keep working. The restore button (`tts-restore`) resumes a dismissed queue.

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
- **D7 — Reader instead of inline expand:** Long replies open a full-height reader on every platform.
- **D8 — Actions hidden at rest:** Hover or focus reveals up to three inline actions on fine pointers; long-press opens a sheet on coarse pointers; hit targets stay 44 px.
- **D9 — Send never presses Enter:** Enter stays a separate, visible action; no setting changes the default.
- **D10 — Sends are never blocked and never routed:** The composer labels state and outcome; it does not gate or redirect bytes.
- **D11 — Echo rows are transient and local:** They resolve against the harness's own user event by normalised text within 60 s.
- **D12 — No post-send view flip.**
- **D13 — History is per device:** 50 entries in localStorage, no server sync.
- **D14 — One search path on FTS5:** Bounded server-side regex over FTS candidates.
- **D15 — Playback is a floating pill that stops when dismissed.**
- **D16 — Transport by events, not polling.**
- **D17 — Prompt detection is server-side.** The client's xterm buffer is read only for the echo row's "still on screen" check.
- **D18 — Activity is pushed on the existing SSE hub** as kind `session_activity`.
- **D19 — Pill, reader, and state slot are scenario-local;** recorded in `docs/reference/component-library-gaps.md`.
- **D20 — The prior plan `web-console-messages-view-open-reliably-at-thousands-of` is superseded.**

## Mockups

- Round-two mockup page: `/home/matthalloran8/.vrooli/plan-artifacts/web-console-messages-view-honest-projection-calm-scrolling/messages-view-redesign-round2.html` (published at https://claude.ai/code/artifact/5be808f9-9665-47e0-8d64-338e3bb7e468).
- Operator decisions: `/home/matthalloran8/.vrooli/plan-artifacts/web-console-messages-view-honest-projection-calm-scrolling/operator-decisions-2026-09-11.md`.
