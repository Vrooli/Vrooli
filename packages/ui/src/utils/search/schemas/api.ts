import { ApiSortBy, FormSchema, endpointsApi } from "@local/shared";
import { toParams } from "./base.js";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common.js";

export function apiSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchApi"),
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

export function apiSearchParams() {
    return toParams(apiSearchSchema(), endpointsApi, ApiSortBy, ApiSortBy.ScoreDesc);
}

