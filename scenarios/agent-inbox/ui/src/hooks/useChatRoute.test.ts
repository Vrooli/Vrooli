/**
 * Tests for useChatRoute hook
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useChatRoute, usePopStateListener } from "./useChatRoute";

// Mock window.history and location
const originalLocation = window.location;
const originalHistory = window.history;

describe("useChatRoute", () => {
  let mockPathname = "/";
  let historyStack: { chatId: string; path: string }[] = [];

  beforeEach(() => {
    mockPathname = "/";
    historyStack = [];

    // Mock location.pathname
    Object.defineProperty(window, "location", {
      value: {
        ...originalLocation,
        get pathname() {
          return mockPathname;
        },
      },
      writable: true,
    });

    // Mock history.pushState and replaceState
    vi.spyOn(window.history, "pushState").mockImplementation((state, _unused, url) => {
      mockPathname = url as string;
      historyStack.push({ chatId: (state as { chatId: string }).chatId, path: url as string });
    });

    vi.spyOn(window.history, "replaceState").mockImplementation((_state, _unused, url) => {
      mockPathname = url as string;
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    Object.defineProperty(window, "location", {
      value: originalLocation,
      writable: true,
    });
  });

  describe("getChatIdFromUrl", () => {
    it("returns empty string for root path", () => {
      mockPathname = "/";
      const { result } = renderHook(() => useChatRoute());
      expect(result.current.getChatIdFromUrl()).toBe("");
    });

    it("extracts chat ID from /chat/:id path", () => {
      mockPathname = "/chat/abc123";
      const { result } = renderHook(() => useChatRoute());
      expect(result.current.getChatIdFromUrl()).toBe("abc123");
    });

    it("handles UUIDs in path", () => {
      mockPathname = "/chat/550e8400-e29b-41d4-a716-446655440000";
      const { result } = renderHook(() => useChatRoute());
      expect(result.current.getChatIdFromUrl()).toBe("550e8400-e29b-41d4-a716-446655440000");
    });

    it("returns empty string for invalid paths", () => {
      mockPathname = "/settings";
      const { result } = renderHook(() => useChatRoute());
      expect(result.current.getChatIdFromUrl()).toBe("");
    });

    it("returns empty string for nested paths", () => {
      mockPathname = "/chat/abc123/messages";
      const { result } = renderHook(() => useChatRoute());
      expect(result.current.getChatIdFromUrl()).toBe("");
    });
  });

  describe("initialChatId", () => {
    it("captures chat ID from URL on initial render", () => {
      mockPathname = "/chat/initial-chat-id";
      const { result } = renderHook(() => useChatRoute());
      expect(result.current.initialChatId).toBe("initial-chat-id");
    });

    it("returns empty string when no chat in URL", () => {
      mockPathname = "/";
      const { result } = renderHook(() => useChatRoute());
      expect(result.current.initialChatId).toBe("");
    });
  });

  describe("setChatInUrl", () => {
    it("pushes /chat/:id when chat is selected", () => {
      mockPathname = "/";
      const { result } = renderHook(() => useChatRoute());

      act(() => {
        result.current.setChatInUrl("my-chat-id");
      });

      expect(window.history.pushState).toHaveBeenCalledWith(
        { chatId: "my-chat-id" },
        "",
        "/chat/my-chat-id"
      );
    });

    it("pushes / when chat is deselected", () => {
      mockPathname = "/chat/abc123";
      const { result } = renderHook(() => useChatRoute());

      act(() => {
        result.current.setChatInUrl("");
      });

      expect(window.history.pushState).toHaveBeenCalledWith({ chatId: "" }, "", "/");
    });

    it("does not push if path unchanged", () => {
      mockPathname = "/chat/abc123";
      const { result } = renderHook(() => useChatRoute());

      act(() => {
        result.current.setChatInUrl("abc123");
      });

      expect(window.history.pushState).not.toHaveBeenCalled();
    });
  });

  describe("replaceChatInUrl", () => {
    it("replaces URL without adding to history", () => {
      mockPathname = "/";
      const { result } = renderHook(() => useChatRoute());

      act(() => {
        result.current.replaceChatInUrl("replacement-chat");
      });

      expect(window.history.replaceState).toHaveBeenCalledWith(
        { chatId: "replacement-chat" },
        "",
        "/chat/replacement-chat"
      );
    });

    it("does not replace if path unchanged", () => {
      mockPathname = "/chat/abc123";
      const { result } = renderHook(() => useChatRoute());

      act(() => {
        result.current.replaceChatInUrl("abc123");
      });

      expect(window.history.replaceState).not.toHaveBeenCalled();
    });
  });
});

describe("usePopStateListener", () => {
  let mockPathname = "/";
  let popStateHandler: ((event: PopStateEvent) => void) | null = null;

  beforeEach(() => {
    mockPathname = "/";
    popStateHandler = null;

    Object.defineProperty(window, "location", {
      value: {
        ...originalLocation,
        get pathname() {
          return mockPathname;
        },
      },
      writable: true,
    });

    vi.spyOn(window, "addEventListener").mockImplementation((event, handler) => {
      if (event === "popstate") {
        popStateHandler = handler as (event: PopStateEvent) => void;
      }
    });

    vi.spyOn(window, "removeEventListener").mockImplementation((event) => {
      if (event === "popstate") {
        popStateHandler = null;
      }
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    Object.defineProperty(window, "location", {
      value: originalLocation,
      writable: true,
    });
  });

  it("registers popstate event listener", () => {
    const onNavigate = vi.fn();
    renderHook(() => usePopStateListener(onNavigate));

    expect(window.addEventListener).toHaveBeenCalledWith("popstate", expect.any(Function));
  });

  it("calls onNavigate with chat ID when popstate fires", () => {
    const onNavigate = vi.fn();
    renderHook(() => usePopStateListener(onNavigate));

    // Simulate navigating back to a chat page
    mockPathname = "/chat/navigated-chat";
    popStateHandler?.(new PopStateEvent("popstate"));

    expect(onNavigate).toHaveBeenCalledWith("navigated-chat");
  });

  it("calls onNavigate with empty string for root path", () => {
    const onNavigate = vi.fn();
    renderHook(() => usePopStateListener(onNavigate));

    mockPathname = "/";
    popStateHandler?.(new PopStateEvent("popstate"));

    expect(onNavigate).toHaveBeenCalledWith("");
  });

  it("removes listener on unmount", () => {
    const onNavigate = vi.fn();
    const { unmount } = renderHook(() => usePopStateListener(onNavigate));

    unmount();

    expect(window.removeEventListener).toHaveBeenCalledWith("popstate", expect.any(Function));
  });
});
