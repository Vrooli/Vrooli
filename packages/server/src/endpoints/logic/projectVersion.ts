import { FindVersionInput, ProjectVersion, ProjectVersionContentsSearchInput, ProjectVersionContentsSearchResult, ProjectVersionCreateInput, ProjectVersionSearchInput, ProjectVersionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsProjectVersion = {
    findOne: ApiEndpoint<FindVersionInput, FindOneResult<ProjectVersion>>;
    findMany: ApiEndpoint<ProjectVersionSearchInput, FindManyResult<ProjectVersion>>;
    contents: ApiEndpoint<ProjectVersionContentsSearchInput, FindManyResult<ProjectVersionContentsSearchResult>>;
    createOne: ApiEndpoint<ProjectVersionCreateInput, CreateOneResult<ProjectVersion>>;
    updateOne: ApiEndpoint<ProjectVersionUpdateInput, UpdateOneResult<ProjectVersion>>;
}

const objectType = "ProjectVersion";
export const projectVersion: EndpointsProjectVersion = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    contents: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        throw new CustomError("0000", "NotImplemented");
        // return ProjectVersionModel.query.searchContents(req, input, info);
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
