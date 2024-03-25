import { FindVersionInput, SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionSearchInput, SmartContractVersionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsSmartContractVersion = {
    Query: {
        smartContractVersion: GQLEndpoint<FindVersionInput, FindOneResult<SmartContractVersion>>;
        smartContractVersions: GQLEndpoint<SmartContractVersionSearchInput, FindManyResult<SmartContractVersion>>;
    },
    Mutation: {
        smartContractVersionCreate: GQLEndpoint<SmartContractVersionCreateInput, CreateOneResult<SmartContractVersion>>;
        smartContractVersionUpdate: GQLEndpoint<SmartContractVersionUpdateInput, UpdateOneResult<SmartContractVersion>>;
    }
}

const objectType = "SmartContractVersion";
export const SmartContractVersionEndpoints: EndpointsSmartContractVersion = {
    Query: {
        smartContractVersion: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        smartContractVersions: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        smartContractVersionCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        smartContractVersionUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
