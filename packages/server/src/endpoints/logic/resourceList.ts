import { FindByIdInput, ResourceList, ResourceListCreateInput, ResourceListSearchInput, ResourceListUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsResourceList = {
    Query: {
        resourceList: ApiEndpoint<FindByIdInput, FindOneResult<ResourceList>>;
        resourceLists: ApiEndpoint<ResourceListSearchInput, FindManyResult<ResourceList>>;
    },
    Mutation: {
        resourceListCreate: ApiEndpoint<ResourceListCreateInput, CreateOneResult<ResourceList>>;
        resourceListUpdate: ApiEndpoint<ResourceListUpdateInput, UpdateOneResult<ResourceList>>;
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
