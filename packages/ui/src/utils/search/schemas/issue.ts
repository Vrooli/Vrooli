import { IssueSortBy } from "@shared/consts";
import { issueFindMany } from "api/generated/endpoints/issue";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const issueSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchIssue', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const issueSearchParams = (lng: string) => toParams(issueSearchSchema(lng), issueFindMany, IssueSortBy, IssueSortBy.ScoreDesc);