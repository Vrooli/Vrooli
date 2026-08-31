// DOC: docs/internal/SEAMS.md#tts-provider-seam
//
// audio-integration is the governed browser capability surface for adopters.
// It includes defensive runtime guards (browser feature detection, race-
// safety checks) that TypeScript's strict-type checking views as
// unnecessary based on declared types but which catch real-world drift
// (older browsers, polyfills, undefined-on-prototype). These guards are
// intentional even where the current type-aware rules don't flag them.
//
// Text-To-Speech Core — Generic, Scenario-Agnostic Hook
// ======================================================
//
// Owns the TTS provider lifecycle (Kokoro vs Browser), synthesize → cache →
// play orchestration, playback state, and queue management for autoplay.
// Capability detection and conversation-aware glue (Claude hook routing,
// Codex tailer ack, conversation cursor) are host concerns and live in the
// adopter's adapter (e.g. web-console's useTextToSpeech.ts).

import { useState, useEffect, useCallback, useRef } from "react";
import type { TTSBackend, TTSPlaybackCapabilities, TTSPlaybackState, TTSProvider, TTSVoiceInfo } from "./types";
import { KokoroProvider } from "./KokoroProvider";
import { BrowserTTSProvider } from "./BrowserTTSProvider";
import { ttsPlaybackRegistry } from "./playbackRegistry";
import type { TTSRuntime } from "./runtime";

/**
 * Playback event emitted by the core. The host wraps this with its own
 * source/sessionId metadata before forwarding to any reporting endpoint.
 */
export interface TTSCorePlaybackEvent {
  stage: string;
  backend?: string;
  message?: string;
}

const NO_CAPABILITIES: TTSPlaybackCapabilities = {
  canPause: false,
  canSeek: false,
  canAdjustSpeed: false,
  canAdjustVolume: false,
};

export interface TTSCoreState {
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
  /** Concrete provider selected by the server, when known. */
  providerId?: string;
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

export interface UseTextToSpeechCoreOptions {
  /** Whether autoplay is enabled (informational; host owns the actual auto trigger). */
  autoEnabled: boolean;
  /** Backend selection. `"auto"` picks the server TTS backend when available, falling back to Browser. */
  backend: "auto" | "kokoro" | "browser";
  /** Initial mute state — `true` keeps playback silent until the user unmutes. */
  startMuted: boolean;
  /** Scenario-local same-origin TTS transport; the package owns no API URL. */
  runtime?: TTSRuntime;
  /** Browser speech-synthesis voice name (rate/pitch live on speak-time options). */
  defaultVoice?: string;
  /** Browser speech-synthesis rate (1.0 = normal). */
  defaultSpeed?: number;
  /**
   * Optional callback for playback lifecycle events (attempt / success / error /
   * autoplay_blocked). The host decorates with its own metadata before reporting.
   */
  onPlaybackEvent?: (ev: TTSCorePlaybackEvent) => void;
  /**
   * Optional probe: returns whether any server-side TTS provider is currently
   * available. Defaults to `() => Promise<true>` when omitted (the server
   * provider then surfaces the actual error at first synthesize). Hosts that
   * own a capabilities API should supply a real probe so auto mode can choose
   * the browser tier only when the complete server tier is unavailable.
   */
  serverTTSAvailable?: () => Promise<boolean>;
  /**
   * Stable key identifying the playback owner (e.g. a session id). When set
   * together with `persistPlaybackAcrossUnmount`, the active provider is held in
   * a process-wide registry so in-progress speech survives this hook unmounting
   * — e.g. a pane being evicted from the workspace warm set — and is re-adopted
   * when the owner remounts. Omit for adopters that do not multiplex playback
   * across mount/unmount boundaries (default: no persistence).
   */
  playbackOwnerKey?: string;
  /**
   * Opt into keeping a speaking provider alive across unmount (see
   * `playbackOwnerKey`). Requires `playbackOwnerKey` to be set. Defaults to
   * false so the generic per-mount lifecycle is unchanged for adopters that do
   * not need it.
   */
  persistPlaybackAcrossUnmount?: boolean;
}

/** Settings the core forwards to the active provider on each speak call. */
export interface TTSCoreSpeakSettings {
  /** Browser-TTS voice name. */
  voice: string;
  /** Browser-TTS rate. */
  rate: number;
  /** Browser-TTS pitch. */
  pitch: number;
  /** Kokoro voice id. */
  kokoroVoice: string;
  /** Kokoro speed multiplier. */
  kokoroSpeed: number;
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

export function useTextToSpeechCore(opts: UseTextToSpeechCoreOptions, settings: TTSCoreSpeakSettings) {
  const initialMuted = opts.startMuted;
  const [state, setState] = useState<TTSCoreState>({
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
    providerId: undefined,
    voices: [],
    error: null,
    backendReason: "Checking TTS backend availability…",
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
  const onPlaybackEventRef = useRef(opts.onPlaybackEvent);
  onPlaybackEventRef.current = opts.onPlaybackEvent;
  const serverTTSAvailableRef = useRef(opts.serverTTSAvailable);
  serverTTSAvailableRef.current = opts.serverTTSAvailable;
  // Resolved playback-owner key: non-null only when persistence is requested
  // AND a key is supplied. Read through a ref so the once-only unmount cleanup
  // sees the latest value.
  const persistKeyRef = useRef<string | null>(null);
  persistKeyRef.current =
    opts.persistPlaybackAcrossUnmount && opts.playbackOwnerKey ? opts.playbackOwnerKey : null;

  const applyEffectiveVolume = useCallback(() => {
    const effective = isMutedRef.current ? 0 : volumeRef.current;
    providerRef.current?.setVolume?.(effective);
    if (activeProviderRef.current && activeProviderRef.current !== providerRef.current) {
      activeProviderRef.current.setVolume?.(effective);
    }
  }, []);

  const emitEvent = useCallback((stage: string, backend: TTSBackend, message?: string) => {
    onPlaybackEventRef.current?.({ stage, backend, message });
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") return;
    // Record that a user gesture has occurred — this is a no-audio boolean that
    // gates BrowserTTSProvider (speechSynthesis). It plays NO audio, so it does
    // not touch the OS audio session.
    //
    // We deliberately do NOT preemptively "unlock" the Kokoro media element here
    // (a silent HTMLAudioElement.play()). On iOS, playing an audio element —
    // even a muted silent one — activates the app's AVAudioSession and ducks/
    // pauses other apps' audio (Spotify/YouTube), even when the user never uses
    // TTS. And because the provider is rebuilt when the backend re-resolves
    // (unlocked resets to false), a persistent gesture→unlock listener re-ducks
    // on essentially every interaction — the "web-console pauses my music on any
    // tap" report. The media element is instead unlocked lazily: a real speak in
    // a user gesture unlocks it naturally, and a programmatic speak blocked by
    // autoplay surfaces `needsUnlock` + the Enable-Audio affordance (a deliberate
    // tap → provider.unlock(true)). See PROBLEMS.md §8b.
    const onGesture = () => {
      audioUnlockedRef.current = true;
      setState((s) => (s.browserAudioReady ? s : { ...s, browserAudioReady: true }));
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

  // Apply opts.startMuted changes (mirrors the legacy startMutedOnLoad effect).
  useEffect(() => {
    if (hasExplicitMuteOverrideRef.current) return;
    isMutedRef.current = opts.startMuted;
    applyEffectiveVolume();
    setState((s) => (s.isMuted === opts.startMuted ? s : { ...s, isMuted: opts.startMuted }));
  }, [applyEffectiveVolume, opts.startMuted]);

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
      setState((s) => ({ ...s, providerId: "browser-tts" }));
      return "browser";
    },
    [settings.pitch, settings.rate, settings.voice],
  );

  const executeSpeak = useCallback(async (
    text: string,
    paragraphs?: string[],
    cacheOpts?: { eventId?: string; version?: "active" | "original" },
  ): Promise<TTSBackend> => {
    const provider = providerRef.current;
    const segments = paragraphs ?? [text];
    if (!provider || segments.length === 0) {
      throw new Error("No TTS backend is available");
    }

    // eventId/version are threaded only into the Kokoro synth path so each
    // paragraph populates the server byte cache under its chunk index; the
    // browser voice ignores them.
    const kokoroOpts = {
      voice: settings.kokoroVoice,
      rate: settings.kokoroSpeed,
      eventId: cacheOpts?.eventId,
      version: cacheOpts?.version,
    };
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
      const providerOpts = backendRef.current === "kokoro" ? kokoroOpts : browserOpts;
      // Use speakSequence for unified playback when the provider supports it
      // and there are multiple segments. This produces a single audio track
      // with accurate total duration and full seek/scrub support.
      if (segments.length > 1 && provider.speakSequence) {
        await provider.speakSequence(segments, providerOpts);
      } else {
        for (const segment of segments) {
          await provider.speak(segment, providerOpts);
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
      if (backendRef.current === "kokoro" && opts.backend === "auto" && isBrowserSupported()) {
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
    opts.backend,
    settings.kokoroSpeed,
    settings.kokoroVoice,
    settings.pitch,
    settings.rate,
    settings.voice,
  ]);

  useEffect(() => {
    let cancelled = false;

    // Adopt a live provider held in the playback registry for this owner key
    // (e.g. on pane remount, so in-progress speech reconnects to a single
    // owner), else create a fresh one and register it. Without persistence this
    // just calls the factory — the generic per-mount lifecycle is unchanged.
    const obtainProvider = <T extends TTSProvider>(
      backend: TTSBackend,
      factory: () => T,
    ): { provider: T; adopted: boolean } => {
      const key = persistKeyRef.current;
      if (key) {
        const existing = ttsPlaybackRegistry.adopt(key, backend);
        if (existing) return { provider: existing as T, adopted: true };
      }
      const provider = factory();
      if (key) ttsPlaybackRegistry.install(key, provider, backend);
      return { provider, adopted: false };
    };

    async function checkBackend() {
      if (opts.backend === "browser") {
        if (isBrowserSupported()) {
          const { provider, adopted } = obtainProvider("browser", () => new BrowserTTSProvider());
          backendRef.current = "browser";
          providerRef.current = provider;
          activeProviderRef.current = provider;
          if (!cancelled) {
            const voices = window.speechSynthesis.getVoices() ?? [];
            setState((s) => ({
              ...s,
              supported: true,
              backend: "browser",
              providerId: "browser-tts",
              capabilities: provider.capabilities,
              voices: voices.map((v) => ({ id: v.name, name: v.name })),
              backendReason: "Browser backend selected explicitly",
              isSpeaking: adopted && provider.isSpeaking ? true : s.isSpeaking,
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

      // Probe server TTS availability. Hosts without a probe assume the server
      // the provider surface the actual error on first synth — the runtime
      // fallback in executeSpeak still routes to Browser when "auto".
      let serverTTSAvailable = true;
      try {
        if (serverTTSAvailableRef.current) {
          serverTTSAvailable = await serverTTSAvailableRef.current();
        }
      } catch {
        serverTTSAvailable = false;
      }

      if (!cancelled && serverTTSAvailable) {
        // Inject synthesizeTTS pulled through the audio-integration barrel so
        // tests using vi.mock("../audio-integration", …) intercept the call.
        const { provider, adopted } = obtainProvider(
          "kokoro",
          () => new KokoroProvider({
            synthesize: opts.runtime?.synthesizeTTS,
            synthesizeWithMetrics: opts.runtime?.synthesizeTTSWithMetrics,
            reportTTSPlayStart: opts.runtime?.reportTTSPlayStart,
            onProviderResolved: (providerId) => setState((s) => ({ ...s, providerId })),
          }),
        );
        // Phase 8 observability: surface per-paragraph degradation (retry /
        // browser fallback / skip) so silent partial TTS loss becomes visible.
        // The outcome hook is non-fatal — it never blocks or halts playback.
        provider.onParagraphOutcome = ({ index, outcome }) => {
          onPlaybackEventRef.current?.({
            stage: `paragraph_${outcome}`,
            backend: "kokoro",
            message: `paragraph ${index} ${outcome}`,
          });
        };
        backendRef.current = "kokoro";
        providerRef.current = provider;
        activeProviderRef.current = provider;
        try {
          const voices = await (opts.runtime?.getTTSVoices?.() ?? Promise.resolve([]));
          if (!cancelled) {
            setState((s) => ({
              ...s,
              supported: true,
              backend: "kokoro",
              providerId: "kokoro-tts",
              capabilities: provider.capabilities,
              voices,
              backendReason: opts.backend === "kokoro"
                ? "Kokoro backend selected explicitly"
                : "A server speech provider is available and preferred over browser speech synthesis",
              isSpeaking: adopted && provider.isSpeaking ? true : s.isSpeaking,
            }));
          }
        } catch {
          if (!cancelled) {
            setState((s) => ({
              ...s,
              supported: true,
              backend: "kokoro",
              providerId: "kokoro-tts",
              capabilities: provider.capabilities,
              voices: [{ id: KOKORO_DEFAULT_VOICE, name: KOKORO_DEFAULT_VOICE }],
              backendReason: opts.backend === "kokoro"
                ? "Kokoro backend selected explicitly"
                : "A server speech provider is available and preferred over browser speech synthesis",
              isSpeaking: adopted && provider.isSpeaking ? true : s.isSpeaking,
            }));
          }
        }
        return;
      }

      // No server-side TTS provider is available.
      if (opts.backend === "kokoro" && !cancelled) {
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
        return;
      }

      if (!cancelled && isBrowserSupported()) {
        const { provider, adopted } = obtainProvider("browser", () => new BrowserTTSProvider());
        backendRef.current = "browser";
        providerRef.current = provider;
        activeProviderRef.current = provider;
        const voices = window.speechSynthesis.getVoices();
        setState((s) => ({
          ...s,
          supported: true,
          backend: "browser",
          providerId: "browser-tts",
          isSpeaking: adopted && provider.isSpeaking ? true : s.isSpeaking,
          capabilities: provider.capabilities,
          voices: voices.map((v) => ({ id: v.name, name: v.name })),
          backendReason: "No server speech provider is available, so browser speech synthesis is active",
        }));
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
          backendReason: opts.backend === "kokoro"
            ? "Kokoro backend was selected explicitly, but Kokoro is unavailable and browser speech synthesis is not supported"
            : "No TTS backend is available. No server speech provider is available and browser speech synthesis is not supported",
        }));
      }
    }

    resolveBackendRef.current = () => checkBackend();
    checkBackend();
    return () => {
      cancelled = true;
      resolveBackendRef.current = null;
    };
  }, [opts.backend]);

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

  // Cleanup on unmount.
  //
  // When playback persistence is enabled and this owner's provider is still
  // speaking, hand it off to the registry instead of disposing it — the audio
  // keeps playing through pane unmount / warm-set eviction, and a remount
  // re-adopts the same provider (single owner, no leak). The registry disposes
  // it once its tail settles while orphaned. Without persistence, or when idle,
  // dispose as before. The browser fallback provider is always disposed (it is
  // never handed off; browser speech is a global singleton).
  useEffect(() => {
    return () => {
      const key = persistKeyRef.current;
      const provider = providerRef.current;
      if (key && provider) {
        ttsPlaybackRegistry.release(key, provider, { keepAliveIfSpeaking: true });
      } else {
        provider?.dispose();
      }
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

      // Single-owner audio: a new playback tears down any orphaned tail still
      // playing under a different owner (e.g. a session we just switched away
      // from) so two sessions never speak at once.
      if (persistKeyRef.current) ttsPlaybackRegistry.stopOrphansExcept(persistKeyRef.current);

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
      speakOpts?: { eventId?: string; version?: "active" | "original" },
    ): Promise<TTSBackend | undefined> => {
      speakChainRef.current?.abort();
      const controller = new AbortController();
      speakChainRef.current = controller;

      if (!providerRef.current || paragraphs.length === 0) return;

      // Single-owner audio (see speak()): drop any orphaned tail from another
      // owner before this session starts speaking.
      if (persistKeyRef.current) ttsPlaybackRegistry.stopOrphansExcept(persistKeyRef.current);

      setState((s) => ({ ...s, isSpeaking: true, isPaused: false }));
      emitEvent("attempt", backendRef.current);

      try {
        // Cache-first path: if eventId is provided and backend is Kokoro, try
        // fetching pre-cached audio (one blob per paragraph, keyed by chunk
        // index) before synthesizing. A FULL hit replays without any synth; a
        // partial/any miss falls through to synthesis, which repopulates every
        // chunk. Empty paragraphs cache as 0-byte hits and are skipped on play.
        const cacheEventId = speakOpts?.eventId;
        if (cacheEventId && backendRef.current === "kokoro" && providerRef.current) {
          const provider = providerRef.current as KokoroProvider;
          const version = speakOpts?.version ?? "active";
          const chunks = await Promise.all(
            paragraphs.map((_, i) =>
              opts.runtime?.fetchCachedTTS?.(
                cacheEventId,
                settings.kokoroVoice,
                settings.kokoroSpeed,
                version,
                controller.signal,
                i,
              ) ?? Promise.resolve(null),
            ),
          );
          const fullyCached = chunks.length > 0 && chunks.every((b) => b !== null);
          if (fullyCached && !controller.signal.aborted) {
            activeProviderRef.current = provider;
            applyEffectiveVolume();
            try {
              await provider.speakFromBlobs(chunks as Blob[]);
            } catch (err) {
              if (controller.signal.aborted || isAbortLikeError(err)) throw err;
              // Cache-first blob playback was rejected. If it was an autoplay
              // block AND we can fall back to browser speech, do so with the
              // joined paragraphs (same behavior as executeSpeak's fallback).
              if (
                isAutoplayBlocked(err)
                && opts.backend === "auto"
                && isBrowserSupported()
              ) {
                await runBrowserSpeak(paragraphs.join("\n\n"), true);
                if (!controller.signal.aborted) {
                  emitEvent("success", "browser");
                  updateSuccess("browser");
                  return "browser";
                }
                return;
              }
              throw err;
            }
            if (!controller.signal.aborted) {
              emitEvent("success", "kokoro");
              updateSuccess("kokoro");
              return "kokoro";
            }
            if (speakChainRef.current === controller) {
              setState((s) => ({ ...s, isSpeaking: false, isPaused: false }));
            }
            return;
          }
        }

        // Fall through to standard synthesis path. Thread eventId/version so
        // each synthesized paragraph populates the server byte cache under its
        // chunk index — the next replay of this message serves fully from cache.
        const usedBackend = await executeSpeak(paragraphs.join("\n\n"), paragraphs, {
          eventId: speakOpts?.eventId,
          version: speakOpts?.version,
        });
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
    [applyEffectiveVolume, emitEvent, executeSpeak, runBrowserSpeak, opts.backend, settings.kokoroSpeed, settings.kokoroVoice, updateSuccess],
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

  // Touch the informational opts so unused-vars lint doesn't fire and so any
  // future host-side coupling has a clear hook.
  void opts.autoEnabled;
  void opts.defaultVoice;
  void opts.defaultSpeed;

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
