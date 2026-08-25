import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import AiSuggestBar from "./AiSuggestBar";
import GridSplitter from "./GridSplitter";
import IntegrationsSection from "./settings/IntegrationsSection";
import NewPaneDefaultsSection from "./settings/NewPaneDefaultsSection";
import { Button } from "./ui/button";
import { renderWithProviders } from "../test-utils";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useTtsPlaybackIntentStore } from "../domains/tts-playback/store";
import type { ConversationEvent } from "../api/conversation";
import type { SessionPlaybackControllerState } from "../domains/tts-playback/types";
import {
  buildPlaybackContext,
  buildPlaybackQueue,
  buildQueueLabel,
  resolvePlaybackParagraphs,
  shouldAutoPlayIncomingEvent,
  shouldShowPlaybackBar,
} from "../domains/tts-playback/utils";

const { generateAISuggestions } = vi.hoisted(() => ({ generateAISuggestions: vi.fn() }));
vi.mock("../api/ai", () => ({ generateAISuggestions }));
vi.mock("./IntegrationsPanel", () => ({ default: ({ open }: { open: boolean }) => <div data-testid="integrations-panel-mock">{String(open)}</div> }));

describe("small interactive UI surfaces", () => {
  beforeEach(() => {
    vi.useRealTimers();
    generateAISuggestions.mockReset();
    useWorkspaceStore.setState({
      defaultHeaderColor: "#000000",
      defaultThemeId: "dracula",
      defaultFontSize: 14,
      plusButtonBehavior: "launcher",
    });
  });

  it("renders AI empty, loading, result, empty-result, and error states", async () => {
    const execute = vi.fn();
    const { rerender } = renderWithProviders(<AiSuggestBar inputText="" onExecute={execute} onClose={vi.fn()} />);
    expect(screen.getByText("aiSuggestBar.empty")).toBeInTheDocument();

    generateAISuggestions.mockResolvedValueOnce({ commands: ["ls -la", "pwd"], provider: "ollama" });
    rerender(<AiSuggestBar inputText="list files" onExecute={execute} onClose={vi.fn()} />);
    await waitFor(() => expect(generateAISuggestions).toHaveBeenCalledWith("list files"));
    expect(screen.getByTestId("ai-suggest-chip-0")).toHaveTextContent("ls -la");
    fireEvent.click(screen.getByTestId("ai-suggest-chip-0"));
    expect(execute).toHaveBeenCalledWith("ls -la\n");

    generateAISuggestions.mockResolvedValueOnce({ commands: [], provider: "" });
    rerender(<AiSuggestBar inputText="no result" onExecute={execute} onClose={vi.fn()} />);
    await waitFor(() => expect(screen.getByText("aiSuggestBar.noSuggestions")).toBeInTheDocument());

    generateAISuggestions.mockRejectedValueOnce(new Error("provider unavailable"));
    rerender(<AiSuggestBar inputText="broken" onExecute={execute} onClose={vi.fn()} />);
    await waitFor(() => expect(screen.getByText(/provider unavailable/i)).toBeInTheDocument());
  });

  it("debounces AI requests and cancels pending work on unmount", async () => {
    vi.useFakeTimers();
    generateAISuggestions.mockResolvedValue({ commands: [], provider: "" });
    const { rerender, unmount } = renderWithProviders(<AiSuggestBar inputText="first" onExecute={vi.fn()} onClose={vi.fn()} />);
    rerender(<AiSuggestBar inputText="second" onExecute={vi.fn()} onClose={vi.fn()} />);
    await vi.advanceTimersByTimeAsync(599);
    expect(generateAISuggestions).not.toHaveBeenCalled();
    unmount();
    await vi.advanceTimersByTimeAsync(10);
    expect(generateAISuggestions).not.toHaveBeenCalled();
  });

  it("forwards splitter geometry and pointer events", () => {
    const onPointerDown = vi.fn();
    renderWithProviders(
      <GridSplitter axis="column" gridColumn="2" gridRow="1 / -1" onPointerDown={onPointerDown} label="Resize columns" />,
    );
    const splitter = screen.getByRole("button", { name: "Resize columns" });
    expect(splitter).toHaveStyle({ gridColumn: "2", gridRow: "1 / -1", cursor: "col-resize", minWidth: "8px" });
    fireEvent.pointerDown(splitter);
    expect(onPointerDown).toHaveBeenCalledOnce();
  });

  it("updates new-pane defaults through the settings controls", () => {
    renderWithProviders(<NewPaneDefaultsSection />);
    fireEvent.click(screen.getByTestId("plus-behavior-new-terminal"));
    expect(useWorkspaceStore.getState().plusButtonBehavior).toBe("new-terminal");
    fireEvent.click(screen.getByTestId("plus-behavior-launcher"));
    expect(useWorkspaceStore.getState().plusButtonBehavior).toBe("launcher");
    expect(screen.getByTestId("defaults-font-value")).toHaveValue("14");
  });

  it("renders shared button variants and the integrations settings seam", () => {
    renderWithProviders(
      <>
        <Button variant="outline" size="lg">Delete</Button>
        <IntegrationsSection open />
      </>,
    );
    expect(screen.getByRole("button", { name: /delete/i })).toHaveClass("h-12");
    expect(screen.getByTestId("integrations-panel-mock")).toHaveTextContent("true");
  });

  it("keeps playback selection and queue decisions deterministic", () => {
    const event: ConversationEvent = {
      id: "event-1", sessionId: "session-1", sequence: 4, role: "assistant", text: "hello",
      speechParagraphs: ["hello"], originalSpeechParagraphs: ["original hello"], summarized: true,
      source: "test", createdAt: "2026-08-24T00:00:00Z", deliveryState: "delivered", ttsState: "idle", consumptionState: "new",
    };
    const state: SessionPlaybackControllerState = {
      preferredVersion: "original" as const,
      selectedVersions: {},
      queueEntries: [{ eventId: "event-1", sequence: 4, version: "original" as const }],
      queueIndex: 0,
      replayTarget: null,
      activeTarget: null,
      queueSessionId: "session-1",
      summarizeLevel: "moderate",
      summarizingEventId: null,
      summarizeErrors: {},
      focusRequest: null,
    };
    expect(resolvePlaybackParagraphs(event, "original")).toEqual(["original hello"]);
    expect(buildPlaybackQueue("session-1", [event], {}, "active")).toHaveLength(1);
    expect(buildQueueLabel(state, event)).toBe("#4");
    const context = buildPlaybackContext({ "session-1": { events: [event] } }, state, { sessionId: "session-1", eventId: "event-1" }, "continuous");
    expect(context?.version).toBe("original");
    expect(shouldAutoPlayIncomingEvent({ autoTtsEnabled: true, playbackIntent: "continuous", activePaneId: "session-1", sessionId: "session-1", event, isSpeaking: false })).toBe(true);
    expect(shouldShowPlaybackBar({ autoTtsEnabled: true, activePaneId: "session-1", context, isSpeaking: false })).toBe(true);
    useTtsPlaybackIntentStore.getState().setPlaybackIntent("paused");
    useTtsPlaybackIntentStore.getState().setSelectedTarget({ sessionId: "session-1", eventId: "event-1" });
    expect(useTtsPlaybackIntentStore.getState().playbackIntent).toBe("paused");
    expect(useTtsPlaybackIntentStore.getState().selectedTarget?.eventId).toBe("event-1");
  });
});
