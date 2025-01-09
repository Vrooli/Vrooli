import { endpointsPullRequest, FormSchema, PullRequestSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function pullRequestSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchPullRequest"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function pullRequestSearchParams() {
    return toParams(pullRequestSearchSchema(), endpointsPullRequest, PullRequestSortBy, PullRequestSortBy.DateCreatedDesc);
}
