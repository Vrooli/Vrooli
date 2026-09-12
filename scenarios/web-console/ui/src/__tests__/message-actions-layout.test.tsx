import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders as render } from "../test-utils";
import MessagesPane from "../components/MessagesPane";
import { createConversationSessionState, useConversationStore } from "../stores/useConversationStore";
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

const assistantEvent: ConversationEvent = {
  id: "assistant-1",
  sessionId: "session-1",
  sequence: 1,
  source: "claude_hook",
  role: "assistant",
  text: "Message body",
  speechParagraphs: ["Message body"],
  summarized: false,
  createdAt: "2026-08-28T00:00:00Z",
  deliveryState: "received",
  ttsState: "idle",
  consumptionState: "seen",
};

const props = {
  sessionId: "session-1",
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
  onSendToComposer: vi.fn(),
};

describe("message action layout", () => {
  beforeEach(() => {
    vi.stubGlobal("IntersectionObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), unobserve: vi.fn(), disconnect: vi.fn() })));
    vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), disconnect: vi.fn() })));
    globalThis.fetch = vi.fn() as typeof fetch;
    useConversationStore.setState({
      sessions: {
        "session-1": createConversationSessionState({
          events: [assistantEvent],
          cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
          hydrated: true,
        }),
      },
    });
  });

  it("caps the row at three inline controls, each the library's icon button with its comfortable tap target", () => {
    render(<MessagesPane {...props} />);

    const card = screen.getByTestId("msg-card-assistant-1");
    expect(card.querySelectorAll("[data-message-action-inline]")).toHaveLength(0);
    fireEvent.mouseEnter(card);
    const inline = card.querySelectorAll<HTMLElement>("[data-message-action-inline]");
    expect(inline).toHaveLength(3);
    // The library grows a comfortable target to the 44px tap floor on a
    // coarse pointer, beyond the icon's visual box.
    for (const control of inline) {
      expect(control).toHaveAttribute("data-rcl-icon-button");
      expect(control).toHaveAttribute("data-rcl-tap-target", "comfortable");
    }
  });

  it("renders inapplicable actions nowhere and every applicable action in ContextMenu/1", () => {
    render(<MessagesPane {...props} />);

    expect(screen.queryByTestId("msg-save-snippet-assistant-1")).toBeNull();
    expect(screen.queryByTestId("msg-handoff-assistant-1")).toBeNull();
    fireEvent.mouseEnter(screen.getByTestId("msg-card-assistant-1"));
    fireEvent.click(screen.getByTestId("msg-actions-more-assistant-1"));

    const menu = screen.getByTestId("msg-actions-menu-assistant-1");
    expect(menu.closest("[data-rcl-context-menu]")).not.toBeNull();
    for (const testId of [
      "msg-copy-assistant-1",
      "msg-speak-from-assistant-1",
      "msg-send-to-composer-assistant-1",
      "msg-render-toggle-assistant-1",
      "msg-assistant-1-mode-control",
    ]) {
      const action = screen.getByTestId(testId).closest("button");
      expect(action).not.toBeNull();
      expect(action?.closest("[data-rcl-context-menu-item-wrap]")).not.toBeNull();
    }
  });
});
