import { FindByIdInput, Quiz, QuizCreateInput, QuizSearchInput, QuizUpdateInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsRole = {
    Query: {
        role: GQLEndpoint<FindByIdInput, FindOneResult<Quiz>>;
        roles: GQLEndpoint<QuizSearchInput, FindManyResult<Quiz>>;
    },
    Mutation: {
        roleCreate: GQLEndpoint<QuizCreateInput, CreateOneResult<Quiz>>;
        roleUpdate: GQLEndpoint<QuizUpdateInput, UpdateOneResult<Quiz>>;
    }
}

const objectType = "Role";
export const RoleEndpoints: EndpointsRole = {
    Query: {
        role: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        roles: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        roleCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        roleUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
