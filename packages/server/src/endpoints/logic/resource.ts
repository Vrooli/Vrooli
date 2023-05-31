import { FindByIdInput, Resource, ResourceCreateInput, ResourceSearchInput, ResourceUpdateInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsResource = {
    Query: {
        resource: GQLEndpoint<FindByIdInput, FindOneResult<Resource>>;
        resources: GQLEndpoint<ResourceSearchInput, FindManyResult<Resource>>;
    },
    Mutation: {
        resourceCreate: GQLEndpoint<ResourceCreateInput, CreateOneResult<Resource>>;
        resourceUpdate: GQLEndpoint<ResourceUpdateInput, UpdateOneResult<Resource>>;
    }
}

const objectType = "Resource";
export const ResourceEndpoints: EndpointsResource = {
    Query: {
        resource: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        resources: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        resourceCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        resourceUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
