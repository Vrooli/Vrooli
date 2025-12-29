/**
 * @file ChatList.test.tsx
 * Tests for the ChatList component, including inline rename functionality.
 * [REQ:NAME-004]
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ChatList } from "./ChatList";
import type { Chat, Label } from "../../lib/api";

// Mock the searchChats API
vi.mock("../../lib/api", async (importOriginal) => {
  const mod = await importOriginal<typeof import("../../lib/api")>();
  return {
    ...mod,
    searchChats: vi.fn().mockResolvedValue([]),
  };
});

// Create a wrapper with QueryClient
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
  };
};

// Mock chat data
const mockChats: Chat[] = [
  {
    id: "chat-1",
    name: "Test Chat 1",
    preview: "Hello world",
    model: "claude-3-5-sonnet",
    view_mode: "bubble",
    is_read: true,
    is_archived: false,
    is_starred: false,
    label_ids: [],
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
    is_read: false,
    is_archived: false,
    is_starred: true,
    label_ids: ["label-1"],
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

describe("ChatList", () => {
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

    // The second argument is now optional (messageId from search)
    expect(defaultProps.onSelectChat).toHaveBeenCalledWith("chat-1", undefined);
  });

  describe("inline rename [NAME-004]", () => {
    it("shows tooltip hint for renaming on hover", () => {
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = screen.getAllByTestId("chat-name")[0];
      expect(chatName).toHaveAttribute("title", "Double-click to rename");
    });

    it("enters edit mode on double-click", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = screen.getAllByTestId("chat-name")[0];
      await user.dblClick(chatName);

      expect(screen.getByTestId("inline-rename-input")).toBeInTheDocument();
    });

    it("pre-fills input with current chat name", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = screen.getAllByTestId("chat-name")[0];
      await user.dblClick(chatName);

      const input = screen.getByTestId("inline-rename-input") as HTMLInputElement;
      expect(input.value).toBe("Test Chat 1");
    });

    it("calls onRenameChat when saving with Enter", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = screen.getAllByTestId("chat-name")[0];
      await user.dblClick(chatName);

      const input = screen.getByTestId("inline-rename-input");
      await user.clear(input);
      await user.type(input, "Renamed Chat{Enter}");

      expect(defaultProps.onRenameChat).toHaveBeenCalledWith("chat-1", "Renamed Chat");
    });

    it("calls onRenameChat when clicking save button", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = screen.getAllByTestId("chat-name")[0];
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

      const chatName = screen.getAllByTestId("chat-name")[0];
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

      const chatName = screen.getAllByTestId("chat-name")[0];
      await user.dblClick(chatName);

      await user.click(screen.getByTestId("inline-rename-cancel"));

      expect(defaultProps.onRenameChat).not.toHaveBeenCalled();
      expect(screen.queryByTestId("inline-rename-input")).not.toBeInTheDocument();
    });

    it("does not call onRenameChat if name unchanged", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = screen.getAllByTestId("chat-name")[0];
      await user.dblClick(chatName);

      // Press Enter without changing the name
      await user.keyboard("{Enter}");

      expect(defaultProps.onRenameChat).not.toHaveBeenCalled();
    });

    it("does not call onRenameChat if name is empty", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = screen.getAllByTestId("chat-name")[0];
      await user.dblClick(chatName);

      const input = screen.getByTestId("inline-rename-input");
      await user.clear(input);
      await user.keyboard("{Enter}");

      expect(defaultProps.onRenameChat).not.toHaveBeenCalled();
    });

    it("trims whitespace from new name", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = screen.getAllByTestId("chat-name")[0];
      await user.dblClick(chatName);

      const input = screen.getByTestId("inline-rename-input");
      await user.clear(input);
      await user.type(input, "  Spaced Name  {Enter}");

      expect(defaultProps.onRenameChat).toHaveBeenCalledWith("chat-1", "Spaced Name");
    });

    it("does not trigger onSelectChat when clicking inside edit input", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatName = screen.getAllByTestId("chat-name")[0];
      await user.dblClick(chatName);

      // Clear the call count after entering edit mode
      vi.clearAllMocks();

      // Click inside the editing area - should not select the chat
      const input = screen.getByTestId("inline-rename-input");
      await user.click(input);

      // Additional clicks inside the edit area should not trigger selection
      expect(defaultProps.onSelectChat).not.toHaveBeenCalled();
    });

    it("disables double-click when onRenameChat is not provided", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} onRenameChat={undefined} />, { wrapper: createWrapper() });

      const chatName = screen.getAllByTestId("chat-name")[0];
      await user.dblClick(chatName);

      // Should not enter edit mode
      expect(screen.queryByTestId("inline-rename-input")).not.toBeInTheDocument();
    });
  });

  describe("keyboard navigation focus [KEY-001, KEY-002]", () => {
    it("shows focus ring when focusedIndex matches item", () => {
      render(<ChatList {...defaultProps} focusedIndex={0} />, { wrapper: createWrapper() });

      const chatItem = screen.getByTestId("chat-item-chat-1");
      expect(chatItem).toHaveAttribute("data-focused", "true");
      expect(chatItem).toHaveClass("ring-2");
    });

    it("does not show focus ring when focusedIndex is -1", () => {
      render(<ChatList {...defaultProps} focusedIndex={-1} />, { wrapper: createWrapper() });

      const chatItem = screen.getByTestId("chat-item-chat-1");
      expect(chatItem).toHaveAttribute("data-focused", "false");
      expect(chatItem).not.toHaveClass("ring-2");
    });

    it("shows focus ring only on the correct item", () => {
      render(<ChatList {...defaultProps} focusedIndex={1} />, { wrapper: createWrapper() });

      const chatItem1 = screen.getByTestId("chat-item-chat-1");
      const chatItem2 = screen.getByTestId("chat-item-chat-2");

      expect(chatItem1).toHaveAttribute("data-focused", "false");
      expect(chatItem2).toHaveAttribute("data-focused", "true");
    });

    it("defaults focusedIndex to -1 when not provided", () => {
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const chatItem1 = screen.getByTestId("chat-item-chat-1");
      const chatItem2 = screen.getByTestId("chat-item-chat-2");

      expect(chatItem1).not.toHaveClass("ring-2");
      expect(chatItem2).not.toHaveClass("ring-2");
    });
  });

  describe("search functionality", () => {
    // NOTE: The search now uses server-side full-text search via useSearch hook.
    // The local filtering has been replaced. These tests verify basic search UI interaction.

    it("renders search input", () => {
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const searchInput = screen.getByTestId("chat-search-input");
      expect(searchInput).toBeInTheDocument();
      expect(searchInput).toHaveAttribute("placeholder", "Search messages... (Ctrl+K)");
    });

    it("displays chats when no search query", () => {
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      expect(screen.getByText("Test Chat 1")).toBeInTheDocument();
      expect(screen.getByText("Test Chat 2")).toBeInTheDocument();
    });

    it("clears search when clicking X button", async () => {
      const user = userEvent.setup();
      render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

      const searchInput = screen.getByTestId("chat-search-input");
      await user.type(searchInput, "test");

      // Clear button should appear
      const clearButton = screen.getByTestId("clear-search-button");
      await user.click(clearButton);

      // Search input should be cleared
      expect(searchInput).toHaveValue("");
    });
  });
});
