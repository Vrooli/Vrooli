// DOC: docs/internal/SEAMS.md#tts-provider-seam
import { useState, useEffect, useCallback, useRef } from "react";
import { fetchCachedTTS, getTTSVoices, synthesizeTTS } from "@audio-tools/embed";
// Playback events flow to web-console's own /api/v1/tts-hook/playback —
// playback success/failure is a web-console concern (it informs the Claude
// hook routing UI), not an audio-tools concern.
import { recordTTSPlaybackEvent } from "../api/ttsHook";
import { fetchCapabilitiesLivenessCached, _resetCapabilitiesCache } from "../api/capabilities";
import type { TTSBackend, TTSPlaybackCapabilities, TTSPlaybackState, TTSProvider, TTSVoiceInfo } from "@audio-tools/embed";
import { KokoroProvider } from "@audio-tools/embed";
import { BrowserTTSProvider } from "@audio-tools/embed";
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

const NO_CAPABILITIES: TTSPlaybackCapabilities = {
  canPause: false,
  canSeek: false,
  canAdjustSpeed: false,
  canAdjustVolume: false,
};

export interface TTSState {
  supported: boolean;
  isSpeaking: boolean;
  isPaused: boolean;
  currentTime: number;
  duration: number | null;
  playbackRate: number;
  volume: number;
  capabilities: TTSPlaybackCapabilities;
  /** Whether playback is muted. Independent of `volume` — the provider receives
   *  `effectiveVolume = isMuted ? 0 : volume` so unmuting restores the slider value. */
  isMuted: boolean;
  backend: TTSBackend;
  voices: TTSVoiceInfo[];
  error: string | null;
  backendReason: string;
  browserAudioReady: boolean;
  lastSuccessfulBackend: TTSBackend;
  lastSuccessfulAt: string | null;
  /**
   * True when a playback attempt was rejected by the browser's autoplay
   * policy and the audio element has not yet been unlocked via a qualifying
   * user gesture. Consumers should surface an "Enable voice" affordance that
   * calls `unlockAudio()` and then retries.
   */
  needsUnlock: boolean;
}

export interface TTSDiagnostics {
  source: string;
  sessionId?: string;
}

/** Default Kokoro voice used when the voice list cannot be fetched. */
const KOKORO_DEFAULT_VOICE = "af_heart";
const TEST_TTS_SAMPLE = "This is a TTS test from web console.";

function isBrowserSupported(): boolean {
  return typeof window !== "undefined" && "speechSynthesis" in window;
}

function isAbortLikeError(err: unknown): boolean {
  return err instanceof Error && (err.name === "AbortError" || err.message === "The operation was aborted.");
}

// The browser's autoplay policy rejects programmatic HTMLAudioElement.play()
// calls that happen outside a qualifying user-gesture call stack. Detecting
// it lets us route to the Enable-Audio affordance instead of flashing a
// transient error banner for every paragraph in the speak chain.
function isAutoplayBlocked(err: unknown): boolean {
  if (!(err instanceof Error)) return false;
  return (
    err.name === "NotAllowedError" ||
    /not allowed by the user agent/i.test(err.message)
  );
}

export function useTextToSpeech(settings: TTSSettings, diagnostics?: TTSDiagnostics) {
  const startMutedOnLoad = useWorkspaceStore((store) => store.startMutedOnLoad);
  const initialMuted = startMutedOnLoad;
  const [state, setState] = useState<TTSState>({
    supported: isBrowserSupported(),
    isSpeaking: false,
    isPaused: false,
    currentTime: 0,
    duration: null,
    playbackRate: 1,
    volume: 1,
    isMuted: initialMuted,
    capabilities: NO_CAPABILITIES,
    backend: "none",
    voices: [],
    error: null,
    backendReason: "Checking TTS backend availability\u2026",
    browserAudioReady: false,
    lastSuccessfulBackend: "none",
    lastSuccessfulAt: null,
    needsUnlock: false,
  });

  const providerRef = useRef<TTSProvider | null>(null);
  const activeProviderRef = useRef<TTSProvider | null>(null);
  const backendRef = useRef<TTSBackend>("none");
  const speakChainRef = useRef<AbortController | null>(null);
  const fallbackProviderRef = useRef<BrowserTTSProvider | null>(null);
  const audioUnlockedRef = useRef(false);
  const resolveBackendRef = useRef<(() => Promise<void>) | null>(null);
  // Mirror state.isMuted/state.volume so helpers can compute effective volume
  // synchronously without stale closures.
  const isMutedRef = useRef(initialMuted);
  const volumeRef = useRef(1);
  const hasExplicitMuteOverrideRef = useRef(false);

  const applyEffectiveVolume = useCallback(() => {
    const effective = isMutedRef.current ? 0 : volumeRef.current;
    providerRef.current?.setVolume?.(effective);
    if (activeProviderRef.current && activeProviderRef.current !== providerRef.current) {
      activeProviderRef.current.setVolume?.(effective);
    }
  }, []);

  const emitEvent = useCallback((stage: string, backend: TTSBackend, message?: string) => {
    if (!diagnostics?.source) return;
    void recordTTSPlaybackEvent({
      source: diagnostics.source,
      sessionId: diagnostics.sessionId,
      stage,
      backend,
      message,
    }).catch(() => {});
  }, [diagnostics]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const onGesture = () => {
      // Always flip the flag that gates BrowserTTSProvider — that only needs
      // to know a gesture occurred. The media-element unlock is separate.
      audioUnlockedRef.current = true;
      setState((s) => (s.browserAudioReady ? s : { ...s, browserAudioReady: true }));

      const provider = providerRef.current;
      if (!provider || provider.isUnlocked()) return;
      // Fire-and-forget: the play() inside unlock() must be kicked off
      // synchronously within this gesture call stack, but we don't await it.
      provider.unlock().then((ok) => {
        if (ok) {
          setState((s) => (s.needsUnlock ? { ...s, needsUnlock: false } : s));
        }
      });
    };
    window.addEventListener("pointerdown", onGesture, { passive: true });
    window.addEventListener("keydown", onGesture, { passive: true });
    window.addEventListener("touchstart", onGesture, { passive: true });
    return () => {
      window.removeEventListener("pointerdown", onGesture);
      window.removeEventListener("keydown", onGesture);
      window.removeEventListener("touchstart", onGesture);
    };
  }, []);

  // Wire progress callback so playback position updates flow into React state.
  // Only the small AudioPlayerBar re-renders from this (~4 Hz from timeupdate).
  useEffect(() => {
    const provider = providerRef.current;
    if (!provider?.onProgress) return;
    provider.onProgress((time, duration) => {
      setState((s) => ({ ...s, currentTime: time, duration }));
    });
    return () => {
      provider.onProgress?.(null);
    };
  }, [state.backend]); // re-wire when provider changes

  // Push effective volume to the active provider whenever it changes, so a
  // muted initial state isn't bypassed by a fresh provider's default volume.
  useEffect(() => {
    applyEffectiveVolume();
  }, [state.backend, applyEffectiveVolume]);

  useEffect(() => {
    if (hasExplicitMuteOverrideRef.current) return;
    isMutedRef.current = startMutedOnLoad;
    applyEffectiveVolume();
    setState((s) => (s.isMuted === startMutedOnLoad ? s : { ...s, isMuted: startMutedOnLoad }));
  }, [applyEffectiveVolume, startMutedOnLoad]);

  const updateSuccess = useCallback((backend: TTSBackend) => {
    setState((s) => ({
      ...s,
      isSpeaking: false,
      isPaused: false,
      error: null,
      needsUnlock: false,
      lastSuccessfulBackend: backend,
      lastSuccessfulAt: new Date().toISOString(),
    }));
  }, []);

  const runBrowserSpeak = useCallback(
    async (text: string, suppressBlockedError = false): Promise<"browser"> => {
      if (!isBrowserSupported()) {
        throw new Error("Browser speech synthesis is not supported");
      }
      if (!audioUnlockedRef.current && !suppressBlockedError) {
        throw new Error("Browser audio is blocked until you interact with the page");
      }
      if (!fallbackProviderRef.current) {
        fallbackProviderRef.current = new BrowserTTSProvider();
      }
      activeProviderRef.current = fallbackProviderRef.current;
      await fallbackProviderRef.current.speak(text, {
        voice: settings.voice,
        rate: settings.rate,
        pitch: settings.pitch,
      });
      return "browser";
    },
    [settings.pitch, settings.rate, settings.voice],
  );

  const executeSpeak = useCallback(async (text: string, paragraphs?: string[]): Promise<TTSBackend> => {
    const provider = providerRef.current;
    const segments = paragraphs ?? [text];
    if (!provider || segments.length === 0) {
      throw new Error("No TTS backend is available");
    }

    const kokoroOpts = { voice: settings.kokoroVoice, rate: settings.kokoroSpeed };
    const browserOpts = {
      voice: settings.voice,
      rate: settings.rate,
      pitch: settings.pitch,
    };

    const speakWithProvider = async () => {
      if (backendRef.current === "browser" && !audioUnlockedRef.current) {
        throw new Error("Browser audio is blocked until you interact with the page");
      }
      activeProviderRef.current = provider;
      // Ensure the provider's effective volume reflects the current mute state
      // before playback begins. Without this, the first speak after a fresh
      // page load can play audibly because the backend-change effect raced
      // with provider creation.
      applyEffectiveVolume();
      const opts = backendRef.current === "kokoro" ? kokoroOpts : browserOpts;
      // Use speakSequence for unified playback when the provider supports it
      // and there are multiple segments. This produces a single audio track
      // with accurate total duration and full seek/scrub support.
      if (segments.length > 1 && provider.speakSequence) {
        await provider.speakSequence(segments, opts);
      } else {
        for (const segment of segments) {
          await provider.speak(segment, opts);
        }
      }
      return backendRef.current;
    };

    try {
      return await speakWithProvider();
    } catch (err) {
      if (isAbortLikeError(err)) {
        throw err;
      }
      if (backendRef.current === "kokoro" && settings.backendPreference === "auto" && isBrowserSupported()) {
        const browserBackend = await runBrowserSpeak(paragraphs ? paragraphs.join("\n\n") : text, true);
        setState((s) => ({
          ...s,
          backendReason: "Kokoro failed at runtime; Browser handled playback for this request",
        }));
        return browserBackend;
      }
      throw err;
    }
  }, [
    applyEffectiveVolume,
    runBrowserSpeak,
    settings.backendPreference,
    settings.kokoroSpeed,
    settings.kokoroVoice,
    settings.pitch,
    settings.rate,
    settings.voice,
  ]);

  useEffect(() => {
    let cancelled = false;

    async function checkBackend(forceRefresh = false) {
      if (forceRefresh) {
        _resetCapabilitiesCache();
      }

      if (settings.backendPreference === "browser") {
        if (isBrowserSupported()) {
          const provider = new BrowserTTSProvider();
          backendRef.current = "browser";
          providerRef.current = provider;
          activeProviderRef.current = provider;
          if (!cancelled) {
            const voices = window.speechSynthesis.getVoices() ?? [];
            setState((s) => ({
              ...s,
              supported: true,
              backend: "browser",
              capabilities: provider.capabilities,
              voices: voices.map((v) => ({ id: v.name, name: v.name })),
              backendReason: "Browser backend selected explicitly",
            }));
          }
        } else if (!cancelled) {
          backendRef.current = "none";
          providerRef.current = null;
          activeProviderRef.current = null;
          setState((s) => ({
            ...s,
            supported: false,
            backend: "none",
            capabilities: NO_CAPABILITIES,
            voices: [],
            backendReason: "Browser backend was selected, but speech synthesis is not supported in this browser",
          }));
        }
        return;
      }

      try {
        const caps = await fetchCapabilitiesLivenessCached();
        const kokoro = caps.capabilities.find(
          (c) => c.id === "kokoro-tts" && c.status === "available",
        );

        if (!cancelled && kokoro) {
          // Inject synthesizeTTS pulled through the embed package barrel so
          // vi.mock("@audio-tools/embed", …) in tests intercepts the call.
          const provider = new KokoroProvider({ synthesize: synthesizeTTS });
          backendRef.current = "kokoro";
          providerRef.current = provider;
          activeProviderRef.current = provider;
          try {
            const voices = await getTTSVoices();
            if (!cancelled) {
              setState((s) => ({
                ...s,
                supported: true,
                backend: "kokoro",
                capabilities: provider.capabilities,
                voices,
                backendReason: settings.backendPreference === "kokoro"
                  ? "Kokoro backend selected explicitly"
                  : "Kokoro is available and preferred over browser speech synthesis",
              }));
            }
          } catch {
            if (!cancelled) {
              setState((s) => ({
                ...s,
                supported: true,
                backend: "kokoro",
                capabilities: provider.capabilities,
                voices: [{ id: KOKORO_DEFAULT_VOICE, name: KOKORO_DEFAULT_VOICE }],
                backendReason: settings.backendPreference === "kokoro"
                  ? "Kokoro backend selected explicitly"
                  : "Kokoro is available and preferred over browser speech synthesis",
              }));
            }
          }
          return;
        }
      } catch {
        if (settings.backendPreference === "kokoro" && !cancelled) {
          backendRef.current = "none";
          providerRef.current = null;
          activeProviderRef.current = null;
          setState((s) => ({
            ...s,
            supported: false,
            backend: "none",
            capabilities: NO_CAPABILITIES,
            voices: [],
            backendReason: "Kokoro backend was selected explicitly, but availability could not be confirmed",
          }));
          return;
        }
      }

      if (!cancelled && isBrowserSupported()) {
        if (settings.backendPreference === "kokoro") {
          backendRef.current = "none";
          providerRef.current = null;
          activeProviderRef.current = null;
          setState((s) => ({
            ...s,
            supported: false,
            backend: "none",
            capabilities: NO_CAPABILITIES,
            voices: [],
            backendReason: "Kokoro backend was selected explicitly, but Kokoro is unavailable",
          }));
        } else {
          const provider = new BrowserTTSProvider();
          backendRef.current = "browser";
          providerRef.current = provider;
          activeProviderRef.current = provider;
          const voices = window.speechSynthesis.getVoices();
          setState((s) => ({
            ...s,
            supported: true,
            backend: "browser",
            capabilities: provider.capabilities,
            voices: voices.map((v) => ({ id: v.name, name: v.name })),
            backendReason: "Kokoro is unavailable, so browser speech synthesis is active",
          }));
        }
      } else if (!cancelled) {
        backendRef.current = "none";
        providerRef.current = null;
        activeProviderRef.current = null;
        setState((s) => ({
          ...s,
          supported: false,
          backend: "none",
          capabilities: NO_CAPABILITIES,
          voices: [],
          backendReason: settings.backendPreference === "kokoro"
            ? "Kokoro backend was selected explicitly, but Kokoro is unavailable and browser speech synthesis is not supported"
            : "No TTS backend is available. Kokoro is unavailable and browser speech synthesis is not supported",
        }));
      }
    }

    resolveBackendRef.current = () => checkBackend(true);
    checkBackend();
    return () => {
      cancelled = true;
      resolveBackendRef.current = null;
    };
  }, [settings.backendPreference]);

  // Load browser voices asynchronously (some browsers load them lazily)
  useEffect(() => {
    if (backendRef.current !== "browser" || !isBrowserSupported()) return;
    const loadVoices = () => {
      const voices = window.speechSynthesis.getVoices() ?? [];
      if (voices.length > 0) {
        setState((s) => ({
          ...s,
          voices: voices.map((v) => ({ id: v.name, name: v.name })),
        }));
      }
    };
    loadVoices();
    window.speechSynthesis.onvoiceschanged = loadVoices;
    return () => {
      window.speechSynthesis.onvoiceschanged = null;
    };
  }, [state.backend]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      providerRef.current?.dispose();
      fallbackProviderRef.current?.dispose();
      activeProviderRef.current = null;
      speakChainRef.current?.abort();
    };
  }, []);

  const speak = useCallback(
    (text: string) => {
      speakChainRef.current?.abort();
      providerRef.current?.stop();

      if (!providerRef.current) return;

      // Track this invocation so async handlers can detect supersession.
      const controller = new AbortController();
      speakChainRef.current = controller;

      setState((s) => ({ ...s, isSpeaking: true, isPaused: false }));
      emitEvent("attempt", backendRef.current);

      executeSpeak(text).then(
        (usedBackend) => {
          if (controller.signal.aborted) return;
          emitEvent("success", usedBackend);
          updateSuccess(usedBackend);
        },
        (err: unknown) => {
          if (isAbortLikeError(err)) {
            // Don't clear isSpeaking if a new chain has superseded us.
            if (speakChainRef.current === controller) {
              setState((s) => ({ ...s, isSpeaking: false, isPaused: false }));
            }
            return;
          }
          const message = err instanceof Error ? err.message : "Speech failed";
          if (isAutoplayBlocked(err)) {
            emitEvent("autoplay_blocked", backendRef.current, message);
            setState((s) => ({
              ...s,
              isSpeaking: false,
              isPaused: false,
              needsUnlock: true,
            }));
            return;
          }
          emitEvent("error", backendRef.current, message);
          setState((s) => ({
            ...s,
            isSpeaking: false,
            isPaused: false,
            error: message,
          }));
        },
      ).finally(() => {
        if (!controller.signal.aborted) {
          speakChainRef.current = null;
        }
      });
    },
    [emitEvent, executeSpeak, updateSuccess],
  );

  const speakParagraphs = useCallback(
    async (
      paragraphs: string[],
      opts?: { eventId?: string; version?: "active" | "original" },
    ): Promise<TTSBackend | undefined> => {
      speakChainRef.current?.abort();
      const controller = new AbortController();
      speakChainRef.current = controller;

      if (!providerRef.current || paragraphs.length === 0) return;

      setState((s) => ({ ...s, isSpeaking: true, isPaused: false }));
      emitEvent("attempt", backendRef.current);

      try {
        // Cache-first path: if eventId is provided and backend is Kokoro,
        // try fetching pre-cached audio before falling back to synthesis.
        if (opts?.eventId && backendRef.current === "kokoro" && providerRef.current) {
          const provider = providerRef.current as KokoroProvider;
          const blob = await fetchCachedTTS(
            opts.eventId,
            settings.kokoroVoice,
            settings.kokoroSpeed,
            opts.version ?? "active",
            controller.signal,
          );
          if (blob && !controller.signal.aborted) {
            activeProviderRef.current = provider;
            applyEffectiveVolume();
            try {
              await provider.speakFromBlob(blob);
            } catch (err) {
              if (controller.signal.aborted || isAbortLikeError(err)) throw err;
              // Cache-first blob playback was rejected. If it was an autoplay
              // block AND we can fall back to browser speech, do so with the
              // joined paragraphs (same behavior as executeSpeak's fallback).
              if (
                isAutoplayBlocked(err)
                && settings.backendPreference === "auto"
                && isBrowserSupported()
              ) {
                await runBrowserSpeak(paragraphs.join("\n\n"), true);
                if (!controller.signal.aborted) {
                  emitEvent("success", "browser");
                  updateSuccess("browser");
                  return "browser" as TTSBackend;
                }
                return;
              }
              throw err;
            }
            if (!controller.signal.aborted) {
              emitEvent("success", "kokoro");
              updateSuccess("kokoro");
              return "kokoro" as TTSBackend;
            }
            if (speakChainRef.current === controller) {
              setState((s) => ({ ...s, isSpeaking: false, isPaused: false }));
            }
            return;
          }
        }

        // Fall through to standard synthesis path.
        const usedBackend = await executeSpeak(paragraphs.join("\n\n"), paragraphs);
        if (controller.signal.aborted) {
          if (speakChainRef.current === controller) {
            setState((s) => ({ ...s, isSpeaking: false, isPaused: false }));
          }
          return;
        }
        emitEvent("success", usedBackend);
        updateSuccess(usedBackend);
        return usedBackend;
      } catch (err: unknown) {
        if (controller.signal.aborted || isAbortLikeError(err)) {
          if (speakChainRef.current === controller) {
            setState((s) => ({ ...s, isSpeaking: false, isPaused: false }));
          }
          return;
        }
        const message =
          err instanceof Error ? err.message : "Speech failed";
        if (isAutoplayBlocked(err)) {
          emitEvent("autoplay_blocked", backendRef.current, message);
          setState((s) => ({
            ...s,
            isSpeaking: false,
            isPaused: false,
            needsUnlock: true,
          }));
          return;
        }
        emitEvent("error", backendRef.current, message);
        setState((s) => ({ ...s, isSpeaking: false, isPaused: false, error: message }));
        throw err;
      } finally {
        if (!controller.signal.aborted) {
          speakChainRef.current = null;
        }
      }
    },
    [applyEffectiveVolume, emitEvent, executeSpeak, runBrowserSpeak, settings.backendPreference, settings.kokoroSpeed, settings.kokoroVoice, updateSuccess],
  );

  const stop = useCallback(() => {
    speakChainRef.current?.abort();
    providerRef.current?.stop();
    fallbackProviderRef.current?.stop();
    activeProviderRef.current = null;
    setState((s) => ({ ...s, isSpeaking: false, isPaused: false }));
  }, []);

  const pause = useCallback(() => {
    const provider = activeProviderRef.current ?? providerRef.current;
    provider?.pause?.();
    setState((s) => (provider ? { ...s, isPaused: true } : s));
  }, []);

  const resume = useCallback(() => {
    const provider = activeProviderRef.current ?? providerRef.current;
    provider?.resume?.();
    setState((s) => (provider ? { ...s, isPaused: false } : s));
  }, []);

  const seek = useCallback((seconds: number) => {
    (activeProviderRef.current ?? providerRef.current)?.seek?.(seconds);
  }, []);

  const setPlaybackRate = useCallback((rate: number) => {
    (activeProviderRef.current ?? providerRef.current)?.setPlaybackRate?.(rate);
    setState((s) => ({ ...s, playbackRate: rate }));
  }, []);

  const setVolume = useCallback((level: number) => {
    volumeRef.current = level;
    applyEffectiveVolume();
    setState((s) => ({ ...s, volume: level }));
  }, [applyEffectiveVolume]);

  const setMuted = useCallback((next: boolean) => {
    hasExplicitMuteOverrideRef.current = true;
    isMutedRef.current = next;
    applyEffectiveVolume();
    setState((s) => (s.isMuted === next ? s : { ...s, isMuted: next }));
  }, [applyEffectiveVolume]);

  const getPlaybackState = useCallback((): TTSPlaybackState | null => {
    const provider = (activeProviderRef.current ?? providerRef.current)?.getPlaybackState?.();
    if (!provider) return null;
    // Override provider's effective volume with the user-configured volume
    // and add the hook-level mute flag, so the audio bar shows the user's
    // chosen slider value rather than the silenced provider output.
    return { ...provider, volume: volumeRef.current, isMuted: isMutedRef.current };
  }, []);

  const refresh = useCallback(async () => {
    await resolveBackendRef.current?.();
  }, []);

  const testSpeak = useCallback(async () => {
    if (!providerRef.current) {
      setState((s) => ({ ...s, error: "No TTS backend is available" }));
      return;
    }
    // testSpeak is always triggered by a direct user action, so reset error
    // upfront — the user wants unambiguous feedback from this click.
    setState((s) => ({ ...s, isSpeaking: true, isPaused: false, error: null }));
    emitEvent("attempt", backendRef.current);
    try {
      const usedBackend = await executeSpeak(TEST_TTS_SAMPLE);
      emitEvent("success", usedBackend);
      updateSuccess(usedBackend);
    } catch (err) {
      if (isAbortLikeError(err)) {
        setState((s) => ({ ...s, isSpeaking: false, isPaused: false }));
        return;
      }
      const message = err instanceof Error ? err.message : "Speech failed";
      if (isAutoplayBlocked(err)) {
        emitEvent("autoplay_blocked", backendRef.current, message);
        setState((s) => ({ ...s, isSpeaking: false, isPaused: false, needsUnlock: true }));
        return;
      }
      emitEvent("error", backendRef.current, message);
      setState((s) => ({ ...s, isSpeaking: false, isPaused: false, error: message }));
      throw err;
    }
  }, [emitEvent, executeSpeak, updateSuccess]);

  const unlockAudio = useCallback(async (): Promise<boolean> => {
    const provider = providerRef.current;
    if (!provider) return false;
    // Force a fresh silent play() on this invocation. Explicit user actions
    // (Enable-Audio banner click) happen after autoplay has already failed
    // once — Chrome's prior activation is gone, so the cached unlocked
    // flag can't be trusted. Preemptive gesture-listener unlocks use the
    // default (no-force) path to avoid thrashing the element during typing.
    const ok = await provider.unlock(true);
    if (ok) {
      audioUnlockedRef.current = true;
      setState((s) => ({ ...s, browserAudioReady: true, needsUnlock: false }));
    }
    return ok;
  }, []);

  const dismissNeedsUnlock = useCallback(() => {
    setState((s) => (s.needsUnlock ? { ...s, needsUnlock: false } : s));
  }, []);

  return {
    ...state,
    speak,
    speakParagraphs,
    stop,
    pause,
    resume,
    seek,
    setPlaybackRate,
    setVolume,
    setMuted,
    getPlaybackState,
    refresh,
    testSpeak,
    unlockAudio,
    dismissNeedsUnlock,
  };
}
