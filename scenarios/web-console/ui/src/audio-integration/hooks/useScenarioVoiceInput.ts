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

import { useCallback, useEffect, useMemo } from "react";
import {
  getVoiceStreamConfig,
  useVoiceCore,
  AUDIO_TOOLS_CAPABILITY_SLUG,
  featureSlug,
  AudioToolsFeature,
} from "../index";
import type { VoiceCapabilityProbe } from "../index";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { parseCommandDirect } from "../../hooks/voice/commandParser";
import type { CommandSuggestion } from "../index";

// Re-export public types and utilities for consumers and tests
export type { TranscriptionProvider, VoiceBackend, VoiceState, VoiceMode, VoiceInputState, VoiceSegment, VoiceRejection, LastTurnAudio, CommandSuggestion, StartRecordingOpts, VoiceActivitySnapshot, VoiceActivityPhase } from "../index";
export { WHISPER_FAILED_SENTINEL, AUDIO_BITRATE, STREAM_CHUNK_INTERVAL_MS, computeFinalTimeout } from "../index";
export { createAudioFilterChain } from "../index";
export type { VadState, VadRefs, VadAction, CachedNoiseFloor } from "../index";
export { VAD_FALLBACK_SILENCE_TIMEOUT_MS, VAD_FALLBACK_SEGMENT_SILENCE_MS, VAD_FLOOR_CACHE_MAX_AGE_MS, createVadRefs, createVadRefsFromCache, extractCacheableFloor, loadNoiseFloorCache, saveNoiseFloorCache, computeSlidingNoiseFloor, vadTick } from "../index";
export { buildVoiceActivitySnapshot, VAD_AUTO_STOP_VISUAL_GRACE_MS } from "../index";
export { getSharedAudioContext, closeSharedAudioContext } from "../index";

export interface UseVoiceInputCallbacks {
  /** Called when a completed transcript is available (both one-shot and persistent). */
  onTranscript: (text: string) => void;
  /** Called when a voice command is confirmed by the user. */
  onCommandExecute?: (suggestion: CommandSuggestion) => void;
}

type CapabilitiesApi = typeof import("../../api/capabilities");

let capabilitiesApiPromise: Promise<CapabilitiesApi> | undefined;

function loadCapabilitiesApi(): Promise<CapabilitiesApi> {
  capabilitiesApiPromise ??= import("../../api/capabilities");
  return capabilitiesApiPromise;
}

/**
 * Probe the web-console capabilities surface for Whisper health. The
 * synchronous liveness snapshot is preferred when available; otherwise we
 * fall back to the full capabilities fetch on the first probe and let the
 * background refresh interval keep the snapshot warm.
 */
export async function probeWhisperHealth(): Promise<VoiceCapabilityProbe> {
  const {
    fetchCapabilities,
    getCapabilitiesLivenessSnapshot,
    refreshCapabilitiesLiveness,
  } = await loadCapabilitiesApi();
  // The capability is registered under audio-tools' scenario slug; the
  // STT features it exposes are sourced from the proto-backed slug map
  // in audio-integration/features.ts. Do NOT hard-code these strings —
  // a "whisper-stt" literal hung around for months after the audio-tools
  // extraction and silently degraded voice input to Web Speech.
  const voiceInputSlug = featureSlug(AudioToolsFeature.VOICE_INPUT);
  const voiceStreamingSlug = featureSlug(AudioToolsFeature.VOICE_STREAMING);

  // Liveness snapshot is populated by refreshCapabilitiesLiveness on a 25s
  // interval. When it's available, prefer it to avoid a network roundtrip.
  const snapshot = getCapabilitiesLivenessSnapshot();
  if (snapshot) {
    const audio = snapshot.capabilities.find((c) => c.id === AUDIO_TOOLS_CAPABILITY_SLUG);
    if (audio?.status === "available" && audio.features?.includes(voiceInputSlug)) {
      return {
        whisperHealthy: true,
        streamingAvailable: audio.features?.includes(voiceStreamingSlug) ?? false,
      };
    }
    return { whisperHealthy: false, streamingAvailable: false };
  }

  // No snapshot yet — fetch fresh capabilities. Also kick off a liveness
  // refresh so subsequent probes have a snapshot to read.
  refreshCapabilitiesLiveness().catch(() => {});
  const caps = await fetchCapabilities();
  const audio = caps.capabilities.find((c) => c.id === AUDIO_TOOLS_CAPABILITY_SLUG);
  if (audio?.status === "available" && audio.features?.includes(voiceInputSlug)) {
    return {
      whisperHealthy: true,
      streamingAvailable: audio.features?.includes(voiceStreamingSlug) ?? false,
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

export function useScenarioVoiceInput(onTranscript: (text: string) => void) {
  const voiceEnabled = useWorkspaceStore((s) => s.voiceEnabled);
  const voiceLanguage = useWorkspaceStore((s) => s.voiceLanguage);
  const vadSilenceTimeoutMs = useWorkspaceStore((s) => s.vadSilenceTimeoutMs);
  const persistentMode = useWorkspaceStore((s) => s.persistentMode);
  const wakeWordEnabled = useWorkspaceStore((s) => s.wakeWordEnabled);
  const wakeWordThreshold = useWorkspaceStore((s) => s.wakeWordThreshold);
  const segmentSilenceMs = useWorkspaceStore((s) => s.segmentSilenceMs);

  // Hydrate workspace store from backend voice-stream config on mount AND
  // whenever the Settings modal opens (cheap unary RPC, gives near-real-time
  // sync without WatchStreamConfig). audio-tools' vad_silence_ms is the
  // single source of truth for both vadSilenceTimeoutMs (auto-stop ring
  // countdown) and segmentSilenceMs (segment boundary emission); both
  // client-side concepts collapse to one server lever.
  const settingsModalOpen = useWorkspaceStore((s) => s.settingsModalOpen);
  useEffect(() => {
    if (!voiceEnabled) return;
    let cancelled = false;
    getVoiceStreamConfig()
      .then((cfg) => {
        if (cancelled) return;
        const store = useWorkspaceStore.getState();
        if (cfg.persistentMode !== store.persistentMode) store.setPersistentMode(cfg.persistentMode);
        if (cfg.wakeWordEnabled !== store.wakeWordEnabled) store.setWakeWordEnabled(cfg.wakeWordEnabled);
        // The server's wake_word_threshold is the durable source of truth for
        // match sensitivity; hydrate the store from it (guard against an unset
        // server value of 0, which would otherwise wipe the user's setting).
        if (cfg.wakeWordThreshold > 0 && cfg.wakeWordThreshold !== store.wakeWordThreshold) {
          store.setWakeWordThreshold(cfg.wakeWordThreshold);
        }
        // Prefer vad_silence_ms (the authoritative server VAD knob) over
        // the legacy segment_silence_ms field. Fall back when zero.
        const serverSilenceMs = cfg.vadSilenceMs > 0 ? cfg.vadSilenceMs : cfg.segmentSilenceMs;
        if (serverSilenceMs > 0) {
          if (serverSilenceMs !== store.segmentSilenceMs) store.setSegmentSilenceMs(serverSilenceMs);
          if (serverSilenceMs !== store.vadSilenceTimeoutMs) store.setVadSilenceTimeoutMs(serverSilenceMs);
        }
      })
      .catch(() => { /* Use store defaults */ });
    return () => { cancelled = true; };
  }, [voiceEnabled, settingsModalOpen]);

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
    wakeWordThreshold,
    segmentSilenceMs,
    capabilityCheck,
    parseCommand,
    onTranscript,
  }), [
    voiceEnabled,
    voiceLanguage,
    vadSilenceTimeoutMs,
    persistentMode,
    wakeWordEnabled,
    wakeWordThreshold,
    segmentSilenceMs,
    capabilityCheck,
    parseCommand,
    onTranscript,
  ]));
}
