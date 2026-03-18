// DOC: docs/internal/SEAMS.md#tts-provider-seam
import { useState, useEffect, useCallback, useRef } from "react";
import { fetchCapabilitiesLivenessCached, getTTSVoices, reportTTSEvent, _resetCapabilitiesCache } from "../lib/api";
import type { TTSBackend, TTSPlaybackCapabilities, TTSPlaybackState, TTSProvider, TTSVoiceInfo } from "./tts/types";
import { KokoroProvider } from "./tts/KokoroProvider";
import { BrowserTTSProvider } from "./tts/BrowserTTSProvider";

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
  backend: TTSBackend;
  voices: TTSVoiceInfo[];
  error: string | null;
  backendReason: string;
  browserAudioReady: boolean;
  lastSuccessfulBackend: TTSBackend;
  lastSuccessfulAt: string | null;
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

export function useTextToSpeech(settings: TTSSettings, diagnostics?: TTSDiagnostics) {
  const [state, setState] = useState<TTSState>({
    supported: isBrowserSupported(),
    isSpeaking: false,
    isPaused: false,
    currentTime: 0,
    duration: null,
    playbackRate: 1,
    volume: 1,
    capabilities: NO_CAPABILITIES,
    backend: "none",
    voices: [],
    error: null,
    backendReason: "Checking TTS backend availability\u2026",
    browserAudioReady: false,
    lastSuccessfulBackend: "none",
    lastSuccessfulAt: null,
  });

  const providerRef = useRef<TTSProvider | null>(null);
  const backendRef = useRef<TTSBackend>("none");
  const speakChainRef = useRef<AbortController | null>(null);
  const fallbackProviderRef = useRef<BrowserTTSProvider | null>(null);
  const audioUnlockedRef = useRef(false);
  const resolveBackendRef = useRef<(() => Promise<void>) | null>(null);

  const emitEvent = useCallback((stage: string, backend: TTSBackend, message?: string) => {
    if (!diagnostics?.source) return;
    void reportTTSEvent({
      source: diagnostics.source,
      sessionId: diagnostics.sessionId,
      stage,
      backend,
      message,
    }).catch(() => {});
  }, [diagnostics]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const unlock = () => {
      audioUnlockedRef.current = true;
      setState((s) => ({ ...s, browserAudioReady: true }));
    };
    window.addEventListener("pointerdown", unlock, { passive: true });
    window.addEventListener("keydown", unlock, { passive: true });
    window.addEventListener("touchstart", unlock, { passive: true });
    return () => {
      window.removeEventListener("pointerdown", unlock);
      window.removeEventListener("keydown", unlock);
      window.removeEventListener("touchstart", unlock);
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

  const updateSuccess = useCallback((backend: TTSBackend) => {
    setState((s) => ({
      ...s,
      isSpeaking: false,
      isPaused: false,
      error: null,
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
          const provider = new KokoroProvider();
          backendRef.current = "kokoro";
          providerRef.current = provider;
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
      speakChainRef.current?.abort();
    };
  }, []);

  const speak = useCallback(
    (text: string) => {
      speakChainRef.current?.abort();
      providerRef.current?.stop();

      if (!providerRef.current) return;

      setState((s) => ({ ...s, isSpeaking: true, isPaused: false, error: null }));
      emitEvent("attempt", backendRef.current);

      executeSpeak(text).then(
        (usedBackend) => {
          emitEvent("success", usedBackend);
          updateSuccess(usedBackend);
        },
        (err: unknown) => {
          if (isAbortLikeError(err)) {
            setState((s) => ({ ...s, isSpeaking: false, isPaused: false }));
            return;
          }
          const message = err instanceof Error ? err.message : "Speech failed";
          emitEvent("error", backendRef.current, message);
          setState((s) => ({
            ...s,
            isSpeaking: false,
            isPaused: false,
            error: message,
          }));
        },
      );
    },
    [emitEvent, executeSpeak, updateSuccess],
  );

  const speakParagraphs = useCallback(
    async (paragraphs: string[]) => {
      speakChainRef.current?.abort();
      const controller = new AbortController();
      speakChainRef.current = controller;

      if (!providerRef.current || paragraphs.length === 0) return;

      setState((s) => ({ ...s, isSpeaking: true, isPaused: false, error: null }));
      emitEvent("attempt", backendRef.current);

      try {
        const usedBackend = await executeSpeak(paragraphs.join("\n\n"), paragraphs);
        if (controller.signal.aborted) {
          setState((s) => ({ ...s, isSpeaking: false, isPaused: false }));
          return;
        }
        emitEvent("success", usedBackend);
        updateSuccess(usedBackend);
        return usedBackend;
      } catch (err: unknown) {
        if (controller.signal.aborted || isAbortLikeError(err)) {
          setState((s) => ({ ...s, isSpeaking: false, isPaused: false }));
          return;
        }
        const message =
          err instanceof Error ? err.message : "Speech failed";
        emitEvent("error", backendRef.current, message);
        setState((s) => ({ ...s, isSpeaking: false, isPaused: false, error: message }));
        throw err;
      } finally {
        if (!controller.signal.aborted) {
          speakChainRef.current = null;
        }
      }
    },
    [emitEvent, executeSpeak, updateSuccess],
  );

  const stop = useCallback(() => {
    speakChainRef.current?.abort();
    providerRef.current?.stop();
    fallbackProviderRef.current?.stop();
    setState((s) => ({ ...s, isSpeaking: false, isPaused: false }));
  }, []);

  const pause = useCallback(() => {
    providerRef.current?.pause?.();
    setState((s) => ({ ...s, isPaused: true }));
  }, []);

  const resume = useCallback(() => {
    providerRef.current?.resume?.();
    setState((s) => ({ ...s, isPaused: false }));
  }, []);

  const seek = useCallback((seconds: number) => {
    providerRef.current?.seek?.(seconds);
  }, []);

  const setPlaybackRate = useCallback((rate: number) => {
    providerRef.current?.setPlaybackRate?.(rate);
    setState((s) => ({ ...s, playbackRate: rate }));
  }, []);

  const setVolume = useCallback((level: number) => {
    providerRef.current?.setVolume?.(level);
    setState((s) => ({ ...s, volume: level }));
  }, []);

  const getPlaybackState = useCallback((): TTSPlaybackState | null => {
    return providerRef.current?.getPlaybackState?.() ?? null;
  }, []);

  const refresh = useCallback(async () => {
    await resolveBackendRef.current?.();
  }, []);

  const testSpeak = useCallback(async () => {
    if (!providerRef.current) {
      setState((s) => ({ ...s, error: "No TTS backend is available" }));
      return;
    }
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
      emitEvent("error", backendRef.current, message);
      setState((s) => ({ ...s, isSpeaking: false, isPaused: false, error: message }));
      throw err;
    }
  }, [emitEvent, executeSpeak, updateSuccess]);

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
    getPlaybackState,
    refresh,
    testSpeak,
  };
}
