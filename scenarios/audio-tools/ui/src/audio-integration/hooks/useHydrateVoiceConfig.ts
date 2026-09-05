// Mounts at app root. Fetches audio-tools' StreamConfig once on mount
// and writes it into useVoiceConfigStore so downstream consumers (mic
// button ring, segment-boundary emission) see authoritative values
// without each issuing its own fetch.

import { useEffect } from "react";

import { getVoiceStreamConfig } from "../api/voice";
import { useAudioToolsUnavailableReason } from "../client";
import { useVoiceConfigStore } from "./useVoiceConfigStore";

export function useHydrateVoiceConfig(): void {
  const unavailableReason = useAudioToolsUnavailableReason();

  useEffect(() => {
    if (unavailableReason) return;
    let cancelled = false;
    getVoiceStreamConfig()
      .then((cfg) => {
        if (cancelled) return;
        useVoiceConfigStore.getState().setFromServer({
          vadSilenceMs: cfg.vadSilenceMs,
          segmentSilenceMs: cfg.segmentSilenceMs,
          persistentMode: cfg.persistentMode,
          wakeWordEnabled: cfg.wakeWordEnabled,
        });
      })
      .catch(() => {
        // Audio-tools unreachable on first paint — leave fallback values
        // in place; useAudioToolsUnavailableReason will already surface
        // the error to the user.
      });
    return () => {
      cancelled = true;
    };
  }, [unavailableReason]);
}
// HOST DIFFERENCE: audio-tools hydrates voice configuration from its own settings surface.
