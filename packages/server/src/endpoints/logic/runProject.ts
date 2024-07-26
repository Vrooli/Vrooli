import { Count, FindByIdInput, RunProject, RunProjectCreateInput, RunProjectSearchInput, RunProjectUpdateInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { assertRequestFrom } from "../../auth/request";
import { rateLimit } from "../../middleware/rateLimit";
import { RunProjectModel } from "../../models/base/runProject";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsRunProject = {
    Query: {
        runProject: GQLEndpoint<FindByIdInput, FindOneResult<RunProject>>;
        runProjects: GQLEndpoint<RunProjectSearchInput, FindManyResult<RunProject>>;
    },
    Mutation: {
        runProjectCreate: GQLEndpoint<RunProjectCreateInput, CreateOneResult<RunProject>>;
        runProjectUpdate: GQLEndpoint<RunProjectUpdateInput, UpdateOneResult<RunProject>>;
        runProjectDeleteAll: GQLEndpoint<Record<string, never>, Count>;
    }
}

const objectType = "RunProject";
export const RunProjectEndpoints: EndpointsRunProject = {
    Query: {
        runProject: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        runProjects: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        runProjectCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return createOneHelper({ info, input, objectType, req });
        },
        runProjectUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        runProjectDeleteAll: async (_p, _d, { req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 25, req });
            return (RunProjectModel as any).danger.deleteAll({ __typename: "User", id: userData.id });
        },
    },
};
