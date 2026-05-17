// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Voice Input Hook — Web-Console Adapter
// =======================================
//
// Thin adapter over `useVoiceCore` (audio-integration). All of the state-
// machine, VAD, RAF loop, wake-word, and provider lifecycle lives in the
// generic core; this adapter wires in web-console-specific concerns:
//   - workspace store reads (voice settings)
//   - capability probe via the web-console `/api/capabilities` surface
//   - voice-command parsing via the local commandParser

import { useCallback, useEffect, useMemo, useRef } from "react";
import { getVoiceStreamConfig, useVoiceCore } from "../audio-integration";
import type { VoiceCapabilityProbe } from "../audio-integration";
import { fetchCapabilities, getCapabilitiesLivenessSnapshot, refreshCapabilitiesLiveness } from "../api/capabilities";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { parseCommandDirect } from "./voice/commandParser";
import type { CommandSuggestion } from "../audio-integration";

// Re-export public types and utilities for consumers and tests
export type { TranscriptionProvider, VoiceBackend, VoiceState, VoiceMode, VoiceInputState, VoiceSegment, VoiceRejection, LastTurnAudio, CommandSuggestion, StartRecordingOpts, VoiceActivitySnapshot, VoiceActivityPhase } from "../audio-integration";
export { WHISPER_FAILED_SENTINEL, CAP_CHECK_FAIL_THRESHOLD, AUDIO_BITRATE, STREAM_CHUNK_INTERVAL_MS, computeFinalTimeout } from "../audio-integration";
export { createAudioFilterChain } from "../audio-integration";
export type { VadState, VadRefs, VadAction, CachedNoiseFloor } from "../audio-integration";
export { VAD_DEFAULT_SILENCE_TIMEOUT_MS, VAD_DEFAULT_SEGMENT_SILENCE_MS, VAD_FLOOR_CACHE_MAX_AGE_MS, createVadRefs, createVadRefsFromCache, extractCacheableFloor, loadNoiseFloorCache, saveNoiseFloorCache, computeSlidingNoiseFloor, vadTick } from "../audio-integration";
export { buildVoiceActivitySnapshot, VAD_AUTO_STOP_VISUAL_GRACE_MS } from "../audio-integration";
export { getSharedAudioContext, ensureAudioContextOnGesture, closeSharedAudioContext } from "../audio-integration";

export interface UseVoiceInputCallbacks {
  /** Called when a completed transcript is available (both one-shot and persistent). */
  onTranscript: (text: string) => void;
  /** Called when a voice command is confirmed by the user. */
  onCommandExecute?: (suggestion: CommandSuggestion) => void;
}

/**
 * Probe the web-console capabilities surface for Whisper health. The
 * synchronous liveness snapshot is preferred when available; otherwise we
 * fall back to the full capabilities fetch on the first probe and let the
 * background refresh interval keep the snapshot warm.
 */
async function probeWhisperHealth(): Promise<VoiceCapabilityProbe> {
  // Liveness snapshot is populated by refreshCapabilitiesLiveness on a 25s
  // interval. When it's available, prefer it to avoid a network roundtrip.
  const snapshot = getCapabilitiesLivenessSnapshot();
  if (snapshot) {
    const whisper = snapshot.capabilities.find((c) => c.id === "whisper-stt");
    if (whisper?.status === "available") {
      return {
        whisperHealthy: true,
        streamingAvailable: whisper.features?.includes("voice-streaming") ?? false,
      };
    }
    return { whisperHealthy: false, streamingAvailable: false };
  }

  // No snapshot yet — fetch fresh capabilities. Also kick off a liveness
  // refresh so subsequent probes have a snapshot to read.
  refreshCapabilitiesLiveness().catch(() => {});
  const caps = await fetchCapabilities();
  const whisper = caps.capabilities.find((c) => c.id === "whisper-stt");
  if (whisper?.status === "available") {
    return {
      whisperHealthy: true,
      streamingAvailable: whisper.features?.includes("voice-streaming") ?? false,
    };
  }
  return { whisperHealthy: false, streamingAvailable: false };
}

/**
 * Map a parsed-command result from `parseCommandDirect` into the
 * audio-integration `CommandSuggestion` shape the core expects.
 */
function buildSuggestion(text: string): CommandSuggestion | null {
  const parsed = parseCommandDirect(text);
  if (!parsed) return null;
  return {
    id: `cmd-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    commandId: parsed.command.id,
    description: parsed.command.description,
    confidence: parsed.confidence,
    rawText: parsed.rawText,
    timestamp: Date.now(),
    args: parsed.args,
  };
}

export function useVoiceInput(onTranscript: (text: string) => void) {
  const voiceEnabled = useWorkspaceStore((s) => s.voiceEnabled);
  const voiceLanguage = useWorkspaceStore((s) => s.voiceLanguage);
  const vadSilenceTimeoutMs = useWorkspaceStore((s) => s.vadSilenceTimeoutMs);
  const persistentMode = useWorkspaceStore((s) => s.persistentMode);
  const wakeWordEnabled = useWorkspaceStore((s) => s.wakeWordEnabled);
  const segmentSilenceMs = useWorkspaceStore((s) => s.segmentSilenceMs);
  const lowLatencyVoice = useWorkspaceStore((s) => s.lowLatencyVoice);

  // Hydrate workspace store from backend voice-stream config on mount, so the
  // store has authoritative values even before the user opens Settings.
  // Lives here (not in the core) because writing to the host store is a
  // host concern — the core takes config via opts.
  const hydratedRef = useRef(false);
  useEffect(() => {
    if (!voiceEnabled || hydratedRef.current) return;
    hydratedRef.current = true;
    getVoiceStreamConfig()
      .then((cfg) => {
        const store = useWorkspaceStore.getState();
        if (cfg.persistentMode !== store.persistentMode) store.setPersistentMode(cfg.persistentMode);
        if (cfg.wakeWordEnabled !== store.wakeWordEnabled) store.setWakeWordEnabled(cfg.wakeWordEnabled);
        if (cfg.segmentSilenceMs && cfg.segmentSilenceMs !== store.segmentSilenceMs) {
          store.setSegmentSilenceMs(cfg.segmentSilenceMs);
        }
      })
      .catch(() => { /* Use store defaults */ });
  }, [voiceEnabled]);

  // Capabilities probe is a stable, ref-driven function — recreating it on
  // every render would force the core to reinstall its capability refs.
  const capabilityCheck = useCallback(probeWhisperHealth, []);
  const parseCommand = useCallback(buildSuggestion, []);

  return useVoiceCore(useMemo(() => ({
    voiceEnabled,
    voiceLanguage,
    vadSilenceTimeoutMs,
    persistentMode,
    wakeWordEnabled,
    segmentSilenceMs,
    lowLatencyVoice,
    capabilityCheck,
    parseCommand,
    onTranscript,
  }), [
    voiceEnabled,
    voiceLanguage,
    vadSilenceTimeoutMs,
    persistentMode,
    wakeWordEnabled,
    segmentSilenceMs,
    lowLatencyVoice,
    capabilityCheck,
    parseCommand,
    onTranscript,
  ]));
}
