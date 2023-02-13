import { StandardSortBy } from "@shared/consts";
import { standardFindMany } from "api/generated/endpoints/standard";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout, starsContainer, starsFields, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const standardSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStandard', lng),
    containers: [
        votesContainer,
        starsContainer,
        languagesContainer,
        tagsContainer,
    ],
    fields: [
        ...votesFields,
        ...starsFields,
        ...languagesFields,
        ...tagsFields,
    ]
})

export const standardSearchParams = (lng: string) => toParams(standardSearchSchema(lng), standardFindMany, StandardSortBy, StandardSortBy.ScoreDesc);