import { FindByIdInput, Issue, IssueCloseInput, IssueCreateInput, IssueSearchInput, IssueSearchResult, IssueUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { ApiEndpoint } from "../../types";

export type EndpointsIssue = {
    findOne: ApiEndpoint<FindByIdInput, Issue>;
    findMany: ApiEndpoint<IssueSearchInput, IssueSearchResult>;
    createOne: ApiEndpoint<IssueCreateInput, Issue>;
    updateOne: ApiEndpoint<IssueUpdateInput, Issue>;
    closeOne: ApiEndpoint<IssueCloseInput, Issue>;
}

const objectType = "Issue";
export const issue: EndpointsIssue = {
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
    closeOne: async ({ input }, { req }, info) => {
        throw new CustomError("0000", "NotImplemented");
        // TODO make sure to set hasBeenClosedOrRejected to true
    },
};
