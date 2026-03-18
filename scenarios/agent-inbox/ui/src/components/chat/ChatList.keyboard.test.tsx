/**
 * Tests for ChatList - Rename edge cases, keyboard navigation, and search
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

describe("ChatList - rename edge cases", () => {
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

  it("does not call onRenameChat if name unchanged", async () => {
    const user = userEvent.setup();
    render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

    const chatName = getFirstElement(screen.getAllByTestId("chat-name"));
    await user.dblClick(chatName);

    await user.keyboard("{Enter}");

    expect(defaultProps.onRenameChat).not.toHaveBeenCalled();
  });

  it("does not call onRenameChat if name is empty", async () => {
    const user = userEvent.setup();
    render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

    const chatName = getFirstElement(screen.getAllByTestId("chat-name"));
    await user.dblClick(chatName);

    const input = screen.getByTestId("inline-rename-input");
    await user.clear(input);
    await user.keyboard("{Enter}");

    expect(defaultProps.onRenameChat).not.toHaveBeenCalled();
  });

  it("trims whitespace from new name", async () => {
    const user = userEvent.setup();
    render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

    const chatName = getFirstElement(screen.getAllByTestId("chat-name"));
    await user.dblClick(chatName);

    const input = screen.getByTestId("inline-rename-input");
    await user.clear(input);
    await user.type(input, "  Spaced Name  {Enter}");

    expect(defaultProps.onRenameChat).toHaveBeenCalledWith("chat-1", "Spaced Name");
  });

  it("does not trigger onSelectChat when clicking inside edit input", async () => {
    const user = userEvent.setup();
    render(<ChatList {...defaultProps} />, { wrapper: createWrapper() });

    const chatName = getFirstElement(screen.getAllByTestId("chat-name"));
    await user.dblClick(chatName);

    vi.clearAllMocks();

    const input = screen.getByTestId("inline-rename-input");
    await user.click(input);

    expect(defaultProps.onSelectChat).not.toHaveBeenCalled();
  });

  it("disables double-click when onRenameChat is not provided", async () => {
    const user = userEvent.setup();
    render(<ChatList {...defaultProps} onRenameChat={undefined} />, { wrapper: createWrapper() });

    const chatName = getFirstElement(screen.getAllByTestId("chat-name"));
    await user.dblClick(chatName);

    expect(screen.queryByTestId("inline-rename-input")).not.toBeInTheDocument();
  });
});

describe("ChatList - keyboard navigation focus [KEY-001, KEY-002]", () => {
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

describe("ChatList - search functionality", () => {
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

    const clearButton = screen.getByTestId("clear-search-button");
    await user.click(clearButton);

    expect(searchInput).toHaveValue("");
  });
});
