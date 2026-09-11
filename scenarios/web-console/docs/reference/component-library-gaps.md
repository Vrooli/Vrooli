# Web Console component-library gaps

This product keeps its specialized frame because the library shell does not
carry the interaction model described in the review record.

```shell-ejection
{"archetype":"navigated-console","reason":"Web Console is a terminal/workspace product with panes, composer, audio/handedness preferences, continuity banners, and terminal headers. Its workspace frame is the product interaction surface and AppShell/2 cannot replace it without losing the terminal contract.","files":["ui/src/App.tsx","ui/src/components/WorkspacePaneShell.tsx","ui/src/components/TerminalHeader.tsx"]}
```

## Scenario-local message components

These stay in Web Console because their behaviour is bound to conversation
events, the message-action registry, and playback. Each notes the generic part
that is a candidate for the component library.

| Component | Why it is local | Canon candidate |
|---|---|---|
| `ui/src/components/messages/MessageRow.tsx` | Renders one conversation event: speaker from the capture source, reveal-on-demand actions from `messageActions.ts`, and the 400 px cap with a reader footer that keeps virtualized row heights stable. | Hover/focus reveal cluster with 44 px hit areas around 28 px glyphs. |
| `ui/src/components/messages/MessageActionList.tsx` | Maps the message-action registry onto `ContextMenu/1` (fine pointer) and `BottomSheet/1` (coarse pointer). | None beyond the two library components it composes. |
| `ui/src/components/messages/MessagesReader.tsx` | A `FullPageDrawer/1` whose header actions (copy, play) and body are bound to one conversation event and the playback queue. | `findInText.ts`: find-in-rendered-text with the CSS Custom Highlight API and a `<mark>` fallback. |
| `ui/src/components/messages/SessionStateSlot.tsx` | Renders one session activity (the working strip and the level-1, -2 and -3 prompt cards) from the session-activity contract, and answers through `TerminalService.AnswerPrompt`. | A status card with a timed strip and an options row is generic; the levels and the answer flow are not. |

## Scenario-local playback components

| Component | Why it is local | Canon candidate |
|---|---|---|
| `ui/src/components/tts/PlaybackPill.tsx` (with `PlaybackPillExpanded.tsx`) | Bound to the playback queue, the pane's transport store, and conversation events (speaker, time, jump to message, summarize mode). | The collapsed/expanded floating pill shell. |
| `ui/src/components/tts/usePillGestures.ts` | Written for the pill's thresholds. | Pointer-event tap, dismiss-drag, track-swipe, and scrub-zone gestures with an axis lock; reuses `usePressGesture`'s move threshold. |

