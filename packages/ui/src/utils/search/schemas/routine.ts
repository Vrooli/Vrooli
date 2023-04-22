import { RoutineSortBy } from "@shared/consts";
import { FormSchema } from "forms/types";
import { routineFindMany } from "../../../api/generated/endpoints/routine_findMany";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const routineSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchRoutine"),
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