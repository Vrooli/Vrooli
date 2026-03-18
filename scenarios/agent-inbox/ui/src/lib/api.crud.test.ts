/**
 * Tests for API client - CRUD operations and URL resolution
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  resolveAttachmentUrl,
  fetchChats,
  fetchChat,
  createChat,
  deleteChat,
  addMessage,
  type Chat,
  type Message,
} from "./api";
import { mockFetch, createMockResponse } from "./api.test.helpers";

describe("resolveAttachmentUrl", () => {
  it("returns undefined for undefined input", () => {
    expect(resolveAttachmentUrl(undefined)).toBeUndefined();
  });

  it("returns undefined for empty string", () => {
    expect(resolveAttachmentUrl("")).toBeUndefined();
  });

  it("returns http URLs as-is", () => {
    const url = "http://example.com/image.png";
    expect(resolveAttachmentUrl(url)).toBe(url);
  });

  it("returns https URLs as-is", () => {
    const url = "https://example.com/image.png";
    expect(resolveAttachmentUrl(url)).toBe(url);
  });

  it("returns data URLs as-is", () => {
    const url = "data:image/png;base64,abc123";
    expect(resolveAttachmentUrl(url)).toBe(url);
  });

  it("resolves relative paths against origin base", () => {
    const path = "/api/v1/uploads/image.png";
    expect(resolveAttachmentUrl(path)).toBe("http://localhost:3000/api/v1/uploads/image.png");
  });
});

describe("fetchChats", () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  it("fetches chats without filters", async () => {
    const mockChats: Chat[] = [
      {
        id: "chat-1",
        name: "Test Chat",
        preview: "Hello",
        model: "gpt-4",
        view_mode: "bubble",
        chat_mode: "llm",
        is_read: true,
        is_archived: false,
        is_starred: false,
        label_ids: [],
        tools_enabled: true,
        web_search_enabled: false,
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-01T00:00:00Z",
      },
    ];

    mockFetch.mockResolvedValueOnce(createMockResponse(mockChats));

    const result = await fetchChats();

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:3000/api/v1/chats",
      expect.objectContaining({
        headers: { "Content-Type": "application/json" },
      })
    );
    expect(result).toEqual(mockChats);
  });

  it("fetches archived chats", async () => {
    mockFetch.mockResolvedValueOnce(createMockResponse([]));
    await fetchChats({ archived: true });
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:3000/api/v1/chats?archived=true",
      expect.any(Object)
    );
  });

  it("fetches starred chats", async () => {
    mockFetch.mockResolvedValueOnce(createMockResponse([]));
    await fetchChats({ starred: true });
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:3000/api/v1/chats?starred=true",
      expect.any(Object)
    );
  });

  it("throws on non-ok response", async () => {
    mockFetch.mockResolvedValueOnce(createMockResponse("Not found", { status: 404, ok: false }));
    await expect(fetchChats()).rejects.toThrow("Failed to fetch chats: 404");
  });
});

describe("fetchChat", () => {
  beforeEach(() => { mockFetch.mockReset(); });

  it("fetches single chat with messages", async () => {
    const mockChatData = {
      chat: { id: "chat-1", name: "Test" },
      messages: [{ id: "msg-1", content: "Hello" }],
      tool_call_records: [],
    };
    mockFetch.mockResolvedValueOnce(createMockResponse(mockChatData));
    const result = await fetchChat("chat-1");
    expect(mockFetch).toHaveBeenCalledWith("http://localhost:3000/api/v1/chats/chat-1", expect.any(Object));
    expect(result).toEqual(mockChatData);
  });
});

describe("createChat", () => {
  beforeEach(() => { mockFetch.mockReset(); });

  it("creates chat with data", async () => {
    const mockChat: Chat = {
      id: "new-chat", name: "New Chat", preview: "", model: "gpt-4",
      view_mode: "bubble", chat_mode: "llm", is_read: true, is_archived: false,
      is_starred: false, label_ids: [], tools_enabled: true, web_search_enabled: false,
      created_at: "2025-01-01T00:00:00Z", updated_at: "2025-01-01T00:00:00Z",
    };
    mockFetch.mockResolvedValueOnce(createMockResponse(mockChat));
    const result = await createChat({ name: "New Chat", model: "gpt-4" });
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:3000/api/v1/chats",
      expect.objectContaining({ method: "POST", body: JSON.stringify({ name: "New Chat", model: "gpt-4" }) })
    );
    expect(result).toEqual(mockChat);
  });

  it("creates chat without data", async () => {
    mockFetch.mockResolvedValueOnce(createMockResponse({ id: "new-chat" }));
    await createChat();
    expect(mockFetch).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ body: JSON.stringify({}) }));
  });
});

describe("deleteChat", () => {
  beforeEach(() => { mockFetch.mockReset(); });

  it("deletes chat", async () => {
    mockFetch.mockResolvedValueOnce(createMockResponse(null, { status: 204 }));
    await deleteChat("chat-1");
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:3000/api/v1/chats/chat-1",
      expect.objectContaining({ method: "DELETE" })
    );
  });
});

describe("addMessage", () => {
  beforeEach(() => { mockFetch.mockReset(); });

  it("adds message to chat", async () => {
    const mockMessage: Message = {
      id: "msg-1", chat_id: "chat-1", role: "user", content: "Hello",
      sibling_index: 0, created_at: "2025-01-01T00:00:00Z",
    };
    mockFetch.mockResolvedValueOnce(createMockResponse(mockMessage));
    const result = await addMessage("chat-1", { role: "user", content: "Hello" });
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:3000/api/v1/chats/chat-1/messages",
      expect.objectContaining({ method: "POST", body: JSON.stringify({ role: "user", content: "Hello" }) })
    );
    expect(result).toEqual(mockMessage);
  });

  it("includes optional fields", async () => {
    mockFetch.mockResolvedValueOnce(createMockResponse({ id: "msg-1" }));
    await addMessage("chat-1", {
      role: "user", content: "Hello",
      attachment_ids: ["attach-1"], web_search: true, skill_ids: ["skill-1"],
    });
    expect(mockFetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        body: JSON.stringify({
          role: "user", content: "Hello",
          attachment_ids: ["attach-1"], web_search: true, skill_ids: ["skill-1"],
        }),
      })
    );
  });
});
