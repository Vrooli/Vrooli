import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { WebSpeechProvider } from "../WebSpeechProvider";

// --- Mock infrastructure ---

const mockTrackStop = vi.fn();
const mockStream = {
  getTracks: () => [{ stop: mockTrackStop }],
} as unknown as MediaStream;

type SRInstance = {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  onresult: ((e: unknown) => void) | null;
  onerror: ((e: unknown) => void) | null;
  onend: (() => void) | null;
  start: ReturnType<typeof vi.fn>;
  stop: ReturnType<typeof vi.fn>;
};

let srInstance: SRInstance | null = null;

function installSpeechRecognition() {
  window.SpeechRecognition = class {
    continuous = false;
    interimResults = false;
    lang = "";
    onresult: ((e: unknown) => void) | null = null;
    onerror: ((e: unknown) => void) | null = null;
    onend: (() => void) | null = null;
    start = vi.fn(() => { srInstance = this as unknown as SRInstance; });
    stop = vi.fn(() => { this.onend?.(); });
    abort() {}
    addEventListener() {}
    removeEventListener() {}
    dispatchEvent() { return false; }
  } as unknown as typeof window.SpeechRecognition;
}

function removeSpeechRecognition() {
  delete window.SpeechRecognition;
  delete window.webkitSpeechRecognition;
}

function installMicMock(success = true) {
  Object.defineProperty(navigator, "mediaDevices", {
    value: {
      getUserMedia: success
        ? vi.fn().mockResolvedValue(mockStream)
        : vi.fn().mockRejectedValue(new Error("Permission denied")),
    },
    configurable: true,
  });
}

/** Build synthetic cumulative results matching browser SpeechRecognition event shape. */
function makeSpeechEvent(results: Array<{ transcript: string; isFinal: boolean }>) {
  const resultList = results.map((r) => {
    const item = { transcript: r.transcript, confidence: 0.95 };
    return Object.assign([item], { isFinal: r.isFinal, length: 1, item: () => item });
  });
  return {
    results: Object.assign(resultList, {
      length: resultList.length,
      item: (i: number) => resultList[i],
    }),
  };
}

describe("WebSpeechProvider", () => {
  beforeEach(() => {
    srInstance = null;
    mockTrackStop.mockClear();
    vi.clearAllMocks();
    installSpeechRecognition();
    installMicMock();
  });

  afterEach(() => {
    removeSpeechRecognition();
  });

  // --- Lifecycle ---

  it("acquires mic and starts recognition on start()", async () => {
    const provider = new WebSpeechProvider();
    provider.onResult = vi.fn();

    await provider.start();

    expect(provider.getStream()).toBe(mockStream);
    expect(srInstance).toBeTruthy();
    expect(srInstance!.start).toHaveBeenCalled();
    expect(srInstance!.continuous).toBe(true);
    expect(srInstance!.interimResults).toBe(true);
  });

  it("sets language from lang property", async () => {
    const provider = new WebSpeechProvider();
    provider.lang = "fr-FR";
    provider.onResult = vi.fn();

    await provider.start();

    expect(srInstance!.lang).toBe("fr-FR");
  });

  it("calls onError when Web Speech API is not available", async () => {
    removeSpeechRecognition();
    const provider = new WebSpeechProvider();
    const onError = vi.fn();
    provider.onError = onError;

    await provider.start();

    expect(onError).toHaveBeenCalledWith("Web Speech API not available");
  });

  it("calls onError when microphone access is denied", async () => {
    installMicMock(false);
    const provider = new WebSpeechProvider();
    const onError = vi.fn();
    provider.onError = onError;

    await provider.start();

    expect(onError).toHaveBeenCalledWith("Microphone access denied");
    expect(provider.getStream()).toBeNull();
  });

  it("releases mic and stops recognition on stop()", async () => {
    const provider = new WebSpeechProvider();
    provider.onResult = vi.fn();

    await provider.start();
    provider.stop();

    expect(mockTrackStop).toHaveBeenCalled();
    expect(provider.getStream()).toBeNull();
  });

  it("dispose is equivalent to stop()", async () => {
    const provider = new WebSpeechProvider();
    provider.onResult = vi.fn();

    await provider.start();
    provider.dispose();

    expect(mockTrackStop).toHaveBeenCalled();
    expect(provider.getStream()).toBeNull();
  });

  // --- Result handling ---

  it("dispatches final results via onResult", async () => {
    const provider = new WebSpeechProvider();
    const onResult = vi.fn();
    provider.onResult = onResult;

    await provider.start();

    srInstance!.onresult?.(makeSpeechEvent([
      { transcript: "hello", isFinal: true },
    ]));

    expect(onResult).toHaveBeenCalledWith("hello");
  });

  it("dispatches interim results via onPartial", async () => {
    const provider = new WebSpeechProvider();
    const onPartial = vi.fn();
    const onResult = vi.fn();
    provider.onPartial = onPartial;
    provider.onResult = onResult;

    await provider.start();

    srInstance!.onresult?.(makeSpeechEvent([
      { transcript: "hel", isFinal: false },
    ]));

    expect(onPartial).toHaveBeenCalledWith("hel");
    expect(onResult).not.toHaveBeenCalled();
  });

  it("deduplicates cumulative results using processedResultCount", async () => {
    const provider = new WebSpeechProvider();
    const onResult = vi.fn();
    provider.onResult = onResult;

    await provider.start();

    // First result
    srInstance!.onresult?.(makeSpeechEvent([
      { transcript: "hello", isFinal: true },
    ]));
    expect(onResult).toHaveBeenCalledTimes(1);
    expect(onResult).toHaveBeenLastCalledWith("hello");

    // Second cumulative event includes old "hello" + new " world"
    srInstance!.onresult?.(makeSpeechEvent([
      { transcript: "hello", isFinal: true },
      { transcript: " world", isFinal: true },
    ]));
    expect(onResult).toHaveBeenCalledTimes(2);
    expect(onResult).toHaveBeenLastCalledWith("world");
  });

  it("preserves processedResultCount across spontaneous restarts", async () => {
    const provider = new WebSpeechProvider();
    const onResult = vi.fn();
    provider.onResult = onResult;

    await provider.start();

    // First result
    srInstance!.onresult?.(makeSpeechEvent([
      { transcript: "hello", isFinal: true },
    ]));
    expect(onResult).toHaveBeenCalledTimes(1);

    // Browser spontaneously ends recognition (triggers restart)
    srInstance!.onend?.();

    // After restart, cumulative event includes old + new
    srInstance!.onresult?.(makeSpeechEvent([
      { transcript: "hello", isFinal: true },
      { transcript: " world", isFinal: true },
    ]));
    expect(onResult).toHaveBeenCalledTimes(2);
    expect(onResult).toHaveBeenLastCalledWith("world");
  });

  // --- Error handling ---

  it("dispatches recognition errors via onError", async () => {
    const provider = new WebSpeechProvider();
    const onError = vi.fn();
    provider.onError = onError;

    await provider.start();

    srInstance!.onerror?.({ error: "network", message: "Network error" });

    expect(onError).toHaveBeenCalledWith("Speech recognition error: network");
  });

  it("ignores aborted errors", async () => {
    const provider = new WebSpeechProvider();
    const onError = vi.fn();
    provider.onError = onError;

    await provider.start();

    srInstance!.onerror?.({ error: "aborted", message: "" });

    expect(onError).not.toHaveBeenCalled();
  });

  it("auto-restarts on spontaneous end when not stopped", async () => {
    const provider = new WebSpeechProvider();
    provider.onResult = vi.fn();

    await provider.start();
    const firstCallCount = srInstance!.start.mock.calls.length;

    // Simulate spontaneous end
    srInstance!.onend?.();

    // Should have called start again
    expect(srInstance!.start.mock.calls.length).toBe(firstCallCount + 1);
  });

  it("does not restart on end when intentionally stopped", async () => {
    const provider = new WebSpeechProvider();
    provider.onResult = vi.fn();

    await provider.start();
    const startCallCount = srInstance!.start.mock.calls.length;

    provider.stop(); // triggers onend via mock

    // Should NOT have restarted
    expect(srInstance!.start.mock.calls.length).toBe(startCallCount);
  });
});
