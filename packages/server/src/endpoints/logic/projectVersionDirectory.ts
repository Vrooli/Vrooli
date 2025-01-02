import { FindByIdInput, ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectorySearchInput, ProjectVersionDirectoryUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsProjectVersionDirectory = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<ProjectVersionDirectory>>;
    findMany: ApiEndpoint<ProjectVersionDirectorySearchInput, FindManyResult<ProjectVersionDirectory>>;
    createOne: ApiEndpoint<ProjectVersionDirectoryCreateInput, CreateOneResult<ProjectVersionDirectory>>;
    updateOne: ApiEndpoint<ProjectVersionDirectoryUpdateInput, UpdateOneResult<ProjectVersionDirectory>>;
}

const objectType = "ProjectVersionDirectory";
export const projectVersionDirectory: EndpointsProjectVersionDirectory = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
