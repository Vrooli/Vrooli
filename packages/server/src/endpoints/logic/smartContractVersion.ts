import { FindVersionInput, SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionSearchInput, SmartContractVersionUpdateInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
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
        smartContractVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        smartContractVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        smartContractVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        smartContractVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
