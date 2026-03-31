# Plan: Add Loading States to Wake Word Sample Buttons

## 1. Purpose

Add visual loading/disabled states to wake word sample recording buttons so users get immediate feedback when async operations are in progress.

## 2. Required Reading

```bash
prompt-manager skill read react-coherence ux
```

## 3. Problem Statement

In `VoiceInputSection.tsx`, the wake word recording UI has several buttons that trigger async operations but provide no loading feedback:

1. **Record button** — calls `startWwRecording(i)` which does `getUserMedia` (permission prompt + stream setup). Between click and recording state there's a gap with no visual indicator.
2. **Save wake word** — calls `saveWakeWord()` which hits `PUT /voice/wakeword`. No spinner during the API call.
3. **Delete wake word** — calls `deleteWakeWord()` which hits `DELETE /voice/wakeword`. No spinner during the API call.
4. **Test cross-match** — calls `testWakeWord()` which runs DTW comparisons. Computation is fast but still synchronous/blocking with no indicator.

The stop button during recording already has a live timer and is adequate.

## 4. Scope

**In scope:**
- Add loading spinners to Save, Delete, and Test cross-match buttons during their async operations
- Add a brief "initializing" state to the Record button between click and `wwRecordingIdx` being set (the getUserMedia gap)
- Disable buttons appropriately during loading states

**Out of scope:**
- Changing recording logic or audio processing
- Modifying the wake word engine
- Backend API changes
- Re-record and remove buttons (these are instant local operations)

## 5. Current Technical Context

**File:** `scenarios/web-console/ui/src/components/settings/VoiceInputSection.tsx`

**Existing state variables (lines 83-98):**
- `wwRecordingIdx: number | null` — which slot is recording
- `wwRecordingSeconds: number` — live timer
- `wakeWordLoading: boolean` — used for initial config load only
- `wakeWordError: string | null`

**Async operations needing loading states:**
| Operation | Function | Lines | Async source |
|-----------|----------|-------|-------------|
| Record | `startWwRecording` | 159-207 | `getUserMedia` + MediaRecorder setup |
| Save | `saveWakeWord` | 370-392 | `updateWakeWordConfig` API call |
| Delete | `deleteWakeWord` | 394-406 | `deleteWakeWordConfig` API call |
| Test | `testWakeWord` | 408-421 | Synchronous DTW computation (may not need spinner) |

**Existing loading pattern in this file:**
- `Loader2` icon from lucide-react with `animate-spin` class (used in speaker enrollment section, ~line 1098)
- Buttons use `disabled` prop during operations

## 6. Target End State

- Save button shows a spinner and "Saving…" text while API call is in flight
- Delete button shows a spinner and "Deleting…" text while API call is in flight
- Record button shows a brief spinner/"Preparing…" state while `getUserMedia` is pending (before recording starts)
- All buttons in the wake word section are disabled while any save/delete operation is in progress
- Test cross-match button shows spinner during computation (if noticeably slow)

## 7. Implementation Strategy

### Phase 1: Add state variables

Add boolean state for each async operation:
```typescript
const [wwSaving, setWwSaving] = useState(false);
const [wwDeleting, setWwDeleting] = useState(false);
const [wwTesting, setWwTesting] = useState(false);
```

The Record button already has a natural loading state via `wwRecordingIdx` being set — but there's a gap between click and `getUserMedia` resolving. A `wwInitializing` state could cover this:
```typescript
const [wwInitializing, setWwInitializing] = useState(false);
```

### Phase 2: Wrap async functions with loading state

In `saveWakeWord`: wrap with `setWwSaving(true/false)` in try/finally.
In `deleteWakeWord`: wrap with `setWwDeleting(true/false)` in try/finally.
In `testWakeWord`: wrap with `setWwTesting(true/false)` in try/finally.
In `startWwRecording`: set `wwInitializing` true at start, false once recorder starts (or on error).

### Phase 3: Update button rendering

- **Save button:** Show `<Loader2 className="mr-1 h-3 w-3 animate-spin" />` + "Saving…" when `wwSaving`, disable when `wwSaving || wwDeleting`
- **Delete button:** Show spinner + "Deleting…" when `wwDeleting`, disable when `wwSaving || wwDeleting`
- **Test button:** Show spinner + "Testing…" when `wwTesting`, disable when `wwTesting`
- **Record button:** Show spinner + "Preparing…" when `wwInitializing` for that slot, disable all record buttons when any operation is in progress
- Add `Loader2` to imports if not already present

## 8. Contract Decisions

No API or data model changes. This is purely a UI state management fix.

## 9. Testing Plan

- **Manual verification:** Click each button and observe spinner appears immediately and disappears when operation completes
- **Error cases:** Verify loading state clears on errors (e.g., denied mic permission, failed API call)
- **Concurrent protection:** Verify you cannot click Save while Delete is in progress and vice versa

## 10. Rollout/Validation Checklist

- [ ] All four button types show loading indicators during async operations
- [ ] Loading state clears on both success and error paths
- [ ] Buttons are properly disabled during operations to prevent double-clicks
- [ ] Existing functionality (recording, playback, save, delete) still works correctly
- [ ] No layout shift when spinner appears/disappears

## 11. Risks + Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Loading state stuck if error not caught | Low | Use try/finally pattern consistently |
| Layout shift from text change | Low | Use fixed-width button or consistent sizing |
| getUserMedia initializing state too brief to notice | Medium | Acceptable — still prevents double-click |

## 12. Non-goals / Prohibited Patterns

- Do not refactor the recording logic
- Do not extract a custom hook for this — the state is local to this component
- Do not add skeleton loading or progress bars — simple spinners match existing patterns

## 13. Definition of Done

- Save, Delete, and Test buttons show `Loader2` spinner with descriptive text during their async operations
- Record button shows initializing state during `getUserMedia`
- All operations are protected from double-click via disabled state
- Loading states clear in both success and error paths (try/finally)
- No regressions to existing wake word functionality
