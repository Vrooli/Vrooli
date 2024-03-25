import { ChatInvite, ChatInviteCreateInput, ChatInviteSearchInput, ChatInviteUpdateInput, FindByIdInput } from "@local/shared";
import { createManyHelper, createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateManyHelper, updateOneHelper } from "../../actions/updates";
import { CustomError } from "../../events/error";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsChatInvite = {
    Query: {
        chatInvite: GQLEndpoint<FindByIdInput, FindOneResult<ChatInvite>>;
        chatInvites: GQLEndpoint<ChatInviteSearchInput, FindManyResult<ChatInvite>>;
    },
    Mutation: {
        chatInviteCreate: GQLEndpoint<ChatInviteCreateInput, CreateOneResult<ChatInvite>>;
        chatInvitesCreate: GQLEndpoint<ChatInviteCreateInput[], CreateOneResult<ChatInvite>[]>;
        chatInviteUpdate: GQLEndpoint<ChatInviteUpdateInput, UpdateOneResult<ChatInvite>>;
        chatInvitesUpdate: GQLEndpoint<ChatInviteUpdateInput[], UpdateOneResult<ChatInvite>[]>;
        chatInviteAccept: GQLEndpoint<FindByIdInput, UpdateOneResult<ChatInvite>>;
        chatInviteDecline: GQLEndpoint<FindByIdInput, UpdateOneResult<ChatInvite>>;
    }
}

const objectType = "ChatInvite";
export const ChatInviteEndpoints: EndpointsChatInvite = {
    Query: {
        chatInvite: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        chatInvites: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        chatInviteCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        chatInvitesCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createManyHelper({ info, input, objectType, req });
        },
        chatInviteUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        chatInvitesUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateManyHelper({ info, input, objectType, req });
        },
        chatInviteAccept: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
        chatInviteDecline: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
    },
};
