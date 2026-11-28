import { act, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { useAgentVoiceStore } from "../../stores/useAgentVoiceStore";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { usePaneSpeech } from "./usePaneSpeech";

const speech = vi.hoisted(() => ({ stop: vi.fn(), speak: vi.fn(() => Promise.resolve(undefined)), prewarm: vi.fn(() => Promise.resolve()) }));
vi.mock("../useTextToSpeech", () => ({ useTextToSpeech: () => ({
  stop: speech.stop, speakParagraphs: speech.speak, prewarmParagraphs: speech.prewarm,
  pause: vi.fn(), resume: vi.fn(), seek: vi.fn(), setPlaybackRate: vi.fn(), setVolume: vi.fn(), setMuted: vi.fn(),
  supported: true, capabilities: {}, backendReason: "test", currentTime: 0, duration: 0,
  isSpeaking: false, needsUnlock: false, isPaused: false, playbackRate: 1, volume: 1, isMuted: false,
}) }));
vi.mock("../useConversationSession", () => ({ useConversationSession: () => ({ persistCursor: vi.fn() }) }));
vi.mock("../../domains/tts-playback/transport", () => ({ publishTransport: vi.fn() }));
function Harness() {
  const { playback } = usePaneSpeech({ sessionId: "native", sendConversationAck: vi.fn() });
  return <button onClick={() => { void playback.speak("An automatic reply", undefined, { initiatedBy: "auto" }); }}>Automatic speech</button>;
}
beforeEach(() => { vi.clearAllMocks(); useAgentVoiceStore.setState({ connected: {} }); useWorkspaceStore.setState({ activePane: "native" }); });
describe("native voice speaker ownership", () => {
  it("stops existing TTS and rejects delayed automatic speech while native voice is connected", async () => {
    renderWithProviders(<Harness />);
    await act(() => { screen.getByText("Automatic speech").click(); return Promise.resolve(); });
    expect(speech.speak).toHaveBeenCalledOnce();
    speech.stop.mockClear();
    act(() => { useAgentVoiceStore.getState().setConnected("native", true); });
    expect(speech.stop).toHaveBeenCalled();
    await act(() => { screen.getByText("Automatic speech").click(); return Promise.resolve(); });
    expect(speech.speak).toHaveBeenCalledOnce();
    act(() => { useAgentVoiceStore.getState().setConnected("native", false); });
    await act(() => { screen.getByText("Automatic speech").click(); return Promise.resolve(); });
    expect(speech.speak).toHaveBeenCalledTimes(2);
  });
});
