import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { apiBaseMock } from "../../test-utils";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { registerVoiceTransport as registerBrowserVoiceTransport } from "@vrooli/audio-capture-browser";
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

function removeSpeechRecognition() {
  const speechWindow = window as { SpeechRecognition?: unknown; webkitSpeechRecognition?: unknown };
  delete speechWindow.SpeechRecognition;
  delete speechWindow.webkitSpeechRecognition;
}

// ---------------------------------------------------------------------------
// Hook integration tests — backend detection and durable refusal
// ---------------------------------------------------------------------------

describe("useVoiceInput", () => {
  let originalFetch: typeof globalThis.fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    vi.clearAllMocks();
    registerBrowserVoiceTransport({
      buildStreamUrl: () => "ws://voice.test/stream",
      transcribeRetained: async () => "",
    });
    removeSpeechRecognition();
    useWorkspaceStore.setState({ voiceEnabled: true });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("refuses voice input when the durable backend is unavailable", async () => {
    mockCapabilities(false);
    mockMediaDevices(true);

    const onTranscript = vi.fn();
    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => {
      await vi.dynamicImportSettled?.();
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(result.current.backend).toBe("none");
    expect(result.current.supported).toBe(false);
    expect(result.current.error).toBe("Durable audio path unavailable");
    expect(result.current.fallbackNotice).toContain("audio-tools cannot be reached");
  });

  it("uses whisper when available", async () => {
    mockCapabilities(true);
    mockMediaDevices(true);

    const onTranscript = vi.fn();
    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
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
    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
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

    const onTranscript = vi.fn();
    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
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
    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(result.current.supported).toBe(false);
    expect(result.current.backend).toBe("none");
  });
});

// ---------------------------------------------------------------------------
// Capture lifecycle ownership (mic-lease honesty + self-healing)
// ---------------------------------------------------------------------------

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

  it("refuses recording without a durable provider and does not acquire the mic", async () => {
    mockCapabilities(false);
    mockMediaDevices(true);

    const onTranscript = vi.fn();
    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    expect(result.current.backend).toBe("none");
    expect(result.current.supported).toBe(false);

    await act(async () => { await result.current.startRecording(); });
    expect(result.current.voiceState).toBe("idle");
    expect(getActiveMicLeases()).toHaveLength(0);
  });

  it("self-heals an orphaned live mic lease while the UI is idle (registry-vs-UI mismatch)", async () => {
    mockCapabilities(true); // whisper backend; idle, not recording
    mockMediaDevices(true);
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});

    const onTranscript = vi.fn();
    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
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
    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
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
    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
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

  it("does not accept a premature server final as a successful turn", async () => {
    mockCapabilities(true);
    const browser = installFinalFrameBrowserFakes();
    const onTranscript = vi.fn();
    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    expect(result.current.backend).toBe("whisper");

    await act(async () => { await result.current.startRecording(); });
    await waitFor(() => expect(result.current.voiceState).toBe("recording"));
    expect(getActiveMicLeases()).toHaveLength(1);
    const firstSocket = FinalFrameWebSocket.instances.at(-1);
    if (!firstSocket) throw new Error("expected streaming socket fake to exist");
    expect(browser.tracks[0]?.readyState).toBe("live");

    await act(async () => {
      firstSocket.emitFinal("server finished");
      await Promise.resolve();
    });

    // A server final while the browser is still recording is a degraded
    // transport signal, not a successful turn. The provider reconnects and
    // replays retained PCM so the words are not silently discarded.
    await waitFor(() => expect(result.current.streamingDegradationNotice).toContain("recovering retained audio"));
    expect(result.current.voiceState).toBe("recording");
    expect(onTranscript).not.toHaveBeenCalled();
    expect(getActiveMicLeases()).toHaveLength(1);
    expect(browser.tracks[0]?.readyState).toBe("live");

    await act(async () => { result.current.releaseMicrophone(); });
    await waitFor(() => expect(result.current.voiceState).toBe("idle"));
    expect(getActiveMicLeases()).toHaveLength(0);
    expect(browser.tracks[0]?.stop).toHaveBeenCalled();

    await act(async () => { await result.current.startRecording(); });
    await waitFor(() => expect(result.current.voiceState).toBe("recording"));
    expect(browser.getUserMedia).toHaveBeenCalledTimes(2);
    expect(getActiveMicLeases()).toHaveLength(1);
  });
});
