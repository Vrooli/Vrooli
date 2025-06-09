import { type FindByPublicIdInput, type Issue, type IssueCloseInput, type IssueCreateInput, type IssueSearchInput, type IssueSearchResult, type IssueUpdateInput } from "@vrooli/shared";
import { CustomError } from "../../events/error.js";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsIssue = {
    findOne: ApiEndpoint<FindByPublicIdInput, Issue>;
    findMany: ApiEndpoint<IssueSearchInput, IssueSearchResult>;
    createOne: ApiEndpoint<IssueCreateInput, Issue>;
    updateOne: ApiEndpoint<IssueUpdateInput, Issue>;
    closeOne: ApiEndpoint<IssueCloseInput, Issue>;
}

export const issue: EndpointsIssue = createStandardCrudEndpoints({
    objectType: "Issue",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            // No permissions required for public issues
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            // No permissions required for public issues
        },
        createOne: {
            rateLimit: RateLimitPresets.STRICT,
            // No permissions required for creating issues
        },
        updateOne: {
            rateLimit: RateLimitPresets.LOW,
            // No permissions required for updating issues
        },
    },
    customEndpoints: {
        closeOne: async (wrapped, { req }, info) => {
            throw new CustomError("0000", "NotImplemented");
        },
    },
});
