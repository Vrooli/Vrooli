import { chat_create, chat_findMany, chat_findOne, chat_update } from "../generated";
import { ChatEndpoints } from "../logic/chat";
import { setupRoutes } from "./base";

export const ChatRest = setupRoutes({
    "/chat/:id": {
        get: [ChatEndpoints.Query.chat, chat_findOne],
        put: [ChatEndpoints.Mutation.chatUpdate, chat_update],
    },
    "/chats": {
        get: [ChatEndpoints.Query.chats, chat_findMany],
    },
    "/chat": {
        post: [ChatEndpoints.Mutation.chatCreate, chat_create],
    },
});
