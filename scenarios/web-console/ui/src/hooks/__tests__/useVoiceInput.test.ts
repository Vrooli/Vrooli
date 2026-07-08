import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { apiBaseMock } from "../../test-utils";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import {
  _resetMicOwnershipForTesting,
  getActiveMicLeases,
  registerMicStream,
} from "../../audio-integration";

vi.mock("@vrooli/api-base", () => apiBaseMock());

const fetchCapabilitiesMock = vi.fn();
const fetchCapabilitiesLivenessCachedMock = vi.fn();
const refreshCapabilitiesLivenessMock = vi.fn();
const getCapabilitiesLivenessSnapshotMock = vi.fn(() => null);

vi.mock("../../api/capabilities", () => ({
  fetchCapabilities: fetchCapabilitiesMock,
  fetchCapabilitiesLiveness: fetchCapabilitiesLivenessCachedMock,
  fetchCapabilitiesLivenessCached: fetchCapabilitiesLivenessCachedMock,
  refreshCapabilitiesLiveness: refreshCapabilitiesLivenessMock,
  getCapabilitiesLivenessSnapshot: getCapabilitiesLivenessSnapshotMock,
}));

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const mockCapabilities = (whisperAvailable: boolean) => {
  const resp = {
    capabilities: [
      {
        id: "audio-tools",
        status: whisperAvailable ? "available" : "unavailable",
        features: whisperAvailable ? ["voice-input", "voice-streaming"] : [],
      },
    ],
    timestamp: new Date().toISOString(),
  };
  fetchCapabilitiesMock.mockResolvedValue(resp);
  fetchCapabilitiesLivenessCachedMock.mockResolvedValue(resp);
  refreshCapabilitiesLivenessMock.mockResolvedValue(resp);
};

const mockMediaDevices = (success: boolean) => {
  const mockStream = {
    getTracks: () => [{ readyState: "live", muted: false, kind: "audio", stop: vi.fn() }],
  } as unknown as MediaStream;
  const getUserMedia = success
    ? vi.fn().mockResolvedValue(mockStream)
    : vi.fn().mockRejectedValue(new Error("Permission denied"));
  Object.defineProperty(navigator, "mediaDevices", {
    value: {
      getUserMedia,
    },
    configurable: true,
  });
  return { getUserMedia, mockStream };
};

/** Minimal SpeechRecognition stub */
function installSpeechRecognition() {
  window.SpeechRecognition = class {
    continuous = false;
    interimResults = false;
    lang = "";
    onresult: ((e: unknown) => void) | null = null;
    onerror: ((e: unknown) => void) | null = null;
    onend: (() => void) | null = null;
    start() {}
    stop() {}
    abort() {}
    addEventListener() {}
    removeEventListener() {}
    dispatchEvent() {
      return false;
    }
  } as unknown as typeof window.SpeechRecognition;
}

function removeSpeechRecognition() {
  delete window.SpeechRecognition;
  delete window.webkitSpeechRecognition;
}

// ---------------------------------------------------------------------------
// Hook integration tests — backend detection and fallback
// ---------------------------------------------------------------------------

describe("useVoiceInput", () => {
  let originalFetch: typeof globalThis.fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    vi.clearAllMocks();
    removeSpeechRecognition();
    useWorkspaceStore.setState({ voiceEnabled: true });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("falls back to web-speech when whisper unavailable", async () => {
    mockCapabilities(false);
    installSpeechRecognition();
    mockMediaDevices(true);

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => {
      await vi.dynamicImportSettled?.();
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(result.current.backend).toBe("web-speech");
    expect(result.current.supported).toBe(true);
  });

  it("uses whisper when available", async () => {
    mockCapabilities(true);
    mockMediaDevices(true);

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(result.current.backend).toBe("whisper");
    expect(result.current.supported).toBe(true);
  });

  it("reports unsupported when no backend available", async () => {
    mockCapabilities(false);
    removeSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(result.current.supported).toBe(false);
    expect(result.current.backend).toBe("none");
  });

  it("disables when voiceEnabled is false", async () => {
    useWorkspaceStore.setState({ voiceEnabled: false });
    mockCapabilities(true);
    installSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(result.current.supported).toBe(false);
    expect(result.current.backend).toBe("none");
  });

  it("reports error when capabilities fetch fails", async () => {
    fetchCapabilitiesMock.mockRejectedValue(new Error("Network error"));
    fetchCapabilitiesLivenessCachedMock.mockRejectedValue(new Error("Network error"));
    refreshCapabilitiesLivenessMock.mockRejectedValue(new Error("Network error"));
    removeSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(result.current.supported).toBe(false);
    expect(result.current.backend).toBe("none");
  });
});

// ---------------------------------------------------------------------------
// WebSpeech deduplication (integration-level via hook)
// ---------------------------------------------------------------------------

/**
 * Controllable SpeechRecognition stub for testing processedResultCount
 * deduplication through the full hook lifecycle.
 */
function installControllableSpeechRecognition() {
  type SRInstance = {
    continuous: boolean;
    interimResults: boolean;
    lang: string;
    onresult: ((e: unknown) => void) | null;
    onerror: ((e: unknown) => void) | null;
    onend: (() => void) | null;
    start(): void;
    stop(): void;
    abort(): void;
    addEventListener(): void;
    removeEventListener(): void;
    dispatchEvent(): boolean;
  };

  let instance: SRInstance | null = null;

  window.SpeechRecognition = class {
    continuous = false;
    interimResults = false;
    lang = "";
    onresult: ((e: unknown) => void) | null = null;
    onerror: ((e: unknown) => void) | null = null;
    onend: (() => void) | null = null;
    start() { instance = this as unknown as SRInstance; }
    stop() { this.onend?.(); }
    abort() {}
    addEventListener() {}
    removeEventListener() {}
    dispatchEvent() { return false; }
  } as unknown as typeof window.SpeechRecognition;

  function fireResult(results: Array<{ transcript: string; isFinal: boolean }>) {
    if (!instance?.onresult) return;
    const resultList = results.map((r) => {
      const item = { transcript: r.transcript, confidence: 0.95 };
      return Object.assign([item], { isFinal: r.isFinal, length: 1, item: () => item });
    });
    const event = {
      results: Object.assign(resultList, {
        length: resultList.length,
        item: (i: number) => resultList[i],
      }),
    };
    instance.onresult(event);
  }

  return {
    getInstance: () => instance,
    fireResult,
    triggerEnd: () => instance?.onend?.(),
  };
}

describe("WebSpeechProvider deduplication (via hook)", () => {
  let originalFetch: typeof globalThis.fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    vi.clearAllMocks();
    removeSpeechRecognition();
    useWorkspaceStore.setState({ voiceEnabled: true });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("dispatches only new final results, not cumulative duplicates", async () => {
    mockCapabilities(false);
    mockMediaDevices(true);
    const ctrl = installControllableSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    expect(result.current.backend).toBe("web-speech");

    await act(async () => { await result.current.startRecording(); });

    act(() => {
      ctrl.fireResult([{ transcript: "hello", isFinal: true }]);
    });
    expect(onTranscript).toHaveBeenCalledTimes(1);
    expect(onTranscript).toHaveBeenLastCalledWith("hello");

    act(() => {
      ctrl.fireResult([
        { transcript: "hello", isFinal: true },
        { transcript: " world", isFinal: true },
      ]);
    });
    expect(onTranscript).toHaveBeenCalledTimes(2);
    expect(onTranscript).toHaveBeenLastCalledWith("world");
  });

  it("interim results update partialTranscript but do not dispatch as final", async () => {
    mockCapabilities(false);
    mockMediaDevices(true);
    const ctrl = installControllableSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    await act(async () => { await result.current.startRecording(); });

    act(() => {
      ctrl.fireResult([{ transcript: "hel", isFinal: false }]);
    });

    expect(onTranscript).not.toHaveBeenCalled();
    expect(result.current.partialTranscript).toBe("hel");
  });

  it("processedResultCount persists across spontaneous recognition restarts", async () => {
    mockCapabilities(false);
    mockMediaDevices(true);
    const ctrl = installControllableSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    await act(async () => { await result.current.startRecording(); });

    act(() => {
      ctrl.fireResult([{ transcript: "hello", isFinal: true }]);
    });
    expect(onTranscript).toHaveBeenCalledTimes(1);

    act(() => { ctrl.triggerEnd(); });

    act(() => {
      ctrl.fireResult([
        { transcript: "hello", isFinal: true },
        { transcript: " world", isFinal: true },
      ]);
    });
    expect(onTranscript).toHaveBeenCalledTimes(2);
    expect(onTranscript).toHaveBeenLastCalledWith("world");
  });
});

// ---------------------------------------------------------------------------
// Capture lifecycle ownership (mic-lease honesty + self-healing)
// ---------------------------------------------------------------------------

/** SpeechRecognition stub that can fire onerror, for the error-path test. */
function installErrorableSpeechRecognition() {
  type SR = {
    onresult: ((e: unknown) => void) | null;
    onerror: ((e: unknown) => void) | null;
    onend: (() => void) | null;
    continuous: boolean;
    interimResults: boolean;
    lang: string;
    start(): void;
    stop(): void;
    abort(): void;
    addEventListener(): void;
    removeEventListener(): void;
    dispatchEvent(): boolean;
  };
  let instance: SR | null = null;
  window.SpeechRecognition = class {
    onresult: ((e: unknown) => void) | null = null;
    onerror: ((e: unknown) => void) | null = null;
    onend: (() => void) | null = null;
    continuous = false;
    interimResults = false;
    lang = "";
    start() { instance = this as unknown as SR; }
    stop() {}
    abort() {}
    addEventListener() {}
    removeEventListener() {}
    dispatchEvent() { return false; }
  } as unknown as typeof window.SpeechRecognition;
  return { fireError: (error: string) => instance?.onerror?.({ error, message: error }) };
}

describe("voice capture lifecycle ownership", () => {
  let originalFetch: typeof globalThis.fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    vi.clearAllMocks();
    removeSpeechRecognition();
    _resetMicOwnershipForTesting();
    useWorkspaceStore.setState({ voiceEnabled: true });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    _resetMicOwnershipForTesting();
  });

  it("a provider error while recording releases the mic lease and returns the UI to idle", async () => {
    mockCapabilities(false); // web-speech backend (testable without MediaRecorder/WS)
    mockMediaDevices(true);
    const ctrl = installErrorableSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    expect(result.current.backend).toBe("web-speech");

    await act(async () => { await result.current.startRecording(); });
    // The provider acquired a registry mic lease for capture.
    expect(getActiveMicLeases().length).toBeGreaterThan(0);

    await act(async () => { ctrl.fireError("network"); });

    // Idle UI AND no live mic lease — the error path disposed the provider.
    expect(result.current.voiceState).toBe("idle");
    expect(getActiveMicLeases()).toHaveLength(0);
  });

  it("self-heals an orphaned live mic lease while the UI is idle (registry-vs-UI mismatch)", async () => {
    mockCapabilities(true); // whisper backend; idle, not recording
    mockMediaDevices(true);
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    expect(result.current.voiceState).toBe("idle");

    // Inject the bug: a live active-recording lease appears while the UI is idle.
    await act(async () => {
      registerMicStream("voice-stream", {
        getTracks: () => [{ readyState: "live", muted: false, kind: "audio", stop: vi.fn(), addEventListener() {}, removeEventListener() {} }],
      } as unknown as MediaStream);
    });

    // The registry subscription detected and self-healed the orphan.
    expect(getActiveMicLeases()).toHaveLength(0);
    expect(warn).toHaveBeenCalledWith(
      expect.stringContaining("INVARIANT VIOLATION"),
      expect.anything(),
      expect.anything(),
    );

    warn.mockRestore();
  });

  it("does not acquire the mic on mount or visibility with persisted wake-word flag", async () => {
    mockCapabilities(true);
    const media = mockMediaDevices(true);
    useWorkspaceStore.setState({
      voiceEnabled: true,
      wakeWordEnabled: true,
      persistentMode: false,
    });

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    expect(result.current.backend).toBe("whisper");
    expect(media.getUserMedia).not.toHaveBeenCalled();
    expect(getActiveMicLeases()).toHaveLength(0);

    await act(async () => {
      Object.defineProperty(document, "visibilityState", { configurable: true, value: "visible" });
      document.dispatchEvent(new Event("visibilitychange"));
      await Promise.resolve();
    });

    expect(media.getUserMedia).not.toHaveBeenCalled();
    expect(getActiveMicLeases()).toHaveLength(0);
  });

  it("prepareRecording never acquires the mic (no pre-warm / no idle mic hold)", async () => {
    // Regression: low-latency pre-warm was removed. Signalling intent must NOT
    // open the microphone — holding the mic idle ducks other apps' audio and
    // churns the iOS audio session. The mic is acquired only on actual record.
    mockCapabilities(true);
    const media = mockMediaDevices(true);
    useWorkspaceStore.setState({
      voiceEnabled: true,
      wakeWordEnabled: false,
      persistentMode: false,
    });

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    expect(media.getUserMedia).not.toHaveBeenCalled();

    await act(async () => {
      result.current.prepareRecording();
      await new Promise((r) => setTimeout(r, 50));
    });

    // Still no mic — prepare only arms intent / passive reconcile.
    expect(media.getUserMedia).not.toHaveBeenCalled();
    expect(getActiveMicLeases()).toHaveLength(0);
  });
});

class FinalFrameWebSocket {
  static OPEN = 1;
  static CLOSED = 3;
  static CONNECTING = 0;
  static instances: FinalFrameWebSocket[] = [];
  readyState = FinalFrameWebSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((e: { data: string }) => void) | null = null;
  send = vi.fn<(data: unknown) => void>();
  close = vi.fn(() => {
    this.readyState = FinalFrameWebSocket.CLOSED;
    this.onclose?.();
  });

  constructor(readonly url: string) {
    FinalFrameWebSocket.instances.push(this);
    queueMicrotask(() => {
      this.readyState = FinalFrameWebSocket.OPEN;
      this.onopen?.();
    });
  }

  emitFinal(text: string): void {
    this.onmessage?.({ data: JSON.stringify({ type: "final", text }) });
  }
}

class FinalFrameMediaRecorder {
  static instances: FinalFrameMediaRecorder[] = [];
  static isTypeSupported = vi.fn(() => true);
  state: "inactive" | "recording" = "inactive";
  ondataavailable: ((e: { data: Blob }) => void) | null = null;
  onstop: (() => void) | null = null;
  start = vi.fn(() => {
    this.state = "recording";
  });
  stop = vi.fn(() => {
    this.state = "inactive";
    this.onstop?.();
  });

  constructor(_stream: MediaStream, _opts?: unknown) {
    FinalFrameMediaRecorder.instances.push(this);
  }
}

function installFinalFrameBrowserFakes() {
  FinalFrameWebSocket.instances = [];
  FinalFrameMediaRecorder.instances = [];
  (globalThis as unknown as { WebSocket: typeof FinalFrameWebSocket }).WebSocket = FinalFrameWebSocket;
  (globalThis as unknown as { MediaRecorder: typeof FinalFrameMediaRecorder }).MediaRecorder = FinalFrameMediaRecorder;
  const tracks: Array<MediaStreamTrack & { stop: ReturnType<typeof vi.fn> }> = [];
  const getUserMedia = vi.fn(async () => {
    const track = {
      kind: "audio",
      muted: false,
      readyState: "live" as MediaStreamTrackState,
      stop: vi.fn(function stop(this: { readyState: MediaStreamTrackState }) {
        this.readyState = "ended";
      }),
      addEventListener() {},
      removeEventListener() {},
    } as unknown as MediaStreamTrack & { stop: ReturnType<typeof vi.fn> };
    tracks.push(track);
    return {
      active: true,
      getTracks: () => [track],
      getAudioTracks: () => [track],
    } as unknown as MediaStream;
  });
  Object.defineProperty(navigator, "mediaDevices", {
    configurable: true,
    value: { getUserMedia },
  });
  return { getUserMedia, tracks };
}

describe("voice capture lifecycle server-final wedge", () => {
  let originalFetch: typeof globalThis.fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    vi.clearAllMocks();
    removeSpeechRecognition();
    _resetMicOwnershipForTesting();
    useWorkspaceStore.setState({
      voiceEnabled: true,
      wakeWordEnabled: false,
      persistentMode: false,
    });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    _resetMicOwnershipForTesting();
  });

  it("tears down capture when the server sends final before the client stops", async () => {
    mockCapabilities(true);
    const browser = installFinalFrameBrowserFakes();
    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    expect(result.current.backend).toBe("whisper");

    await act(async () => { await result.current.startRecording(); });
    await waitFor(() => expect(result.current.voiceState).toBe("recording"));
    expect(getActiveMicLeases()).toHaveLength(1);
    const firstRecorder = FinalFrameMediaRecorder.instances.at(-1);
    const firstSocket = FinalFrameWebSocket.instances.at(-1);
    if (!firstRecorder || !firstSocket) throw new Error("expected recording fakes to exist");
    expect(firstRecorder.state).toBe("recording");

    await act(async () => {
      firstSocket.emitFinal("server finished");
      await Promise.resolve();
    });

    await waitFor(() => expect(result.current.voiceState).toBe("idle"));
    expect(onTranscript).toHaveBeenCalledWith("server finished");
    expect(getActiveMicLeases()).toHaveLength(0);
    expect(firstRecorder.stop).toHaveBeenCalled();
    expect(firstRecorder.state).toBe("inactive");
    expect(browser.tracks[0]?.stop).toHaveBeenCalled();

    await act(async () => { await result.current.startRecording(); });
    await waitFor(() => expect(result.current.voiceState).toBe("recording"));
    expect(browser.getUserMedia).toHaveBeenCalledTimes(2);
    expect(getActiveMicLeases()).toHaveLength(1);
  });
});
