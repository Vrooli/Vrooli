import { FindVersionInput, ProjectVersion, ProjectVersionContentsSearchInput, ProjectVersionContentsSearchResult, ProjectVersionCreateInput, ProjectVersionSearchInput, ProjectVersionSearchResult, ProjectVersionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { ApiEndpoint } from "../../types";

export type EndpointsProjectVersion = {
    findOne: ApiEndpoint<FindVersionInput, ProjectVersion>;
    findMany: ApiEndpoint<ProjectVersionSearchInput, ProjectVersionSearchResult>;
    contents: ApiEndpoint<ProjectVersionContentsSearchInput, ProjectVersionContentsSearchResult>;
    createOne: ApiEndpoint<ProjectVersionCreateInput, ProjectVersion>;
    updateOne: ApiEndpoint<ProjectVersionUpdateInput, ProjectVersion>;
}

const objectType = "ProjectVersion";
export const projectVersion: EndpointsProjectVersion = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    contents: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        throw new CustomError("0000", "NotImplemented");
        // return ProjectVersionModel.query.searchContents(req, input, info);
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
