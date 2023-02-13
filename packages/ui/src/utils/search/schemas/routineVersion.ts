import { RoutineVersionSortBy } from "@shared/consts";
import { routineVersionFindMany } from "api/generated/endpoints/routineVersion";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const routineVersionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchRoutineVersion', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const routineVersionSearchParams = (lng: string) => toParams(routineVersionSearchSchema(lng), routineVersionFindMany, RoutineVersionSortBy, RoutineVersionSortBy.DateCreatedDesc);