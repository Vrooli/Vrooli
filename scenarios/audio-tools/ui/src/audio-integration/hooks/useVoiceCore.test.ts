// Tests for useVoiceCore — the generic voice-input orchestrator. Every
// dependency pulled through the audio-integration barrel ("../index") is
// stubbed with controllable fakes, so the hook's own state machine
// (mount/capability check, start/stop, transcription callbacks, VAD tick,
// passive wake-word, rejection retry) runs against deterministic seams.
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";

import { _resetServerVadStateForTesting } from "./useServerVadStateStore";

// ── Hoisted shared fakes for the "../index" barrel ──────────────────────────
const h = vi.hoisted(() => {
  const stableStream = {
    active: true,
    getTracks: () => [{ kind: "audio", readyState: "live", stop: vi.fn() }],
  };

  class FakeVoiceStreamProvider {
    static instances: FakeVoiceStreamProvider[] = [];
    onResult: ((t: string) => void) | null = null;
    onError: ((e: string) => void) | null = null;
    onPartial: ((t: string) => void) | null = null;
    onSegmentFinal: ((t: string, i: number) => void) | null = null;
    onSegmentAccepted: ((i: number, s: number, t: number) => void) | null = null;
    onSegmentRejected: ((i: number, s: number, t: number) => void) | null = null;
    onVadState: ((snap: unknown) => void) | null = null;
    onSpeakerStatus: ((a: boolean, b: boolean) => void) | null = null;
    retainStream = false;
    language = "";
    streamResult: unknown = stableStream;
    start = vi.fn(() => Promise.resolve());
    stop = vi.fn();
    dispose = vi.fn();
    getStream = vi.fn(() => this.streamResult);
    getLastTurnAudio = vi.fn((): unknown => null);
    disposeLastTurn = vi.fn();
    dropTail = vi.fn();
    preConnect = vi.fn();
    sendVadState = vi.fn();
    sendSegmentBoundary = vi.fn();
    constructor() {
      FakeVoiceStreamProvider.instances.push(this);
    }
  }

  class FakeWhisperProvider {
    onResult: ((t: string) => void) | null = null;
    onError: ((e: string) => void) | null = null;
    onPartial: ((t: string) => void) | null = null;
    language = "";
    start = vi.fn(() => Promise.resolve());
    stop = vi.fn();
    dispose = vi.fn();
    getStream = vi.fn(() => stableStream);
    getLastTurnAudio = vi.fn(() => null);
    disposeLastTurn = vi.fn();
    dropTail = vi.fn();
  }

  class FakeWebSpeechProvider {
    static instances: FakeWebSpeechProvider[] = [];
    onResult: ((t: string) => void) | null = null;
    onError: ((e: string) => void) | null = null;
    onPartial: ((t: string) => void) | null = null;
    lang = "";
    start = vi.fn(() => Promise.resolve());
    stop = vi.fn();
    dispose = vi.fn();
    getStream = vi.fn(() => stableStream);
    getLastTurnAudio = vi.fn(() => null);
    disposeLastTurn = vi.fn();
    dropTail = vi.fn();
    constructor() {
      FakeWebSpeechProvider.instances.push(this);
    }
  }

  class FakePassiveListener {
    static instances: FakePassiveListener[] = [];
    opts: Record<string, unknown>;
    start = vi.fn(() => Promise.resolve());
    dispose = vi.fn();
    getAudioContext = vi.fn(() => null);
    constructor(opts: Record<string, unknown>) {
      this.opts = opts;
      FakePassiveListener.instances.push(this);
    }
  }

  const fakeCtx = {
    state: "running",
    sampleRate: 48_000,
    destination: {},
    resume: vi.fn(() => Promise.resolve()),
    close: vi.fn(() => Promise.resolve()),
    createMediaStreamSource: vi.fn(() => ({ connect: vi.fn(), disconnect: vi.fn() })),
  };

  return {
    stableStream,
    fakeCtx,
    FakeVoiceStreamProvider,
    FakeWhisperProvider,
    FakeWebSpeechProvider,
    FakePassiveListener,
    getVoiceStreamConfig: vi.fn(() => Promise.resolve({})),
    transcribeAudioBypassFilter: vi.fn(() => Promise.resolve("retried text")),
    getWakeWordConfig: vi.fn(
      (): Promise<{ configured: boolean; template?: unknown }> => Promise.resolve({ configured: false }),
    ),
    createAudioFilterChain: vi.fn(() => ({
      analyser: { frequencyBinCount: 8, getByteTimeDomainData: (d: Uint8Array) => d.fill(128) },
      nodes: [{ disconnect: vi.fn() }],
    })),
    playRecordingStartCue: vi.fn(),
    playRecordingStopCue: vi.fn(),
    createVadRefs: vi.fn(() => ({
      state: "idle",
      speechThreshold: 0.1,
      silenceThreshold: 0.05,
      silenceStart: 0,
      recordingStart: 0,
      segmentSilenceMs: 0,
      segmentBoundaryEmitted: false,
    })),
    loadNoiseFloorCache: vi.fn(() => null),
    saveNoiseFloorCache: vi.fn(),
    extractCacheableFloor: vi.fn(() => ({ speechThreshold: 0.1, silenceThreshold: 0.05 })),
    createVadRefsFromCache: vi.fn(() => ({ state: "watchingSilence" })),
    vadTick: vi.fn((): string | null => null),
    createWakeWordEngine: vi.fn(() => ({
      extractFeatures: vi.fn(),
      compare: vi.fn(),
      compareBest: vi.fn(),
    })),
    acquireStream: vi.fn(() => Promise.resolve(stableStream)),
    releaseStream: vi.fn(),
    getStream: vi.fn(() => stableStream),
    isStreamAlive: vi.fn(() => false),
    installVisibilityHandler: vi.fn(() => vi.fn()),
    getSharedAudioContext: vi.fn(() => fakeCtx),
    ensureAudioContextOnGesture: vi.fn(),
    installAudioContextKeepalive: vi.fn(),
    teardownAudioContextKeepalive: vi.fn(),
  };
});

vi.mock("../index", () => ({
  VoiceStreamProvider: h.FakeVoiceStreamProvider,
  WhisperProvider: h.FakeWhisperProvider,
  WebSpeechProvider: h.FakeWebSpeechProvider,
  PassiveListener: h.FakePassiveListener,
  getVoiceStreamConfig: h.getVoiceStreamConfig,
  transcribeAudioBypassFilter: h.transcribeAudioBypassFilter,
  getWakeWordConfig: h.getWakeWordConfig,
  createAudioFilterChain: h.createAudioFilterChain,
  playRecordingStartCue: h.playRecordingStartCue,
  playRecordingStopCue: h.playRecordingStopCue,
  buildVoiceActivitySnapshot: vi.fn(() => ({ idle: true })),
  IDLE_VOICE_ACTIVITY: { idle: true },
  voiceActivitySnapshotsEqual: vi.fn(() => true),
  createVadRefs: h.createVadRefs,
  createVadRefsFromCache: h.createVadRefsFromCache,
  extractCacheableFloor: h.extractCacheableFloor,
  loadNoiseFloorCache: h.loadNoiseFloorCache,
  saveNoiseFloorCache: h.saveNoiseFloorCache,
  vadTick: h.vadTick,
  VAD_FLOOR_CACHE_MAX_AGE_MS: 60_000,
  getSharedAudioContext: h.getSharedAudioContext,
  ensureAudioContextOnGesture: h.ensureAudioContextOnGesture,
  installAudioContextKeepalive: h.installAudioContextKeepalive,
  teardownAudioContextKeepalive: h.teardownAudioContextKeepalive,
  acquireStream: h.acquireStream,
  releaseStream: h.releaseStream,
  getStream: h.getStream,
  isStreamAlive: h.isStreamAlive,
  installVisibilityHandler: h.installVisibilityHandler,
  createWakeWordEngine: h.createWakeWordEngine,
  CAP_CHECK_FAIL_THRESHOLD: 3,
  WHISPER_FAILED_SENTINEL: "whisper-failed",
}));

import { useVoiceCore, type UseVoiceCoreOptions } from "./useVoiceCore";

let rafCb: (() => void) | null = null;

function defaultOpts(over: Partial<UseVoiceCoreOptions> = {}): UseVoiceCoreOptions {
  return {
    voiceEnabled: true,
    voiceLanguage: "en-US",
    vadSilenceTimeoutMs: 1_500,
    persistentMode: false,
    wakeWordEnabled: false,
    segmentSilenceMs: 800,
    lowLatencyVoice: false,
    onTranscript: vi.fn(),
    ...over,
  };
}

function renderVoice(over: Partial<UseVoiceCoreOptions> = {}) {
  const opts = defaultOpts(over);
  const view = renderHook((o: UseVoiceCoreOptions) => useVoiceCore(o), { initialProps: opts });
  return { ...view, opts };
}

/** Mount, then wait for the optimistic backend confirmation. */
async function mountReady(over: Partial<UseVoiceCoreOptions> = {}) {
  const view = renderVoice(over);
  await waitFor(() => expect(view.result.current.supported).toBe(true));
  // Let the mount capability check settle (preConnect / streamingAvailable).
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
  return view;
}

const lastVSP = () => h.FakeVoiceStreamProvider.instances[h.FakeVoiceStreamProvider.instances.length - 1]!;

beforeEach(() => {
  _resetServerVadStateForTesting();
  h.FakeVoiceStreamProvider.instances = [];
  h.FakeWebSpeechProvider.instances = [];
  h.FakePassiveListener.instances = [];
  rafCb = null;
  vi.spyOn(performance, "now").mockReturnValue(10_000);
  (globalThis as unknown as { requestAnimationFrame: (cb: () => void) => number }).requestAnimationFrame = (cb) => {
    rafCb = cb;
    return 1;
  };
  (globalThis as unknown as { cancelAnimationFrame: (id: number) => void }).cancelAnimationFrame = vi.fn();
  // Default: no Web Speech fallback unless a test opts in.
  Reflect.deleteProperty(window, "SpeechRecognition");
  Reflect.deleteProperty(window, "webkitSpeechRecognition");
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  vi.restoreAllMocks();
});

describe("useVoiceCore — mount", () => {
  it("is unsupported when voice is disabled", async () => {
    const { result } = renderVoice({ voiceEnabled: false });
    await waitFor(() => expect(result.current.supported).toBe(false));
    expect(result.current.backend).toBe("none");
  });

  it("optimistically supports Whisper and pre-connects the streaming provider", async () => {
    const { result } = await mountReady();
    expect(result.current.backend).toBe("whisper");
    expect(h.ensureAudioContextOnGesture).toHaveBeenCalled();
    // The mount capability check pre-connects a VoiceStreamProvider.
    expect(h.FakeVoiceStreamProvider.instances.length).toBeGreaterThanOrEqual(1);
    expect(lastVSP().preConnect).toHaveBeenCalled();
  });

  it("downgrades to web-speech when Whisper is unhealthy and SpeechRecognition exists", async () => {
    (window as unknown as { SpeechRecognition: unknown }).SpeechRecognition = function () {};
    const { result } = renderVoice({
      capabilityCheck: () => Promise.resolve({ whisperHealthy: false }),
    });
    await waitFor(() => expect(result.current.backend).toBe("web-speech"));
  });

  it("reports unsupported when Whisper is unhealthy and no fallback exists", async () => {
    const { result } = renderVoice({
      capabilityCheck: () => Promise.resolve({ whisperHealthy: false }),
    });
    await waitFor(() => expect(result.current.backend).toBe("none"));
    expect(result.current.supported).toBe(false);
  });

  it("tolerates a rejecting capability probe (web-speech fallback)", async () => {
    (window as unknown as { SpeechRecognition: unknown }).SpeechRecognition = function () {};
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const { result } = renderVoice({
      capabilityCheck: () => Promise.reject(new Error("probe down")),
    });
    await waitFor(() => expect(result.current.backend).toBe("web-speech"));
    warn.mockRestore();
  });

  it("marks wake word configured when the backend reports a template", async () => {
    h.getWakeWordConfig.mockResolvedValueOnce({
      configured: true,
      template: { samples: [], label: "hey", threshold: 0.6, updatedAt: "" },
    });
    const { result } = await mountReady();
    await waitFor(() => expect(result.current.wakeWordConfigured).toBe(true));
  });

  it("reflects persistent mode in voiceMode", async () => {
    const { result } = await mountReady({ persistentMode: true });
    expect(result.current.voiceMode).toBe("persistent");
  });
});

describe("useVoiceCore — recording lifecycle", () => {
  it("starts a one-shot recording and plays the start cue", async () => {
    const { result } = await mountReady();
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));
    expect(h.playRecordingStartCue).toHaveBeenCalled();
    expect(lastVSP().start).toHaveBeenCalled();
  });

  it("delivers a final transcript via onResult and returns to idle", async () => {
    const onTranscript = vi.fn();
    const { result } = await mountReady({ onTranscript });
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));

    act(() => lastVSP().onResult?.("hello there"));
    await waitFor(() => expect(result.current.voiceState).toBe("idle"));
    expect(onTranscript).toHaveBeenCalledWith("hello there");
  });

  it("surfaces a provider error", async () => {
    const { result } = await mountReady();
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));
    act(() => lastVSP().onError?.("mic exploded"));
    await waitFor(() => expect(result.current.error).toBe("mic exploded"));
  });

  it("forwards partial transcripts", async () => {
    const { result } = await mountReady();
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));
    act(() => lastVSP().onPartial?.("partial…"));
    expect(result.current.partialTranscript).toBe("partial…");
  });

  it("falls back to web-speech when Whisper fails after retry", async () => {
    (window as unknown as { SpeechRecognition: unknown }).SpeechRecognition = function () {};
    const { result } = await mountReady();
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));
    act(() => lastVSP().onError?.("whisper-failed"));
    await waitFor(() => expect(result.current.backend).toBe("web-speech"));
    expect(result.current.fallbackNotice).toMatch(/browser speech recognition/);
  });

  it("user-stop transitions to transcribing and stops the provider after the settle delay", async () => {
    vi.useFakeTimers();
    try {
      const view = renderVoice();
      // Drive the mount capability check under fake timers.
      await vi.advanceTimersByTimeAsync(0);
      const provider = lastVSP();
      await act(async () => {
        await result_startRecording(view);
      });
      expect(view.result.current.isRecording).toBe(true);

      act(() => view.result.current.stopRecording());
      expect(view.result.current.voiceState).toBe("transcribing");
      expect(provider.dropTail).not.toHaveBeenCalled();
      await act(async () => {
        await vi.advanceTimersByTimeAsync(120);
      });
      expect(provider.stop).toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });

  it("auto-stop arms tail-drop and stops immediately", async () => {
    const { result } = await mountReady();
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));
    const provider = lastVSP();
    act(() => result.current.stopRecording({ reason: "auto" }));
    expect(provider.dropTail).toHaveBeenCalled();
    expect(provider.stop).toHaveBeenCalled();
  });

  it("pre-warms the mic and installs a visibility handler in low-latency mode", async () => {
    const { result, unmount } = await mountReady({ lowLatencyVoice: true });
    expect(h.acquireStream).toHaveBeenCalled();
    expect(h.installVisibilityHandler).toHaveBeenCalled();
    // onResult in low-latency mode releases then re-acquires the stream.
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));
    act(() => lastVSP().onResult?.("done"));
    await waitFor(() => expect(result.current.voiceState).toBe("idle"));
    expect(h.releaseStream).toHaveBeenCalled();
    unmount();
  });

  it("records through the Web Speech provider when that backend is active", async () => {
    (window as unknown as { SpeechRecognition: unknown }).SpeechRecognition = function () {};
    const { result } = renderVoice({
      capabilityCheck: () => Promise.resolve({ whisperHealthy: false }),
    });
    await waitFor(() => expect(result.current.backend).toBe("web-speech"));
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));
    expect(h.FakeWebSpeechProvider.instances.length).toBeGreaterThanOrEqual(1);
    const wsp = h.FakeWebSpeechProvider.instances[h.FakeWebSpeechProvider.instances.length - 1]!;
    expect(wsp.start).toHaveBeenCalled();
  });

  it("ignores a start while not idle", async () => {
    const { result } = await mountReady();
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));
    const startCalls = lastVSP().start.mock.calls.length;
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    expect(lastVSP().start.mock.calls.length).toBe(startCalls);
  });
});

describe("useVoiceCore — VAD tick", () => {
  it("auto-stops on a client VAD no-speech verdict", async () => {
    h.vadTick.mockReturnValue("no-speech");
    const { result } = await mountReady();
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));
    // Drive one animation frame; the tick reads vadTick → "no-speech".
    await act(async () => {
      rafCb?.();
      await Promise.resolve();
    });
    await waitFor(() => expect(result.current.error).toBe("No speech detected"));
  });
});

describe("useVoiceCore — transcription cancel", () => {
  it("cancelTranscription disposes the provider and returns to idle", async () => {
    const { result } = await mountReady();
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));
    const provider = lastVSP();
    act(() => result.current.stopRecording());
    await waitFor(() => expect(result.current.isTranscribing).toBe(true));
    act(() => result.current.cancelTranscription());
    expect(provider.dispose).toHaveBeenCalled();
    expect(result.current.voiceState).toBe("idle");
  });
});

describe("useVoiceCore — rejection banner", () => {
  it("retryWithoutFilter re-transcribes the retained audio on success", async () => {
    const onTranscript = vi.fn();
    const { result } = await mountReady({ onTranscript });
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));
    const provider = lastVSP();
    provider.getLastTurnAudio = vi.fn(() => ({
      blob: new Blob(["x"]),
      mimeType: "audio/wav",
      durationMs: 1_000,
    }));

    // Simulate a verification rejection during the turn, then end the turn.
    act(() => provider.onSegmentRejected?.(0, 0.4, 0.7));
    act(() => provider.onResult?.(""));
    await waitFor(() => expect(result.current.rejectedAudio).not.toBeNull());
    expect(result.current.rejectedAudio?.kind).toBe("retryable");

    await act(async () => {
      await result.current.retryWithoutFilter();
    });
    expect(h.transcribeAudioBypassFilter).toHaveBeenCalled();
    expect(onTranscript).toHaveBeenCalledWith("retried text");
    await waitFor(() => expect(result.current.rejectedAudio).toBeNull());
  });

  it("retryWithoutFilter surfaces a failure", async () => {
    h.transcribeAudioBypassFilter.mockRejectedValueOnce(new Error("server boom"));
    const { result } = await mountReady();
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));
    const provider = lastVSP();
    provider.getLastTurnAudio = vi.fn(() => ({
      blob: new Blob(["x"]),
      mimeType: "audio/wav",
      durationMs: 1_000,
    }));
    act(() => provider.onSegmentRejected?.(0, 0.4, 0.7));
    act(() => provider.onResult?.(""));
    await waitFor(() => expect(result.current.rejectedAudio).not.toBeNull());

    await act(async () => {
      await result.current.retryWithoutFilter();
    });
    const rejected = () =>
      result.current.rejectedAudio as { status?: string; errorMessage?: string } | null;
    await waitFor(() => expect(rejected()?.status).toBe("failed"));
    expect(rejected()?.errorMessage).toMatch(/server boom/);
  });

  it("dismissRejection clears the banner", async () => {
    const { result } = await mountReady();
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isRecording).toBe(true));
    const provider = lastVSP();
    provider.getLastTurnAudio = vi.fn(() => ({
      blob: new Blob(["x"]),
      mimeType: "audio/wav",
      durationMs: 1_000,
    }));
    act(() => provider.onSegmentRejected?.(0, 0.4, 0.7));
    act(() => provider.onResult?.(""));
    await waitFor(() => expect(result.current.rejectedAudio).not.toBeNull());
    act(() => result.current.dismissRejection());
    expect(result.current.rejectedAudio).toBeNull();
    expect(provider.disposeLastTurn).toHaveBeenCalled();
  });
});

describe("useVoiceCore — command suggestion + segments", () => {
  it("emits a command suggestion from a persistent-mode segment", async () => {
    const onCommandSuggest = vi.fn();
    const parseCommand = vi.fn(() => ({ id: "do-thing" }) as unknown as ReturnType<NonNullable<UseVoiceCoreOptions["parseCommand"]>>);
    const { result } = await mountReady({
      persistentMode: true,
      parseCommand,
      onCommandSuggest,
    });
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isListening).toBe(true));
    act(() => lastVSP().onSegmentFinal?.("open the door", 0));
    expect(parseCommand).toHaveBeenCalledWith("open the door");
    expect(onCommandSuggest).toHaveBeenCalled();
    expect(result.current.commandSuggestion).toEqual({ id: "do-thing" });

    act(() => result.current.dismissCommandSuggestion());
    expect(result.current.commandSuggestion).toBeNull();
  });

  it("appends a non-command segment as dictation text", async () => {
    const onTranscript = vi.fn();
    const { result } = await mountReady({ persistentMode: true, onTranscript });
    await act(async () => {
      await result.current.startRecording({ vadEnabled: true });
    });
    await waitFor(() => expect(result.current.isListening).toBe(true));
    act(() => lastVSP().onSegmentFinal?.("first sentence.", 0));
    expect(onTranscript).toHaveBeenCalledWith("first sentence.");
    expect(result.current.segments).toEqual([{ text: "first sentence.", isFinal: true }]);
  });
});

describe("useVoiceCore — passive wake word", () => {
  it("warns and no-ops when no wake word is configured", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const { result } = await mountReady();
    await act(async () => {
      await result.current.enterPassiveMode();
    });
    expect(result.current.isPassive).toBe(false);
    expect(h.FakePassiveListener.instances.length).toBe(0);
    warn.mockRestore();
  });

  it("enters passive mode when a wake word is configured", async () => {
    h.getWakeWordConfig.mockResolvedValueOnce({
      configured: true,
      template: { samples: [], label: "hey", threshold: 0.6, updatedAt: "" },
    });
    const { result } = await mountReady();
    await waitFor(() => expect(result.current.wakeWordConfigured).toBe(true));
    await act(async () => {
      await result.current.enterPassiveMode();
    });
    expect(result.current.isPassive).toBe(true);
    expect(h.FakePassiveListener.instances.length).toBe(1);
    expect(h.FakePassiveListener.instances[0]!.start).toHaveBeenCalled();

    act(() => result.current.exitPassiveMode());
    expect(result.current.isPassive).toBe(false);
  });
});

/** Trigger startRecording through the latest render result (fake-timer test). */
async function result_startRecording(view: ReturnType<typeof renderVoice>): Promise<void> {
  await view.result.current.startRecording({ vadEnabled: true });
}
