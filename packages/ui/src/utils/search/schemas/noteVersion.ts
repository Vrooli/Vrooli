import { endpointsNoteVersion, FormSchema, NoteVersionSortBy } from "@local/shared";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export function noteVersionSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchNoteVersion"),
        containers: [
            isLatestContainer,
            votesRootContainer(),
            bookmarksRootContainer(),
            languagesContainer(),
            tagsRootContainer(),
        ],
        elements: [
            ...isLatestFields(),
            ...votesRootFields(),
            ...bookmarksRootFields(),
            ...languagesFields(),
            ...tagsRootFields(),
        ],
    };
}

export function noteVersionSearchParams() {
    return toParams(noteVersionSearchSchema(), endpointsNoteVersion, NoteVersionSortBy, NoteVersionSortBy.DateCreatedDesc);
}
