import { endpointGetNote, endpointGetNotes, FormSchema, NoteSortBy } from "@local/shared";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

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
    return toParams(noteSearchSchema(), endpointGetNotes, endpointGetNote, NoteSortBy, NoteSortBy.ScoreDesc);
}

