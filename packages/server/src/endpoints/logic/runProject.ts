import { Count, FindByIdInput, RunProject, RunProjectCreateInput, RunProjectSearchInput, RunProjectUpdateInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { RunProjectModel } from "../../models/base/runProject";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsRunProject = {
    Query: {
        runProject: ApiEndpoint<FindByIdInput, FindOneResult<RunProject>>;
        runProjects: ApiEndpoint<RunProjectSearchInput, FindManyResult<RunProject>>;
    },
    Mutation: {
        runProjectCreate: ApiEndpoint<RunProjectCreateInput, CreateOneResult<RunProject>>;
        runProjectUpdate: ApiEndpoint<RunProjectUpdateInput, UpdateOneResult<RunProject>>;
        runProjectDeleteAll: ApiEndpoint<Record<string, never>, Count>;
    }
}

const objectType = "RunProject";
export const RunProjectEndpoints: EndpointsRunProject = {
    Query: {
        runProject: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        runProjects: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        runProjectCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return createOneHelper({ info, input, objectType, req });
        },
        runProjectUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        runProjectDeleteAll: async (_p, _d, { req }) => {
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 25, req });
            return (RunProjectModel as any).danger.deleteAll({ __typename: "User", id: userData.id });
        },
    },
};
