import { renderWithProviders as render, setMobileViewport } from "../test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { useConversationStore, createConversationSessionState } from "../stores/useConversationStore";
import { useMessagesViewStore } from "../stores/useMessagesViewStore";
import type { ConversationEvent } from "../api/conversation";

vi.mock("../hooks/useConversationSession", () => ({
  refreshConversationSession: vi.fn().mockResolvedValue({ ok: true, addedEvents: 0 }),
  loadOlderConversationPage: vi.fn().mockResolvedValue(false),
  loadConversationPageContaining: vi.fn().mockResolvedValue(false),
}));

vi.mock("../components/markdown", () => ({
  MarkdownRenderer: ({ content }: { content: string }) => <div>{content}</div>,
}));

vi.mock("../hooks/useHandoffSuggestions", () => ({
  useHandoffSuggestions: () => ({ forEvent: () => [], dismiss: vi.fn() }),
}));

function event(sequence: number): ConversationEvent {
  return {
    id: `e${String(sequence)}`, sessionId: "sess-1", sequence, source: "claude_hook", role: "assistant",
    text: `Message ${String(sequence)}`, speechParagraphs: [`Message ${String(sequence)}`], summarized: false,
    createdAt: new Date().toISOString(), deliveryState: "received", ttsState: "idle", consumptionState: "seen",
  };
}

const props = {
  sessionId: "sess-1",
  onPlayFromHere: vi.fn(),
  onPlayEvent: vi.fn(),
  activeSpeakingEventId: null,
  isTtsSpeaking: false,
  summarizeLevel: "moderate" as const,
  selectedVersionForEvent: vi.fn(() => "active" as const),
  summarizingEventId: null,
  getSummarizeError: vi.fn(() => null),
  onClearSummarizeError: vi.fn(),
  onToggleSummarized: vi.fn(),
  onChangeLevel: vi.fn(),
  playbackFocusRequest: null,
  toolbarTrailingAction: <button type="button" data-testid="workspace-toggle-view">toggle</button>,
};

describe("messages toolbar", () => {
  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), disconnect: vi.fn() })));
    globalThis.fetch = vi.fn() as typeof fetch;
    useMessagesViewStore.setState({ viewModes: {}, positions: {} });
    useConversationStore.setState({
      sessions: {
        "sess-1": {
          ...createConversationSessionState({ events: [event(41), event(42)], cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, hydrated: true }),
          totalCount: 1242,
        },
      },
    });
  });

  it("[REQ:P0-017c] holds exactly one search field and the view toggle", () => {
    render(<MessagesPane {...props} />);

    const strip = screen.getByTestId("messages-control-strip");
    expect(within(strip).getAllByTestId("messages-search-field")).toHaveLength(1);
    expect(within(strip).getByTestId("workspace-toggle-view")).toBeInTheDocument();
    expect(within(strip).getAllByRole("button")).toHaveLength(2);
    for (const gone of ["messages-nav-up", "messages-nav-down", "messages-refresh-btn", "msg-jump-trigger", "messages-search-btn"]) {
      expect(screen.queryByTestId(gone)).toBeNull();
    }
  });

  it("[REQ:P0-017c] names how many messages it searches", () => {
    render(<MessagesPane {...props} />);
    expect(screen.getByTestId("messages-search-field")).toHaveAttribute("data-count", "1242");
  });

  it("[REQ:P0-017c] opens the navigator with focus in its search box", async () => {
    render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("messages-search-field"));

    expect(screen.getByTestId("msg-jump-list")).toBeInTheDocument();
    await waitFor(() => { expect(document.activeElement).toBe(screen.getByTestId("msg-nav-search")); });
  });

  it("[REQ:P0-017c] Ctrl+K opens the navigator from anywhere in the pane", async () => {
    render(<MessagesPane {...props} />);
    fireEvent.keyDown(screen.getByTestId("messages-scroll"), { key: "k", ctrlKey: true });
    expect(screen.getByTestId("msg-jump-list")).toBeInTheDocument();
    await waitFor(() => { expect(document.activeElement).toBe(screen.getByTestId("msg-nav-search")); });
  });

  it("[REQ:P0-017c] on a phone, the search field opens the navigator with focus in the search box", async () => {
    setMobileViewport();
    render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("messages-search-field"));

    // The sheet's overlay surface applies its initial focus asynchronously
    // (component library), so focus lands shortly after the tap.
    await waitFor(() => { expect(document.activeElement).toBe(screen.getByTestId("msg-nav-search")); });
  });

  it("[REQ:P0-017c] Cmd+K opens the navigator on macOS", () => {
    render(<MessagesPane {...props} />);
    fireEvent.keyDown(screen.getByTestId("messages-scroll"), { key: "k", metaKey: true });
    expect(screen.getByTestId("msg-jump-list")).toBeInTheDocument();
  });
});
