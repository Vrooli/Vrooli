// DOC: docs/internal/SEAMS.md#tts-provider-seam
//
// Text-To-Speech Hook — Web-Console Adapter
// ==========================================
//
// Thin adapter over `useTextToSpeechCore` (audio-integration). The core owns
// provider lifecycle (Kokoro vs Browser), synthesize → cache → play, playback
// state, and the queue-management for autoplay. This adapter contributes the
// web-console-specific glue:
//   - workspace-store reads (startMutedOnLoad)
//   - Kokoro availability probe via the local capabilities surface
//   - forwarding playback lifecycle events to `/api/v1/tts-hook/playback`

import { useCallback, useMemo } from "react";
import { useTextToSpeechCore, AUDIO_TOOLS_CAPABILITY_SLUG, featureSlug, AudioToolsFeature } from "../audio-integration";
import type { TTSCorePlaybackEvent } from "../audio-integration";
import { recordTTSPlaybackEvent } from "../api/ttsHook";
import { fetchCapabilitiesLivenessCached, _resetCapabilitiesCache } from "../api/capabilities";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";

export interface TTSSettings {
  /** Browser TTS voice name */
  voice: string;
  rate: number;
  pitch: number;
  /** Kokoro voice ID */
  kokoroVoice: string;
  kokoroSpeed: number;
  /** User preference: "auto" picks best available */
  backendPreference: "auto" | "kokoro" | "browser";
}

export interface TTSDiagnostics {
  source: string;
  sessionId?: string;
}

/** Probe `/api/capabilities` for the audio-tools TTS surface. Used by the
 *  core to decide between the audio-tools-backed Kokoro path and Browser
 *  speech synthesis on mount. */
async function probeKokoroAvailable(): Promise<boolean> {
  // Reset the cached capabilities so a manual `refresh()` (which is what
  // resolveBackend ultimately calls) sees fresh data on each invocation,
  // matching the pre-extraction behavior of `checkBackend(forceRefresh)`.
  _resetCapabilitiesCache();
  try {
    const caps = await fetchCapabilitiesLivenessCached();
    const voiceOutputSlug = featureSlug(AudioToolsFeature.VOICE_OUTPUT);
    return caps.capabilities.some(
      (c) => c.status === "available" && (
        (c.id === AUDIO_TOOLS_CAPABILITY_SLUG && c.features.includes(voiceOutputSlug)) ||
        // Backward-compatible for older web-console builds/tests that still
        // exposed the pre-audio-tools resource capability directly.
        c.id === "kokoro-tts"
      ),
    );
  } catch {
    return false;
  }
}

export function useTextToSpeech(settings: TTSSettings, diagnostics?: TTSDiagnostics) {
  const startMutedOnLoad = useWorkspaceStore((store) => store.startMutedOnLoad);

  const onPlaybackEvent = useCallback((ev: TTSCorePlaybackEvent) => {
    if (!diagnostics?.source) return;
    void recordTTSPlaybackEvent({
      source: diagnostics.source,
      sessionId: diagnostics.sessionId,
      stage: ev.stage,
      backend: ev.backend,
      message: ev.message,
    }).catch(() => {});
  }, [diagnostics?.source, diagnostics?.sessionId]);

  const coreOpts = useMemo(() => ({
    autoEnabled: true,
    backend: settings.backendPreference,
    startMuted: startMutedOnLoad,
    defaultVoice: settings.voice,
    defaultSpeed: settings.rate,
    onPlaybackEvent,
    kokoroAvailable: probeKokoroAvailable,
    // Keep in-progress speech alive across pane unmount / warm-set eviction:
    // the workspace only mounts a warm set of panes, so a session evicted
    // mid-utterance would otherwise have its provider disposed and its audio
    // truncated. Keying persistence on the session id lets a remounted pane
    // re-adopt the same, still-playing provider (single owner, no leak).
    playbackOwnerKey: diagnostics?.sessionId,
    persistPlaybackAcrossUnmount: Boolean(diagnostics?.sessionId),
  }), [
    onPlaybackEvent,
    settings.backendPreference,
    settings.rate,
    settings.voice,
    startMutedOnLoad,
    diagnostics?.sessionId,
  ]);

  return useTextToSpeechCore(coreOpts, settings);
}
