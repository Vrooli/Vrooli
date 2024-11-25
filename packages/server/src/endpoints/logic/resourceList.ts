import { FindByIdInput, ResourceList, ResourceListCreateInput, ResourceListSearchInput, ResourceListUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
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
        resourceList: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        resourceLists: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        resourceListCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        resourceListUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
