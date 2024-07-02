import { endpointGetPullRequest, endpointGetPullRequests, PullRequestSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const pullRequestSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchPullRequest"),
    containers: [], //TODO
    elements: [], //TODO
});

export const pullRequestSearchParams = () => toParams(pullRequestSearchSchema(), endpointGetPullRequests, endpointGetPullRequest, PullRequestSortBy, PullRequestSortBy.DateCreatedDesc);
