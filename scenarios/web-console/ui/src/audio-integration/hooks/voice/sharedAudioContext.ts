// DOC: docs/internal/VOICE-LATENCY.md#audiocontext-lifecycle
//
// Shared AudioContext Singleton
// ==============================
//
// Browsers impose two constraints on AudioContext:
//
//   1. **User gesture requirement**: An AudioContext created (or resumed) outside
//      a user gesture handler starts in "suspended" state. It must be resumed
//      inside a click/touch/keydown handler. We resume it lazily, inside the
//      actual voice/cue gesture (the mic-button press, the record cue) — NOT
//      eagerly on the first arbitrary tap. Eager resume was a ~20-50ms latency
//      micro-optimization that, on iOS, activated the app's AVAudioSession on the
//      first interaction of ANY kind and interrupted other apps' audio (Spotify /
//      YouTube) even when the user never used voice. The latency win is not worth
//      hijacking the audio session; see PROBLEMS.md.
//
//   2. **Context limit**: Browsers allow only 6-8 AudioContexts per page. Before
//      this module, the codebase created separate contexts for audio cues
//      (audioCues.ts) and level monitoring (useVoiceInput.ts). Consolidating
//      into a single shared context reduces resource usage and avoids hitting
//      the limit as more audio features are added.
//
// Lifecycle — release the audio session when idle. The shared context is created
// lazily on first real audio need and SUSPENDED whenever it is idle: on page
// background (hidden / pagehide / freeze) and shortly after a capture turn ends.
// A running-but-idle AudioContext still holds the iOS audio session active (and
// keeps other apps' audio ducked), so leaving it running app-lifetime is exactly
// the "hold the session forever" anti-pattern this module now avoids. Consumers
// (level monitor, audio cues) resume it on demand inside their own gesture, so
// suspending when idle is transparent to them. `armIdleSuspend()` schedules a
// suspend; `keepAudioContextAwake()` cancels a pending one when audio resumes.

let _sharedCtx: AudioContext | null = null;
let _idleSuspendTimer: ReturnType<typeof setTimeout> | null = null;

/** Delay before an idle context is suspended, long enough for a stop cue
 *  (~150ms) to finish playing so we never clip it. */
const IDLE_SUSPEND_DELAY_MS = 1500;

/**
 * Get the shared AudioContext singleton, creating it if necessary.
 *
 * If the context is in "suspended" state (created before a user gesture),
 * callers must await `ctx.resume()` before scheduling audio. The level
 * monitor already handles this in startLevelMonitor().
 */
export function getSharedAudioContext(): AudioContext {
  // Rebuild on "closed" AND "interrupted". "interrupted" (Safari/iOS, and some
  // Chrome cases after audio-device changes) is a terminal state that resume()
  // cannot reliably recover; reusing it leaves the level meter permanently dead.
  if (!_sharedCtx || _sharedCtx.state === "closed" || (_sharedCtx.state as string) === "interrupted") {
    _sharedCtx = new AudioContext();
  }
  return _sharedCtx;
}

/**
 * Return a shared AudioContext that is actually in "running" state, healing a
 * wedged one if necessary.
 *
 * The plain getter only rebuilds on closed/interrupted. A context can also get
 * stuck in "suspended" where resume() silently fails to bring it back (e.g.
 * after the machine sleeps/wakes or the audio device changes). Before this,
 * such a context was reused forever and the only fix was a full page reload.
 * This function resumes, and if that doesn't reach "running", discards the
 * wedged context and builds a fresh one. Best-effort: a context rebuilt outside
 * a user gesture may remain suspended, but the capture path does not depend on
 * it (only the level meter does), so callers should not fail the session on it.
 */
export async function ensureRunningSharedAudioContext(): Promise<AudioContext> {
  // Real audio need — cancel any idle-suspend armed by a prior turn.
  keepAudioContextAwake();
  let ctx = getSharedAudioContext();
  // Cast to string: TS narrows ctx.state across the resume() call (property
  // reads aren't re-widened), which would flag the post-resume comparison.
  if ((ctx.state as string) === "running") return ctx;

  try { await ctx.resume(); } catch { /* fall through to rebuild */ }
  if ((ctx.state as string) === "running") return ctx;

  // Wedged: resume() did not recover it. Discard and rebuild once.
  console.warn("[voice] Shared AudioContext wedged (state=%s); rebuilding", ctx.state);
  try { await ctx.close(); } catch { /* ignore */ }
  _sharedCtx = null;
  ctx = getSharedAudioContext();
  try { await ctx.resume(); } catch { /* ignore — may need a gesture */ }
  return ctx;
}

/**
 * Cancel any pending idle-suspend. Call this whenever audio is (re)activated —
 * the level monitor starting, a cue about to play, or a recording start — so a
 * timer armed by a just-ended turn cannot suspend the context out from under a
 * new one.
 */
export function keepAudioContextAwake(): void {
  if (_idleSuspendTimer) {
    clearTimeout(_idleSuspendTimer);
    _idleSuspendTimer = null;
  }
}

/**
 * Suspend the shared AudioContext immediately if it exists and is running.
 * Suspending releases the iOS audio session so other apps' audio is no longer
 * held; the context is resumed on demand (in a gesture) the next time voice or a
 * cue needs it. No-op if there is no context or it is already suspended/closed.
 * Used by the page-background backstop (hidden / pagehide / freeze).
 */
export function suspendSharedAudioContext(): void {
  keepAudioContextAwake();
  if (_sharedCtx && _sharedCtx.state === "running") {
    _sharedCtx.suspend().catch(() => { /* best-effort; harmless if it races a close */ });
  }
}

/**
 * Arm a deferred suspend for when the context goes idle after a capture turn.
 * Debounced: a later call replaces the pending timer, and any resume cancels it
 * via `keepAudioContextAwake()`. The delay lets a trailing stop-cue finish
 * before the session is released.
 */
export function armIdleSuspend(delayMs: number = IDLE_SUSPEND_DELAY_MS): void {
  keepAudioContextAwake();
  if (!_sharedCtx || _sharedCtx.state !== "running") return;
  _idleSuspendTimer = setTimeout(() => {
    _idleSuspendTimer = null;
    // Re-check state: a new session may have started (and will keep it awake).
    if (_sharedCtx && _sharedCtx.state === "running") {
      _sharedCtx.suspend().catch(() => {});
    }
  }, delayMs);
}

/**
 * Close the shared AudioContext and reset state. Primarily for test cleanup.
 * In production the context is suspended when idle, not closed.
 */
export function closeSharedAudioContext(): void {
  keepAudioContextAwake();
  if (_sharedCtx && _sharedCtx.state !== "closed") {
    _sharedCtx.close().catch(() => {});
  }
  _sharedCtx = null;
}
