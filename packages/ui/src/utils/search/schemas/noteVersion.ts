import { NoteVersionSortBy } from "@shared/consts";
import { noteVersionFindMany } from "api/generated/endpoints/noteVersion";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const noteVersionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchNoteVersion', lng),
    containers: [
        isLatestContainer,
        votesRootContainer,
        bookmarksRootContainer,
        languagesContainer,
        tagsRootContainer,
    ],
    fields: [
        ...isLatestFields,
        ...votesRootFields,
        ...bookmarksRootFields,
        ...languagesFields,
        ...tagsRootFields,
    ]
})

export const noteVersionSearchParams = (lng: string) => toParams(noteVersionSearchSchema(lng), noteVersionFindMany, NoteVersionSortBy, NoteVersionSortBy.DateCreatedDesc);