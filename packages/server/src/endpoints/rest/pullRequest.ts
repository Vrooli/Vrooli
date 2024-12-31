import { endpointsPullRequest } from "@local/shared";
import { pullRequest_create, pullRequest_findMany, pullRequest_findOne, pullRequest_update } from "../generated";
import { PullRequestEndpoints } from "../logic/pullRequest";
import { setupRoutes } from "./base";

export const PullRequestRest = setupRoutes([
    [endpointsPullRequest.findOne, PullRequestEndpoints.Query.pullRequest, pullRequest_findOne],
    [endpointsPullRequest.findMany, PullRequestEndpoints.Query.pullRequests, pullRequest_findMany],
    [endpointsPullRequest.createOne, PullRequestEndpoints.Mutation.pullRequestCreate, pullRequest_create],
    [endpointsPullRequest.updateOne, PullRequestEndpoints.Mutation.pullRequestUpdate, pullRequest_update],
]);
