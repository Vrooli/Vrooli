import { IssueSortBy, endpointsIssue, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, votesContainer, votesFields } from "./common.js";

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
