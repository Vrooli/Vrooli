import { NoteVersionSortBy } from "@shared/consts";
import { FormSchema } from "forms/types";
import { noteVersionFindMany } from "../../../api/generated/endpoints/noteVersion_findMany";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const noteVersionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchNoteVersion"),
    containers: [
        isLatestContainer,
        votesRootContainer(),
        bookmarksRootContainer(),
        languagesContainer(),
        tagsRootContainer(),
    ],
    fields: [
        ...isLatestFields(),
        ...votesRootFields(),
        ...bookmarksRootFields(),
        ...languagesFields(),
        ...tagsRootFields(),
    ]
})

export const noteVersionSearchParams = () => toParams(noteVersionSearchSchema(), noteVersionFindMany, NoteVersionSortBy, NoteVersionSortBy.DateCreatedDesc);