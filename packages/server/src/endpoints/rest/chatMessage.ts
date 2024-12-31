import { endpointsChatMessage } from "@local/shared";
import { chatMessage_create, chatMessage_findMany, chatMessage_findOne, chatMessage_findTree, chatMessage_regenerateResponse, chatMessage_update } from "../generated";
import { ChatMessageEndpoints } from "../logic/chatMessage";
import { setupRoutes } from "./base";

export const ChatMessageRest = setupRoutes([
    [endpointsChatMessage.findOne, ChatMessageEndpoints.Query.chatMessage, chatMessage_findOne],
    [endpointsChatMessage.updateOne, ChatMessageEndpoints.Mutation.chatMessageUpdate, chatMessage_update],
    [endpointsChatMessage.findMany, ChatMessageEndpoints.Query.chatMessages, chatMessage_findMany],
    [endpointsChatMessage.findTree, ChatMessageEndpoints.Query.chatMessageTree, chatMessage_findTree],
    [endpointsChatMessage.createOne, ChatMessageEndpoints.Mutation.chatMessageCreate, chatMessage_create],
    [endpointsChatMessage.regenerateResponse, ChatMessageEndpoints.Mutation.regenerateResponse, chatMessage_regenerateResponse],
]);
