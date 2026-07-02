// DOC: docs/internal/SEAMS.md#wake-word-engine-seam
//
// Pure decision for the passive wake-word auto-arm reconciliation. Extracted
// from useVoiceCore so the always-on lifecycle (enter / exit / hold) is
// unit-testable without rendering the hook or mocking getUserMedia. Mirrors
// voice/autoStopDecision.ts.
//
// The reconciliation is the heart of the "always-on" feature: when wake word
// is enabled and a template is loaded, the app must passively listen whenever
// voice is idle, then re-arm after each detected turn ends. Before this seam,
// nothing ever entered passive mode — the toggle flipped a config bit but the
// mic was never opened.

/** Voice states the core can be in. Kept loose to avoid importing the full union. */
export type VoiceStateLite =
  | "idle"
  | "preparing"
  | "recording"
  | "listening"
  | "transcribing";

export interface PassiveArmInput {
  /** Whether voice input is enabled at all. */
  voiceEnabled: boolean;
  /** Whether the wake-word toggle is on. */
  wakeWordEnabled: boolean;
  /** Whether a wake-word template is loaded (engine + sample features ready). */
  wakeWordConfigured: boolean;
  /** Current voice state machine value. */
  voiceState: VoiceStateLite;
  /** Whether a passive listener is currently live. */
  listenerActive: boolean;
  /** Whether the last passive start failed and we're holding off auto-retries. */
  startBlocked: boolean;
  /**
   * Whether the document is currently visible. Passive listening must NOT open
   * the mic while the tab/PWA is hidden (the iOS-PWA background-mic leak). The
   * visibility handler re-arms explicitly on becoming visible.
   */
  documentVisible: boolean;
}

export type PassiveArmAction = "enter" | "exit" | "none";

/**
 * Decide whether to enter, exit, or leave passive listening untouched.
 *
 * - Voice/wake-word off → tear down any live listener ("exit"), else "none".
 * - Wake word on + template loaded + idle + visible + nothing already listening
 *   + not blocked by a prior failure → arm passive listening ("enter").
 * - Hidden document, anything mid-flight (preparing/recording/listening/
 *   transcribing), a listener already running, or a failure latch → "none".
 */
export function decidePassiveArm(input: PassiveArmInput): PassiveArmAction {
  const {
    voiceEnabled,
    wakeWordEnabled,
    wakeWordConfigured,
    voiceState,
    listenerActive,
    startBlocked,
    documentVisible,
  } = input;

  if (!voiceEnabled || !wakeWordEnabled) {
    return listenerActive ? "exit" : "none";
  }

  if (wakeWordConfigured && voiceState === "idle" && documentVisible && !listenerActive && !startBlocked) {
    return "enter";
  }

  return "none";
}
