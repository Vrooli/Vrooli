import { PullRequestSortBy } from "@shared/consts";
import { FormSchema } from "forms/types";
import { pullRequestFindMany } from "../../api/generated/endpoints/pullRequest_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const pullRequestSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchPullRequest'),
    containers: [], //TODO
    fields: [], //TODO
})

export const pullRequestSearchParams = () => toParams(pullRequestSearchSchema(), pullRequestFindMany, PullRequestSortBy, PullRequestSortBy.DateCreatedDesc)