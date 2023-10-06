import { FindByIdInput, SmartContract, SmartContractCreateInput, SmartContractSearchInput, SmartContractUpdateInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
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
        smartContract: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        smartContracts: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        smartContractCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        smartContractUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
