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
//   - voice/speed/summarize knobs → ../audio-integration (calls audio-tools).
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

const mockUpdateSummarizeConfig = vi.fn((patch?: Partial<{
  enabled: boolean;
  charThreshold: number;
  level: "light" | "moderate" | "heavy";
  model: string;
  timeoutSeconds: number;
}>) => Promise.resolve({
  enabled: patch?.enabled ?? false,
  charThreshold: patch?.charThreshold ?? 500,
  level: patch?.level ?? "moderate",
  model: patch?.model ?? "llama3.2:3b",
  timeoutSeconds: patch?.timeoutSeconds ?? 120,
}));

vi.mock("../audio-integration", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../audio-integration")>();
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
      model: "llama3.2:3b",
      timeoutSeconds: 120,
    }),
    updateTTSSummarizeConfig: vi.fn((patch: Parameters<typeof mockUpdateSummarizeConfig>[0]) => mockUpdateSummarizeConfig(patch)),
    listTTSSummarizeModels: vi.fn().mockResolvedValue([
      {
        id: "llama3.2:3b",
        displayName: "Llama 3.2 3B",
        installed: true,
        recommended: true,
        defaultEligible: true,
        reasoning: false,
        statusLabel: "Installed, recommended",
        pullCommand: "ollama pull llama3.2:3b",
        sizeBytes: 1n,
        parameterSize: "3B",
        sourceUrl: "https://ollama.com/library/llama3.2",
        notes: "Installed fallback validated for fast local TTS summarization.",
      },
      {
        id: "gemma3:4b",
        displayName: "Gemma 3 4B",
        installed: false,
        recommended: true,
        defaultEligible: true,
        reasoning: false,
        statusLabel: "Recommended, not installed",
        pullCommand: "ollama pull gemma3:4b",
        sizeBytes: 0n,
        parameterSize: "4B",
        sourceUrl: "https://ollama.com/library/gemma3",
        notes: "Benchmark locally before making it the default.",
      },
      {
        id: "qwen3:4b",
        displayName: "Qwen3 4B",
        installed: true,
        recommended: false,
        defaultEligible: false,
        reasoning: true,
        statusLabel: "Installed reasoning model",
        pullCommand: "ollama pull qwen3:4b",
        sizeBytes: 2n,
        parameterSize: "4B",
        sourceUrl: "https://ollama.com/library/qwen3:4b",
        notes: "Reasoning-capable; too slow/noisy for default TTS summaries.",
      },
    ]),
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
    mockUpdateSummarizeConfig.mockClear();
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

  it("renders summarize model catalog status", async () => {
    await renderSection();
    const select = screen.getByTestId("summarize-model-select") as HTMLSelectElement;
    expect(select.value).toBe("llama3.2:3b");
    expect(screen.getByText("Installed, recommended")).toBeTruthy();
  });

  it("shows a pull command for missing recommended summarize models", async () => {
    await renderSection();
    fireEvent.change(screen.getByTestId("summarize-model-select"), { target: { value: "gemma3:4b" } });
    await waitFor(() => expect(mockUpdateSummarizeConfig).toHaveBeenCalledWith({ model: "gemma3:4b" }));
    expect(screen.getByTestId("summarize-model-pull-command").textContent).toContain("ollama pull gemma3:4b");
  });

  it("warns when a reasoning summarize model is selected", async () => {
    await renderSection();
    fireEvent.change(screen.getByTestId("summarize-model-select"), { target: { value: "qwen3:4b" } });
    await waitFor(() => expect(mockUpdateSummarizeConfig).toHaveBeenCalledWith({ model: "qwen3:4b" }));
    expect(screen.getByTestId("summarize-model-reasoning-warning").textContent).toContain("Reasoning models are slower");
  });
});
