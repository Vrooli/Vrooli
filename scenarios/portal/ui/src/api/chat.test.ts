import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const clients = vi.hoisted(() => ({
  chat: {
    listChats: vi.fn(),
    createChat: vi.fn(),
    createGroup: vi.fn(),
    updateGroup: vi.fn(),
  },
  message: {
    getTree: vi.fn(),
    sendMessage: vi.fn(),
    editMessage: vi.fn(),
    regenerate: vi.fn(),
    streamCompletion: vi.fn(),
  },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn()
    .mockReturnValueOnce(clients.chat)
    .mockReturnValueOnce(clients.message),
}));

import {
  ChatMode,
  createPortalChat,
  createPortalGroup,
  editPortalMessage,
  getMessageTree,
  listChats,
  regeneratePortalMessage,
  sendPortalMessage,
  streamPortalCompletion,
  updatePortalGroupCollapsed,
} from "./chat";

describe("api/chat Connect wrappers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("maps chat CRUD calls onto generated Connect clients", async () => {
    clients.chat.listChats.mockResolvedValueOnce({ chats: [], groups: [] });
    clients.chat.createChat.mockResolvedValueOnce({ chat: { id: "chat-1" } });
    clients.chat.createGroup.mockResolvedValueOnce({ group: { id: "grp-1" } });
    clients.chat.updateGroup.mockResolvedValueOnce({ group: { id: "grp-1", collapsed: true } });

    await expect(listChats()).resolves.toEqual({ chats: [], groups: [] });
    await expect(createPortalChat({ title: "New" })).resolves.toEqual({ id: "chat-1" });
    await expect(createPortalGroup("Core", "#123456")).resolves.toEqual({ id: "grp-1" });
    await expect(updatePortalGroupCollapsed("grp-1", true)).resolves.toEqual({
      id: "grp-1",
      collapsed: true,
    });

    expect(clients.chat.listChats).toHaveBeenCalledWith({});
    expect(clients.chat.createChat).toHaveBeenCalledWith({
      title: "New",
      groupId: "",
      model: "",
      webSearchEnabled: false,
      mode: ChatMode.LLM,
    });
    expect(clients.chat.createGroup).toHaveBeenCalledWith({ name: "Core", color: "#123456" });
    expect(clients.chat.updateGroup).toHaveBeenCalledWith({
      id: "grp-1",
      collapsed: true,
      hasCollapsed: true,
    });
  });

  it("maps message tree, send, edit, regenerate, and stream calls", async () => {
    const signal = new AbortController().signal;
    async function* events() {
      await Promise.resolve();
      yield { kind: 1, text: "token" };
    }
    clients.message.getTree.mockResolvedValueOnce({
      messages: [{ id: "msg-1" }],
      activeLeafMessageId: "msg-1",
    });
    clients.message.sendMessage.mockResolvedValueOnce({ userMessage: { id: "msg-2" } });
    clients.message.editMessage.mockResolvedValueOnce({ message: { id: "msg-1", content: "edited" } });
    clients.message.regenerate.mockResolvedValueOnce({ assistantMessage: { id: "msg-3" } });
    clients.message.streamCompletion.mockReturnValueOnce(events());

    await expect(getMessageTree("chat-1")).resolves.toEqual({
      messages: [{ id: "msg-1" }],
      activeLeafMessageId: "msg-1",
    });
    await expect(sendPortalMessage({ chatId: "chat-1", content: "hello" })).resolves.toEqual({ id: "msg-2" });
    await expect(editPortalMessage("msg-1", "edited")).resolves.toEqual({ id: "msg-1", content: "edited" });
    await expect(regeneratePortalMessage("msg-1", "model-a")).resolves.toEqual({ id: "msg-3" });
    const streamed = [];
    for await (const event of streamPortalCompletion({ chatId: "chat-1", fromMessageId: "msg-2", signal })) {
      streamed.push(event);
    }

    expect(streamed).toEqual([{ kind: 1, text: "token" }]);
    expect(clients.message.sendMessage).toHaveBeenCalledWith({
      chatId: "chat-1",
      parentMessageId: "",
      content: "hello",
      model: "",
      webSearchEnabled: false,
      selectedSkillIds: [],
    });
    expect(clients.message.streamCompletion).toHaveBeenCalledWith(
      {
        chatId: "chat-1",
        fromMessageId: "msg-2",
        model: "",
        webSearchEnabled: false,
        selectedSkillIds: [],
        mode: ChatMode.LLM,
      },
      { signal },
    );
  });

  it("throws when required response payloads are absent", async () => {
    clients.chat.createChat.mockResolvedValueOnce({});
    clients.message.sendMessage.mockResolvedValueOnce({});
    clients.message.editMessage.mockResolvedValueOnce({});
    clients.message.regenerate.mockResolvedValueOnce({});

    await expect(createPortalChat({ title: "bad" })).rejects.toThrow("create chat response");
    await expect(sendPortalMessage({ chatId: "chat-1", content: "bad" })).rejects.toThrow("send message response");
    await expect(editPortalMessage("msg-1", "bad")).rejects.toThrow("edit message response");
    await expect(regeneratePortalMessage("msg-1", "model-a")).rejects.toThrow("regenerate response");
  });
});
