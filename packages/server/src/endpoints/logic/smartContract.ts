import { FindByIdInput, SmartContract, SmartContractCreateInput, SmartContractSearchInput, SmartContractUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsSmartContract = {
    Query: {
        smartContract: GQLEndpoint<FindByIdInput, FindOneResult<SmartContract>>;
        smartContracts: GQLEndpoint<SmartContractSearchInput, FindManyResult<SmartContract>>;
    },
    Mutation: {
        smartContractCreate: GQLEndpoint<SmartContractCreateInput, CreateOneResult<SmartContract>>;
        smartContractUpdate: GQLEndpoint<SmartContractUpdateInput, UpdateOneResult<SmartContract>>;
    }
}

const objectType = "SmartContract";
export const SmartContractEndpoints: EndpointsSmartContract = {
    Query: {
        smartContract: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        smartContracts: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        smartContractCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        smartContractUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
