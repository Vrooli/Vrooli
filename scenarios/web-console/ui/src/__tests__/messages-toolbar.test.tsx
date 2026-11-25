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

vi.mock("@vrooli/file-viewer", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@vrooli/file-viewer")>()),
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

  it("[REQ:P0-017c] holds exactly one search field and the shared pane actions", () => {
    render(<MessagesPane {...props} />);

    const strip = screen.getByTestId("messages-control-strip");
    expect(within(strip).getAllByTestId("messages-search-field")).toHaveLength(1);
    expect(within(strip).getByTestId("workspace-toggle-view")).toBeInTheDocument();
    expect(within(strip).getByTestId("messages-filter-toggle")).toBeInTheDocument();
    expect(within(strip).getByTestId("messages-select-toggle")).toBeInTheDocument();
    expect(within(strip).getAllByRole("button")).toHaveLength(3);
    for (const gone of ["messages-nav-up", "messages-nav-down", "messages-refresh-btn", "msg-jump-trigger", "messages-search-btn"]) {
      expect(screen.queryByTestId(gone)).toBeNull();
    }
  });

  it("[REQ:P0-017c] filters the messages list from the always-available filter popover", () => {
    useConversationStore.setState({
      sessions: {
        "sess-1": {
          ...createConversationSessionState({
            events: [event(41), { ...event(42), role: "user", text: "A user question" }],
            cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
            hydrated: true,
          }),
          totalCount: 2,
        },
      },
    });
    render(<MessagesPane {...props} />);
    expect(screen.getByTestId("msg-card-e41")).toBeInTheDocument();
    expect(screen.getByTestId("msg-card-e42")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("messages-filter-toggle"));
    fireEvent.click(screen.getByText("messageJumpList.filterUser"));
    expect(screen.getByTestId("msg-card-e42")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-card-e41")).toBeNull();
  });

  it("[REQ:P0-017c] shows loaded matches immediately while the full search runs", () => {
    render(<MessagesPane {...props} />);
    const field = screen.getByTestId("messages-search-field");
    fireEvent.focus(field);
    fireEvent.change(field, { target: { value: "Message 41" } });

    expect(screen.getByTestId("messages-search-result-e41")).toBeInTheDocument();
    expect(screen.getByTestId("messages-search-searching-all")).toBeInTheDocument();
  });

  it("[REQ:P0-017c] enters selection from the toolbar and exposes the export action", () => {
    render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("messages-select-toggle"));
    expect(screen.getByTestId("messages-selection-bar")).toBeInTheDocument();
    expect(screen.getByTestId("messages-select-export")).toBeDisabled();
    fireEvent.click(screen.getByTestId("msg-select-e41"));
    expect(screen.getByTestId("messages-select-export")).toBeEnabled();
    fireEvent.click(screen.getByTestId("messages-select-export"));
    expect(screen.getByTestId("msg-export-drawer")).toBeInTheDocument();
  });

  it("[REQ:P0-017c] names how many messages it searches", () => {
    render(<MessagesPane {...props} />);
    expect(screen.getByTestId("messages-search-field")).toHaveAttribute("data-count", "1242");
  });

  it("[REQ:P0-017c] opens a search results dropdown with focus in the field", async () => {
    render(<MessagesPane {...props} />);
    const field = screen.getByTestId("messages-search-field");
    field.focus();
    fireEvent.focus(field);

    expect(screen.getByTestId("messages-search-results")).toBeInTheDocument();
    await waitFor(() => { expect(document.activeElement).toBe(field); });
  });

  it("[REQ:P0-017c] Ctrl+K focuses the search field and opens results", async () => {
    render(<MessagesPane {...props} />);
    fireEvent.keyDown(screen.getByTestId("messages-scroll"), { key: "k", ctrlKey: true });
    expect(screen.getByTestId("messages-search-results")).toBeInTheDocument();
    await waitFor(() => { expect(document.activeElement).toBe(screen.getByTestId("messages-search-field")); });
  });

  it("[REQ:P0-017c] on a phone, the search field opens the same results dropdown", async () => {
    setMobileViewport();
    render(<MessagesPane {...props} />);
    const field = screen.getByTestId("messages-search-field");
    field.focus();
    fireEvent.focus(field);

    await waitFor(() => { expect(document.activeElement).toBe(field); });
    expect(screen.getByTestId("messages-search-results")).toBeInTheDocument();
  });

  it("[REQ:P0-017c] lists recent messages and opens the chosen one", () => {
    render(<MessagesPane {...props} />);
    const field = screen.getByTestId("messages-search-field");
    field.focus();
    fireEvent.focus(field);

    const result = screen.getByTestId("messages-search-result-e41");
    fireEvent.click(result);
    expect(screen.queryByTestId("messages-search-results")).toBeNull();
  });

  it("[REQ:P0-017c] Cmd+K focuses the search field on macOS", () => {
    render(<MessagesPane {...props} />);
    fireEvent.keyDown(screen.getByTestId("messages-scroll"), { key: "k", metaKey: true });
    expect(screen.getByTestId("messages-search-results")).toBeInTheDocument();
  });
});
