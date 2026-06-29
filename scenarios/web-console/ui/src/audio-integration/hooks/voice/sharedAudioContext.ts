// DOC: docs/internal/VOICE-LATENCY.md#pre-create-audiocontext-on-first-gesture
//
// Shared AudioContext Singleton
// ==============================
//
// Browsers impose two constraints on AudioContext:
//
//   1. **User gesture requirement**: An AudioContext created (or resumed) outside
//      a user gesture handler starts in "suspended" state. It must be resumed
//      inside a click/touch/keydown handler. By creating the context on the very
//      first user gesture (anywhere in the app), we guarantee it is ready before
//      the mic button is pressed, eliminating ~20-50ms from the recording start.
//
//   2. **Context limit**: Browsers allow only 6-8 AudioContexts per page. Before
//      this module, the codebase created separate contexts for audio cues
//      (audioCues.ts) and level monitoring (useVoiceInput.ts). Consolidating
//      into a single shared context reduces resource usage and avoids hitting
//      the limit as more audio features are added.
//
// Lifecycle: The shared context is app-lifetime. It is created once and never
// closed during normal operation. Only `closeSharedAudioContext()` (used in
// tests) closes it. Individual consumers (level monitor, audio cues) connect
// and disconnect their own nodes without affecting the shared context.

let _sharedCtx: AudioContext | null = null;
let _gestureInstalled = false;

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
 * Install a one-shot event listener that creates the AudioContext on the first
 * user gesture (pointerdown or keydown). This ensures the context is available
 * and in "running" state before the user presses the mic button.
 *
 * Safe to call multiple times — subsequent calls are no-ops.
 *
 * Why pointerdown instead of click: pointerdown fires earlier in the event
 * chain, giving the AudioContext a few extra milliseconds to initialize
 * before any audio-dependent code runs.
 */
export function ensureAudioContextOnGesture(): void {
  if (_gestureInstalled || typeof document === "undefined") return;
  _gestureInstalled = true;

  const handler = () => {
    // Create and immediately resume the context. If it's already created
    // (e.g., by an earlier call to getSharedAudioContext), just resume it.
    const ctx = getSharedAudioContext();
    if (ctx.state === "suspended") {
      ctx.resume().catch(() => {});
    }
    console.info("[voice] AudioContext pre-created on user gesture (state=%s)", ctx.state);

    // Self-remove after first trigger — we only need one gesture.
    document.removeEventListener("pointerdown", handler, true);
    document.removeEventListener("keydown", handler, true);
  };

  // Use capture phase so we fire before any stopPropagation in the app.
  document.addEventListener("pointerdown", handler, { capture: true, passive: true, once: true });
  document.addEventListener("keydown", handler, { capture: true, passive: true, once: true });
}

/**
 * Close the shared AudioContext and reset state. Primarily for test cleanup.
 * In production, the context is app-lifetime and should never be closed.
 */
export function closeSharedAudioContext(): void {
  if (_sharedCtx && _sharedCtx.state !== "closed") {
    _sharedCtx.close().catch(() => {});
  }
  _sharedCtx = null;
  _gestureInstalled = false;
}
