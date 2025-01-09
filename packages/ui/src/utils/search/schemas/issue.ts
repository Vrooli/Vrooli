import { endpointsIssue, FormSchema, IssueSortBy } from "@local/shared";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, votesContainer, votesFields } from "./common";

export function issueSearchSchema(): FormSchema {
    return {
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
    };
}

export function issueSearchParams() {
    return toParams(issueSearchSchema(), endpointsIssue, IssueSortBy, IssueSortBy.ScoreDesc);
}
