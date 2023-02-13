import { RoutineSortBy } from "@shared/consts";
import { routineFindMany } from "api/generated/endpoints/routine";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { complexityContainer, complexityFields, isCompleteContainer, isCompleteFields, languagesContainer, languagesFields, searchFormLayout, simplicityContainer, simplicityFields, starsContainer, starsFields, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const routineSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchRoutine', lng),
    containers: [
        isCompleteContainer,
        votesContainer,
        starsContainer,
        simplicityContainer,
        complexityContainer,
        languagesContainer,
        tagsContainer,
    ],
    fields: [
        ...isCompleteFields,
        ...votesFields,
        ...starsFields,
        ...simplicityFields,
        ...complexityFields,
        ...languagesFields,
        ...tagsFields,
    ]
})

export const routineSearchParams = (lng: string) => toParams(routineSearchSchema(lng), routineFindMany, RoutineSortBy, RoutineSortBy.ScoreDesc)