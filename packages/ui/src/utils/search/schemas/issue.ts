import { endpointGetIssue, endpointGetIssues, IssueSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, votesContainer, votesFields } from "./common";

export const issueSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchIssue"),
    containers: [
        votesContainer(),
        bookmarksContainer(),
        languagesContainer(),
    ],
    elements: [
        ...votesFields(),
        ...bookmarksFields(),
        ...languagesFields(),
    ],
});

export const issueSearchParams = () => toParams(issueSearchSchema(), endpointGetIssues, endpointGetIssue, IssueSortBy, IssueSortBy.ScoreDesc);
