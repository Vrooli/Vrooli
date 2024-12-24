import { FindByIdInput, MemberInvite, MemberInviteCreateInput, MemberInviteSearchInput, MemberInviteUpdateInput } from "@local/shared";
import { createManyHelper, createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateManyHelper, updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsMemberInvite = {
    Query: {
        memberInvite: ApiEndpoint<FindByIdInput, FindOneResult<MemberInvite>>;
        memberInvites: ApiEndpoint<MemberInviteSearchInput, FindManyResult<MemberInvite>>;
    },
    Mutation: {
        memberInviteCreate: ApiEndpoint<MemberInviteCreateInput, CreateOneResult<MemberInvite>>;
        memberInvitesCreate: ApiEndpoint<MemberInviteCreateInput[], CreateOneResult<MemberInvite>[]>;
        memberInviteUpdate: ApiEndpoint<MemberInviteUpdateInput, UpdateOneResult<MemberInvite>>;
        memberInvitesUpdate: ApiEndpoint<MemberInviteUpdateInput[], UpdateOneResult<MemberInvite>[]>;
        memberInviteAccept: ApiEndpoint<FindByIdInput, UpdateOneResult<MemberInvite>>;
        memberInviteDecline: ApiEndpoint<FindByIdInput, UpdateOneResult<MemberInvite>>;
    }
}

const objectType = "MemberInvite";
export const MemberInviteEndpoints: EndpointsMemberInvite = {
    Query: {
        memberInvite: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        memberInvites: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        memberInviteCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        memberInvitesCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createManyHelper({ info, input, objectType, req });
        },
        memberInviteUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        memberInvitesUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateManyHelper({ info, input, objectType, req });
        },
        memberInviteAccept: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented");
        },
        memberInviteDecline: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented");
        },
    },
};
