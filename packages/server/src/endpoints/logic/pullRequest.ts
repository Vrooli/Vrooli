import { FindByIdInput, PullRequest, PullRequestCreateInput, PullRequestSearchInput, PullRequestUpdateInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsPullRequest = {
    Query: {
        pullRequest: GQLEndpoint<FindByIdInput, FindOneResult<PullRequest>>;
        pullRequests: GQLEndpoint<PullRequestSearchInput, FindManyResult<PullRequest>>;
    },
    Mutation: {
        pullRequestCreate: GQLEndpoint<PullRequestCreateInput, CreateOneResult<PullRequest>>;
        pullRequestUpdate: GQLEndpoint<PullRequestUpdateInput, UpdateOneResult<PullRequest>>;
    }
}

const objectType = "PullRequest";
export const PullRequestEndpoints: EndpointsPullRequest = {
    Query: {
        pullRequest: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        pullRequests: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        pullRequestCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        pullRequestUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
            // TODO make sure to set hasBeenClosedOrRejected to true if status is closed or rejected
            // TODO 2 permissions for this differ from normal objects. Some fields can be updated by creator, and some by owner of object the pull request is for. Probably need to make custom endpoints like for transfers
        },
    },
};
