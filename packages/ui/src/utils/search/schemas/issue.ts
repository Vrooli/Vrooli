import { IssueSortBy } from "@shared/consts";
import { issueFindMany } from "api/generated/endpoints/issue";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, votesContainer, votesFields } from "./common";

export const issueSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchIssue', lng),
    containers: [
        votesContainer,
        bookmarksContainer,
        languagesContainer,
    ],
    fields: [
        ...votesFields,
        ...bookmarksFields,
        ...languagesFields,
    ]
})

export const issueSearchParams = (lng: string) => toParams(issueSearchSchema(lng), issueFindMany, IssueSortBy, IssueSortBy.ScoreDesc);