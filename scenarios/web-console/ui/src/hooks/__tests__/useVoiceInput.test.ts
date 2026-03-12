import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { apiBaseMock } from "../../test-utils";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";

vi.mock("@vrooli/api-base", () => apiBaseMock());

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const mockCapabilities = (whisperAvailable: boolean) => {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () =>
      Promise.resolve({
        capabilities: [
          {
            id: "whisper-stt",
            status: whisperAvailable ? "available" : "unavailable",
          },
        ],
        timestamp: new Date().toISOString(),
      }),
  }) as typeof fetch;
};

const mockMediaDevices = (success: boolean) => {
  const mockStream = {
    getTracks: () => [{ stop: vi.fn() }],
  } as unknown as MediaStream;
  Object.defineProperty(navigator, "mediaDevices", {
    value: {
      getUserMedia: success
        ? vi.fn().mockResolvedValue(mockStream)
        : vi.fn().mockRejectedValue(new Error("Permission denied")),
    },
    configurable: true,
  });
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
// Tests
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
    // Dynamic import so vi.mock is applied
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    // Wait for the async capability detection to settle
    await act(async () => {
      await vi.dynamicImportSettled?.();
      // Allow microtasks from the effect to flush
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
    globalThis.fetch = vi.fn().mockRejectedValue(
      new Error("Network error"),
    ) as typeof fetch;
    removeSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    // Should gracefully fall through to unsupported
    expect(result.current.supported).toBe(false);
    expect(result.current.backend).toBe("none");
  });
});
