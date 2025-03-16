import { FindByIdInput, ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectorySearchInput, ProjectVersionDirectorySearchResult, ProjectVersionDirectoryUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsProjectVersionDirectory = {
    findOne: ApiEndpoint<FindByIdInput, ProjectVersionDirectory>;
    findMany: ApiEndpoint<ProjectVersionDirectorySearchInput, ProjectVersionDirectorySearchResult>;
    createOne: ApiEndpoint<ProjectVersionDirectoryCreateInput, ProjectVersionDirectory>;
    updateOne: ApiEndpoint<ProjectVersionDirectoryUpdateInput, ProjectVersionDirectory>;
}

const objectType = "ProjectVersionDirectory";
export const projectVersionDirectory: EndpointsProjectVersionDirectory = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
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
