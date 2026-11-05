import { AgentTasksProvider } from "./useAgentTask";
import { create } from "@bufbuild/protobuf";
import { act, cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  ChatGroupSchema,
  ChatSchema,
} from "@vrooli/proto-types/portal/v1/chat/chat_pb";
import {
  CompletionEventKind,
  CompletionEventSchema,
  MessageRole,
  MessageSchema,
} from "@vrooli/proto-types/portal/v1/message/message_pb";
import { AgentHarness, ChatMode } from "@vrooli/proto-types/portal/v1/shared/common_pb";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { CompanionPresentation, CompanionToolbar } from "../companion/CompanionPresentation";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { ChatWorkspace } from "./ChatWorkspace";

const chatApiMock = vi.hoisted(() => {
  return {
    listPortalAgentAdmissions: vi.fn(),
    getPortalAgentRun: vi.fn(),
    stopPortalAgentRun: vi.fn(),
    listChats: vi.fn(),
    createPortalChat: vi.fn(),
    createPortalGroup: vi.fn(),
    updatePortalGroupCollapsed: vi.fn(),
    getMessageTree: vi.fn(),
    sendPortalMessage: vi.fn(),
    editPortalMessage: vi.fn(),
    regeneratePortalMessage: vi.fn(),
    streamPortalCompletion: vi.fn(),
    AgentHarness: {
      UNSPECIFIED: 0,
      CLAUDE_CODE: 1,
      CODEX: 2,
      OPENCODE: 3,
      GROK: 4,
    },
    ChatMode: {
      UNSPECIFIED: 0,
      LLM: 1,
      AGENT: 2,
    },
  };
});

vi.mock("../../api/chat", () => chatApiMock);

vi.mock("../../api/brief", () => ({
  BriefConsumer: { UNSPECIFIED: 0 },
  BriefUseKind: { OPENED: 1, COPIED: 2 },
  listPortalBriefs: vi.fn().mockResolvedValue([]),
  getPortalBrief: vi.fn(),
  recordPortalBriefUse: vi.fn().mockResolvedValue(true),
}));

vi.mock("../search/EcosystemOmnibox", () => ({
  EcosystemOmnibox: () => <div data-testid={selectors.search.omnibox} />,
}));

const group = create(ChatGroupSchema, {
  id: "grp-1",
  name: "Core",
  color: "var(--color-primary)",
  sortOrder: 1,
});

const chat = create(ChatSchema, {
  id: "chat-1",
  title: "Operator thread",
  groupId: "grp-1",
  model: "openai/gpt-4.1-mini",
  mode: ChatMode.LLM,
  agentHarness: AgentHarness.CLAUDE_CODE,
  webSearchEnabled: true,
  activeLeafMessageId: "msg-1",
  createdAt: "2026-07-06T00:00:00Z",
  updatedAt: "2026-07-06T00:00:00Z",
});

const userMessage = create(MessageSchema, {
  id: "msg-1",
  chatId: "chat-1",
  role: MessageRole.USER,
  content: "Hello Portal",
  createdAt: "2026-07-06T00:00:00Z",
  updatedAt: "2026-07-06T00:00:00Z",
});

const assistantMessage = create(MessageSchema, {
  id: "msg-assistant",
  chatId: "chat-1",
  parentMessageId: "msg-1",
  siblingIndex: 0,
  role: MessageRole.ASSISTANT,
  content: "Assistant answer",
  createdAt: "2026-07-06T00:00:02Z",
  updatedAt: "2026-07-06T00:00:02Z",
});

const alternateAssistantMessage = create(MessageSchema, {
  id: "msg-alt",
  chatId: "chat-1",
  parentMessageId: "msg-1",
  siblingIndex: 1,
  role: MessageRole.ASSISTANT,
  content: "Alternate answer",
  createdAt: "2026-07-06T00:00:03Z",
  updatedAt: "2026-07-06T00:00:03Z",
});

const sentMessage = create(MessageSchema, {
  id: "msg-2",
  chatId: "chat-1",
  parentMessageId: "msg-1",
  role: MessageRole.USER,
  content: "Next turn",
  createdAt: "2026-07-06T00:00:01Z",
  updatedAt: "2026-07-06T00:00:01Z",
});

async function* streamPortalCompletion() {
  await Promise.resolve();
  yield create(CompletionEventSchema, {
    kind: CompletionEventKind.STATUS,
    text: "started",
  });
  yield create(CompletionEventSchema, {
    kind: CompletionEventKind.TOKEN,
    text: "done",
  });
  yield create(CompletionEventSchema, {
    kind: CompletionEventKind.DONE,
  });
}

describe("ChatWorkspace", () => {
  beforeEach(() => {
    chatApiMock.listPortalAgentAdmissions.mockResolvedValue({admissions:[],nextPageToken:""});
    vi.clearAllMocks();
    chatApiMock.listChats.mockResolvedValue({ chats: [chat], groups: [group] });
    chatApiMock.createPortalChat.mockResolvedValue(chat);
    chatApiMock.createPortalGroup.mockResolvedValue(group);
    chatApiMock.updatePortalGroupCollapsed.mockResolvedValue(group);
    chatApiMock.getMessageTree.mockResolvedValue({
      messages: [userMessage],
      activeLeafMessageId: userMessage.id,
    });
    chatApiMock.sendPortalMessage.mockResolvedValue(sentMessage);
    chatApiMock.editPortalMessage.mockResolvedValue(userMessage);
    chatApiMock.regeneratePortalMessage.mockResolvedValue(userMessage);
    chatApiMock.streamPortalCompletion.mockImplementation(streamPortalCompletion);
  });

  afterEach(() => {
    cleanup();
    delete window.desktopPresentation;
  });

  it("keeps the selected branch and unsent draft across companion views", async () => {
    const user = userEvent.setup();
    window.desktopPresentation = {
      get: () => Promise.resolve({ version: 1, mode: "expanded" }),
      set: mode => Promise.resolve({ version: 1, mode }),
    };
    chatApiMock.getMessageTree.mockResolvedValue({
      messages: [userMessage, assistantMessage, alternateAssistantMessage],
      activeLeafMessageId: assistantMessage.id,
    });
    renderWithProviders(<CompanionPresentation><CompanionToolbar /><AgentTasksProvider><ChatWorkspace /></AgentTasksProvider></CompanionPresentation>);
    await screen.findByTestId(selectors.chat.message({ id: "msg-assistant" }));
    await user.click(screen.getByTestId(selectors.chat.branchNext));
    const branch = await screen.findByTestId(selectors.chat.message({ id: "msg-alt" }));
    const input = screen.getByTestId(selectors.chat.composerInput);
    await user.type(input, "Preserve this draft");
    const treeReads = chatApiMock.getMessageTree.mock.calls.length;
    for (const mode of ["palette", "pill", "expanded"]) {
      await user.selectOptions(screen.getByLabelText(strings.companion.presentation), mode);
      await waitFor(() => expect(screen.getByLabelText(strings.companion.presentation)).toHaveValue(mode));
      expect(screen.getByTestId(selectors.chat.composerInput)).toBe(input);
      expect(input).toHaveValue("Preserve this draft");
      expect(screen.getByTestId(selectors.chat.message({ id: "msg-alt" }))).toBe(branch);
      expect(chatApiMock.getMessageTree).toHaveBeenCalledTimes(treeReads);
    }
    expect(chatApiMock.sendPortalMessage).not.toHaveBeenCalled();
  });

  it("renders grouped chats and the active message tree", async () => {
    renderWithProviders(<AgentTasksProvider><ChatWorkspace /></AgentTasksProvider>);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.chat.chat({ id: "chat-1" }))).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.chat.group({ id: "grp-1" }))).toBeInTheDocument();
    expect(await screen.findByTestId(selectors.chat.message({ id: "msg-1" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.chat.composer)).toBeInTheDocument();
  });

  it("registers Stop during message admission and never starts completion after cancellation", async () => {
    window.desktopPresentation = { get: () => Promise.resolve({ version: 1, mode: "expanded" }), set: mode => Promise.resolve({ version: 1, mode }) };
    let resolveMessage!: (message: typeof sentMessage) => void;
    chatApiMock.sendPortalMessage.mockImplementation(() => new Promise(resolve => { resolveMessage = resolve; }));
    const user = userEvent.setup();
    renderWithProviders(<CompanionPresentation><CompanionToolbar /><AgentTasksProvider><ChatWorkspace /></AgentTasksProvider></CompanionPresentation>);
    const input = await screen.findByTestId(selectors.chat.composerInput);
    await user.type(input, "Cancel before generation");
    await user.click(screen.getByTestId(selectors.chat.sendButton));
    await waitFor(() => expect(chatApiMock.sendPortalMessage).toHaveBeenCalledOnce());
    await user.click(await screen.findByRole("button", { name: strings.companion.stopChat }));
    expect(chatApiMock.sendPortalMessage.mock.calls[0]?.[0]?.signal.aborted).toBe(true);
    act(() => { resolveMessage(sentMessage); });
    await waitFor(() => expect(screen.queryByTestId(selectors.chat.stopButton)).not.toBeInTheDocument());
    expect(chatApiMock.streamPortalCompletion).not.toHaveBeenCalled();
  });

  it("allows the active task to be stopped from the keyboard", async () => { // UI-08
    window.desktopPresentation = { get: () => Promise.resolve({ version: 1, mode: "expanded" }), set: mode => Promise.resolve({ version: 1, mode }) };
    const user = userEvent.setup();
    chatApiMock.streamPortalCompletion.mockImplementation(async function* (input: { signal: AbortSignal }) {
      yield create(CompletionEventSchema, { kind: CompletionEventKind.AGENT_ACTIVITY, text: "Running fixture" });
      await new Promise<void>(resolve => { if (input.signal.aborted) resolve(); else input.signal.addEventListener("abort", () => resolve(), { once: true }); });
    });
    renderWithProviders(<CompanionPresentation><CompanionToolbar /><AgentTasksProvider><ChatWorkspace /></AgentTasksProvider></CompanionPresentation>);
    const input = await screen.findByTestId(selectors.chat.composerInput);
    await user.type(input, "Keyboard stop");
    await user.click(screen.getByTestId(selectors.chat.sendButton));
    const stop = await screen.findByTestId(selectors.chat.stopButton);
    stop.focus();
    await user.keyboard("{Enter}");
    await waitFor(() => expect(screen.queryByTestId(selectors.chat.stopButton)).not.toBeInTheDocument());
  });

  it("does not start generation when stopped during the admitted message tree refresh", async () => {
    const tree = { messages: [userMessage], activeLeafMessageId: userMessage.id };
    let resolveTree!: (value: typeof tree) => void;
    chatApiMock.getMessageTree.mockResolvedValueOnce(tree).mockImplementationOnce(() => new Promise(resolve => { resolveTree = resolve; }));
    const user = userEvent.setup();
    renderWithProviders(<AgentTasksProvider><ChatWorkspace /></AgentTasksProvider>);
    await user.type(await screen.findByTestId(selectors.chat.composerInput), "Stop during refresh");
    await user.click(screen.getByTestId(selectors.chat.sendButton));
    await waitFor(() => expect(chatApiMock.getMessageTree).toHaveBeenCalledTimes(2));
    await user.click(await screen.findByTestId(selectors.chat.stopButton));
    act(() => { resolveTree(tree); });
    await waitFor(() => expect(screen.queryByTestId(selectors.chat.stopButton)).not.toBeInTheDocument());
    expect(chatApiMock.streamPortalCompletion).not.toHaveBeenCalled();
  });

  it("retains an uncertain agent task through pill mode and checks status without repeating Stop", async () => {
    window.desktopPresentation = { get: () => Promise.resolve({version:1,mode:"expanded"}), set: mode => Promise.resolve({version:1,mode}) };
    chatApiMock.stopPortalAgentRun.mockRejectedValueOnce(new Error("lost reply"));
    chatApiMock.getPortalAgentRun.mockResolvedValue({runId:"owned-run",status:"cancelled",terminal:true});
    chatApiMock.streamPortalCompletion.mockImplementation(async function* (input: {signal:AbortSignal}) {
      yield create(CompletionEventSchema,{kind:CompletionEventKind.AGENT_ACTIVITY,text:"Running fixture"});
      await new Promise<void>(resolve => { if(input.signal.aborted)resolve();else input.signal.addEventListener("abort",()=>resolve(),{once:true}); });
    });
    const user=userEvent.setup();
    const {container}=renderWithProviders(<CompanionPresentation><CompanionToolbar /><AgentTasksProvider><ChatWorkspace /></AgentTasksProvider></CompanionPresentation>);
    await user.type(await screen.findByTestId(selectors.chat.composerInput),"Run fixture");
    await user.selectOptions(screen.getByTestId(selectors.chat.modeSelect),String(ChatMode.AGENT));
    await user.click(screen.getByTestId(selectors.chat.sendButton));
    await waitFor(()=>expect(chatApiMock.streamPortalCompletion).toHaveBeenCalledOnce());
    await user.click(screen.getByTestId(selectors.chat.stopButton));
    await waitFor(()=>expect(screen.getByTestId(selectors.chat.stopButton)).toHaveTextContent(strings.companion.checkStatus));
    await user.selectOptions(screen.getByLabelText(strings.companion.presentation),"pill");
    const toolbar=container.querySelector<HTMLElement>(".companion-toolbar");
    if (!toolbar) throw new Error("expected companion toolbar");
    await user.click(within(toolbar).getByRole("button",{name:strings.companion.checkStatus}));
    await waitFor(()=>expect(screen.queryByTestId(selectors.chat.stopButton)).not.toBeInTheDocument());
    expect(chatApiMock.stopPortalAgentRun).toHaveBeenCalledTimes(1);
    expect(chatApiMock.stopPortalAgentRun).toHaveBeenCalledWith("chat-1",sentMessage.id);
    expect(chatApiMock.getPortalAgentRun).toHaveBeenCalledWith("chat-1",sentMessage.id);
  });

  it("sends a message and consumes streamed completion events", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AgentTasksProvider><ChatWorkspace /></AgentTasksProvider>);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.chat.composerInput)).toBeInTheDocument();
    });
    await user.type(screen.getByTestId(selectors.chat.composerInput), "Next turn");
    await user.click(screen.getByTestId(selectors.chat.sendButton));

    await waitFor(() => {
      expect(chatApiMock.sendPortalMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          chatId: "chat-1",
          content: "Next turn",
        }),
        undefined,
      );
    });
    expect(chatApiMock.streamPortalCompletion).toHaveBeenCalledWith(
      expect.objectContaining({
        chatId: "chat-1",
        fromMessageId: "msg-2",
      }),
      undefined,
    );
  });

  it("creates chats, agent chats, groups, and toggles group collapse", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AgentTasksProvider><ChatWorkspace /></AgentTasksProvider>);

    await screen.findByTestId(selectors.chat.chat({ id: "chat-1" }));
    await user.click(screen.getByTestId(selectors.chat.newChatButton));
    await user.click(screen.getByTestId(selectors.chat.newAgentChatButton));
    await user.click(screen.getByTestId(selectors.chat.newGroupButton));
    const groupButtons = within(screen.getByTestId(selectors.chat.group({ id: "grp-1" }))).getAllByRole("button");
    const groupHeader = groupButtons[0];
    if (!groupHeader) {
      throw new Error("expected group header button");
    }
    await user.click(groupHeader);

    expect(chatApiMock.createPortalChat).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ mode: ChatMode.LLM }),
      undefined,
    );
    expect(chatApiMock.createPortalChat).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ mode: ChatMode.AGENT }),
      undefined,
    );
    expect(chatApiMock.createPortalGroup).toHaveBeenCalledWith(expect.any(String), "var(--color-success)", undefined);
    expect(chatApiMock.updatePortalGroupCollapsed).toHaveBeenCalledWith("grp-1", true, undefined);
  });

  it("edits user messages and regenerates assistant branches", async () => {
    const user = userEvent.setup();
    const promptSpy = vi.spyOn(window, "prompt").mockReturnValue("Edited text");
    chatApiMock.getMessageTree.mockResolvedValue({
      messages: [userMessage, assistantMessage, alternateAssistantMessage],
      activeLeafMessageId: assistantMessage.id,
    });
    renderWithProviders(<AgentTasksProvider><ChatWorkspace /></AgentTasksProvider>);

    await screen.findByTestId(selectors.chat.message({ id: "msg-assistant" }));
    await user.click(screen.getByTestId(selectors.chat.editButton));
    await user.click(screen.getByTestId(selectors.chat.regenerateButton));
    await user.click(screen.getByTestId(selectors.chat.branchNext));

    expect(promptSpy).toHaveBeenCalledWith(expect.any(String), "Hello Portal");
    expect(chatApiMock.editPortalMessage).toHaveBeenCalledWith("msg-1", "Edited text", undefined);
    expect(chatApiMock.regeneratePortalMessage).toHaveBeenCalledWith("msg-assistant", "openai/gpt-4.1-mini", undefined);
    expect(await screen.findByTestId(selectors.chat.message({ id: "msg-alt" }))).toBeInTheDocument();
  });
});
