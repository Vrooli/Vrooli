import { ChatInvite, ChatInviteCreateInput, ChatInviteSearchInput, ChatInviteUpdateInput, FindByIdInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { CustomError } from "../../events";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsChatInvite = {
    Query: {
        chatInvite: GQLEndpoint<FindByIdInput, FindOneResult<ChatInvite>>;
        chatInvites: GQLEndpoint<ChatInviteSearchInput, FindManyResult<ChatInvite>>;
    },
    Mutation: {
        chatInviteCreate: GQLEndpoint<ChatInviteCreateInput, CreateOneResult<ChatInvite>>;
        chatInviteUpdate: GQLEndpoint<ChatInviteUpdateInput, UpdateOneResult<ChatInvite>>;
        chatInviteAccept: GQLEndpoint<FindByIdInput, UpdateOneResult<ChatInvite>>;
        chatInviteDecline: GQLEndpoint<FindByIdInput, UpdateOneResult<ChatInvite>>;
    }
}

const objectType = "ChatInvite";
export const ChatInviteEndpoints: EndpointsChatInvite = {
    Query: {
        chatInvite: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        chatInvites: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        chatInviteCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        chatInviteUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        chatInviteAccept: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
        chatInviteDecline: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
    },
};
