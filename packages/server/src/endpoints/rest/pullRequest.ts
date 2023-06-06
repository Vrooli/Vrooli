import { pullRequest_create, pullRequest_findMany, pullRequest_findOne, pullRequest_update } from "../generated";
import { PullRequestEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const PullRequestRest = setupRoutes({
    "/pullRequest/:id": {
        get: [PullRequestEndpoints.Query.pullRequest, pullRequest_findOne],
        put: [PullRequestEndpoints.Mutation.pullRequestUpdate, pullRequest_update],
    },
    "/pullRequests": {
        get: [PullRequestEndpoints.Query.pullRequests, pullRequest_findMany],
    },
    "/pullRequest": {
        post: [PullRequestEndpoints.Mutation.pullRequestCreate, pullRequest_create],
    },
});
