import { FindByIdInput, Member, MemberSearchInput, MemberUpdateInput } from "@local/shared";
import { readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";


export type EndpointsMember = {
    Query: {
        member: GQLEndpoint<FindByIdInput, FindOneResult<Member>>;
        members: GQLEndpoint<MemberSearchInput, FindManyResult<Member>>;
    },
    Mutation: {
        memberUpdate: GQLEndpoint<MemberUpdateInput, UpdateOneResult<Member>>;
    }
}

const objectType = "Member";
export const MemberEndpoints: EndpointsMember = {
    Query: {
        member: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        members: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        memberUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
