import { FindVersionInput, StandardVersion, StandardVersionCreateInput, StandardVersionSearchInput, StandardVersionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
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
        standardVersion: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        standardVersions: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        standardVersionCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return createOneHelper({ info, input, objectType, req });
        },
        standardVersionUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
