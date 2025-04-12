import { FindByIdInput, RunProject, RunProjectCreateInput, RunProjectSearchInput, RunProjectSearchResult, RunProjectUpdateInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsRunProject = {
    findOne: ApiEndpoint<FindByIdInput, RunProject>;
    findMany: ApiEndpoint<RunProjectSearchInput, RunProjectSearchResult>;
    createOne: ApiEndpoint<RunProjectCreateInput, RunProject>;
    updateOne: ApiEndpoint<RunProjectUpdateInput, RunProject>;
}

const objectType = "RunProject";
export const runProject: EndpointsRunProject = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
