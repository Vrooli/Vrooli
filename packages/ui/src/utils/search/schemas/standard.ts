import { endpointsStandard, FormSchema, StandardSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common.js";

export function standardSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchStandard"),
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

export function standardSearchParams() {
    return toParams(standardSearchSchema(), endpointsStandard, StandardSortBy, StandardSortBy.ScoreDesc);
}
