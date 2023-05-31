import { Chat, ChatCreateInput, ChatSearchInput, ChatUpdateInput, FindByIdInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsChat = {
    Query: {
        chat: GQLEndpoint<FindByIdInput, FindOneResult<Chat>>;
        chats: GQLEndpoint<ChatSearchInput, FindManyResult<Chat>>;
    },
    Mutation: {
        chatCreate: GQLEndpoint<ChatCreateInput, CreateOneResult<Chat>>;
        chatUpdate: GQLEndpoint<ChatUpdateInput, UpdateOneResult<Chat>>;
    }
}

const objectType = "Chat";
export const ChatEndpoints: EndpointsChat = {
    Query: {
        chat: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        chats: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        chatCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        chatUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
