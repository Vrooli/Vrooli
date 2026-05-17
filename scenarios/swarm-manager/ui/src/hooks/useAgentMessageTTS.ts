// useAgentMessageTTS — scenario-side adapter over useTextToSpeechCore.
// Exposes a stable speak/stop pair keyed by message id so multiple
// bubbles render their own speaking state without bookkeeping.
//
// Voice/speed settings come from audio-tools' shared config; auto-speak
// is a swarm-manager-local pref in useAudioPrefs.

import { useCallback, useState } from "react";
import { useTextToSpeechCore, useAudioToolsUnavailableReason } from "../audio-integration";

interface UseAgentMessageTTSResult {
  speak: (messageId: string, text: string) => void;
  stop: () => void;
  isSpeaking: boolean;
  speakingMessageId: string | null;
  unavailable: boolean;
}

export function useAgentMessageTTS(): UseAgentMessageTTSResult {
  const unavailableReason = useAudioToolsUnavailableReason();
  const unavailable = Boolean(unavailableReason);
  const [speakingMessageId, setSpeakingMessageId] = useState<string | null>(null);

  const tts = useTextToSpeechCore(
    {
      autoEnabled: false,
      backend: "auto",
      startMuted: false,
    },
    {
      voice: "",
      rate: 1.0,
      pitch: 1.0,
      kokoroVoice: "af_heart",
      kokoroSpeed: 1.0,
    },
  );

  const speak = useCallback(
    (messageId: string, text: string) => {
      if (unavailable || !text.trim()) return;
      setSpeakingMessageId(messageId);
      tts.speak(text);
    },
    [tts, unavailable],
  );

  const stop = useCallback(() => {
    tts.stop();
    setSpeakingMessageId(null);
  }, [tts]);

  // Clear the speaking id when the core finishes (isSpeaking flips false).
  if (!tts.isSpeaking && speakingMessageId !== null) {
    // Defer to next tick to avoid set-during-render warning.
    queueMicrotask(() => setSpeakingMessageId((curr) => (tts.isSpeaking ? curr : null)));
  }

  return {
    speak,
    stop,
    isSpeaking: tts.isSpeaking,
    speakingMessageId,
    unavailable,
  };
}
