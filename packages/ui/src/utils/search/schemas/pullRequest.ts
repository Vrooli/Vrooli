import { PullRequestSortBy, endpointsPullRequest, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

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
