import { chatMessage_create, chatMessage_findMany, chatMessage_findOne, chatMessage_update } from "@local/shared";
import { ChatMessageEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ChatMessageRest = setupRoutes({
    "/chatMessage/:id": {
        get: [ChatMessageEndpoints.Query.chatMessage, chatMessage_findOne],
        put: [ChatMessageEndpoints.Mutation.chatMessageUpdate, chatMessage_update],
    },
    "/chatMessages": {
        get: [ChatMessageEndpoints.Query.chatMessages, chatMessage_findMany],
    },
    "/chatMessage": {
        post: [ChatMessageEndpoints.Mutation.chatMessageCreate, chatMessage_create],
    },
});
