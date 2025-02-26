import { endpointsNote, FormSchema, NoteSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { bookmarksContainer, bookmarksFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common.js";

export function noteSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchNote"),
        containers: [
            votesContainer(),
            bookmarksContainer(),
            languagesVersionContainer(),
            tagsContainer(),
        ],
        elements: [
            ...votesFields(),
            ...bookmarksFields(),
            ...languagesVersionFields(),
            ...tagsFields(),
        ],
    };
}

export function noteSearchParams() {
    return toParams(noteSearchSchema(), endpointsNote, NoteSortBy, NoteSortBy.ScoreDesc);
}

