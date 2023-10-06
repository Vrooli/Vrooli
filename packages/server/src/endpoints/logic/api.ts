import { Api, ApiCreateInput, ApiSearchInput, ApiUpdateInput, FindByIdInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsApi = {
    Query: {
        api: GQLEndpoint<FindByIdInput, FindOneResult<Api>>;
        apis: GQLEndpoint<ApiSearchInput, FindManyResult<Api>>;
    },
    Mutation: {
        apiCreate: GQLEndpoint<ApiCreateInput, CreateOneResult<Api>>;
        apiUpdate: GQLEndpoint<ApiUpdateInput, UpdateOneResult<Api>>;
    }
}

const objectType = "Api";
export const ApiEndpoints: EndpointsApi = {
    Query: {
        api: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        apis: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        apiCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        apiUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
