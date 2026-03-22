# Persistent Voice Mode with Voice Commands — Implementation Plan

## 1. Purpose

Add a "persistent voice mode" to the web-console scenario where pressing the mic button toggles always-on listening (instead of one-shot recording). In this mode, continuous speech flows into the text input as dictation, while utterances prefixed with a configurable keyword are recognized as terminal commands (tab switching, key combos, etc.) and surfaced as confirmable suggestions above the mobile toolbar.

This plan also addresses a transcription quality gap: in persistent mode the current "final pass" (full-buffer retranscription on stop) never fires, so we introduce silence-bounded segment finals — VAD-detected pauses trigger high-quality retranscription of each speech segment, replacing the rough streaming partials.

## Greenfield Constraint

This is a **greenfield** implementation. There is no prior version of persistent voice mode or voice commands to maintain compatibility with.

**What this means:**
- **No compatibility shims**: Do not add backward-compatibility wrappers, feature flags to gate old-vs-new behavior, or migration bridges. Write the correct implementation directly.
- **No legacy code**: Do not leave dead code paths, commented-out old logic, or unused types/interfaces behind. If something is replaced, delete it entirely.
- **No technical debt by default**: Every abstraction must earn its place by solving a current need, not a hypothetical future one. If you wouldn't write it from scratch today, don't keep it.
- **Clean integration with existing code**: The existing one-shot voice mode is not legacy — it remains a supported mode. New code integrates with it cleanly (e.g., shared VAD, shared providers), but persistent mode logic should not be shoehorned into existing functions via flags. Prefer new, focused functions/modules over bolting conditional branches onto existing ones.

## 2. Required Reading

```bash
# Skills
prompt-manager skill read implementation-plan-authoring documentation-health seam-discovery-and-enforcement test interoperability-steer utils-unification boundary-of-responsibility-enforcement

# Existing voice implementation (read ALL of these before starting)
cat scenarios/web-console/ui/src/hooks/useVoiceInput.ts
cat scenarios/web-console/ui/src/hooks/voice/vad.ts
cat scenarios/web-console/ui/src/hooks/voice/types.ts
cat scenarios/web-console/ui/src/hooks/voice/VoiceStreamProvider.ts
cat scenarios/web-console/ui/src/hooks/voice/WhisperProvider.ts
cat scenarios/web-console/ui/src/hooks/voice/WebSpeechProvider.ts
cat scenarios/web-console/ui/src/hooks/voice/audioUtils.ts
cat scenarios/web-console/ui/src/hooks/voice/audioCues.ts
cat scenarios/web-console/ui/src/components/VoiceMicButton.tsx
cat scenarios/web-console/ui/src/components/MobileToolbar.tsx
cat scenarios/web-console/ui/src/components/AiSuggestBar.tsx
cat scenarios/web-console/ui/src/components/settings/VoiceInputSection.tsx
cat scenarios/web-console/api/voice_stream_ws.go
cat scenarios/web-console/api/voice_config.go
cat scenarios/web-console/api/voice_transcribe.go

# Existing seams & architecture
cat scenarios/web-console/docs/internal/SEAMS.md
cat scenarios/web-console/docs/internal/TEMPORAL-FLOWS.md
cat scenarios/web-console/docs/internal/INVARIANTS.md
```

## 3. Problem Statement

**Current behavior:** The mic button starts a one-shot recording session. Speech is transcribed via streaming partials (fast, rough) and a single final retranscription (accurate) when recording stops. VAD auto-stop ends the recording after a configurable silence timeout. The result is placed in the text input or sent to the terminal.

**Desired behavior:** A toggle mode where pressing the mic enters persistent listening. The user speaks freely — continuous dictation flows into the text input with high accuracy. When the user says a configurable prefix keyword followed by a command (e.g., "zap new tab"), it's recognized as a command and shown as a confirmable action above the mobile toolbar. Pressing the mic again exits the mode.

**Key challenges:**
1. In persistent mode, there's no recording-stop event to trigger a final retranscription, so accuracy degrades to partial-only quality
2. Commands must be reliably distinguished from dictation text
3. The VAD silence detection currently triggers recording stop — it needs to be repurposed as a segment boundary trigger instead
4. Streaming partials may split a command prefix across two chunks

## 4. Scope

### In scope
- Persistent voice mode toggle (mic on/off)
- Two-tier transcription: fast partials for visual feedback + silence-bounded segment finals for accuracy
- Configurable command prefix keyword
- Fixed initial command vocabulary (tab management, key combos)
- Command suggestion UI above mobile toolbar with tap-to-execute
- Settings for persistent mode, prefix keyword, and segment silence threshold
- Unit and integration tests for new logic
- SEAMS.md updates

### Out of scope
- Wake-word detection (no always-on-from-app-open)
- Auto-execute countdown timer (V2 feature)
- Custom/extensible command vocabulary beyond the initial fixed set
- Desktop-specific command UI (desktop can use the same suggestion bar)
- Changes to the Whisper server itself
- TTS integration with commands

## 5. Current Technical Context

### Key files and their roles

| File | Role | Changes needed |
|------|------|----------------|
| `ui/src/hooks/voice/vad.ts` | Pure VAD functions: calibration, silence detection, state transitions | Add `"segment-boundary"` return value alongside existing `"stop"` and `"no-speech"` |
| `ui/src/hooks/voice/types.ts` | Shared voice types and constants | Add persistent mode types, command types, segment types |
| `ui/src/hooks/useVoiceInput.ts` | Main orchestrator hook — state machine, provider lifecycle, VAD integration | Major changes: persistent mode state machine, segment management, command detection |
| `ui/src/hooks/voice/VoiceStreamProvider.ts` | WebSocket streaming provider | Adapt for segment-aware streaming (reset buffer on segment boundary, request segment finals) |
| `api/voice_stream_ws.go` | Backend WebSocket handler — partial ticker, final retranscription, deduplication | Add segment-final message type, support continuous session with multiple segment finals |
| `ui/src/components/VoiceMicButton.tsx` | Mic button UI | Add persistent mode visual state (different color/animation for "listening") |
| `ui/src/components/MobileToolbar.tsx` | Mobile toolbar | Add command suggestion bar slot |
| `ui/src/components/settings/VoiceInputSection.tsx` | Voice settings | Add persistent mode toggle, prefix keyword input, segment threshold slider |
| `api/voice_config.go` | Voice config persistence | Add persistent mode config fields |

### Existing seams to leverage
- **VAD pure functions** (`vad.ts`): `vadTick()` already returns action strings — adding `"segment-boundary"` is a clean extension
- **TranscriptionProvider interface**: All 3 providers implement `start/stop/dispose/onResult/onPartial` — persistent mode extends this contract
- **VoiceStreamProvider WebSocket protocol**: Already has `partial`/`final`/`done`/`error` message types — adding `segment-final` is additive
- **MobileToolbar imperative handle**: `appendText()` already exists for inserting transcription results
- **AiSuggestBar component**: Existing suggestion display pattern above toolbar — command suggestions follow same UX pattern

## 6. Target End State

### User experience
1. User taps mic button → enters persistent listening mode (button turns green/cyan with a persistent pulse)
2. User speaks naturally → text appears in real-time in the text input (partials for feedback, segment finals for accuracy)
3. User pauses → VAD detects silence → segment final fires, replacing rough partials with accurate text for that segment
4. User says "[prefix] new tab" → command is detected in the segment final → suggestion appears above toolbar: "New Tab ✓ ✕"
5. User taps ✓ → command executes (new tab created)
6. User taps mic again → persistent mode ends, final segment is transcribed

### Architecture
```
Mic audio (continuous while persistent mode active)
  │
  ├─→ Tier 1: Streaming partials (every 500ms)
  │     → Live text feedback in UI (same as today)
  │
  └─→ VAD monitors silence gaps
        │
        ├─ Silence < segment threshold → continue accumulating
        │
        └─ Silence ≥ segment threshold →
             │
             ├─→ Backend: segment-final transcription
             │     → Transcode segment audio to 16kHz WAV
             │     → Single Whisper pass (high accuracy)
             │     → Send "segment-final" message to client
             │
             └─→ Client: receive segment-final
                   │
                   ├─ Check for command prefix
                   │   ├─ Found → parse command → show suggestion UI
                   │   └─ Not found → replace partials with final text in input
                   │
                   └─ Reset segment buffer, continue listening
```

## 7. Implementation Strategy

### Phase 1: Two-Tier Segment Transcription (Backend + VAD)

**Goal:** Establish the segment-final transcription pipeline without changing the UI mode yet. This can be tested by adding segment-boundary detection to the existing one-shot recording flow.

#### 1a. Extend VAD to emit segment boundaries (`vad.ts`)

- Add a new return value `"segment-boundary"` from `vadTick()` when silence exceeds a configurable `segmentSilenceMs` threshold (separate from the existing `vadSilenceTimeoutMs`)
- The segment boundary fires **before** the auto-stop timeout: `segmentSilenceMs` (default 1.5s) < `vadSilenceTimeoutMs` (default 2s)
- After emitting `"segment-boundary"`, reset the VAD silence timer so it can detect the next segment or eventually fire `"stop"` if silence continues beyond `vadSilenceTimeoutMs`
- Add `segmentSilenceMs` to `VadConfig` type

**Key design decision:** In persistent mode, the `"stop"` action from VAD is suppressed — only `"segment-boundary"` fires. The user must manually tap the mic to stop. In one-shot mode, behavior is unchanged (both can fire, stop takes precedence at its threshold).

#### 1b. Extend WebSocket protocol (`voice_stream_ws.go`)

- Add a new client→server text message: `{ type: "segment-boundary" }`
- On receiving `segment-boundary`:
  1. Pause the partial ticker
  2. Take a snapshot of the current audio buffer (from segment start offset to current position)
  3. Transcode the snapshot to 16kHz WAV via existing `transcodeToWav()`
  4. Send to Whisper with empty `initial_prompt` for maximum accuracy
  5. Send server→client message: `{ type: "segment-final", text: "...", segmentIndex: N }`
  6. Update the segment start offset to current buffer position
  7. Resume the partial ticker for the next segment
- Keep the accumulated transcript across segments so `initial_prompt` for partials in the next segment has context
- Add `segmentIndex` counter to the session for ordering

#### 1c. Extend VoiceStreamProvider (`VoiceStreamProvider.ts`)

- Add method `sendSegmentBoundary()` that sends the `{ type: "segment-boundary" }` message
- Add `onSegmentFinal` callback: `(text: string, segmentIndex: number) => void`
- Handle incoming `segment-final` messages from the WebSocket

#### 1d. Wire segment boundaries in useVoiceInput (`useVoiceInput.ts`)

- When `vadTick()` returns `"segment-boundary"`, call `provider.sendSegmentBoundary()`
- On `onSegmentFinal` callback: replace the current partial text for that segment with the final text
- Track segment text accumulation: `segments: Array<{ text: string; isFinal: boolean }>`

**Testing:**
- Unit test `vadTick()` segment-boundary emission with mock time progression
- Unit test backend segment-final flow with mock Whisper responses
- Integration test: simulated audio → verify segment-final replaces partials

### Phase 2: Persistent Voice Mode

**Goal:** Add the toggle behavior — mic button stays active until pressed again.

#### 2a. Add persistent mode state to useVoiceInput

- New state: `voiceMode: "one-shot" | "persistent"` (configurable in settings, default "one-shot")
- In persistent mode:
  - `startRecording()` enters persistent listening state
  - VAD `"stop"` action is suppressed — silence only triggers segment boundaries
  - `stopRecording()` is only called on explicit mic tap
  - On stop: send `"done"` for the final segment, await its final transcription
- New `voiceState` value: `"listening"` (distinct from `"recording"` — indicates persistent mode is active)

#### 2b. Update VoiceMicButton for persistent mode

- New visual state for `"listening"`: green/cyan color, subtle pulse (distinct from red recording pulse)
- Tap toggles between listening and idle (no push-to-talk in persistent mode)
- Show segment count or duration indicator
- Partial transcript display continues to work as today

#### 2c. Update VoiceInputSection settings

- Add "Persistent mode" toggle (default off)
- Add "Segment silence threshold" slider (0.8s – 3s, default 1.5s) — only visible when persistent mode enabled
- Help text explaining the mode

#### 2d. Update voice_config.go

- Add `PersistentMode bool` and `SegmentSilenceMs int` to config struct
- Persist to `voice-config.json`
- Add validation (segment silence must be less than auto-stop silence when both enabled)

**Testing:**
- Unit test persistent mode state transitions in useVoiceInput
- Unit test that VAD stop is suppressed in persistent mode
- Unit test settings persistence round-trip
- Manual test: toggle persistent mode, speak in segments, verify segment finals fire

### Phase 3: Command Detection and Execution

**Goal:** Parse segment-final text for command prefix, map to terminal actions, show confirmable suggestions.

#### 3a. Define command vocabulary (`ui/src/hooks/voice/commands.ts` — new file)

```typescript
interface VoiceCommand {
  id: string;
  patterns: string[];        // e.g., ["new tab", "add tab", "open tab"]
  execute: (context: CommandContext) => void;
  description: string;       // For suggestion UI display
}

interface CommandContext {
  sessionManager: SessionManagerHandle;
  sendToTerminal: (data: string) => void;
  // ... other action handles
}
```

**Initial command set:**

| Command | Patterns | Action |
|---------|----------|--------|
| New tab | "new tab", "add tab", "open tab" | Create new terminal session |
| Switch tab N | "tab [number]", "switch tab [number]", "go to tab [number]" | Switch to pane by index |
| Close tab | "close tab", "close this tab" | Close active pane |
| Send | "send", "enter", "submit" | Press Enter in terminal |
| Cancel | "cancel", "interrupt", "stop" | Send Ctrl-C |
| Copy | "copy" | Send Ctrl-Shift-C (terminal copy) |
| Paste | "paste" | Send Ctrl-Shift-V (terminal paste) |
| Clear | "clear", "clear screen" | Send Ctrl-L |
| Tab key | "tab key", "autocomplete" | Send Tab character |
| Scroll up | "scroll up" | Send Shift-PageUp |
| Scroll down | "scroll down" | Send Shift-PageDown |
| Stop listening | "stop listening", "mic off" | Exit persistent mode |

#### 3b. Command parser (`ui/src/hooks/voice/commandParser.ts` — new file)

- `parseCommand(text: string, prefix: string): ParsedCommand | null`
- Strip the prefix from the beginning of the text (case-insensitive)
- Fuzzy-match the remainder against command patterns (handle Whisper misrecognitions like "knew tab" → "new tab")
- Use Levenshtein distance with a threshold (e.g., max 2 edits for short commands)
- Return `{ command: VoiceCommand, confidence: number, rawText: string }` or null

**Key design decision:** Command detection runs only on segment-final text (not partials). This ensures we're parsing the highest-quality transcription and avoids false positives from incomplete partials.

#### 3c. Adaptive segment silence for commands

When the partial transcript stream detects the command prefix (even in rough partials), temporarily reduce the segment silence threshold to ~600-800ms so commands feel snappy. Reset to the normal threshold after the segment final resolves.

- In `useVoiceInput`: watch partial transcript for prefix match
- If prefix detected in partial: set `vadRef.current.segmentSilenceMs` to a shorter value
- On segment final (whether command or not): restore original value

#### 3d. Command suggestion UI component (`ui/src/components/VoiceCommandSuggestion.tsx` — new file)

- Rendered above the MobileToolbar (same slot as AiSuggestBar, or adjacent)
- Shows: command icon + description + confirm (✓) + dismiss (✕) buttons
- Auto-dismiss after 5 seconds if not acted on
- Only one suggestion visible at a time (queue if multiple fire quickly)
- On confirm: execute the command via CommandContext
- On dismiss: discard and optionally append the raw text to the input instead

#### 3e. Wire command detection into useVoiceInput

- On `onSegmentFinal`:
  1. Run `parseCommand(text, prefix)`
  2. If command detected with sufficient confidence: emit command suggestion event
  3. If no command: append final text to accumulated dictation
- New callback: `onCommandSuggestion: (suggestion: CommandSuggestion) => void`
- MobileToolbar receives command suggestions and renders VoiceCommandSuggestion

#### 3f. Update MobileToolbar to show command suggestions

- Add a `commandSuggestion` prop slot
- Render VoiceCommandSuggestion above the toolbar when present
- Animate in/out with a slide transition

**Testing:**
- Unit test `parseCommand()` with exact matches, fuzzy matches, non-matches
- Unit test command execution for each command in the vocabulary
- Unit test adaptive silence threshold switching
- Unit test VoiceCommandSuggestion component rendering and interaction
- Integration test: segment final with prefix → command suggestion appears → confirm → action executes

### Phase 4: Settings and Configuration

**Goal:** Expose all new configuration to users in a coherent settings UI.

#### 4a. VoiceInputSection additions

- **Persistent mode** toggle with description
- **Command prefix** text input (default: configurable, suggest "hey do" or "run" as defaults)
- **Segment silence threshold** slider (0.8s – 3s, only shown in persistent mode)
- **Command vocabulary** display (read-only list of available commands and their trigger phrases)

#### 4b. Voice config schema updates

Add to VoiceStreamConfig (both frontend type and backend struct):
```
persistentMode: boolean     (default: false)
commandPrefix: string       (default: "hey do")
segmentSilenceMs: number    (default: 1500, range: 800-3000)
```

### Phase 5: Documentation and Seams

**Goal:** Update internal docs to reflect the new architecture.

#### 5a. Update SEAMS.md

- Add Voice Command subsystem under Section 1 (Entry/Presentation)
- Document the segment-final seam in the voice pipeline
- Document the command parser as a testability boundary
- Document the command execution context as an integration seam

#### 5b. Update TEMPORAL-FLOWS.md

- Add persistent voice mode timing flow
- Document the two-tier transcription timing (partial vs. segment-final)
- Document adaptive silence threshold behavior

#### 5c. Update PROBLEMS.md / PROGRESS.md

- Record completion of persistent voice mode feature
- Note any known limitations or edge cases discovered during implementation

## 8. Contract Decisions

### WebSocket Protocol Extension

New message types (additive, backward compatible):

**Client → Server:**
```json
{ "type": "segment-boundary" }
```

**Server → Client:**
```json
{ "type": "segment-final", "text": "transcribed segment text", "segmentIndex": 0 }
```

### Voice Config API Extension

`GET/PUT /api/v1/voice/config` — existing endpoint, extended payload:

```json
{
  "flushIntervalMs": 500,
  "minDeltaBytes": 4096,
  "overlapBytes": 2048,
  "persistentMode": false,
  "commandPrefix": "hey do",
  "segmentSilenceMs": 1500
}
```

### Command Suggestion Event Shape

Internal UI event (not an API contract):

```typescript
interface CommandSuggestion {
  id: string;
  command: VoiceCommand;
  confidence: number;
  rawText: string;
  timestamp: number;
}
```

## 9. Testing Plan

### Unit Tests

| Component | Test file | Key cases |
|-----------|-----------|-----------|
| `vadTick()` segment boundary | `vad.test.ts` | Silence crosses segment threshold → returns "segment-boundary"; resets timer after; doesn't fire in one-shot mode when stop fires first |
| `parseCommand()` | `commandParser.test.ts` | Exact match, fuzzy match (Levenshtein ≤ 2), no match, prefix stripping, case insensitivity, number extraction ("tab 3") |
| Command execution | `commands.test.ts` | Each command maps to correct action call on CommandContext |
| Segment text accumulation | `useVoiceInput.test.ts` | Partials accumulate per-segment; segment-final replaces partials; segments concatenate in order |
| Adaptive silence threshold | `useVoiceInput.test.ts` | Prefix detected in partial → threshold drops; segment final → threshold restores |
| VoiceCommandSuggestion | `VoiceCommandSuggestion.test.tsx` | Renders command, confirm executes, dismiss discards, auto-dismiss after timeout |
| Config round-trip | `voice_config_test.go` | New fields persist and validate (segment silence < auto-stop silence) |

### Integration Tests

| Flow | Validation |
|------|------------|
| Persistent mode lifecycle | Start → speak → segments fire → stop → all segments finalized |
| Command flow end-to-end | Speak with prefix → segment final detects command → suggestion shown → confirm → action executes |
| Fallback behavior | Persistent mode with no Whisper → falls back gracefully (persistent mode disabled or uses WebSpeech) |
| Config changes mid-session | Changing segment threshold takes effect on next segment (snapshot pattern) |

### Manual Validation

- Speak naturally with pauses — verify segments fire at natural boundaries, not mid-sentence
- Speak a command with prefix — verify it's detected and suggestion appears
- Speak without prefix — verify no false command detection
- Toggle persistent mode on/off — verify clean transitions
- Test on mobile Chrome and desktop Chrome

## 10. Rollout / Validation Checklist

- [ ] Phase 1 complete: segment-final transcriptions work in existing one-shot mode
- [ ] Phase 2 complete: persistent mode toggles correctly, segments fire on silence
- [ ] Phase 3 complete: commands detected, suggestions shown, actions execute
- [ ] Phase 4 complete: all settings exposed and persisted
- [ ] Phase 5 complete: SEAMS.md, TEMPORAL-FLOWS.md updated
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Manual validation on mobile and desktop
- [ ] No regressions in existing one-shot voice mode
- [ ] `cd scenarios/web-console && make test` passes
- [ ] `cd scenarios/web-console/api && go build ./... && go test ./... -timeout 300s` passes

## 11. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Whisper server bottleneck on segment finals (transcoding + inference blocks partials for next segment) | Delayed partials in next segment | Segment final runs in a goroutine with its own Whisper request; partials continue on the ticker. If Whisper is single-threaded, queue segment finals and let partials skip while a final is in-flight. |
| Segment silence threshold too short → fragments sentences | Poor transcription quality | Default to 1.5s (generous). Provide user-configurable slider. Adaptive threshold only reduces for prefix-detected segments. |
| Segment silence threshold too long → commands feel sluggish | Poor UX | Adaptive threshold: when prefix detected in partials, drop to 600-800ms for that segment only. |
| Command prefix false positives (prefix word appears naturally in speech) | Unintended command execution | Two-word prefix (e.g., "hey do") reduces collision probability. Commands require confirmation tap. Fuzzy matching has a confidence threshold. |
| Battery/resource drain from always-on mic + Whisper | Mobile device overheating | Document that persistent mode is designed for active work sessions, not background. Consider adding a maximum session duration with warning. |
| Partials split the command prefix across chunks | Missed command detection | Command detection only runs on segment finals (complete, accurate transcriptions), never on partials. The adaptive silence shortening uses prefix detection on partials as a hint only, not a commitment. |
| WebSpeech fallback doesn't support segment protocol | Feature degradation | Persistent mode requires Whisper backend. If only WebSpeech available, persistent mode toggle is disabled with explanatory message. |

## 12. Non-Goals / Prohibited Patterns

- **No auto-execute without user confirmation** — all commands require a tap to confirm in V1
- **No wake-word detection** — persistent mode is manually toggled, not always-on from app open
- **No changes to the Whisper server** — all changes are in the web-console API and UI
- **No command vocabulary extensibility in V1** — fixed command set, extensibility is V2
- **No compatibility shims, legacy code, or dead code** — this is greenfield (see constraint section above); if the voice config schema changes, migrate forward cleanly; do not leave unused types, commented-out logic, or feature-flag gates for old behavior
- **Do not** scatter command detection logic across multiple files — keep it in `commandParser.ts`
- **Do not** put command execution logic in UI components — keep it in `commands.ts` with a clean CommandContext interface
- **Do not** modify the existing one-shot voice mode behavior — persistent mode is an additive feature toggled by a setting

## 13. Definition of Done

1. Persistent voice mode can be toggled on/off via settings and functions correctly
2. Two-tier transcription works: fast partials for feedback, segment finals for accuracy on silence boundaries
3. Configurable command prefix reliably distinguishes commands from dictation
4. All commands in the initial vocabulary execute correctly when confirmed
5. Command suggestions appear above the mobile toolbar and can be confirmed or dismissed
6. All settings are exposed, validated, and persisted
7. All unit and integration tests pass
8. No regressions in existing one-shot voice mode
9. SEAMS.md and TEMPORAL-FLOWS.md updated to reflect new architecture
10. `make test` passes for the web-console scenario
