import { FindByIdInput, ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectorySearchInput, ProjectVersionDirectoryUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsProjectVersionDirectory = {
    Query: {
        projectVersionDirectory: ApiEndpoint<FindByIdInput, FindOneResult<ProjectVersionDirectory>>;
        projectVersionDirectories: ApiEndpoint<ProjectVersionDirectorySearchInput, FindManyResult<ProjectVersionDirectory>>;
    },
    Mutation: {
        projectVersionDirectoryCreate: ApiEndpoint<ProjectVersionDirectoryCreateInput, CreateOneResult<ProjectVersionDirectory>>;
        projectVersionDirectoryUpdate: ApiEndpoint<ProjectVersionDirectoryUpdateInput, UpdateOneResult<ProjectVersionDirectory>>;
    }
}

const objectType = "ProjectVersionDirectory";
export const ProjectVersionDirectoryEndpoints: EndpointsProjectVersionDirectory = {
    Query: {
        projectVersionDirectory: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        projectVersionDirectories: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        projectVersionDirectoryCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        projectVersionDirectoryUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
