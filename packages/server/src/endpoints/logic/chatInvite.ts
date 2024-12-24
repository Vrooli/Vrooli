import { ChatInvite, ChatInviteCreateInput, ChatInviteSearchInput, ChatInviteUpdateInput, FindByIdInput } from "@local/shared";
import { createManyHelper, createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateManyHelper, updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsChatInvite = {
    Query: {
        chatInvite: ApiEndpoint<FindByIdInput, FindOneResult<ChatInvite>>;
        chatInvites: ApiEndpoint<ChatInviteSearchInput, FindManyResult<ChatInvite>>;
    },
    Mutation: {
        chatInviteCreate: ApiEndpoint<ChatInviteCreateInput, CreateOneResult<ChatInvite>>;
        chatInvitesCreate: ApiEndpoint<ChatInviteCreateInput[], CreateOneResult<ChatInvite>[]>;
        chatInviteUpdate: ApiEndpoint<ChatInviteUpdateInput, UpdateOneResult<ChatInvite>>;
        chatInvitesUpdate: ApiEndpoint<ChatInviteUpdateInput[], UpdateOneResult<ChatInvite>[]>;
        chatInviteAccept: ApiEndpoint<FindByIdInput, UpdateOneResult<ChatInvite>>;
        chatInviteDecline: ApiEndpoint<FindByIdInput, UpdateOneResult<ChatInvite>>;
    }
}

const objectType = "ChatInvite";
export const ChatInviteEndpoints: EndpointsChatInvite = {
    Query: {
        chatInvite: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        chatInvites: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        chatInviteCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        chatInvitesCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createManyHelper({ info, input, objectType, req });
        },
        chatInviteUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        chatInvitesUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateManyHelper({ info, input, objectType, req });
        },
        chatInviteAccept: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented");
        },
        chatInviteDecline: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented");
        },
    },
};
