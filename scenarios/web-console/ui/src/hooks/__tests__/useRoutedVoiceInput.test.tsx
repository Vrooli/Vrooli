import { act, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { useRoutedVoiceInput } from "../useRoutedVoiceInput";

const mocks = vi.hoisted(() => ({
  transcript: null as null | ((text: string) => void),
  dictationStart: vi.fn(() => Promise.resolve()),
  dictationStop: vi.fn(), cancel: vi.fn(), passive: vi.fn(), prepare: vi.fn(),
  nativeStart: vi.fn(() => Promise.resolve()), nativeStop: vi.fn(), selected: false, available: true,
}));
vi.mock("../../audio-integration", () => ({ useScenarioVoiceInput: (callback: (text: string) => void) => {
  mocks.transcript = callback;
  return { supported: true, backend: "whisper", voiceState: "idle", startRecording: mocks.dictationStart, stopRecording: mocks.dictationStop, cancelTranscription: mocks.cancel, exitPassiveMode: mocks.passive, prepareRecording: mocks.prepare };
} }));
vi.mock("../useAgentVoice", async (importOriginal) => ({ ...await importOriginal<typeof import("../useAgentVoice")>(), useAgentVoice: () => ({ selected: mocks.selected, capability: { available: mocks.available, nativeActive: mocks.selected, agent: "codex" }, voiceState: "idle", error: null, audioLevel: 0, partialTranscript: "", start: mocks.nativeStart, stop: mocks.nativeStop }) }));
function Harness({ onTranscript }: { onTranscript: (text: string) => void }) {
  const voice = useRoutedVoiceInput(onTranscript);
  return <><button onClick={() => { void voice.startRecording(); }}>Start</button><button onClick={() => { void voice.startRecordingWithMode("agent"); }}>Choose agent</button><output data-testid="recovery">{String(voice.needsRouteSelection)}</output><output data-testid="route">{voice.backend}</output></>;
}
beforeEach(() => { vi.clearAllMocks(); mocks.selected = false; mocks.available = true; useWorkspaceStore.setState({ activePane: "first", agentVoiceMode: "auto", persistentMode: false }); });
describe("shared microphone routing", () => {
  it("uses dictation for text input when native voice is inactive", () => {
    const transcript = vi.fn(); renderWithProviders(<Harness onTranscript={transcript} />);
    act(() => { screen.getByText("Start").click(); });
    act(() => { mocks.transcript?.("Review this draft"); });
    expect(mocks.dictationStart).toHaveBeenCalledOnce(); expect(mocks.nativeStart).not.toHaveBeenCalled(); expect(transcript).toHaveBeenCalledWith("Review this draft");
  });
  it("routes active native voice to its adapter and releases passive dictation", () => {
    mocks.selected = true; renderWithProviders(<Harness onTranscript={vi.fn()} />);
    act(() => { screen.getByText("Start").click(); });
    expect(mocks.nativeStart).toHaveBeenCalledOnce(); expect(mocks.dictationStart).not.toHaveBeenCalled(); expect(mocks.passive).toHaveBeenCalled(); expect(screen.getByTestId("route")).toHaveTextContent("agent");
  });
  it("starts an explicit first-use selection immediately before the stored route rerenders", () => {
    renderWithProviders(<Harness onTranscript={vi.fn()} />);
    act(() => { screen.getByText("Choose agent").click(); });
    expect(mocks.nativeStart).toHaveBeenCalledOnce();
    expect(mocks.dictationStart).not.toHaveBeenCalled();
  });
  it("keeps an unavailable saved native route recoverable instead of a dead microphone", () => {
    mocks.selected = true; mocks.available = false;
    useWorkspaceStore.setState({ agentVoiceMode: "agent" });
    renderWithProviders(<Harness onTranscript={vi.fn()} />);
    expect(screen.getByTestId("recovery")).toHaveTextContent("true");
    expect(mocks.nativeStart).not.toHaveBeenCalled();
  });
  it("cancels on pane change and rejects a late transcript for the old pane", () => {
    const transcript = vi.fn(); renderWithProviders(<Harness onTranscript={transcript} />);
    act(() => { screen.getByText("Start").click(); });
    act(() => { useWorkspaceStore.setState({ activePane: "second" }); });
    act(() => { mocks.transcript?.("Old recording"); });
    expect(mocks.cancel).toHaveBeenCalled(); expect(transcript).not.toHaveBeenCalled();
  });
});
