import { ApiVersion, ApiVersionCreateInput, ApiVersionSearchInput, ApiVersionUpdateInput, FindVersionInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
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
        apiVersion: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        apiVersions: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        apiVersionCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        apiVersionUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};

