import { endpointGetNoteVersion, endpointGetNoteVersions, FormSchema, NoteVersionSortBy } from "@local/shared";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const noteVersionSearchSchema = (): FormSchema => ({
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
});

export const noteVersionSearchParams = () => toParams(noteVersionSearchSchema(), endpointGetNoteVersions, endpointGetNoteVersion, NoteVersionSortBy, NoteVersionSortBy.DateCreatedDesc);
