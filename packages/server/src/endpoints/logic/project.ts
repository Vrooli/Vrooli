import { FindByIdOrHandleInput, Project, ProjectCreateInput, ProjectSearchInput, ProjectUpdateInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsProject = {
    Query: {
        project: GQLEndpoint<FindByIdOrHandleInput, FindOneResult<Project>>;
        projects: GQLEndpoint<ProjectSearchInput, FindManyResult<Project>>;
    },
    Mutation: {
        projectCreate: GQLEndpoint<ProjectCreateInput, CreateOneResult<Project>>;
        projectUpdate: GQLEndpoint<ProjectUpdateInput, UpdateOneResult<Project>>;
    }
}

const objectType = "Project";
export const ProjectEndpoints: EndpointsProject = {
    Query: {
        project: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        projects: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        projectCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        projectUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
