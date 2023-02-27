import { StandardSortBy } from "@shared/consts";
import { standardFindMany } from "api/generated/endpoints/standard_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, votesContainer, votesFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields } from "./common";

export const standardSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchStandard'),
    containers: [
        hasCompleteVersionContainer,
        votesContainer(),
        bookmarksContainer(),
        languagesVersionContainer(),
        tagsContainer(),
    ],
    fields: [
        ...hasCompleteVersionFields(),
        ...votesFields(),
        ...bookmarksFields(),
        ...languagesVersionFields(),
        ...tagsFields(),
    ]
})

export const standardSearchParams = () => toParams(standardSearchSchema(), standardFindMany, StandardSortBy, StandardSortBy.ScoreDesc);