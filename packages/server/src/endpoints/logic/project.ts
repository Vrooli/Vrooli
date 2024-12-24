import { FindByIdOrHandleInput, Project, ProjectCreateInput, ProjectSearchInput, ProjectUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsProject = {
    Query: {
        project: ApiEndpoint<FindByIdOrHandleInput, FindOneResult<Project>>;
        projects: ApiEndpoint<ProjectSearchInput, FindManyResult<Project>>;
    },
    Mutation: {
        projectCreate: ApiEndpoint<ProjectCreateInput, CreateOneResult<Project>>;
        projectUpdate: ApiEndpoint<ProjectUpdateInput, UpdateOneResult<Project>>;
    }
}

const objectType = "Project";
export const ProjectEndpoints: EndpointsProject = {
    Query: {
        project: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        projects: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        projectCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        projectUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
