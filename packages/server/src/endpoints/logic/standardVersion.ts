import { FindVersionInput, StandardVersion, StandardVersionCreateInput, StandardVersionSearchInput, StandardVersionUpdateInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsStandardVersion = {
    Query: {
        standardVersion: GQLEndpoint<FindVersionInput, FindOneResult<StandardVersion>>;
        standardVersions: GQLEndpoint<StandardVersionSearchInput, FindManyResult<StandardVersion>>;
    },
    Mutation: {
        standardVersionCreate: GQLEndpoint<StandardVersionCreateInput, CreateOneResult<StandardVersion>>;
        standardVersionUpdate: GQLEndpoint<StandardVersionUpdateInput, UpdateOneResult<StandardVersion>>;
    }
}

const objectType = "StandardVersion";
export const StandardVersionEndpoints: EndpointsStandardVersion = {
    Query: {
        standardVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        standardVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        standardVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        standardVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
