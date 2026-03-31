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
// ── Audio Cue Contract ──
//
// Cues are scoped to the RECORDING SESSION, not the mic hardware lifecycle.
// They must ONLY play during these transitions:
//
//   Start cue: user pressed mic → provider started → ready to accept speech
//   Stop cue:  user pressed stop, VAD auto-stopped, or abort during startup
//
// They must NEVER play during these events:
//
//   - Mic pre-warm (low-latency mode acquireStream)
//   - Mic release (visibility handler, cleanup, audio ducking cycle)
//   - Component unmount / app close
//   - Error recovery or backend fallback
//   - Transcription cancellation
//   - Wake word passive listening start/stop
//
// The `cueSessionActiveRef` in useVoiceInput.ts enforces this contract. The
// start cue sets it to true; the stop cue checks it before playing and sets
// it to false. All non-stop exit paths (cleanup, error, cancel) clear it
// WITHOUT playing the stop cue.
//
// DOC: docs/internal/VOICE-LATENCY.md#audio-cue-contract
//
// Implementation: pure Web Audio API oscillators with soft gain envelopes.
// No external dependencies or audio files needed. The tones are designed to
// be gentle and unobtrusive — a soft two-note chime, not a harsh beep.

// Uses the shared AudioContext singleton to avoid creating a separate context
// for cue playback. This keeps the total AudioContext count at 1 instead of 2,
// leaving more headroom under the browser's 6-8 context limit.
// DOC: docs/internal/VOICE-LATENCY.md#pre-create-audiocontext-on-first-gesture
import { getSharedAudioContext } from "./sharedAudioContext";

async function getCueContext(): Promise<AudioContext> {
  const ctx = getSharedAudioContext();
  // Resume if suspended (browsers suspend until user gesture).
  // We must await this — scheduling oscillators on a suspended context
  // means they fire against a frozen currentTime and are already in the
  // past when playback finally starts, so they never produce sound.
  if (ctx.state === "suspended") {
    await ctx.resume();
  }
  return ctx;
}

/**
 * Play a soft two-note chime.
 *
 * @param freq1 - First note frequency (Hz)
 * @param freq2 - Second note frequency (Hz)
 * @param volume - Peak gain (0–1). Kept low so it doesn't blast through
 *                 headphones or compete with TTS playback.
 */
async function playChime(freq1: number, freq2: number, volume = 0.18): Promise<void> {
  try {
    const ctx = await getCueContext();
    const now = ctx.currentTime;

    // ── Note 1 ──
    const osc1 = ctx.createOscillator();
    const gain1 = ctx.createGain();
    osc1.type = "sine";
    osc1.frequency.value = freq1;
    // Gentle envelope — longer attack/decay avoids clicks on mobile audio hardware
    gain1.gain.setValueAtTime(0, now);
    gain1.gain.linearRampToValueAtTime(volume, now + 0.06);   // 60ms attack
    gain1.gain.setValueAtTime(volume, now + 0.10);             // hold to 100ms
    gain1.gain.exponentialRampToValueAtTime(0.001, now + 0.25); // smooth decay
    osc1.connect(gain1).connect(ctx.destination);
    osc1.start(now);
    osc1.stop(now + 0.28);

    // ── Note 2 (offset by 140ms for a two-note "chime" feel) ──
    const osc2 = ctx.createOscillator();
    const gain2 = ctx.createGain();
    osc2.type = "sine";
    osc2.frequency.value = freq2;
    gain2.gain.setValueAtTime(0, now + 0.14);
    gain2.gain.linearRampToValueAtTime(volume, now + 0.20);    // 60ms attack
    gain2.gain.setValueAtTime(volume, now + 0.26);              // hold
    gain2.gain.exponentialRampToValueAtTime(0.001, now + 0.45); // smooth decay
    osc2.connect(gain2).connect(ctx.destination);
    osc2.start(now + 0.14);
    osc2.stop(now + 0.48);
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
  void playChime(523, 659);
}

/**
 * Falling two-note chime: "recording stopped."
 * Notes: E5 (659 Hz) → C5 (523 Hz) — the inverse interval, signalling "done."
 */
export function playRecordingStopCue(): void {
  void playChime(659, 523);
}
