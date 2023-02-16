import { NoteSortBy } from "@shared/consts";
import { noteFindMany } from "api/generated/endpoints/note";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const noteSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchNote', lng),
    containers: [
        votesContainer,
        bookmarksContainer,
        languagesVersionContainer,
        tagsContainer,
    ],
    fields: [
        ...votesFields,
        ...bookmarksFields,
        ...languagesVersionFields,
        ...tagsFields,
    ]
})

export const noteSearchParams = (lng: string) => toParams(noteSearchSchema(lng), noteFindMany, NoteSortBy, NoteSortBy.ScoreDesc);