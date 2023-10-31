import { FindByIdInput, MeetingInvite, MeetingInviteCreateInput, MeetingInviteSearchInput, MeetingInviteUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { CustomError } from "../../events/error";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsMeetingInvite = {
    Query: {
        meetingInvite: GQLEndpoint<FindByIdInput, FindOneResult<MeetingInvite>>;
        meetingInvites: GQLEndpoint<MeetingInviteSearchInput, FindManyResult<MeetingInvite>>;
    },
    Mutation: {
        meetingInviteCreate: GQLEndpoint<MeetingInviteCreateInput, CreateOneResult<MeetingInvite>>;
        meetingInviteUpdate: GQLEndpoint<MeetingInviteUpdateInput, UpdateOneResult<MeetingInvite>>;
        meetingInviteAccept: GQLEndpoint<FindByIdInput, UpdateOneResult<MeetingInvite>>;
        meetingInviteDecline: GQLEndpoint<FindByIdInput, UpdateOneResult<MeetingInvite>>;
    }
}

const objectType = "MeetingInvite";
export const MeetingInviteEndpoints: EndpointsMeetingInvite = {
    Query: {
        meetingInvite: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        meetingInvites: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        meetingInviteCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, prisma, req });
        },
        meetingInviteUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, prisma, req });
        },
        meetingInviteAccept: async (_, { input }, { prisma, req }, info) => {
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
        meetingInviteDecline: async (_, { input }, { prisma, req }, info) => {
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
    },
};
