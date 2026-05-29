// useAgentMessageTTS — scenario-side adapter over useTextToSpeechCore.
// Exposes a stable speak/stop pair keyed by message id so multiple
// bubbles render their own speaking state without bookkeeping.
//
// Voice/speed settings come from audio-tools' shared config; auto-speak
// is a swarm-manager-local pref in useAudioPrefs.

import { useCallback, useEffect, useRef, useState } from "react";
import { useTextToSpeechCore, useAudioToolsUnavailableReason } from "../audio-integration";

interface UseAgentMessageTTSResult {
  speak: (messageId: string, text: string) => void;
  stop: () => void;
  isSpeaking: boolean;
  speakingMessageId: string | null;
  loadingMessageId: string | null;
  unavailable: boolean;
}

export function useAgentMessageTTS(): UseAgentMessageTTSResult {
  const unavailableReason = useAudioToolsUnavailableReason();
  const unavailable = Boolean(unavailableReason);
  const [speakingMessageId, setSpeakingMessageId] = useState<string | null>(null);
  // "Loading" is the synth→play gap: speak() was called but the core hasn't
  // flipped isSpeaking yet. The core no longer exposes a dedicated isLoading
  // flag, so we track it locally and clear it once playback actually starts.
  const [loadingMessageId, setLoadingMessageId] = useState<string | null>(null);

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
      setLoadingMessageId(messageId);
      tts.speak(text);
    },
    [tts, unavailable],
  );

  const stop = useCallback(() => {
    tts.stop();
    setSpeakingMessageId(null);
    setLoadingMessageId(null);
  }, [tts]);

  // Drive the message-id bookkeeping off the core's isSpeaking transitions.
  // Tracking the previous value lets us tell the synth→play gap (never spoke
  // yet) apart from a natural end (was speaking, now stopped).
  const wasSpeakingRef = useRef(false);
  useEffect(() => {
    if (tts.isSpeaking) {
      wasSpeakingRef.current = true;
      setLoadingMessageId(null);
    } else if (wasSpeakingRef.current) {
      wasSpeakingRef.current = false;
      setSpeakingMessageId(null);
    }
  }, [tts.isSpeaking]);

  // If synthesis fails before playback ever starts, clear the loading/speaking
  // ids so the bubble doesn't hang on a spinner.
  useEffect(() => {
    if (tts.error && !tts.isSpeaking) {
      setLoadingMessageId(null);
      setSpeakingMessageId(null);
    }
  }, [tts.error, tts.isSpeaking]);

  return {
    speak,
    stop,
    isSpeaking: tts.isSpeaking,
    speakingMessageId,
    loadingMessageId,
    unavailable,
  };
}
