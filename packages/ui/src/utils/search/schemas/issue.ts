import { IssueSortBy } from "@shared/consts";
import { issueFindMany } from "api/generated/endpoints/issue";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, votesContainer, votesFields } from "./common";

export const issueSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchIssue', lng),
    containers: [
        votesContainer(lng),
        bookmarksContainer(lng),
        languagesContainer(lng),
    ],
    fields: [
        ...votesFields(lng),
        ...bookmarksFields(lng),
        ...languagesFields(lng),
    ]
})

export const issueSearchParams = (lng: string) => toParams(issueSearchSchema(lng), issueFindMany, IssueSortBy, IssueSortBy.ScoreDesc);