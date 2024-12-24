import { FindByIdInput, MeetingInvite, MeetingInviteCreateInput, MeetingInviteSearchInput, MeetingInviteUpdateInput } from "@local/shared";
import { createManyHelper, createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateManyHelper, updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsMeetingInvite = {
    Query: {
        meetingInvite: ApiEndpoint<FindByIdInput, FindOneResult<MeetingInvite>>;
        meetingInvites: ApiEndpoint<MeetingInviteSearchInput, FindManyResult<MeetingInvite>>;
    },
    Mutation: {
        meetingInviteCreate: ApiEndpoint<MeetingInviteCreateInput, CreateOneResult<MeetingInvite>>;
        meetingInvitesCreate: ApiEndpoint<MeetingInviteCreateInput[], CreateOneResult<MeetingInvite>[]>;
        meetingInviteUpdate: ApiEndpoint<MeetingInviteUpdateInput, UpdateOneResult<MeetingInvite>>;
        meetingInvitesUpdate: ApiEndpoint<MeetingInviteUpdateInput[], UpdateOneResult<MeetingInvite>[]>;
        meetingInviteAccept: ApiEndpoint<FindByIdInput, UpdateOneResult<MeetingInvite>>;
        meetingInviteDecline: ApiEndpoint<FindByIdInput, UpdateOneResult<MeetingInvite>>;
    }
}

const objectType = "MeetingInvite";
export const MeetingInviteEndpoints: EndpointsMeetingInvite = {
    Query: {
        meetingInvite: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        meetingInvites: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        meetingInviteCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        meetingInvitesCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createManyHelper({ info, input, objectType, req });
        },
        meetingInviteUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        meetingInvitesUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateManyHelper({ info, input, objectType, req });
        },
        meetingInviteAccept: async (_, { input }, { req }, info) => {
            throw new CustomError("0000", "NotImplemented");
        },
        meetingInviteDecline: async (_, { input }, { req }, info) => {
            throw new CustomError("0000", "NotImplemented");
        },
    },
};
