import { StandardSortBy } from "@shared/consts";
import { standardFindMany } from "api/generated/endpoints/standard";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const standardSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStandard', lng),
    containers: [
        votesContainer,
        bookmarksContainer,
        languagesContainer,
        tagsContainer,
    ],
    fields: [
        ...votesFields,
        ...bookmarksFields,
        ...languagesFields,
        ...tagsFields,
    ]
})

export const standardSearchParams = (lng: string) => toParams(standardSearchSchema(lng), standardFindMany, StandardSortBy, StandardSortBy.ScoreDesc);