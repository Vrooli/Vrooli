import { ChatInvite, ChatInviteCreateInput, ChatInviteSearchInput, ChatInviteSearchResult, ChatInviteUpdateInput, FindByIdInput } from "@local/shared";
import { createManyHelper, createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateManyHelper, updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { CustomError } from "../../events/error.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsChatInvite = {
    findOne: ApiEndpoint<FindByIdInput, ChatInvite>;
    findMany: ApiEndpoint<ChatInviteSearchInput, ChatInviteSearchResult>;
    createOne: ApiEndpoint<ChatInviteCreateInput, ChatInvite>;
    createMany: ApiEndpoint<ChatInviteCreateInput[], ChatInvite[]>;
    updateOne: ApiEndpoint<ChatInviteUpdateInput, ChatInvite>;
    updateMany: ApiEndpoint<ChatInviteUpdateInput[], ChatInvite[]>;
    acceptOne: ApiEndpoint<FindByIdInput, ChatInvite>;
    declineOne: ApiEndpoint<FindByIdInput, ChatInvite>;
}

const objectType = "ChatInvite";
export const chatInvite: EndpointsChatInvite = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    createMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createManyHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
    updateMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateManyHelper({ info, input, objectType, req });
    },
    acceptOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        throw new CustomError("0000", "NotImplemented");
    },
    declineOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        throw new CustomError("0000", "NotImplemented");
    },
};
