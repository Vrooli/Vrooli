import { RoutineSortBy } from "@shared/consts";
import { routineFindMany } from "api/generated/endpoints/routine_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { hasCompleteVersionContainer, hasCompleteVersionFields, searchFormLayout, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, votesContainer, votesFields, languagesVersionContainer, languagesVersionFields } from "./common";

export const routineSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchRoutine'),
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

export const routineSearchParams = () => toParams(routineSearchSchema(), routineFindMany, RoutineSortBy, RoutineSortBy.ScoreDesc)