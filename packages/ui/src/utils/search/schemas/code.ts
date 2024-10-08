import { CodeSortBy, FormSchema, endpointGetCode, endpointGetCodes } from "@local/shared";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

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
    return toParams(codeSearchSchema(), endpointGetCodes, endpointGetCode, CodeSortBy, CodeSortBy.ScoreDesc);
}

