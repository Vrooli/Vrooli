import { ApiSortBy } from "@shared/consts";
import { apiFindMany } from "api/generated/endpoints/api";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const apiSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchApi', lng),
    containers: [
        hasCompleteVersionContainer,
        votesContainer,
        bookmarksContainer,
        languagesVersionContainer,
        tagsContainer,
    ],
    fields: [
        ...hasCompleteVersionFields,
        ...votesFields,
        ...bookmarksFields,
        ...languagesVersionFields,
        ...tagsFields,
    ]
})

export const apiSearchParams = (lng: string) => toParams(apiSearchSchema(lng), apiFindMany, ApiSortBy, ApiSortBy.ScoreDesc);