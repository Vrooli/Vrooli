import { FindByIdInput, Issue, IssueCloseInput, IssueCreateInput, IssueSearchInput, IssueUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsIssue = {
    Query: {
        issue: ApiEndpoint<FindByIdInput, FindOneResult<Issue>>;
        issues: ApiEndpoint<IssueSearchInput, FindManyResult<Issue>>;
    },
    Mutation: {
        issueCreate: ApiEndpoint<IssueCreateInput, CreateOneResult<Issue>>;
        issueUpdate: ApiEndpoint<IssueUpdateInput, UpdateOneResult<Issue>>;
        issueClose: ApiEndpoint<IssueCloseInput, UpdateOneResult<Issue>>;
    }
}

const objectType = "Issue";
export const IssueEndpoints: EndpointsIssue = {
    Query: {
        issue: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        issues: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        issueCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        issueUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        issueClose: async (_, { input }, { req }, info) => {
            throw new CustomError("0000", "NotImplemented");
            // TODO make sure to set hasBeenClosedOrRejected to true
        },
    },
};
