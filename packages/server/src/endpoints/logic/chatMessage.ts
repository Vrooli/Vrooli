import { ChatMessage, ChatMessageCreateInput, ChatMessageSearchInput, ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatMessageUpdateInput, FindByIdInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsChatMessage = {
    Query: {
        chatMessage: GQLEndpoint<FindByIdInput, FindOneResult<ChatMessage>>;
        chatMessages: GQLEndpoint<ChatMessageSearchInput, FindManyResult<ChatMessage>>;
        chatMessageTree: GQLEndpoint<ChatMessageSearchTreeInput, ChatMessageSearchTreeResult>;
    },
    Mutation: {
        chatMessageCreate: GQLEndpoint<ChatMessageCreateInput, CreateOneResult<ChatMessage>>;
        chatMessageUpdate: GQLEndpoint<ChatMessageUpdateInput, UpdateOneResult<ChatMessage>>;
    }
}

const objectType = "ChatMessage";
export const ChatMessageEndpoints: EndpointsChatMessage = {
    Query: {
        chatMessage: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        chatMessages: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
        chatMessageTree: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return ChatMessageModel.query.searchTree(prisma, req, input, info);
        },
    },
    Mutation: {
        chatMessageCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        chatMessageUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
