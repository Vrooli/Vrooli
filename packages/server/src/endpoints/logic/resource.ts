import { FindByIdInput, Resource, ResourceCreateInput, ResourceSearchInput, ResourceUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsResource = {
    Query: {
        resource: ApiEndpoint<FindByIdInput, FindOneResult<Resource>>;
        resources: ApiEndpoint<ResourceSearchInput, FindManyResult<Resource>>;
    },
    Mutation: {
        resourceCreate: ApiEndpoint<ResourceCreateInput, CreateOneResult<Resource>>;
        resourceUpdate: ApiEndpoint<ResourceUpdateInput, UpdateOneResult<Resource>>;
    }
}

const objectType = "Resource";
export const ResourceEndpoints: EndpointsResource = {
    Query: {
        resource: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        resources: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        resourceCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return createOneHelper({ info, input, objectType, req });
        },
        resourceUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
