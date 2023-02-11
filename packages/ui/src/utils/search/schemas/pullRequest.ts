import { PullRequestSortBy } from "@shared/consts";
import { pullRequestFindMany } from "api/generated/endpoints/pullRequest";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const pullRequestSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchPullRequests', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const pullRequestSearchParams = (lng: string) => toParams(pullRequestSearchSchema(lng), pullRequestFindMany, PullRequestSortBy, PullRequestSortBy.DateCreatedDesc)