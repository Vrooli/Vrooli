import { useState } from "react";

export interface VoiceInputButtonProps {
  /** Called with the final transcript when STT completes. */
  onTranscript: (text: string) => void;
  /**
   * Optional command-handler callback. Consumers like web-console wire their
   * terminal-command vocabulary into this prop; audio-tools intentionally
   * does not own command parsing.
   */
  commandHandler?: (text: string) => void;
  /** Language hint passed to STT (empty = auto-detect). */
  language?: string;
  /**
   * "push-to-talk" or "wake-word"; defaults to push-to-talk. Mode
   * implementation is the consumer's choice — this surface only declares
   * the prop so consumers can configure behavior without forking.
   */
  mode?: "push-to-talk" | "wake-word";
  /** Optional ARIA label override. */
  ariaLabel?: string;
}

/**
 * Generalized mic button. P0 skeleton: clicking toggles a "listening" state
 * via the consumer's wired-in STT hook (passed via context in a follow-up
 * iteration). The full streaming pipeline integration lands when this
 * package starts re-exporting the ported web-console VoiceStreamProvider.
 */
export function VoiceInputButton(props: VoiceInputButtonProps): JSX.Element {
  const [listening, setListening] = useState(false);
  return (
    <button
      type="button"
      aria-label={props.ariaLabel ?? "Voice input"}
      aria-pressed={listening}
      onClick={() => setListening((v) => !v)}
      className="audio-tools-embed-voice-input-button"
    >
      {listening ? "Listening…" : "🎙"}
    </button>
  );
}
