import { FindByIdInput, ResourceList, ResourceListCreateInput, ResourceListSearchInput, ResourceListUpdateInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsResourceList = {
    Query: {
        resourceList: GQLEndpoint<FindByIdInput, FindOneResult<ResourceList>>;
        resourceLists: GQLEndpoint<ResourceListSearchInput, FindManyResult<ResourceList>>;
    },
    Mutation: {
        resourceListCreate: GQLEndpoint<ResourceListCreateInput, CreateOneResult<ResourceList>>;
        resourceListUpdate: GQLEndpoint<ResourceListUpdateInput, UpdateOneResult<ResourceList>>;
    }
}

const objectType = "ResourceList";
export const ResourceListEndpoints: EndpointsResourceList = {
    Query: {
        resourceList: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        resourceLists: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        resourceListCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        resourceListUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
