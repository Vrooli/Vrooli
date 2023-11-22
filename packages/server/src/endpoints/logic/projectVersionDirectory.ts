import { ProjectVersionDirectory, ProjectVersionDirectorySearchInput } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { FindManyResult, GQLEndpoint } from "../../types";

export type EndpointsProjectVersionDirectory = {
    Query: {
        projectVersionDirectories: GQLEndpoint<ProjectVersionDirectorySearchInput, FindManyResult<ProjectVersionDirectory>>;
    },
}

const objectType = "ProjectVersionDirectory";
export const ProjectVersionDirectoryEndpoints: EndpointsProjectVersionDirectory = {
    Query: {
        projectVersionDirectories: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
