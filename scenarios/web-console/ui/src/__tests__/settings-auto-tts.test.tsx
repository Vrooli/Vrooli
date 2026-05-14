import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import TtsSettingsSection from "../components/settings/TtsSettingsSection";
import { strings } from "../consts/strings";

const mockStoreState: Record<string, unknown> = {
  ttsVoice: "",
  setTtsVoice: vi.fn(),
  ttsRate: 1.0,
  setTtsRate: vi.fn(),
  ttsPitch: 1.0,
  setTtsPitch: vi.fn(),
  autoTtsEnabled: false,
  setAutoTtsEnabled: vi.fn(),
  ttsBackendPreference: "auto",
  setTtsBackendPreference: vi.fn(),
  kokoroVoice: "af_heart",
  setKokoroVoice: vi.fn(),
  kokoroSpeed: 1.0,
  setKokoroSpeed: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) => selector(mockStoreState),
}));

const mockUpdateTTSConfig = vi.fn((patch?: Partial<{
  autoEnabled: boolean;
  backend: "auto" | "kokoro" | "browser";
  kokoroVoice: string;
  kokoroSpeed: number;
}>) => Promise.resolve({
  autoEnabled: patch?.autoEnabled ?? true,
  backend: patch?.backend ?? "auto",
  kokoroVoice: patch?.kokoroVoice ?? "af_heart",
  kokoroSpeed: patch?.kokoroSpeed ?? 1.0,
}));

vi.mock("../api/tts", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/tts")>();
  return {
    ...actual,
    getTTSStatus: vi.fn().mockResolvedValue({
      config: {
        autoEnabled: false,
        backend: "auto",
        kokoroVoice: "af_heart",
        kokoroSpeed: 1.0,
      },
      hookRegistered: false,
      hookCode: "hook_missing",
      hookReason: "Claude Stop hook is not registered in project settings",
      lastHookRouting: {
        routed: false,
        code: "tts_target_missing",
        reason: "No terminal session was available for TTS routing",
        source: "claude_hook",
      },
      lastTailerRouting: {
        routed: false,
        code: "tts_target_missing",
        reason: "No terminal session was available for TTS routing",
        source: "codex_tailer",
      },
      lastHookAck: {
        eventId: "evt-1",
        source: "claude_hook",
        sessionId: "s1",
        stage: "rejected",
        message: "Assistant text did not match the rendered terminal buffer",
      },
      lastTailerAck: {
        eventId: "evt-2",
        source: "codex_tailer",
        sessionId: "s2",
        stage: "playback_succeeded",
        backend: "browser",
      },
      kokoroCapabilityLabel: "resource is not responding",
    }),
    updateTTSConfig: vi.fn((...args: unknown[]) => mockUpdateTTSConfig(args[0] as Parameters<typeof mockUpdateTTSConfig>[0])),
  };
});

vi.mock("../hooks/useTextToSpeech", () => ({
  useTextToSpeech: () => ({
    supported: true,
    isSpeaking: false,
    backend: mockStoreState.ttsBackendPreference === "kokoro" ? "kokoro" : "browser",
    voices: [{ id: "af_heart", name: "af_heart" }],
    error: null,
    backendReason: mockStoreState.ttsBackendPreference === "kokoro"
      ? "Kokoro backend selected explicitly"
      : "Kokoro is unavailable, so browser speech synthesis is active",
    browserAudioReady: false,
    lastSuccessfulBackend: "none",
    lastSuccessfulAt: null,
    refresh: vi.fn(),
    testSpeak: vi.fn(),
  }),
}));

Object.defineProperty(window, "speechSynthesis", {
  value: {
    speak: vi.fn(),
    cancel: vi.fn(),
    getVoices: vi.fn(() => []),
    speaking: false,
    paused: false,
    onvoiceschanged: null,
  },
  writable: true,
  configurable: true,
});

describe("TtsSettingsSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStoreState.autoTtsEnabled = false;
    mockStoreState.ttsBackendPreference = "auto";
    mockStoreState.kokoroSpeed = 1.0;
  });

  async function renderSection() {
    render(<TtsSettingsSection />);
    await waitFor(() => {
      expect(screen.getByText(strings.settings.voiceOutputSection.claudeHookPrefix, { exact: false })).toBeTruthy();
    });
  }

  it("renders auto-TTS toggle with correct initial state", async () => {
    await renderSection();
    expect(screen.getByTestId("auto-tts-toggle").getAttribute("aria-checked")).toBe("false");
  });

  it("clicking toggle updates store and calls updateTTSConfig", async () => {
    await renderSection();
    fireEvent.click(screen.getByTestId("auto-tts-toggle"));
    expect(mockStoreState.setAutoTtsEnabled).toHaveBeenCalledWith(true);
    await waitFor(() => {
      expect(mockUpdateTTSConfig).toHaveBeenCalledWith({ autoEnabled: true });
    });
  });

  it("changing backend preference persists to the API", async () => {
    await renderSection();
    fireEvent.change(screen.getByTestId("tts-backend-select"), { target: { value: "kokoro" } });
    expect(mockStoreState.setTtsBackendPreference).toHaveBeenCalledWith("kokoro");
    await waitFor(() => {
      expect(mockUpdateTTSConfig).toHaveBeenCalledWith({ backend: "kokoro" });
    });
  });

  it("changing kokoro speed persists to the API", async () => {
    mockStoreState.ttsBackendPreference = "kokoro";
    await renderSection();
    fireEvent.change(screen.getByTestId("kokoro-speed-slider"), { target: { value: "1.6" } });
    expect(mockStoreState.setKokoroSpeed).toHaveBeenCalledWith(1.6);
    await waitFor(() => {
      expect(mockUpdateTTSConfig).toHaveBeenCalledWith({ kokoroSpeed: 1.6 });
    });
  });

  it("renders structured hook diagnostics from TTS status", async () => {
    await renderSection();
    await waitFor(() => {
      expect(screen.getByText(strings.settings.voiceOutputSection.hookStatusCode, { exact: false })).toBeTruthy();
      expect(screen.getByText(strings.settings.voiceOutputSection.lastHookRouting, { exact: false })).toBeTruthy();
      expect(screen.getByText(strings.settings.voiceOutputSection.lastHookAck, { exact: false })).toBeTruthy();
      expect(screen.getByText(strings.settings.voiceOutputSection.lastTailerRouting, { exact: false })).toBeTruthy();
      expect(screen.getByText(strings.settings.voiceOutputSection.lastTailerAck, { exact: false })).toBeTruthy();
    });
  });
});
