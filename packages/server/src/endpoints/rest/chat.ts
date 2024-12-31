import { endpointsChat } from "@local/shared";
import { chat_create, chat_findMany, chat_findOne, chat_update } from "../generated";
import { ChatEndpoints } from "../logic/chat";
import { setupRoutes } from "./base";

export const ChatRest = setupRoutes([
    [endpointsChat.findOne, ChatEndpoints.Query.chat, chat_findOne],
    [endpointsChat.findMany, ChatEndpoints.Query.chats, chat_findMany],
    [endpointsChat.createOne, ChatEndpoints.Mutation.chatCreate, chat_create],
    [endpointsChat.updateOne, ChatEndpoints.Mutation.chatUpdate, chat_update],
]);
