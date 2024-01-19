import { FindByIdInput, ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectorySearchInput, ProjectVersionDirectoryUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsProjectVersionDirectory = {
    Query: {
        projectVersionDirectory: GQLEndpoint<FindByIdInput, FindOneResult<ProjectVersionDirectory>>;
        projectVersionDirectories: GQLEndpoint<ProjectVersionDirectorySearchInput, FindManyResult<ProjectVersionDirectory>>;
    },
    Mutation: {
        projectVersionDirectoryCreate: GQLEndpoint<ProjectVersionDirectoryCreateInput, CreateOneResult<ProjectVersionDirectory>>;
        projectVersionDirectoryUpdate: GQLEndpoint<ProjectVersionDirectoryUpdateInput, UpdateOneResult<ProjectVersionDirectory>>;
    }
}

const objectType = "ProjectVersionDirectory";
export const ProjectVersionDirectoryEndpoints: EndpointsProjectVersionDirectory = {
    Query: {
        projectVersionDirectory: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        projectVersionDirectories: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        projectVersionDirectoryCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, prisma, req });
        },
        projectVersionDirectoryUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, prisma, req });
        },
    },
};
