import { StandardSortBy } from "@shared/consts";
import { standardFindMany } from "api/generated/endpoints/standard";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, votesContainer, votesFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields } from "./common";

export const standardSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStandard', lng),
    containers: [
        hasCompleteVersionContainer,
        votesContainer(lng),
        bookmarksContainer(lng),
        languagesVersionContainer(lng),
        tagsContainer(lng),
    ],
    fields: [
        ...hasCompleteVersionFields(lng),
        ...votesFields(lng),
        ...bookmarksFields(lng),
        ...languagesVersionFields(lng),
        ...tagsFields(lng),
    ]
})

export const standardSearchParams = (lng: string) => toParams(standardSearchSchema(lng), standardFindMany, StandardSortBy, StandardSortBy.ScoreDesc);