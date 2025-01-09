import { endpointsProject, FormSchema, ProjectSortBy } from "@local/shared";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export function projectSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchProject"),
        containers: [
            hasCompleteVersionContainer,
            votesContainer(),
            bookmarksContainer(),
            languagesVersionContainer(),
            tagsContainer(),
        ],
        elements: [
            ...hasCompleteVersionFields(),
            ...votesFields(),
            ...bookmarksFields(),
            ...languagesVersionFields(),
            ...tagsFields(),
        ],
    };
}

export function projectSearchParams() {
    return toParams(projectSearchSchema(), endpointsProject, ProjectSortBy, ProjectSortBy.ScoreDesc);
}
