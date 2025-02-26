import { CodeSortBy, FormSchema, endpointsCode } from "@local/shared";
import { toParams } from "./base.js";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common.js";

export function codeSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchCode"),
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

export function codeSearchParams() {
    return toParams(codeSearchSchema(), endpointsCode, CodeSortBy, CodeSortBy.ScoreDesc);
}

