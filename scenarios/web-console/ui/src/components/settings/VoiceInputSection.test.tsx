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
  updateSpeakerVerificationConfig: vi.fn(),
  clearSpeakerVerificationProfile: vi.fn(),
  deleteSpeakerVerificationProfile: vi.fn(),
  deleteWakeWordConfig: vi.fn(),
  fetchCapabilities: vi.fn(),
  probeWhisperHealth: vi.fn(),
}));

vi.mock("../../audio-integration", async (importActual) => ({
  ...(await importActual<typeof import("../../audio-integration")>()),
  getWakeWordConfig: audioMocks.getWakeWordConfig,
  getVoiceStreamConfig: audioMocks.getVoiceStreamConfig,
  getSpeakerVerificationStatus: audioMocks.getSpeakerVerificationStatus,
  updateVoiceStreamConfig: audioMocks.updateVoiceStreamConfig,
  updateSpeakerVerificationConfig: audioMocks.updateSpeakerVerificationConfig,
  clearSpeakerVerificationProfile: audioMocks.clearSpeakerVerificationProfile,
  deleteSpeakerVerificationProfile: audioMocks.deleteSpeakerVerificationProfile,
  deleteWakeWordConfig: audioMocks.deleteWakeWordConfig,
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
      config: { enabled: false, profileIds: ["profile-1"], threshold: 0.35, mode: "filter", rejectBehavior: "drop", fallbackWithoutVerification: true },
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
    audioMocks.updateSpeakerVerificationConfig.mockResolvedValue({ enabled: false, profileIds: ["profile-1"], threshold: 0.35, mode: "filter", rejectBehavior: "drop", fallbackWithoutVerification: true });
    audioMocks.clearSpeakerVerificationProfile.mockResolvedValue({ enabled: false, profileIds: [], threshold: 0.35, mode: "filter", rejectBehavior: "drop", fallbackWithoutVerification: true });
    audioMocks.deleteWakeWordConfig.mockResolvedValue({ configured: false, template: null });
    audioMocks.fetchCapabilities.mockResolvedValue({ capabilities: [] });
    audioMocks.probeWhisperHealth.mockResolvedValue({ whisperHealthy: false, streamingAvailable: false });
  });

  it("renders the disabled voice settings and keeps browser-only work dormant", () => {
    renderWithProviders(<VoiceInputSection />);
    expect(screen.getByTestId("voice-enabled-toggle")).toHaveAttribute("aria-checked", "false");
    fireEvent.click(screen.getByTestId("voice-enabled-toggle"));
    expect(useWorkspaceStore.getState().voiceEnabled).toBe(true);
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
    fireEvent.change(screen.getByTestId("vs-min-delta"), { target: { value: "8192" } });
    fireEvent.change(screen.getByTestId("vs-overlap"), { target: { value: "4096" } });
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

  it("exercises the durable voice, wake-word, speaker, capability, and reset controls", async () => {
    useWorkspaceStore.setState({ voiceEnabled: true });
    audioMocks.getWakeWordConfig.mockResolvedValue({ configured: true, template: null });
    renderWithProviders(<VoiceInputSection />);

    await waitFor(() => expect(screen.getByTestId("advanced-streaming-toggle")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("vad-auto-stop-toggle"));
    fireEvent.click(screen.getByTestId("vad-auto-stop-toggle"));
    fireEvent.change(screen.getByTestId("vad-silence-timeout-slider"), { target: { value: "900" } });
    fireEvent.click(screen.getByTestId("persistent-mode-toggle"));
    fireEvent.change(await screen.findByTestId("segment-silence-slider"), { target: { value: "1100" } });
    fireEvent.change(screen.getByTestId("wake-word-threshold-slider"), { target: { value: "0.75" } });
    fireEvent.change(screen.getByTestId("wake-word-label"), { target: { value: "Hey Console" } });
    fireEvent.click(screen.getByTestId("wake-word-delete"));
    fireEvent.click(screen.getByTestId("speaker-refresh"));
    fireEvent.change(screen.getByTestId("speaker-mode-select"), { target: { value: "advisory" } });
    fireEvent.change(screen.getByTestId("speaker-threshold-slider"), { target: { value: "0.6" } });
    fireEvent.click(screen.getByTestId("speaker-clear-profile"));
    fireEvent.click(screen.getByTestId("speaker-verification-toggle"));
    fireEvent.click(screen.getByTestId("advanced-streaming-toggle"));
    fireEvent.click(screen.getByTestId("vs-reset-defaults"));
    fireEvent.click(screen.getByTestId("voice-caps-refresh"));
    fireEvent.click(screen.getByTestId("mic-test-refresh-detection"));
    fireEvent.click(screen.getByTestId("mic-request-permission"));

    await waitFor(() => {
      expect(audioMocks.updateVoiceStreamConfig).toHaveBeenCalled();
      expect(audioMocks.updateSpeakerVerificationConfig).toHaveBeenCalled();
      expect(audioMocks.clearSpeakerVerificationProfile).toHaveBeenCalled();
      expect(audioMocks.deleteWakeWordConfig).toHaveBeenCalled();
      expect(audioMocks.fetchCapabilities).toHaveBeenCalled();
    });
  });

  it("removes an active speaker profile and routes the mic test refresh", async () => {
    useWorkspaceStore.setState({ voiceEnabled: true });
    renderWithProviders(<VoiceInputSection />);
    await waitFor(() => expect(screen.getByTestId("speaker-active-profiles")).toBeInTheDocument());
    fireEvent.click(screen.getByTitle(/removeProfileTitle/));
    await waitFor(() => expect(audioMocks.updateSpeakerVerificationConfig).toHaveBeenCalled());
  });
});
