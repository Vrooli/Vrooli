import { Chat, ChatCreateInput, ChatSearchInput, ChatUpdateInput, FindByIdInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsChat = {
    Query: {
        chat: ApiEndpoint<FindByIdInput, FindOneResult<Chat>>;
        chats: ApiEndpoint<ChatSearchInput, FindManyResult<Chat>>;
    },
    Mutation: {
        chatCreate: ApiEndpoint<ChatCreateInput, CreateOneResult<Chat>>;
        chatUpdate: ApiEndpoint<ChatUpdateInput, UpdateOneResult<Chat>>;
    }
}

const objectType = "Chat";
export const ChatEndpoints: EndpointsChat = {
    Query: {
        chat: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        chats: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        chatCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        chatUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
