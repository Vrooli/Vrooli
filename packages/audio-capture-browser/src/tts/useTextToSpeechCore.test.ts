// Tests for useTextToSpeechCore — the generic TTS orchestrator. The provider
// implementations (Kokoro/Browser) and the audio-integration API surface are
// stubbed at their concrete module paths so the hook's backend selection, speak /
// cache / fallback orchestration, playback controls, and error paths run
// against controllable fakes.
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";

const h = vi.hoisted(() => {
  const caps = { canPause: true, canSeek: true, canAdjustSpeed: true, canAdjustVolume: true };
  interface FakeProvider {
    capabilities: typeof caps;
    speak: ReturnType<typeof vi.fn>;
    speakSequence: ReturnType<typeof vi.fn>;
    speakFromBlob: ReturnType<typeof vi.fn>;
    speakFromBlobs: ReturnType<typeof vi.fn>;
    stop: ReturnType<typeof vi.fn>;
    pause: ReturnType<typeof vi.fn>;
    resume: ReturnType<typeof vi.fn>;
    seek: ReturnType<typeof vi.fn>;
    setVolume: ReturnType<typeof vi.fn>;
    setPlaybackRate: ReturnType<typeof vi.fn>;
    getPlaybackState: ReturnType<typeof vi.fn>;
    onProgress: ReturnType<typeof vi.fn>;
    isUnlocked: ReturnType<typeof vi.fn>;
    unlock: ReturnType<typeof vi.fn>;
    dispose: ReturnType<typeof vi.fn>;
  }
  const kokoroInstances: FakeProvider[] = [];
  const browserInstances: FakeProvider[] = [];
  const newProvider = (store: FakeProvider[]): FakeProvider => {
    const p: FakeProvider = {
      capabilities: caps,
      speak: vi.fn(() => Promise.resolve()),
      speakSequence: vi.fn(() => Promise.resolve()),
      speakFromBlob: vi.fn(() => Promise.resolve()),
      speakFromBlobs: vi.fn(() => Promise.resolve()),
      stop: vi.fn(),
      pause: vi.fn(),
      resume: vi.fn(),
      seek: vi.fn(),
      setVolume: vi.fn(),
      setPlaybackRate: vi.fn(),
      getPlaybackState: vi.fn(() => ({
        isPlaying: false,
        isPaused: false,
        currentTime: 1,
        duration: 5,
        playbackRate: 1,
        volume: 1,
      })),
      onProgress: vi.fn(),
      isUnlocked: vi.fn(() => false),
      unlock: vi.fn(() => Promise.resolve(true)),
      dispose: vi.fn(),
    };
    store.push(p);
    return p;
  };
  return { caps, kokoroInstances, browserInstances, newProvider };
});

vi.mock("./KokoroProvider", () => ({
  // eslint-disable-next-line @typescript-eslint/no-extraneous-class
  KokoroProvider: class {
    constructor(_opts?: unknown) {
      return h.newProvider(h.kokoroInstances);
    }
  },
}));

vi.mock("./BrowserTTSProvider", () => ({
  // eslint-disable-next-line @typescript-eslint/no-extraneous-class
  BrowserTTSProvider: class {
    constructor() {
      return h.newProvider(h.browserInstances);
    }
  },
}));

import { useTextToSpeechCore, type TTSCoreSpeakSettings, type UseTextToSpeechCoreOptions } from "./useTextToSpeechCore";

const lastKokoro = () => h.kokoroInstances[h.kokoroInstances.length - 1]!;
const lastBrowser = () => h.browserInstances[h.browserInstances.length - 1]!;
const runtime = {
  synthesizeTTS: vi.fn(() => Promise.resolve(new Blob(["audio"]))),
  getTTSVoices: vi.fn(() => Promise.resolve([{ id: "af_heart", name: "Heart" }])),
  fetchCachedTTS: vi.fn(() => Promise.resolve(null)),
};

function settings(): TTSCoreSpeakSettings {
  return { voice: "Alice", rate: 1, pitch: 1, kokoroVoice: "af_heart", kokoroSpeed: 1 };
}

function options(over: Partial<UseTextToSpeechCoreOptions> = {}): UseTextToSpeechCoreOptions {
  return {
    autoEnabled: true,
    backend: "auto",
    startMuted: false,
    runtime,
    ...over,
  };
}

function renderTTS(over: Partial<UseTextToSpeechCoreOptions> = {}, sett = settings()) {
  return renderHook(({ o, s }) => useTextToSpeechCore(o, s), {
    initialProps: { o: options(over), s: sett },
  });
}

beforeEach(() => {
  h.kokoroInstances.length = 0;
  h.browserInstances.length = 0;
  runtime.getTTSVoices.mockResolvedValue([{ id: "af_heart", name: "Heart" }]);
  runtime.fetchCachedTTS.mockResolvedValue(null);
  // Provide a minimal speechSynthesis so isBrowserSupported() is true and the
  // browser-voice loader has something to read.
  Object.defineProperty(window, "speechSynthesis", {
    configurable: true,
    value: {
      getVoices: vi.fn(() => [{ name: "Alice" }, { name: "Bob" }]),
      onvoiceschanged: null,
    },
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("useTextToSpeechCore — backend selection", () => {
  it("selects Kokoro under auto and loads voices", async () => {
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    expect(result.current.supported).toBe(true);
    expect(result.current.voices).toEqual([{ id: "af_heart", name: "Heart" }]);
  });

  it("falls back to a default Kokoro voice when the voice list fails", async () => {
    runtime.getTTSVoices.mockRejectedValueOnce(new Error("no voices"));
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    expect(result.current.voices).toEqual([{ id: "af_heart", name: "af_heart" }]);
  });

  it("selects the Browser backend when requested explicitly", async () => {
    const { result } = renderTTS({ backend: "browser" });
    await waitFor(() => expect(result.current.backend).toBe("browser"));
    expect(result.current.voices.map((v) => v.id)).toEqual(["Alice", "Bob"]);
  });

  it("reports no backend when browser is requested but unsupported", async () => {
    // Remove speech synthesis entirely so `"speechSynthesis" in window` is false.
    Reflect.deleteProperty(window, "speechSynthesis");
    const { result } = renderTTS({ backend: "browser" });
    await waitFor(() => expect(result.current.backend).toBe("none"));
    expect(result.current.supported).toBe(false);
  });

  it("downgrades to none when Kokoro is requested but unavailable", async () => {
    const { result } = renderTTS({ backend: "kokoro", kokoroAvailable: () => Promise.resolve(false) });
    await waitFor(() => expect(result.current.backend).toBe("none"));
    expect(result.current.backendReason).toMatch(/Kokoro backend was selected/);
  });

  it("falls back to Browser under auto when Kokoro is unavailable", async () => {
    const { result } = renderTTS({ kokoroAvailable: () => Promise.resolve(false) });
    await waitFor(() => expect(result.current.backend).toBe("browser"));
    expect(result.current.backendReason).toMatch(/browser speech synthesis is active/);
  });

  it("treats a throwing kokoroAvailable probe as unavailable", async () => {
    const { result } = renderTTS({ kokoroAvailable: () => Promise.reject(new Error("boom")) });
    await waitFor(() => expect(result.current.backend).toBe("browser"));
  });
});

describe("useTextToSpeechCore — speak", () => {
  it("speaks successfully and records the last successful backend", async () => {
    const onPlaybackEvent = vi.fn();
    const { result } = renderTTS({ onPlaybackEvent });
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));

    await act(async () => {
      result.current.speak("hello");
      await Promise.resolve();
    });
    await waitFor(() => expect(result.current.lastSuccessfulBackend).toBe("kokoro"));
    expect(lastKokoro().speak).toHaveBeenCalledWith("hello", expect.anything());
    expect(onPlaybackEvent).toHaveBeenCalledWith(expect.objectContaining({ stage: "attempt" }));
    expect(onPlaybackEvent).toHaveBeenCalledWith(expect.objectContaining({ stage: "success" }));
  });

  it("surfaces a speak error", async () => {
    const onPlaybackEvent = vi.fn();
    const { result } = renderTTS({ backend: "browser", onPlaybackEvent });
    await waitFor(() => expect(result.current.backend).toBe("browser"));
    lastBrowser().speak.mockRejectedValueOnce(new Error("synth boom"));
    // Browser backend requires an audio-unlock gesture first.
    act(() => {
      window.dispatchEvent(new Event("pointerdown"));
    });

    await act(async () => {
      result.current.speak("oops");
      await Promise.resolve();
    });
    await waitFor(() => expect(result.current.error).toBe("synth boom"));
    expect(onPlaybackEvent).toHaveBeenCalledWith(expect.objectContaining({ stage: "error" }));
  });

  it("routes an autoplay-block into needsUnlock", async () => {
    const { result } = renderTTS({ backend: "browser" });
    await waitFor(() => expect(result.current.backend).toBe("browser"));
    act(() => {
      window.dispatchEvent(new Event("pointerdown"));
    });
    const blocked = new Error("play() not allowed");
    blocked.name = "NotAllowedError";
    lastBrowser().speak.mockRejectedValueOnce(blocked);

    await act(async () => {
      result.current.speak("blocked");
      await Promise.resolve();
    });
    await waitFor(() => expect(result.current.needsUnlock).toBe(true));
  });

  it("clears isSpeaking quietly on an abort-like error", async () => {
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    const abort = new Error("The operation was aborted.");
    abort.name = "AbortError";
    lastKokoro().speak.mockRejectedValueOnce(abort);

    await act(async () => {
      result.current.speak("cancel-me");
      await Promise.resolve();
    });
    await waitFor(() => expect(result.current.isSpeaking).toBe(false));
    expect(result.current.error).toBeNull();
  });

  it("falls back to Browser when Kokoro fails at runtime (auto)", async () => {
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    act(() => {
      window.dispatchEvent(new Event("pointerdown"));
    });
    lastKokoro().speak.mockRejectedValueOnce(new Error("kokoro down"));

    await act(async () => {
      result.current.speak("hi");
      await Promise.resolve();
    });
    await waitFor(() => expect(result.current.backendReason).toMatch(/Kokoro failed at runtime/));
    expect(lastBrowser().speak).toHaveBeenCalled();
  });
});

describe("useTextToSpeechCore — speakParagraphs", () => {
  it("plays cached audio when an eventId is provided and the backend is Kokoro", async () => {
    runtime.fetchCachedTTS
      .mockResolvedValueOnce(new Blob(["cached-a"]))
      .mockResolvedValueOnce(new Blob(["cached-b"]));
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));

    let used: string | undefined;
    await act(async () => {
      used = await result.current.speakParagraphs(["a", "b"], { eventId: "evt-1" });
    });
    expect(used).toBe("kokoro");
    expect(lastKokoro().speakFromBlobs).toHaveBeenCalled();
  });

  it("uses speakSequence for multi-paragraph synthesis", async () => {
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    let used: string | undefined;
    await act(async () => {
      used = await result.current.speakParagraphs(["one", "two", "three"]);
    });
    expect(used).toBe("kokoro");
    expect(lastKokoro().speakSequence).toHaveBeenCalledWith(["one", "two", "three"], expect.anything());
  });

  it("returns undefined for an empty paragraph list", async () => {
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    let used: string | undefined;
    await act(async () => {
      used = await result.current.speakParagraphs([]);
    });
    expect(used).toBeUndefined();
  });
});

describe("useTextToSpeechCore — playback controls", () => {
  it("stop aborts the chain and resets speaking state", async () => {
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    act(() => result.current.stop());
    expect(lastKokoro().stop).toHaveBeenCalled();
    expect(result.current.isSpeaking).toBe(false);
  });

  it("pause and resume toggle isPaused and call the provider", async () => {
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    act(() => result.current.pause());
    expect(result.current.isPaused).toBe(true);
    expect(lastKokoro().pause).toHaveBeenCalled();
    act(() => result.current.resume());
    expect(result.current.isPaused).toBe(false);
    expect(lastKokoro().resume).toHaveBeenCalled();
  });

  it("seek, setPlaybackRate and setVolume forward to the provider", async () => {
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    act(() => result.current.seek(3));
    expect(lastKokoro().seek).toHaveBeenCalledWith(3);
    act(() => result.current.setPlaybackRate(1.5));
    expect(result.current.playbackRate).toBe(1.5);
    act(() => result.current.setVolume(0.25));
    expect(result.current.volume).toBe(0.25);
    expect(lastKokoro().setVolume).toHaveBeenCalled();
  });

  it("setMuted updates state and pushes effective volume", async () => {
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    act(() => result.current.setMuted(true));
    expect(result.current.isMuted).toBe(true);
    // Effective volume sent to the provider is 0 when muted.
    expect(lastKokoro().setVolume).toHaveBeenCalledWith(0);
  });

  it("getPlaybackState overlays the hook-level volume and mute", async () => {
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    act(() => result.current.setVolume(0.4));
    act(() => result.current.setMuted(true));
    const ps = result.current.getPlaybackState();
    expect(ps).toMatchObject({ volume: 0.4, isMuted: true });
  });

  it("refresh re-runs backend resolution", async () => {
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    runtime.getTTSVoices.mockClear();
    await act(async () => {
      await result.current.refresh();
    });
    expect(runtime.getTTSVoices).toHaveBeenCalled();
  });
});

describe("useTextToSpeechCore — unlock + test", () => {
  it("testSpeak runs a sample utterance", async () => {
    const onPlaybackEvent = vi.fn();
    const { result } = renderTTS({ onPlaybackEvent });
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    await act(async () => {
      await result.current.testSpeak();
    });
    expect(lastKokoro().speak).toHaveBeenCalled();
    expect(onPlaybackEvent).toHaveBeenCalledWith(expect.objectContaining({ stage: "success" }));
  });

  it("unlockAudio flips browserAudioReady and clears needsUnlock", async () => {
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    let ok: boolean | undefined;
    await act(async () => {
      ok = await result.current.unlockAudio();
    });
    expect(ok).toBe(true);
    expect(result.current.browserAudioReady).toBe(true);
    expect(lastKokoro().unlock).toHaveBeenCalledWith(true);
  });

  it("a user gesture marks browser audio ready", async () => {
    const { result } = renderTTS();
    await waitFor(() => expect(result.current.backend).toBe("kokoro"));
    await act(async () => {
      window.dispatchEvent(new Event("keydown"));
      await Promise.resolve();
    });
    expect(result.current.browserAudioReady).toBe(true);
  });

  it("dismissNeedsUnlock clears the flag", async () => {
    const { result } = renderTTS({ backend: "browser" });
    await waitFor(() => expect(result.current.backend).toBe("browser"));
    act(() => {
      window.dispatchEvent(new Event("pointerdown"));
    });
    const blocked = new Error("blocked");
    blocked.name = "NotAllowedError";
    lastBrowser().speak.mockRejectedValueOnce(blocked);
    await act(async () => {
      result.current.speak("x");
      await Promise.resolve();
    });
    await waitFor(() => expect(result.current.needsUnlock).toBe(true));
    act(() => result.current.dismissNeedsUnlock());
    expect(result.current.needsUnlock).toBe(false);
  });
});
