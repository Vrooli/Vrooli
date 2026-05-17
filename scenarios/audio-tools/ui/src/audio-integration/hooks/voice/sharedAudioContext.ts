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
/** Keepalive oscillator that maintains an active audio path to ctx.destination.
 *  Without this, Chrome's power-saving optimisation idles the audio rendering
 *  thread between recording sessions, causing AnalyserNodes in subsequent
 *  sessions to return stale/zero data (volume indicator stuck at 0, VAD sees
 *  rms=0 → premature stop). A silent DC oscillator is the cheapest way to
 *  keep the renderer alive (~zero CPU, no audible output).
 *
 *  Installed lazily — only between recording sessions, where Chrome's renderer
 *  is at risk of idling. Not installed before the first recording. This matters
 *  on iOS, where any oscillator routed to ctx.destination keeps the
 *  AVAudioSession active and shows the Dynamic Island audio indicator (which
 *  also flickers per-keystroke as keyboard click sounds preempt the session).
 *
 *  Two mitigations are in effect:
 *    1. The keepalive is skipped entirely on iOS — Safari/WebKit does not have
 *       the Chrome renderer-idle bug this was working around, and the OS
 *       audio-indicator cost outweighs any benefit.
 *    2. On other platforms, the keepalive auto-tears-down after
 *       KEEPALIVE_IDLE_TIMEOUT_MS of no new recording, so back-to-back
 *       recordings stay fast but a user who recorded once and then went back
 *       to typing doesn't keep paying the OS audio-session cost forever. */
let _keepaliveOsc: OscillatorNode | null = null;
let _keepaliveGain: GainNode | null = null;
let _keepaliveIdleTimer: ReturnType<typeof setTimeout> | null = null;

/** How long the keepalive stays installed after a recording session ends.
 *  Long enough to cover natural back-to-back recordings (user stops, glances
 *  at transcript, taps mic again), short enough that the OS audio session is
 *  released before the user goes back to general typing. */
const KEEPALIVE_IDLE_TIMEOUT_MS = 30_000;

/** Detect iOS (including iPadOS 13+, which reports as Mac with touch). */
function isIOS(): boolean {
  if (typeof navigator === "undefined") return false;
  const ua = navigator.userAgent || "";
  if (/iPad|iPhone|iPod/.test(ua)) return true;
  // iPadOS 13+ masquerades as macOS but has touch points.
  return ua.includes("Mac") && typeof document !== "undefined" && "ontouchend" in document;
}

/**
 * Get the shared AudioContext singleton, creating it if necessary.
 *
 * If the context is in "suspended" state (created before a user gesture),
 * callers must await `ctx.resume()` before scheduling audio. The level
 * monitor already handles this in startLevelMonitor().
 */
export function getSharedAudioContext(): AudioContext {
  if (!_sharedCtx || _sharedCtx.state === "closed") {
    _sharedCtx = new AudioContext();
  }
  return _sharedCtx;
}

/**
 * Install the keepalive oscillator on the shared AudioContext. Called when a
 * recording session ends, to prevent Chrome's renderer from idling before the
 * next session starts. Idempotent — calling while already installed is a no-op.
 */
export function installAudioContextKeepalive(): void {
  // iOS: skip entirely. The Chrome renderer-idle bug this works around does
  // not occur on WebKit, and on iOS any oscillator routed to ctx.destination
  // keeps the AVAudioSession active and the Dynamic Island audio indicator
  // flickering on every keystroke. Net negative for iOS users.
  if (isIOS()) return;
  const ctx = _sharedCtx;
  if (!ctx || ctx.state === "closed") return;
  if (!_keepaliveOsc) {
    _installKeepalive(ctx);
  }
  // (Re)arm the idle teardown timer. Each new install — or re-install via a
  // subsequent stopLevelMonitor — extends the window by another full timeout.
  if (_keepaliveIdleTimer) clearTimeout(_keepaliveIdleTimer);
  _keepaliveIdleTimer = setTimeout(() => {
    _keepaliveIdleTimer = null;
    _teardownKeepalive();
  }, KEEPALIVE_IDLE_TIMEOUT_MS);
}

/**
 * Tear down the keepalive oscillator. Called when a recording session is about
 * to start (the live mic stream itself keeps the renderer alive, so the
 * keepalive is redundant during recording) and on iOS, where it would
 * otherwise keep the AVAudioSession active.
 */
export function teardownAudioContextKeepalive(): void {
  _teardownKeepalive();
}

/**
 * Install a silent keepalive oscillator on the AudioContext. This ensures
 * there is always an active audio path to ctx.destination, preventing Chrome
 * from idling the audio rendering thread between recording sessions.
 *
 * Uses a DC oscillator (0 Hz, constant signal) routed through a zero-gain
 * node — inaudible, negligible CPU cost, but keeps the renderer alive.
 */
function _installKeepalive(ctx: AudioContext): void {
  if (_keepaliveOsc) return; // already installed

  try {
    const osc = ctx.createOscillator();
    osc.type = "square";
    osc.frequency.value = 0; // DC — constant signal, no audible tone
    const gain = ctx.createGain();
    gain.gain.value = 0; // Silent
    osc.connect(gain);
    gain.connect(ctx.destination);
    osc.start();
    _keepaliveOsc = osc;
    _keepaliveGain = gain;
  } catch {
    // AudioContext might not support oscillators in some test environments
  }
}

function _teardownKeepalive(): void {
  if (_keepaliveIdleTimer) {
    clearTimeout(_keepaliveIdleTimer);
    _keepaliveIdleTimer = null;
  }
  try { _keepaliveOsc?.stop(); } catch { /* already stopped */ }
  try { _keepaliveOsc?.disconnect(); } catch { /* already disconnected */ }
  try { _keepaliveGain?.disconnect(); } catch { /* already disconnected */ }
  _keepaliveOsc = null;
  _keepaliveGain = null;
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
  _teardownKeepalive();
  if (_sharedCtx && _sharedCtx.state !== "closed") {
    _sharedCtx.close().catch(() => {});
  }
  _sharedCtx = null;
  _gestureInstalled = false;
}
