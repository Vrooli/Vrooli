import { RoutineSortBy } from "@shared/consts";
import { routineFindMany } from "api/generated/endpoints/routine";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { hasCompleteVersionContainer, hasCompleteVersionFields, searchFormLayout, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, votesContainer, votesFields, languagesVersionContainer, languagesVersionFields } from "./common";

export const routineSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchRoutine', lng),
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

export const routineSearchParams = (lng: string) => toParams(routineSearchSchema(lng), routineFindMany, RoutineSortBy, RoutineSortBy.ScoreDesc)