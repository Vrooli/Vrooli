import { RoutineSortBy } from "@shared/consts";
import { routineFindMany } from "api/generated/endpoints/routine";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { complexityContainer, complexityFields, isCompleteContainer, isCompleteFields, languagesContainer, languagesFields, searchFormLayout, simplicityContainer, simplicityFields, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const routineSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchRoutine', lng),
    containers: [
        isCompleteContainer,
        votesContainer,
        bookmarksContainer,
        simplicityContainer,
        complexityContainer,
        languagesContainer,
        tagsContainer,
    ],
    fields: [
        ...isCompleteFields,
        ...votesFields,
        ...bookmarksFields,
        ...simplicityFields,
        ...complexityFields,
        ...languagesFields,
        ...tagsFields,
    ]
})

export const routineSearchParams = (lng: string) => toParams(routineSearchSchema(lng), routineFindMany, RoutineSortBy, RoutineSortBy.ScoreDesc)