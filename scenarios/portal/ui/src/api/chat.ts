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

export type ChatAccess={actor:string;signal:AbortSignal;account?:{token:string;expires:number}};
export function checkChatAccess(access?:ChatAccess,signal?:AbortSignal){
 access?.signal.throwIfAborted();signal?.throwIfAborted();
 if(access?.account&&(!access.account.token||access.account.expires<=Date.now()))throw new Error("Chat account authentication required");
}
function options(access?:ChatAccess,signal?:AbortSignal):{headers?:Record<string,string>;signal?:AbortSignal}|undefined{
 checkChatAccess(access,signal);
 const request:{headers?:Record<string,string>;signal?:AbortSignal}={};
 if (access?.account) request.headers={Authorization:`Bearer ${access.account.token}`};
 const combined=access?signal?AbortSignal.any([access.signal,signal]):access.signal:signal;
 if (combined) request.signal=combined;
 return request.headers || request.signal ? request : undefined;
}

export interface CreatePortalChatInput {
  title: string;
  groupId?: string;
  mode?: ChatMode;
  model?: string;
  webSearchEnabled?: boolean;
}

export interface SendPortalMessageInput {
  signal?: AbortSignal;
  chatId: string;
  parentMessageId?: string;
  content: string;
  model?: string;
  webSearchEnabled?: boolean;
  selectedSkillIds?: string[];
  contextDocumentIds?: string[];
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

export async function listChats(access?:ChatAccess): Promise<ListChatsResponse> {
  const request=options(access);const response=request?await chatClient.listChats({},request):await chatClient.listChats({});checkChatAccess(access);return response;
}

export async function createPortalChat(input: CreatePortalChatInput,access?:ChatAccess): Promise<Chat> {
  const inputMessage = {
    title: input.title,
    groupId: input.groupId ?? "",
    model: input.model ?? "",
    webSearchEnabled: input.webSearchEnabled ?? false,
    mode: input.mode ?? ChatMode.LLM,
  };
  const request=options(access);const response=request?await chatClient.createChat(inputMessage,request):await chatClient.createChat(inputMessage);
  checkChatAccess(access);
  if (!response.chat) {
    throw new Error("create chat response did not include a chat");
  }
  return response.chat;
}

export async function createPortalGroup(name: string, color: string,access?:ChatAccess): Promise<ChatGroup> {
  const inputMessage={ name, color };const request=options(access);const response=request?await chatClient.createGroup(inputMessage,request):await chatClient.createGroup(inputMessage);
  checkChatAccess(access);
  if (!response.group) {
    throw new Error("create group response did not include a group");
  }
  return response.group;
}

export async function updatePortalGroupCollapsed(id: string, collapsed: boolean,access?:ChatAccess): Promise<ChatGroup> {
  const inputMessage={
    id,
    collapsed,
    hasCollapsed: true,
  };const request=options(access);const response=request?await chatClient.updateGroup(inputMessage,request):await chatClient.updateGroup(inputMessage);
  checkChatAccess(access);
  if (!response.group) {
    throw new Error("update group response did not include a group");
  }
  return response.group;
}

export async function getMessageTree(chatId: string,access?:ChatAccess): Promise<{
  messages: Message[];
  activeLeafMessageId: string;
}> {
  const inputMessage={chatId};const request=options(access);const response=request?await messageClient.getTree(inputMessage,request):await messageClient.getTree(inputMessage);
  checkChatAccess(access);
  return {
    messages: response.messages,
    activeLeafMessageId: response.activeLeafMessageId,
  };
}

export async function sendPortalMessage(input: SendPortalMessageInput,access?:ChatAccess): Promise<Message> {
  const inputMessage={
    chatId: input.chatId,
    parentMessageId: input.parentMessageId ?? "",
    content: input.content,
    model: input.model ?? "",
    webSearchEnabled: input.webSearchEnabled ?? false,
    selectedSkillIds: input.selectedSkillIds ?? [],
    contextDocumentIds: input.contextDocumentIds ?? [],
  };const request=options(access,input.signal);const response=request?await messageClient.sendMessage(inputMessage,request):await messageClient.sendMessage(inputMessage);
  checkChatAccess(access,input.signal);
  if (!response.userMessage) {
    throw new Error("send message response did not include a user message");
  }
  return response.userMessage;
}

export async function editPortalMessage(messageId: string, content: string,access?:ChatAccess): Promise<Message> {
  const inputMessage={ messageId, content };const request=options(access);const response=request?await messageClient.editMessage(inputMessage,request):await messageClient.editMessage(inputMessage);
  checkChatAccess(access);
  if (!response.message) {
    throw new Error("edit message response did not include a message");
  }
  return response.message;
}

export async function regeneratePortalMessage(messageId: string, model: string,access?:ChatAccess): Promise<Message> {
  const inputMessage={ messageId, model };const request=options(access);const response=request?await messageClient.regenerate(inputMessage,request):await messageClient.regenerate(inputMessage);
  checkChatAccess(access);
  if (!response.assistantMessage) {
    throw new Error("regenerate response did not include an assistant message");
  }
  return response.assistantMessage;
}

export async function* streamPortalCompletion(
  input: StreamPortalCompletionInput,access?:ChatAccess,
): AsyncIterable<CompletionEvent> {
  const inputMessage={
      chatId: input.chatId,
      fromMessageId: input.fromMessageId,
      model: input.model ?? "",
      webSearchEnabled: input.webSearchEnabled ?? false,
      selectedSkillIds: input.selectedSkillIds ?? [],
      mode: input.mode ?? ChatMode.LLM,
    };const request=options(access,input.signal);const events=request?messageClient.streamCompletion(inputMessage,request):messageClient.streamCompletion(inputMessage);
  for await(const event of events){checkChatAccess(access,input.signal);yield event;}
}

export { AgentHarness, ChatMode };
export type { Chat, ChatGroup, CompletionEvent, Message };

export async function getPortalAgentRun(chatId: string, messageId: string,access?:ChatAccess) {
  const inputMessage={ chatId, messageId };const request=options(access);const response=request?await messageClient.getAgentRun(inputMessage,request):await messageClient.getAgentRun(inputMessage);checkChatAccess(access);return response;
}

export async function stopPortalAgentRun(chatId: string, messageId: string,access?:ChatAccess) {
  const inputMessage={ chatId, messageId };const request=options(access);const response=request?await messageClient.stopAgentRun(inputMessage,request):await messageClient.stopAgentRun(inputMessage);checkChatAccess(access);return response;
}

/** Each page includes unknown admissions; absence of a run ID is not completion. */
export async function listPortalAgentAdmissions(pageToken = "",access?:ChatAccess) {
  const inputMessage={ pageToken, pageSize: 50 };const request=options(access);const response=request?await messageClient.listAgentAdmissions(inputMessage,request):await messageClient.listAgentAdmissions(inputMessage);checkChatAccess(access);return response;
}
