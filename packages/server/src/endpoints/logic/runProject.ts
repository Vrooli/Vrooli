import { FindByIdInput, RunProject, RunProjectCreateInput, RunProjectSearchInput, RunProjectUpdateInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsRunProject = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<RunProject>>;
    findMany: ApiEndpoint<RunProjectSearchInput, FindManyResult<RunProject>>;
    createOne: ApiEndpoint<RunProjectCreateInput, CreateOneResult<RunProject>>;
    updateOne: ApiEndpoint<RunProjectUpdateInput, UpdateOneResult<RunProject>>;
}

const objectType = "RunProject";
export const runProject: EndpointsRunProject = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
    },
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
