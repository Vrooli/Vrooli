import { FindByIdInput, MemberInvite, MemberInviteCreateInput, MemberInviteSearchInput, MemberInviteUpdateInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { CustomError } from "../../events";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsMemberInvite = {
    Query: {
        memberInvite: GQLEndpoint<FindByIdInput, FindOneResult<MemberInvite>>;
        memberInvites: GQLEndpoint<MemberInviteSearchInput, FindManyResult<MemberInvite>>;
    },
    Mutation: {
        memberInviteCreate: GQLEndpoint<MemberInviteCreateInput, CreateOneResult<MemberInvite>>;
        memberInviteUpdate: GQLEndpoint<MemberInviteUpdateInput, UpdateOneResult<MemberInvite>>;
        memberInviteAccept: GQLEndpoint<FindByIdInput, UpdateOneResult<MemberInvite>>;
        memberInviteDecline: GQLEndpoint<FindByIdInput, UpdateOneResult<MemberInvite>>;
    }
}

const objectType = "MemberInvite";
export const MemberInviteEndpoints: EndpointsMemberInvite = {
    Query: {
        memberInvite: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        memberInvites: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        memberInviteCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        memberInviteUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        memberInviteAccept: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
        memberInviteDecline: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
    },
};
