import { ChatInvite, ChatInviteCreateInput, ChatInviteSearchInput, ChatInviteUpdateInput, FindByIdInput } from "@local/shared";
import { createManyHelper, createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateManyHelper, updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsChatInvite = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<ChatInvite>>;
    findMany: ApiEndpoint<ChatInviteSearchInput, FindManyResult<ChatInvite>>;
    createOne: ApiEndpoint<ChatInviteCreateInput, CreateOneResult<ChatInvite>>;
    createMany: ApiEndpoint<ChatInviteCreateInput[], CreateOneResult<ChatInvite>[]>;
    updateOne: ApiEndpoint<ChatInviteUpdateInput, UpdateOneResult<ChatInvite>>;
    updateMany: ApiEndpoint<ChatInviteUpdateInput[], UpdateOneResult<ChatInvite>[]>;
    acceptOne: ApiEndpoint<FindByIdInput, UpdateOneResult<ChatInvite>>;
    declineOne: ApiEndpoint<FindByIdInput, UpdateOneResult<ChatInvite>>;
}

const objectType = "ChatInvite";
export const chatInvite: EndpointsChatInvite = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    createMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createManyHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
    updateMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateManyHelper({ info, input, objectType, req });
    },
    acceptOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        throw new CustomError("0000", "NotImplemented");
    },
    declineOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        throw new CustomError("0000", "NotImplemented");
    },
};
