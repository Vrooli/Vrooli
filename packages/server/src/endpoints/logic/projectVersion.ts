import { FindVersionInput, ProjectVersion, ProjectVersionContentsSearchInput, ProjectVersionContentsSearchResult, ProjectVersionCreateInput, ProjectVersionSearchInput, ProjectVersionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
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
        projectVersion: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        projectVersions: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
        projectVersionContents: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            throw new CustomError("0000", "NotImplemented");
            // return ProjectVersionModel.query.searchContents(req, input, info);
        },
    },
    Mutation: {
        projectVersionCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        projectVersionUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
