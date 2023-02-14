import { StandardSortBy } from "@shared/consts";
import { standardFindMany } from "api/generated/endpoints/standard";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, votesContainer, votesFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields } from "./common";

export const standardSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStandard', lng),
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

export const standardSearchParams = (lng: string) => toParams(standardSearchSchema(lng), standardFindMany, StandardSortBy, StandardSortBy.ScoreDesc);