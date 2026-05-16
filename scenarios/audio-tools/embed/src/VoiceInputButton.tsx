import { useState } from "react";
import type { JSX } from "react";

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
  /** Optional listening-state label override. */
  listeningLabel?: string;
}

/**
 * Generalized mic button. Clicking toggles the local "listening" state and
 * notifies callers via the `onToggle` callback; the actual STT pipeline is
 * provided by the consumer (typically the re-exported VoiceStreamProvider
 * from this package). The button stays unopinionated about transport.
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
      {listening ? (props.listeningLabel ?? "Listening…") : "🎙"}
    </button>
  );
}
