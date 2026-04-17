/**
 * Tests for ChatList - Basic rendering, selection, and inline rename
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ChatList } from "./ChatList";
import type { Chat, Label } from "../../lib/api";

function getFirstElement(elements: HTMLElement[]): HTMLElement {
  const first = elements[0];
  if (!first) throw new Error("Expected at least one element");
  return first;
}

vi.mock("../../lib/api", async (importOriginal) => {
  const mod = await importOriginal<typeof import("../../lib/api")>();
  return {
    ...mod,
    searchChats: vi.fn().mockResolvedValue([]),
  };
});

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
  };
};

const mockChats: Chat[] = [
  {
    id: "chat-1",
    name: "Test Chat 1",
    preview: "Hello world",
    model: "claude-3-5-sonnet",
    view_mode: "bubble",
    chat_mode: "llm",
    is_read: true,
    is_archived: false,
    is_starred: false,
    label_ids: [],
    tools_enabled: true,
    web_search_enabled: false,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "chat-2",
    name: "Test Chat 2",
    preview: "Another message",
    model: "claude-3-5-sonnet",
    view_mode: "bubble",
    chat_mode: "llm",
    is_read: false,
    is_archived: false,
    is_starred: true,
    label_ids: ["label-1"],
    tools_enabled: true,
    web_search_enabled: false,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
];

const mockLabels: Label[] = [
  {
    id: "label-1",
    name: "Important",
    color: "#ff0000",
    created_at: new Date().toISOString(),
  },
];

describe("ChatList - basic rendering and selection", () => {
  const defaultProps = {
    chats: mockChats,
    labels: mockLabels,
    selectedChatId: null,
    currentView: "inbox" as const,
    isLoading: false,
    onSelectChat: vi.fn(),
    onNewChat: vi.fn(),
    onRenameChat: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders chat list items", () => {
    render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

    expect(screen.getByText("Test Chat 1")).toBeInTheDocument();
    expect(screen.getByText("Test Chat 2")).toBeInTheDocument();
  });

  it("displays labels on chat items", () => {
    render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

    expect(screen.getByText("Important")).toBeInTheDocument();
  });

  it("shows unread indicator for unread chats", () => {
    render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

    expect(screen.getByTestId("unread-indicator")).toBeInTheDocument();
  });

  it("calls onSelectChat when clicking a chat item", async () => {
    const user = userEvent.setup();
    render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

    await user.click(screen.getByTestId("chat-item-chat-1"));

    expect(defaultProps.onSelectChat).toHaveBeenCalledWith("chat-1", undefined);
  });

  describe("inline rename [NAME-004]", () => {
    it("shows tooltip hint for renaming on hover", () => {
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = getFirstElement(screen.getAllByTestId("chat-name"));
      expect(chatName).toHaveAttribute("title", "Double-click to rename");
    });

    it("enters edit mode on double-click", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = getFirstElement(screen.getAllByTestId("chat-name"));
      await user.dblClick(chatName);

      expect(screen.getByTestId("inline-rename-input")).toBeInTheDocument();
    });

    it("pre-fills input with current chat name", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = getFirstElement(screen.getAllByTestId("chat-name"));
      await user.dblClick(chatName);

      const input = screen.getByTestId("inline-rename-input");
      expect((input as HTMLInputElement).value).toBe("Test Chat 1");
    });

    it("calls onRenameChat when saving with Enter", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = getFirstElement(screen.getAllByTestId("chat-name"));
      await user.dblClick(chatName);

      const input = screen.getByTestId("inline-rename-input");
      await user.clear(input);
      await user.type(input, "Renamed Chat{Enter}");

      expect(defaultProps.onRenameChat).toHaveBeenCalledWith("chat-1", "Renamed Chat");
    });

    it("calls onRenameChat when clicking save button", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = getFirstElement(screen.getAllByTestId("chat-name"));
      await user.dblClick(chatName);

      const input = screen.getByTestId("inline-rename-input");
      await user.clear(input);
      await user.type(input, "Another Name");

      await user.click(screen.getByTestId("inline-rename-save"));

      expect(defaultProps.onRenameChat).toHaveBeenCalledWith("chat-1", "Another Name");
    });

    it("cancels rename on Escape key", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = getFirstElement(screen.getAllByTestId("chat-name"));
      await user.dblClick(chatName);

      const input = screen.getByTestId("inline-rename-input");
      await user.clear(input);
      await user.type(input, "Changed Name{Escape}");

      expect(defaultProps.onRenameChat).not.toHaveBeenCalled();
      expect(screen.queryByTestId("inline-rename-input")).not.toBeInTheDocument();
    });

    it("cancels rename when clicking cancel button", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = getFirstElement(screen.getAllByTestId("chat-name"));
      await user.dblClick(chatName);

      await user.click(screen.getByTestId("inline-rename-cancel"));

      expect(defaultProps.onRenameChat).not.toHaveBeenCalled();
      expect(screen.queryByTestId("inline-rename-input")).not.toBeInTheDocument();
    });
  });
});
