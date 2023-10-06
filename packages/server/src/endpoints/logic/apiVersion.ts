import { ApiVersion, ApiVersionCreateInput, ApiVersionSearchInput, ApiVersionUpdateInput, FindVersionInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsApiVersion = {
    Query: {
        apiVersion: GQLEndpoint<FindVersionInput, FindOneResult<ApiVersion>>;
        apiVersions: GQLEndpoint<ApiVersionSearchInput, FindManyResult<ApiVersion>>;
    },
    Mutation: {
        apiVersionCreate: GQLEndpoint<ApiVersionCreateInput, CreateOneResult<ApiVersion>>;
        apiVersionUpdate: GQLEndpoint<ApiVersionUpdateInput, UpdateOneResult<ApiVersion>>;
    }
}

const objectType = "ApiVersion";
export const ApiVersionEndpoints: EndpointsApiVersion = {
    Query: {
        apiVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        apiVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        apiVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        apiVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};

