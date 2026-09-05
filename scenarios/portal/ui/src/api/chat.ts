import { createClient } from "@connectrpc/connect";
import {
  ChatService,
  type Chat,
  type ChatGroup,
  type ListChatsResponse,
} from "@vrooli/proto-types/portal/v1/chat/chat_pb";
import {
  MessageService,
  type CompletionEvent,
  type Message,
} from "@vrooli/proto-types/portal/v1/message/message_pb";
import { AgentHarness, ChatMode } from "@vrooli/proto-types/portal/v1/shared/common_pb";

import { transport } from "./client";

const chatClient = createClient(ChatService, transport);
const messageClient = createClient(MessageService, transport);

export interface CreatePortalChatInput {
  title: string;
  groupId?: string;
  mode?: ChatMode;
  model?: string;
  webSearchEnabled?: boolean;
}

export interface SendPortalMessageInput {
  chatId: string;
  parentMessageId?: string;
  content: string;
  model?: string;
  webSearchEnabled?: boolean;
  selectedSkillIds?: string[];
}

export interface StreamPortalCompletionInput {
  chatId: string;
  fromMessageId: string;
  model?: string;
  webSearchEnabled?: boolean;
  selectedSkillIds?: string[];
  mode?: ChatMode;
  signal?: AbortSignal;
}

export async function listChats(): Promise<ListChatsResponse> {
  return chatClient.listChats({});
}

export async function createPortalChat(input: CreatePortalChatInput): Promise<Chat> {
  const response = await chatClient.createChat({
    title: input.title,
    groupId: input.groupId ?? "",
    model: input.model ?? "",
    webSearchEnabled: input.webSearchEnabled ?? false,
    mode: input.mode ?? ChatMode.LLM,
  });
  if (!response.chat) {
    throw new Error("create chat response did not include a chat");
  }
  return response.chat;
}

export async function createPortalGroup(name: string, color: string): Promise<ChatGroup> {
  const response = await chatClient.createGroup({ name, color });
  if (!response.group) {
    throw new Error("create group response did not include a group");
  }
  return response.group;
}

export async function updatePortalGroupCollapsed(id: string, collapsed: boolean): Promise<ChatGroup> {
  const response = await chatClient.updateGroup({
    id,
    collapsed,
    hasCollapsed: true,
  });
  if (!response.group) {
    throw new Error("update group response did not include a group");
  }
  return response.group;
}

export async function getMessageTree(chatId: string): Promise<{
  messages: Message[];
  activeLeafMessageId: string;
}> {
  const response = await messageClient.getTree({ chatId });
  return {
    messages: response.messages,
    activeLeafMessageId: response.activeLeafMessageId,
  };
}

export async function sendPortalMessage(input: SendPortalMessageInput): Promise<Message> {
  const response = await messageClient.sendMessage({
    chatId: input.chatId,
    parentMessageId: input.parentMessageId ?? "",
    content: input.content,
    model: input.model ?? "",
    webSearchEnabled: input.webSearchEnabled ?? false,
    selectedSkillIds: input.selectedSkillIds ?? [],
  });
  if (!response.userMessage) {
    throw new Error("send message response did not include a user message");
  }
  return response.userMessage;
}

export async function editPortalMessage(messageId: string, content: string): Promise<Message> {
  const response = await messageClient.editMessage({ messageId, content });
  if (!response.message) {
    throw new Error("edit message response did not include a message");
  }
  return response.message;
}

export async function regeneratePortalMessage(messageId: string, model: string): Promise<Message> {
  const response = await messageClient.regenerate({ messageId, model });
  if (!response.assistantMessage) {
    throw new Error("regenerate response did not include an assistant message");
  }
  return response.assistantMessage;
}

export async function* streamPortalCompletion(
  input: StreamPortalCompletionInput,
): AsyncIterable<CompletionEvent> {
  yield* messageClient.streamCompletion(
    {
      chatId: input.chatId,
      fromMessageId: input.fromMessageId,
      model: input.model ?? "",
      webSearchEnabled: input.webSearchEnabled ?? false,
      selectedSkillIds: input.selectedSkillIds ?? [],
      mode: input.mode ?? ChatMode.LLM,
    },
    { signal: input.signal },
  );
}

export { AgentHarness, ChatMode };
export type { Chat, ChatGroup, CompletionEvent, Message };
