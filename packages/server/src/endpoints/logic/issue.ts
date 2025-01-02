import { FindByIdInput, Issue, IssueCloseInput, IssueCreateInput, IssueSearchInput, IssueUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsIssue = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<Issue>>;
    findMany: ApiEndpoint<IssueSearchInput, FindManyResult<Issue>>;
    createOne: ApiEndpoint<IssueCreateInput, CreateOneResult<Issue>>;
    updateOne: ApiEndpoint<IssueUpdateInput, UpdateOneResult<Issue>>;
    closeOne: ApiEndpoint<IssueCloseInput, UpdateOneResult<Issue>>;
}

const objectType = "Issue";
export const issue: EndpointsIssue = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
    closeOne: async (_, { input }, { req }, info) => {
        throw new CustomError("0000", "NotImplemented");
        // TODO make sure to set hasBeenClosedOrRejected to true
    },
};
