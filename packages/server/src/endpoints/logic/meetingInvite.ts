import { FindByIdInput, MeetingInvite, MeetingInviteCreateInput, MeetingInviteSearchInput, MeetingInviteUpdateInput } from "@local/shared";
import { createManyHelper, createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateManyHelper, updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsMeetingInvite = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<MeetingInvite>>;
    findMany: ApiEndpoint<MeetingInviteSearchInput, FindManyResult<MeetingInvite>>;
    createOne: ApiEndpoint<MeetingInviteCreateInput, CreateOneResult<MeetingInvite>>;
    createMany: ApiEndpoint<MeetingInviteCreateInput[], CreateOneResult<MeetingInvite>[]>;
    updateOne: ApiEndpoint<MeetingInviteUpdateInput, UpdateOneResult<MeetingInvite>>;
    updateMany: ApiEndpoint<MeetingInviteUpdateInput[], UpdateOneResult<MeetingInvite>[]>;
    acceptOne: ApiEndpoint<FindByIdInput, UpdateOneResult<MeetingInvite>>;
    declineOne: ApiEndpoint<FindByIdInput, UpdateOneResult<MeetingInvite>>;
}

const objectType = "MeetingInvite";
export const meetingInvite: EndpointsMeetingInvite = {
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
        throw new CustomError("0000", "NotImplemented");
    },
    declineOne: async (_, { input }, { req }, info) => {
        throw new CustomError("0000", "NotImplemented");
    },
};
