import { FindByIdInput, PullRequest, PullRequestCreateInput, PullRequestSearchInput, PullRequestUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsPullRequest = {
    Query: {
        pullRequest: ApiEndpoint<FindByIdInput, FindOneResult<PullRequest>>;
        pullRequests: ApiEndpoint<PullRequestSearchInput, FindManyResult<PullRequest>>;
    },
    Mutation: {
        pullRequestCreate: ApiEndpoint<PullRequestCreateInput, CreateOneResult<PullRequest>>;
        pullRequestUpdate: ApiEndpoint<PullRequestUpdateInput, UpdateOneResult<PullRequest>>;
    }
}

const objectType = "PullRequest";
export const PullRequestEndpoints: EndpointsPullRequest = {
    Query: {
        pullRequest: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        pullRequests: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        pullRequestCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        pullRequestUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
            // TODO make sure to set hasBeenClosedOrRejected to true if status is closed or rejected
            // TODO 2 permissions for this differ from normal objects. Some fields can be updated by creator, and some by owner of object the pull request is for. Probably need to make custom endpoints like for transfers
        },
    },
};
