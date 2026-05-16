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
  startMutedOnLoad: false,
  setStartMutedOnLoad: vi.fn(),
  kokoroVoice: "af_heart",
  setKokoroVoice: vi.fn(),
  kokoroSpeed: 1.0,
  setKokoroSpeed: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) => selector(mockStoreState),
}));

// Mock the new split-of-concerns sources:
//   - hook config / routing status / playback events → ../api/ttsHook
//     (web-console-internal REST against /api/v1/tts-hook/*).
//   - voice/speed/summarize knobs → @audio-tools/embed (calls audio-tools).
const mockUpdateHookConfig = vi.fn((patch?: Partial<{
  autoEnabled: boolean;
  backend: "auto" | "kokoro" | "browser";
  startMuted: boolean;
}>) => Promise.resolve({
  autoEnabled: patch?.autoEnabled ?? false,
  backend: patch?.backend ?? "auto",
  startMuted: patch?.startMuted ?? false,
}));

vi.mock("../api/ttsHook", () => ({
  getTTSHookStatus: vi.fn().mockResolvedValue({
    config: { autoEnabled: false, backend: "auto", startMuted: false },
    hookRegistered: false,
    hookCode: "hook_missing",
    hookReason: "Claude Stop hook is not registered in project settings",
    lastHookRouting: {
      appended: false,
      code: "tts_target_missing",
      reason: "No terminal session was available for TTS routing",
      source: "claude_hook",
    },
    lastTailerRouting: {
      appended: false,
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
    audioToolsCapabilityLabel: "resource is not responding",
  }),
  updateTTSHookConfig: vi.fn((patch: Parameters<typeof mockUpdateHookConfig>[0]) => mockUpdateHookConfig(patch)),
  recordTTSPlaybackEvent: vi.fn().mockResolvedValue(undefined),
  recordTTSHookAck: vi.fn().mockResolvedValue(undefined),
}));

const mockUpdateVoiceConfig = vi.fn((patch?: Partial<{
  defaultVoice: string;
  defaultSpeed: number;
}>) => Promise.resolve({
  defaultVoice: patch?.defaultVoice ?? "af_heart",
  defaultSpeed: patch?.defaultSpeed ?? 1.0,
  defaultResponseFormat: "mp3",
}));

vi.mock("@audio-tools/embed", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@audio-tools/embed")>();
  return {
    ...actual,
    getTTSConfig: vi.fn().mockResolvedValue({
      defaultVoice: "af_heart",
      defaultSpeed: 1.0,
      defaultResponseFormat: "mp3",
    }),
    updateTTSConfig: vi.fn((patch: Parameters<typeof mockUpdateVoiceConfig>[0]) => mockUpdateVoiceConfig(patch)),
    getTTSSummarizeConfig: vi.fn().mockResolvedValue({
      enabled: false,
      charThreshold: 500,
      level: "moderate",
      model: "qwen3:1.7b",
      timeoutSeconds: 30,
    }),
    updateTTSSummarizeConfig: vi.fn().mockResolvedValue({
      enabled: false,
      charThreshold: 500,
      level: "moderate",
      model: "qwen3:1.7b",
      timeoutSeconds: 30,
    }),
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
    mockStoreState.startMutedOnLoad = false;
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

  it("clicking toggle updates store and persists via tts-hook config endpoint", async () => {
    await renderSection();
    fireEvent.click(screen.getByTestId("auto-tts-toggle"));
    expect(mockStoreState.setAutoTtsEnabled).toHaveBeenCalledWith(true);
    await waitFor(() => {
      expect(mockUpdateHookConfig).toHaveBeenCalledWith({ autoEnabled: true });
    });
  });

  it("changing backend preference persists via tts-hook config endpoint", async () => {
    await renderSection();
    fireEvent.change(screen.getByTestId("tts-backend-select"), { target: { value: "kokoro" } });
    expect(mockStoreState.setTtsBackendPreference).toHaveBeenCalledWith("kokoro");
    await waitFor(() => {
      expect(mockUpdateHookConfig).toHaveBeenCalledWith({ backend: "kokoro" });
    });
  });

  it("changing kokoro speed persists to audio-tools via updateTTSConfig", async () => {
    mockStoreState.ttsBackendPreference = "kokoro";
    await renderSection();
    fireEvent.change(screen.getByTestId("kokoro-speed-slider"), { target: { value: "1.6" } });
    expect(mockStoreState.setKokoroSpeed).toHaveBeenCalledWith(1.6);
    await waitFor(() => {
      expect(mockUpdateVoiceConfig).toHaveBeenCalledWith({ defaultSpeed: 1.6 });
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
