// useAgentMessageTTS — scenario-side adapter over useTextToSpeechCore.
// Exposes a stable speak/stop pair keyed by message id so multiple
// bubbles render their own speaking state without bookkeeping.
//
// Voice/speed settings come from audio-tools' shared config; auto-speak
// is a swarm-manager-local pref in useAudioPrefs.

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTextToSpeechCore, useAudioToolsUnavailableReason } from "../audio-integration";
import * as ttsRuntime from "../audio-integration/api/tts";

/** The slice of useTextToSpeechCore's result this adapter actually consumes. */
interface TTSCoreSlice {
  speak: (text: string) => void;
  stop: () => void;
  isSpeaking: boolean;
  error: string | null;
}

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
      runtime: {
        synthesizeTTS: ttsRuntime.synthesizeTTS,
        synthesizeTTSWithMetrics: ttsRuntime.synthesizeTTSWithMetrics,
        fetchCachedTTS: ttsRuntime.fetchCachedTTS,
        getTTSVoices: ttsRuntime.getTTSVoices,
        reportTTSPlayStart: ttsRuntime.reportTTSPlayStart,
      },
    },
    {
      voice: "",
      rate: 1.0,
      pitch: 1.0,
      kokoroVoice: "af_heart",
      kokoroSpeed: 1.0,
    },
  );

  // Destructured rather than closing over `tts`: useTextToSpeechCore returns a
  // fresh object literal on every render (it spreads its state into the
  // result), so depending on the whole controller would invalidate these
  // callbacks — and everything memoized downstream of them — continuously. The
  // individual functions are useCallback'd inside the core and are the actual
  // dependencies.
  //
  // Narrowed through an explicit type because the shared audio package resolves
  // as `any` here; without it every use below is an unchecked call.
  const { speak: coreSpeak, stop: coreStop, isSpeaking: coreIsSpeaking, error: coreError } = tts as TTSCoreSlice;

  const speak = useCallback(
    (messageId: string, text: string) => {
      if (unavailable || !text.trim()) return;
      setSpeakingMessageId(messageId);
      setLoadingMessageId(messageId);
      coreSpeak(text);
    },
    [coreSpeak, unavailable],
  );

  const stop = useCallback(() => {
    coreStop();
    setSpeakingMessageId(null);
    setLoadingMessageId(null);
  }, [coreStop]);

  // Drive the message-id bookkeeping off the core's isSpeaking transitions.
  // Tracking the previous value lets us tell the synth→play gap (never spoke
  // yet) apart from a natural end (was speaking, now stopped).
  const wasSpeakingRef = useRef(false);
  useEffect(() => {
    if (coreIsSpeaking) {
      wasSpeakingRef.current = true;
      setLoadingMessageId(null);
    } else if (wasSpeakingRef.current) {
      wasSpeakingRef.current = false;
      setSpeakingMessageId(null);
    }
  }, [coreIsSpeaking]);

  // If synthesis fails before playback ever starts, clear the loading/speaking
  // ids so the bubble doesn't hang on a spinner.
  useEffect(() => {
    if (coreError && !coreIsSpeaking) {
      setLoadingMessageId(null);
      setSpeakingMessageId(null);
    }
  }, [coreError, coreIsSpeaking]);

  // Memoized because this controller is handed to every message bubble and is
  // a dependency of ChatThread's auto-speak effect. Returned as a fresh object
  // literal it defeated bubble memoization outright and re-ran that effect on
  // every render of the thread, including the 3s session poll.
  return useMemo(
    () => ({
      speak,
      stop,
      isSpeaking: coreIsSpeaking,
      speakingMessageId,
      loadingMessageId,
      unavailable,
    }),
    [speak, stop, coreIsSpeaking, speakingMessageId, loadingMessageId, unavailable],
  );
}
