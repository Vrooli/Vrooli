// ── Voice Recording Audio Cues ──
//
// Short, pleasant chime sounds that play when voice recording starts and stops.
// These give the user audible confirmation of state changes, which is critical
// in two scenarios:
//
//   1. **Recording start**: The user may begin speaking before the mic is
//      actually active (especially on mobile while walking). A rising chime
//      confirms "I'm listening now."
//
//   2. **Recording stop (especially VAD auto-stop)**: When Voice Activity
//      Detection ends recording after a silence timeout, the user may not
//      realise recording stopped and keep talking into the void. A falling
//      chime makes the cutoff obvious.
//
// Implementation: pure Web Audio API oscillators with soft gain envelopes.
// No external dependencies or audio files needed. The tones are designed to
// be gentle and unobtrusive — a soft two-note chime, not a harsh beep.

/** Shared AudioContext for cue playback — reused to avoid hitting the
 *  browser's AudioContext limit (typically 6–8 per page). */
let cueCtx: AudioContext | null = null;

function getCueContext(): AudioContext {
  if (!cueCtx || cueCtx.state === "closed") {
    cueCtx = new AudioContext();
  }
  // Resume if suspended (browsers suspend until user gesture)
  if (cueCtx.state === "suspended") {
    cueCtx.resume().catch(() => {});
  }
  return cueCtx;
}

/**
 * Play a soft two-note chime.
 *
 * @param freq1 - First note frequency (Hz)
 * @param freq2 - Second note frequency (Hz)
 * @param volume - Peak gain (0–1). Kept low so it doesn't blast through
 *                 headphones or compete with TTS playback.
 */
function playChime(freq1: number, freq2: number, volume = 0.15): void {
  try {
    const ctx = getCueContext();
    const now = ctx.currentTime;

    // ── Note 1 ──
    const osc1 = ctx.createOscillator();
    const gain1 = ctx.createGain();
    osc1.type = "sine";
    osc1.frequency.value = freq1;
    // Soft attack → sustain → quick fade
    gain1.gain.setValueAtTime(0, now);
    gain1.gain.linearRampToValueAtTime(volume, now + 0.03);   // 30ms attack
    gain1.gain.linearRampToValueAtTime(0, now + 0.12);        // fade out by 120ms
    osc1.connect(gain1).connect(ctx.destination);
    osc1.start(now);
    osc1.stop(now + 0.15);

    // ── Note 2 (offset by 100ms for a two-note "chime" feel) ──
    const osc2 = ctx.createOscillator();
    const gain2 = ctx.createGain();
    osc2.type = "sine";
    osc2.frequency.value = freq2;
    gain2.gain.setValueAtTime(0, now + 0.10);
    gain2.gain.linearRampToValueAtTime(volume, now + 0.13);   // 30ms attack
    gain2.gain.linearRampToValueAtTime(0, now + 0.25);        // fade by 250ms
    osc2.connect(gain2).connect(ctx.destination);
    osc2.start(now + 0.10);
    osc2.stop(now + 0.30);
  } catch {
    // AudioContext not available — silently skip. Cues are a nice-to-have,
    // not a hard requirement. The UI still shows visual state changes.
  }
}

/**
 * Rising two-note chime: "recording started."
 * Notes: C5 (523 Hz) → E5 (659 Hz) — a pleasant major third rising interval.
 */
export function playRecordingStartCue(): void {
  playChime(523, 659);
}

/**
 * Falling two-note chime: "recording stopped."
 * Notes: E5 (659 Hz) → C5 (523 Hz) — the inverse interval, signalling "done."
 */
export function playRecordingStopCue(): void {
  playChime(659, 523);
}
