import { FindByIdInput, PullRequest, PullRequestCreateInput, PullRequestSearchInput, PullRequestSearchResult, PullRequestUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsPullRequest = {
    findOne: ApiEndpoint<FindByIdInput, PullRequest>;
    findMany: ApiEndpoint<PullRequestSearchInput, PullRequestSearchResult>;
    createOne: ApiEndpoint<PullRequestCreateInput, PullRequest>;
    updateOne: ApiEndpoint<PullRequestUpdateInput, PullRequest>;
}

const objectType = "PullRequest";
export const pullRequest: EndpointsPullRequest = {
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
        // TODO make sure to set hasBeenClosedOrRejected to true if status is closed or rejected
        // TODO 2 permissions for this differ from normal objects. Some fields can be updated by creator, and some by owner of object the pull request is for. Probably need to make custom endpoints like for transfers
    },
};
