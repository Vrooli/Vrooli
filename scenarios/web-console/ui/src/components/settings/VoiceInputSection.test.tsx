import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import VoiceInputSection from "./VoiceInputSection";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { renderWithProviders } from "../../test-utils";

const audioMocks = vi.hoisted(() => ({
  getWakeWordConfig: vi.fn(),
  getVoiceStreamConfig: vi.fn(),
  getSpeakerVerificationStatus: vi.fn(),
  updateVoiceStreamConfig: vi.fn(),
  fetchCapabilities: vi.fn(),
  probeWhisperHealth: vi.fn(),
}));

vi.mock("../../audio-integration", async (importActual) => ({
  ...(await importActual<typeof import("../../audio-integration")>()),
  getWakeWordConfig: audioMocks.getWakeWordConfig,
  getVoiceStreamConfig: audioMocks.getVoiceStreamConfig,
  getSpeakerVerificationStatus: audioMocks.getSpeakerVerificationStatus,
  updateVoiceStreamConfig: audioMocks.updateVoiceStreamConfig,
  probeWhisperHealth: audioMocks.probeWhisperHealth,
}));
vi.mock("../../api/capabilities", () => ({ fetchCapabilities: audioMocks.fetchCapabilities }));

describe("VoiceInputSection", () => {
  beforeEach(() => {
    vi.useRealTimers();
    useWorkspaceStore.setState({ voiceEnabled: false, vadAutoStop: true, voiceLanguage: "auto", voiceShortcut: "Control+Space" });
    audioMocks.getWakeWordConfig.mockResolvedValue({ configured: false, template: null });
    audioMocks.getVoiceStreamConfig.mockResolvedValue({
      flushIntervalMs: 500,
      minDeltaBytes: 4096,
      overlapBytes: 2048,
      persistentMode: false,
      wakeWordEnabled: false,
      wakeWordThreshold: 0.65,
      segmentSilenceMs: 1500,
      vadSilenceMs: 1500,
      overlapWindowMs: 0,
      overlapCommitRuns: 0,
    });
    audioMocks.getSpeakerVerificationStatus.mockResolvedValue({
      capability: "available",
      resourceReady: true,
      profileConfigured: false,
      profileExists: false,
      profileCount: 0,
      config: { enabled: false, profileIds: [], threshold: 0.35, mode: "filter", rejectBehavior: "drop", fallbackWithoutVerification: true },
      profiles: [],
    });
    audioMocks.updateVoiceStreamConfig.mockImplementation(async (patch) => ({
      flushIntervalMs: 500,
      minDeltaBytes: 4096,
      overlapBytes: 2048,
      persistentMode: false,
      wakeWordEnabled: false,
      wakeWordThreshold: 0.65,
      segmentSilenceMs: 1500,
      vadSilenceMs: 1500,
      overlapWindowMs: 0,
      overlapCommitRuns: 0,
      ...patch,
    }));
    audioMocks.fetchCapabilities.mockResolvedValue({ capabilities: [] });
    audioMocks.probeWhisperHealth.mockResolvedValue({ whisperHealthy: false, streamingAvailable: false });
  });

  it("renders the disabled voice settings and keeps browser-only work dormant", () => {
    renderWithProviders(<VoiceInputSection />);
    expect(screen.getByTestId("voice-enabled-toggle")).toHaveAttribute("aria-checked", "false");
  });

  it("hydrates enabled voice settings and exercises the durable controls", async () => {
    useWorkspaceStore.setState({ voiceEnabled: true });
    renderWithProviders(<VoiceInputSection />);

    await waitFor(() => expect(audioMocks.getVoiceStreamConfig).toHaveBeenCalled());
    expect(screen.getByTestId("persistent-mode-toggle")).toHaveAttribute("aria-checked", "false");
    fireEvent.click(screen.getByTestId("persistent-mode-toggle"));
    fireEvent.change(screen.getByTestId("voice-language-select"), { target: { value: "en-US" } });
    expect(useWorkspaceStore.getState().voiceLanguage).toBe("en-US");

    fireEvent.click(screen.getByTestId("wake-word-toggle"));
    expect(screen.getByText("settings.voiceInputSection.wakeWordRequireSaveFirst")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("advanced-streaming-toggle"));
    expect(screen.getByTestId("vs-flush-interval")).toBeInTheDocument();
    fireEvent.change(screen.getByTestId("vs-flush-interval"), { target: { value: "1000" } });
    await waitFor(() => expect(audioMocks.updateVoiceStreamConfig).toHaveBeenCalled());
  });

  it("records a new voice shortcut and reports unavailable microphone backends", async () => {
    useWorkspaceStore.setState({ voiceEnabled: true });
    renderWithProviders(<VoiceInputSection />);
    await waitFor(() => expect(audioMocks.probeWhisperHealth).toHaveBeenCalled());
    fireEvent.click(screen.getByTestId("voice-shortcut-change"));
    const recorder = screen.getByTestId("voice-shortcut-recording");
    fireEvent.keyDown(recorder, { key: "k", ctrlKey: true, code: "KeyK" });
    expect(useWorkspaceStore.getState().voiceShortcut).toBe("Ctrl+K");
    await waitFor(() => expect(screen.getByTestId("mic-test-detected-backend")).toHaveTextContent("settings.voiceInputSection.testMicBackendNone"));
  });
});
