import { chatMessage_autoFill, chatMessage_create, chatMessage_findMany, chatMessage_findOne, chatMessage_findTree, chatMessage_regenerateResponse, chatMessage_update } from "../generated";
import { ChatMessageEndpoints } from "../logic/chatMessage";
import { setupRoutes } from "./base";

export const ChatMessageRest = setupRoutes({
    "/chatMessage/:id": {
        get: [ChatMessageEndpoints.Query.chatMessage, chatMessage_findOne],
        put: [ChatMessageEndpoints.Mutation.chatMessageUpdate, chatMessage_update],
    },
    "/chatMessages": {
        get: [ChatMessageEndpoints.Query.chatMessages, chatMessage_findMany],
    },
    "/chatMessageTree": {
        get: [ChatMessageEndpoints.Query.chatMessageTree, chatMessage_findTree],
    },
    "/chatMessage": {
        post: [ChatMessageEndpoints.Mutation.chatMessageCreate, chatMessage_create],
    },
    "/regenerateResponse": {
        post: [ChatMessageEndpoints.Mutation.regenerateResponse, chatMessage_regenerateResponse],
    },
    "/autoFill": {
        get: [ChatMessageEndpoints.Mutation.autoFill, chatMessage_autoFill],
    },
});
