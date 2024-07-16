import { ApiSortBy, endpointGetApi, endpointGetApis } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

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
    return toParams(apiSearchSchema(), endpointGetApis, endpointGetApi, ApiSortBy, ApiSortBy.ScoreDesc);
}

