import { PullRequestSortBy } from "@local/consts";
import { pullRequestFindMany } from "../../../api/generated/endpoints/pullRequest_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const pullRequestSearchSchema = () => ({
    formLayout: searchFormLayout("SearchPullRequest"),
    containers: [],
    fields: [],
});
export const pullRequestSearchParams = () => toParams(pullRequestSearchSchema(), pullRequestFindMany, PullRequestSortBy, PullRequestSortBy.DateCreatedDesc);
//# sourceMappingURL=pullRequest.js.map