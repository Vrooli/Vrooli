import { FindVersionInput, ProjectVersion, ProjectVersionContentsSearchInput, ProjectVersionContentsSearchResult, ProjectVersionCreateInput, ProjectVersionSearchInput, ProjectVersionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { CustomError } from "../../events/error";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsProjectVersion = {
    Query: {
        projectVersion: GQLEndpoint<FindVersionInput, FindOneResult<ProjectVersion>>;
        projectVersions: GQLEndpoint<ProjectVersionSearchInput, FindManyResult<ProjectVersion>>;
        projectVersionContents: GQLEndpoint<ProjectVersionContentsSearchInput, FindManyResult<ProjectVersionContentsSearchResult>>;
    },
    Mutation: {
        projectVersionCreate: GQLEndpoint<ProjectVersionCreateInput, CreateOneResult<ProjectVersion>>;
        projectVersionUpdate: GQLEndpoint<ProjectVersionUpdateInput, UpdateOneResult<ProjectVersion>>;
    }
}

const objectType = "ProjectVersion";
export const ProjectVersionEndpoints: EndpointsProjectVersion = {
    Query: {
        projectVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        projectVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
        projectVersionContents: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            throw new CustomError("0000", "NotImplemented", ["en"]);
            // return ProjectVersionModel.query.searchContents(prisma, req, input, info);
        },
    },
    Mutation: {
        projectVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, prisma, req });
        },
        projectVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, prisma, req });
        },
    },
};
