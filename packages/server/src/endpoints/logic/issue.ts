import { FindByIdInput, Issue, IssueCloseInput, IssueCreateInput, IssueSearchInput, IssueUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { CustomError } from "../../events/error";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsIssue = {
    Query: {
        issue: GQLEndpoint<FindByIdInput, FindOneResult<Issue>>;
        issues: GQLEndpoint<IssueSearchInput, FindManyResult<Issue>>;
    },
    Mutation: {
        issueCreate: GQLEndpoint<IssueCreateInput, CreateOneResult<Issue>>;
        issueUpdate: GQLEndpoint<IssueUpdateInput, UpdateOneResult<Issue>>;
        issueClose: GQLEndpoint<IssueCloseInput, UpdateOneResult<Issue>>;
    }
}

const objectType = "Issue";
export const IssueEndpoints: EndpointsIssue = {
    Query: {
        issue: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        issues: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        issueCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, prisma, req });
        },
        issueUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, prisma, req });
        },
        issueClose: async (_, { input }, { prisma, req }, info) => {
            throw new CustomError("0000", "NotImplemented", ["en"]);
            // TODO make sure to set hasBeenClosedOrRejected to true
        },
    },
};
