import { Count, FindByIdInput, RunProject, RunProjectCancelInput, RunProjectCompleteInput, RunProjectCreateInput, RunProjectSearchInput, RunProjectUpdateInput, VisibilityType } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { assertRequestFrom } from "../../auth/request";
import { rateLimit } from "../../middleware/rateLimit";
import { RunProjectModel } from "../../models/base/runProject";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, RecursivePartial, UpdateOneResult } from "../../types";

export type EndpointsRunProject = {
    Query: {
        runProject: GQLEndpoint<FindByIdInput, FindOneResult<RunProject>>;
        runProjects: GQLEndpoint<RunProjectSearchInput, FindManyResult<RunProject>>;
    },
    Mutation: {
        runProjectCreate: GQLEndpoint<RunProjectCreateInput, CreateOneResult<RunProject>>;
        runProjectUpdate: GQLEndpoint<RunProjectUpdateInput, UpdateOneResult<RunProject>>;
        runProjectDeleteAll: GQLEndpoint<Record<string, never>, Count>;
        runProjectComplete: GQLEndpoint<RunProjectCompleteInput, RecursivePartial<RunProject>>;
        runProjectCancel: GQLEndpoint<RunProjectCancelInput, RecursivePartial<RunProject>>;
    }
}

const objectType = "RunProject";
export const RunProjectEndpoints: EndpointsRunProject = {
    Query: {
        runProject: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        runProjects: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        runProjectCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        runProjectUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        runProjectDeleteAll: async (_p, _d, { prisma, req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 25, req });
            return (RunProjectModel as any).danger.deleteAll(prisma, { __typename: "User", id: userData.id });
        },
        runProjectComplete: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });
            return (RunProjectModel as any).run.complete(prisma, userData, input, info);
        },
        runProjectCancel: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });
            return (RunProjectModel as any).run.cancel(prisma, userData, input, info);
        },
    },
};
