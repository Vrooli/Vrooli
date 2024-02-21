import { FindByIdInput, Standard, StandardCreateInput, StandardSearchInput, StandardUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsStandard = {
    Query: {
        standard: GQLEndpoint<FindByIdInput, FindOneResult<Standard>>;
        standards: GQLEndpoint<StandardSearchInput, FindManyResult<Standard>>;
    },
    Mutation: {
        standardCreate: GQLEndpoint<StandardCreateInput, CreateOneResult<Standard>>;
        standardUpdate: GQLEndpoint<StandardUpdateInput, UpdateOneResult<Standard>>;
    }
}

const objectType = "Standard";
export const StandardEndpoints: EndpointsStandard = {
    Query: {
        standard: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        standards: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        standardCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return createOneHelper({ info, input, objectType, prisma, req });
        },
        standardUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return updateOneHelper({ info, input, objectType, prisma, req });
        },
    },
};
