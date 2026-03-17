import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";

// Mock workspace store with direct access (same pattern as existing settings-modal tests)
const mockStoreState: Record<string, unknown> = {
  settingsModalOpen: true,
  setSettingsModalOpen: vi.fn(),
  isMinimapVisible: true,
  setMinimapVisible: vi.fn(),
  displayMode: "grid",
  setDisplayMode: vi.fn(),
  defaultHeaderColor: "transparent",
  defaultThemeId: "slate-ocean",
  defaultFontSize: 14,
  setDefaultHeaderColor: vi.fn(),
  setDefaultThemeId: vi.fn(),
  setDefaultFontSize: vi.fn(),
  voiceEnabled: true,
  setVoiceEnabled: vi.fn(),
  voiceShortcut: "Ctrl+Shift+Space",
  setVoiceShortcut: vi.fn(),
  vadAutoStop: true,
  setVadAutoStop: vi.fn(),
  vadSilenceTimeoutMs: 2000,
  setVadSilenceTimeoutMs: vi.fn(),
  voiceLanguage: "en-US",
  setVoiceLanguage: vi.fn(),
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
  toolbarLayout: "expanded",
  setToolbarLayout: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector(mockStoreState),
}));

// Mock draggable position hook
vi.mock("../hooks/useDraggablePosition", () => ({
  useDraggablePosition: () => ({
    elementRef: { current: null },
    floatingStyle: { transform: "translate3d(100px, 100px, 0)" },
    pointerHandlers: {
      onPointerDown: vi.fn(),
      onPointerMove: vi.fn(),
      onPointerUp: vi.fn(),
      onPointerCancel: vi.fn(),
    },
    handleClickCapture: vi.fn(),
    resetPosition: vi.fn(),
    moveTo: vi.fn(),
    isDragging: false,
    position: { x: 100, y: 100 },
  }),
}));

// Mock API module
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

vi.mock("../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api")>();
  return {
    ...actual,
    listShortcutProfiles: vi.fn().mockResolvedValue([]),
    upsertShortcutProfile: vi.fn(),
    deleteShortcutProfile: vi.fn(),
    fetchCapabilities: vi.fn().mockResolvedValue({ capabilities: [] }),
    getVoiceStreamConfig: vi.fn().mockResolvedValue({ enabled: false }),
    updateVoiceStreamConfig: vi.fn(),
    getTTSStatus: vi.fn().mockResolvedValue({
      config: {
        autoEnabled: false,
        backend: "auto",
        kokoroVoice: "af_heart",
        kokoroSpeed: 1.0,
      },
      hookRegistered: false,
      hookReason: "Claude Stop hook is not registered in project settings",
      kokoroCapability: "unavailable",
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
    speak: vi.fn(),
    speakParagraphs: vi.fn(),
    stop: vi.fn(),
    refresh: vi.fn(),
    testSpeak: vi.fn(),
  }),
}));

vi.mock("../components/IntegrationsPanel", () => ({
  default: () => <div data-testid="integrations-panel">IntegrationsPanel</div>,
}));

// Mock window.speechSynthesis for the TTS section to render
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

import SettingsModal from "../components/SettingsModal";

describe("SettingsModal auto-TTS toggle", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStoreState.settingsModalOpen = true;
    mockStoreState.autoTtsEnabled = false;
    mockStoreState.ttsBackendPreference = "auto";
    mockStoreState.setAutoTtsEnabled = vi.fn();
  });

  it("renders auto-TTS toggle with correct initial state (off)", () => {
    render(<SettingsModal />);
    const toggle = screen.getByTestId("auto-tts-toggle");
    expect(toggle).toBeTruthy();
    expect(toggle.getAttribute("aria-checked")).toBe("false");
  });

  it("renders auto-TTS toggle with correct initial state (on)", () => {
    mockStoreState.autoTtsEnabled = true;
    render(<SettingsModal />);
    const toggle = screen.getByTestId("auto-tts-toggle");
    expect(toggle.getAttribute("aria-checked")).toBe("true");
  });

  it("clicking toggle updates store and calls updateTTSConfig API", async () => {
    render(<SettingsModal />);
    const toggle = screen.getByTestId("auto-tts-toggle");

    fireEvent.click(toggle);

    // Store should be updated to the opposite value
    expect(mockStoreState.setAutoTtsEnabled).toHaveBeenCalledWith(true);

    // API should be called with new value
    await waitFor(() => {
      expect(mockUpdateTTSConfig).toHaveBeenCalledWith({ autoEnabled: true });
    });
  });

  it("clicking toggle when enabled sets autoEnabled to false", async () => {
    mockStoreState.autoTtsEnabled = true;
    render(<SettingsModal />);
    const toggle = screen.getByTestId("auto-tts-toggle");

    fireEvent.click(toggle);

    expect(mockStoreState.setAutoTtsEnabled).toHaveBeenCalledWith(false);

    await waitFor(() => {
      expect(mockUpdateTTSConfig).toHaveBeenCalledWith({ autoEnabled: false });
    });
  });

  it("toggle has switch role for accessibility", () => {
    render(<SettingsModal />);
    const toggle = screen.getByTestId("auto-tts-toggle");
    expect(toggle.getAttribute("role")).toBe("switch");
  });

  it("changing backend preference persists to the API", async () => {
    render(<SettingsModal />);
    const select = screen.getByTestId("tts-backend-select");

    fireEvent.change(select, { target: { value: "kokoro" } });

    expect(mockStoreState.setTtsBackendPreference).toHaveBeenCalledWith("kokoro");
    await waitFor(() => {
      expect(mockUpdateTTSConfig).toHaveBeenCalledWith({ backend: "kokoro" });
    });
  });

  it("changing kokoro speed persists to the API", async () => {
    mockStoreState.ttsBackendPreference = "kokoro";
    render(<SettingsModal />);
    const slider = screen.getByTestId("kokoro-speed-slider");

    fireEvent.change(slider, { target: { value: "1.6" } });

    expect(mockStoreState.setKokoroSpeed).toHaveBeenCalledWith(1.6);
    await waitFor(() => {
      expect(mockUpdateTTSConfig).toHaveBeenCalledWith({ kokoroSpeed: 1.6 });
    });
  });
});
