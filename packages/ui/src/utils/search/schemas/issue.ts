import { IssueSortBy } from "@shared/consts";
import { issueFindMany } from "api/generated/endpoints/issue";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, votesContainer, votesFields } from "./common";

export const issueSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchIssue'),
    containers: [
        votesContainer(),
        bookmarksContainer(),
        languagesContainer(),
    ],
    fields: [
        ...votesFields(),
        ...bookmarksFields(),
        ...languagesFields(),
    ]
})

export const issueSearchParams = () => toParams(issueSearchSchema(), issueFindMany, IssueSortBy, IssueSortBy.ScoreDesc);