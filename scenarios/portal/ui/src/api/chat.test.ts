import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const clients = vi.hoisted(() => ({
  chat: {
    listChats: vi.fn(),
    createChat: vi.fn(),
    createGroup: vi.fn(),
    updateGroup: vi.fn(),
  },
  message: {
    listAgentAdmissions: vi.fn(),
    getAgentRun: vi.fn(),
    stopAgentRun: vi.fn(),
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
  listPortalAgentAdmissions,
  getPortalAgentRun,
  stopPortalAgentRun,
  createPortalChat,
  createPortalGroup,
  editPortalMessage,
  getMessageTree,
  listChats,
  regeneratePortalMessage,
  sendPortalMessage,
  streamPortalCompletion,
  updatePortalGroupCollapsed,
  type ChatAccess,
} from "./chat";

describe("api/chat Connect wrappers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("preserves recovery page tokens and never launches or stops while listing", async () => {
    const page = { admissions: [{chatId:"chat",messageId:"unknown-message"}], nextPageToken:"2" };
    clients.message.listAgentAdmissions.mockResolvedValueOnce(page);
    await expect(listPortalAgentAdmissions("1")).resolves.toEqual(page);
    expect(clients.message.listAgentAdmissions).toHaveBeenCalledWith({pageToken:"1",pageSize:50});
    expect(clients.message.stopAgentRun).not.toHaveBeenCalled();
    expect(clients.message.streamCompletion).not.toHaveBeenCalled();
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

  it("uses chat/message identity for Stop and independent readback", async () => {
    clients.message.stopAgentRun.mockRejectedValueOnce(new Error("lost reply"));
    clients.message.getAgentRun.mockResolvedValueOnce({runId:"owned-run",status:"cancelled",terminal:true});
    await expect(stopPortalAgentRun("chat-1","msg-1")).rejects.toThrow("lost reply");
    await expect(getPortalAgentRun("chat-1","msg-1")).resolves.toMatchObject({terminal:true});
    expect(clients.message.stopAgentRun).toHaveBeenCalledTimes(1);
    expect(clients.message.getAgentRun).toHaveBeenCalledTimes(1);
    expect(clients.message.stopAgentRun).toHaveBeenCalledWith({chatId:"chat-1",messageId:"msg-1"});
    expect(clients.message.getAgentRun).toHaveBeenCalledWith({chatId:"chat-1",messageId:"msg-1"});
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
    await expect(sendPortalMessage({ chatId: "chat-1", content: "hello", signal })).resolves.toEqual({ id: "msg-2" });
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
      contextDocumentIds: [],
    }, { signal });
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

  it("binds every authenticated operation to its account and fences late replies", async () => {
    const controller = new AbortController();
    const access: ChatAccess = { actor: "[\"realm\",\"alice\"]", signal: controller.signal, account: { token: "alice-token", expires: Date.now() + 60_000 } };
    let release!: (value: { chats: never[]; groups: never[] }) => void;
    clients.chat.listChats.mockImplementationOnce(() => new Promise(resolve => { release = resolve; }));
    const pending = listChats(access);
    expect(clients.chat.listChats).toHaveBeenCalledWith({}, { headers: { Authorization: "Bearer alice-token" }, signal: access.signal });
    controller.abort();
    release({ chats: [], groups: [] });
    await expect(pending).rejects.toThrow();
  });

  it("rejects expired account access before touching the transport", async () => {
    const access: ChatAccess = { actor: "alice", signal: new AbortController().signal, account: { token: "expired", expires: Date.now() - 1 } };
    await expect(listChats(access)).rejects.toThrow("Chat account authentication required");
    expect(clients.chat.listChats).not.toHaveBeenCalled();
  });
});
