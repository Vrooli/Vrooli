import { NoteVersionSortBy } from "@shared/consts";
import { noteVersionFindMany } from "api/generated/endpoints/noteVersion";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const noteVersionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchNoteVersions', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const noteVersionSearchParams = (lng: string) => toParams(noteVersionSearchSchema(lng), noteVersionFindMany, NoteVersionSortBy, NoteVersionSortBy.DateCreatedDesc);