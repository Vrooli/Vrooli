import { create } from "@bufbuild/protobuf";
import { cleanup, screen, waitFor } from "@testing-library/react";
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

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { ChatWorkspace } from "./ChatWorkspace";

const chatApiMock = vi.hoisted(() => {
  return {
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
  });

  it("renders grouped chats and the active message tree", async () => {
    renderWithProviders(<ChatWorkspace />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.chat.chat({ id: "chat-1" }))).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.chat.group({ id: "grp-1" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.chat.message({ id: "msg-1" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.chat.composer)).toBeInTheDocument();
  });

  it("enables the agent harness selector when agent mode is selected", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ChatWorkspace />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.chat.modeSelect)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.chat.harnessSelect)).toBeDisabled();
    await user.selectOptions(screen.getByTestId(selectors.chat.modeSelect), String(ChatMode.AGENT));
    expect(screen.getByTestId(selectors.chat.harnessSelect)).toBeEnabled();
  });

  it("sends a message and consumes streamed completion events", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ChatWorkspace />);

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
      );
    });
    expect(chatApiMock.streamPortalCompletion).toHaveBeenCalledWith(
      expect.objectContaining({
        chatId: "chat-1",
        fromMessageId: "msg-2",
      }),
    );
  });
});
